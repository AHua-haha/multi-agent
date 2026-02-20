package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type TaskItem struct {
	ID            int
	Completed     bool
	Goal          string
	ConclusionReq string
	ContextReq    string

	Conclusion string
	Context    map[int]*ContextItem
}

type ContextItem struct {
	ID      int
	Desc    string
	ToolLog *ToolExecLog
}

func (item *TaskItem) formatString() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Task %d: %s\n", item.ID, item.Goal))
	if !item.Completed {
		builder.WriteString(fmt.Sprintf("Conclusions Requirements:\n%s\nBackground Context Requirements:\n%s\n", item.ConclusionReq, item.ContextReq))
	} else {
		if item.Conclusion != "" {
			builder.WriteString(fmt.Sprintf("Conclusions:\n%s\n", item.Conclusion))
		}
		if len(item.Context) != 0 {
			builder.WriteString("Retrievaled Context: ")
			for _, ctxItem := range item.Context {
				builder.WriteString(fmt.Sprintf("#%d: %s, ", ctxItem.ID, ctxItem.Desc))
			}
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

type TaskMgr struct {
	UserGoal    string
	PreTasks    []*TaskItem
	CurrentTask *TaskItem
}

func (mgr *TaskMgr) Reset(userGoal string) {
	mgr.UserGoal = userGoal
	mgr.PreTasks = nil
	mgr.CurrentTask = nil
}
func (mgr *TaskMgr) FillToolLog(toolLog []*ToolExecLog) {
	task := mgr.PreTasks[len(mgr.PreTasks)-1]
	for i := range task.Context {
		task.Context[i].ToolLog = toolLog[task.Context[i].ID]
	}
}

func (mgr *TaskMgr) refineContext(oldID int, newID int) error {
	task := mgr.PreTasks[len(mgr.PreTasks)-1]
	item, exist := task.Context[oldID]
	if !exist {
		return fmt.Errorf("contex ID %d not found", oldID)
	}
	item.ID = newID
	return nil
}

func (mgr *TaskMgr) createTask(goal string, answerSpec string, contextSpec string) error {
	if mgr.CurrentTask != nil {
		return fmt.Errorf("Current Task %s not finished, can not create new task", mgr.CurrentTask.Goal)
	}
	mgr.CurrentTask = &TaskItem{
		ID:            len(mgr.PreTasks) + 1,
		Completed:     false,
		Goal:          goal,
		ConclusionReq: answerSpec,
		ContextReq:    contextSpec,
		Context:       map[int]*ContextItem{},
	}
	return nil
}

func (mgr *TaskMgr) finishTask(answer string, contexts []ContextItem) error {
	if mgr.CurrentTask == nil {
		return fmt.Errorf("There is no current task, can not finish task")
	}

	mgr.CurrentTask.Completed = true
	mgr.CurrentTask.Conclusion = answer
	for i, ctx := range contexts {
		mgr.CurrentTask.Context[i] = &ctx
	}
	mgr.PreTasks = append(mgr.PreTasks, mgr.CurrentTask)

	mgr.CurrentTask = nil
	return nil
}
func (mgr *TaskMgr) GetInputForRefineContext() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("** USER PRIMARY GOAL **: %s\n", mgr.UserGoal))
	builder.WriteString("### TASK HISTORY\n")
	length := len(mgr.PreTasks)
	if len(mgr.PreTasks) != 0 {
		builder.WriteString("** Completed Tasks **\n")
		for _, task := range mgr.PreTasks[:length-1] {
			builder.WriteString(task.formatString())
		}
		builder.WriteByte('\n')
	}

	lastTask := mgr.PreTasks[length-1]
	builder.WriteString(fmt.Sprintf("Task %d: %s\n", lastTask.ID, lastTask.Goal))
	builder.WriteString(fmt.Sprintf("Conclusions Requirements:\n%s\nBackground Context Requirements:\n%s\n", lastTask.ConclusionReq, lastTask.ContextReq))
	if lastTask.Conclusion != "" {
		builder.WriteString(fmt.Sprintf("Conclusions:\n%s\n", lastTask.Conclusion))
	}

	builder.WriteString("Refine the following tool call Context\n")

	for _, ctx := range lastTask.Context {
		builder.WriteString(fmt.Sprintf("Context ID: %d, Description: %s\n", ctx.ID, ctx.Desc))
		builder.WriteString("<Tool Call>\n")
		builder.WriteString(fmt.Sprintf("Tool Call Name: %s, Tool Call Args: %s\n", ctx.ToolLog.ToolCallName, ctx.ToolLog.ToolCallArgs))
		builder.WriteString(fmt.Sprintf("** Result **\n%s", ctx.ToolLog.ToolCallRes))
		builder.WriteString("</Tool Call>\n")
	}

	builder.WriteByte('\n')
	return builder.String()
}

func (mgr *TaskMgr) GetTaskContextPrompt() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("** USER PRIMARY GOAL **: %s\n", mgr.UserGoal))
	builder.WriteString("### TASK HISTORY\n")
	if len(mgr.PreTasks) != 0 {
		builder.WriteString("** Completed Tasks **\n")
		for _, task := range mgr.PreTasks {
			builder.WriteString(task.formatString())
		}
		builder.WriteByte('\n')
	}

	if mgr.CurrentTask == nil {
		builder.WriteString("NO TASK IN PROGRESS\n")
	} else {
		builder.WriteString("Focus MAINLY on this 'Current Task', accomplish the 'Current Task', make best effort to meet the 'Conclusion Requirements' and 'Backgournd Context Requirements'\n")
		builder.WriteString("** Current Tasks **\n")
		builder.WriteString(mgr.CurrentTask.formatString())
	}
	builder.WriteByte('\n')
	return builder.String()
}

type CreateTaskArgs struct {
	Goal          string
	ConclusionReq string
	ContextReq    string
}
type FinishTaskArgs struct {
	Conclusion string
	Context    []ContextItem
}
type RefineContextArgs struct {
	OldID int
	NewID int
}

func (mgr *TaskMgr) CreateTaskTool() ToolEndPoint {
	def := openai.FunctionDefinition{
		Name:        "create_task",
		Description: "Defines a structured task by outlining the objective, the specific instruction about target conclusion to extract, and the background context to observe and record.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Goal": {
					Type:        jsonschema.String,
					Description: "The high-level objective of the task.",
				},
				"ConclusionReq": {
					Type:        jsonschema.String,
					Description: "Specific requirements on what direct facts to extract as the output of the task, can be empty if do not need",
				},
				"ContextReq": {
					Type:        jsonschema.String,
					Description: "Specific requirements on what background context to observe and record. can be empty if do not need",
				},
			},
			Required: []string{"Goal", "ConclusionReq", "ContextReq"},
		},
	}
	Handler := func(args string) (string, error) {
		var para CreateTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		err = mgr.createTask(para.Goal, para.ConclusionReq, para.ContextReq)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	endpoint := ToolEndPoint{
		Name:    "create_task",
		Def:     def,
		Handler: Handler,
	}
	return endpoint
}

func (mgr *TaskMgr) FinishTaskTool() ToolEndPoint {
	def := openai.FunctionDefinition{
		Name:        "finish_task",
		Description: "Finish the current in progress task with the answer and context infomation",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Conclusion": {
					Type:        jsonschema.String,
					Description: "The short and concise conclusions and facts required by the task conclusion requirements.",
				},
				"Context": {
					Type:        jsonschema.Array,
					Description: "A list of the tool execute result, which is the output of the task context requirements",
					Items: &jsonschema.Definition{
						Type: jsonschema.Object,
						Properties: map[string]jsonschema.Definition{
							"ID": {
								Type:        jsonschema.Integer,
								Description: "the id of the tool log",
							},
							"Desc": {
								Type:        jsonschema.String,
								Description: "the description of this background context",
							},
						},
					},
				},
			},
			Required: []string{"Conclusion", "Context"},
		},
	}
	Handler := func(args string) (string, error) {
		var para FinishTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		err = mgr.finishTask(para.Conclusion, para.Context)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	endpoint := ToolEndPoint{
		Name:    "finish_task",
		Def:     def,
		Handler: Handler,
	}
	return endpoint
}
func (mgr *TaskMgr) RefineContextTool() ToolEndPoint {
	def := openai.FunctionDefinition{
		Name:        "refine_context",
		Description: "replace the context with the refined context",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"OldID": {
					Type:        jsonschema.Integer,
					Description: "the old id of the context",
				},
				"NewID": {
					Type:        jsonschema.Integer,
					Description: "the new id of the context",
				},
			},
			Required: []string{"OldD", "NewID"},
		},
	}
	Handler := func(args string) (string, error) {
		var para RefineContextArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		err = mgr.refineContext(para.OldID, para.NewID)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	endpoint := ToolEndPoint{
		Name:    "refine_context",
		Def:     def,
		Handler: Handler,
	}
	return endpoint
}
