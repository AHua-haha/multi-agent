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

func (item *TaskItem) GetTask() string {
	return item.Goal
}

type ContextItem struct {
	ID      int
	Desc    string
	ToolLog *ToolExecLog
}

func (item *TaskItem) FormatString() string {
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
	PreTasks    []Task
	CurrentTask Task
}

func (mgr *TaskMgr) Reset(userGoal string) {
	mgr.UserGoal = userGoal
	mgr.PreTasks = nil
	mgr.CurrentTask = nil
}

// func (mgr *TaskMgr) FillToolLog(toolLog []*ToolExecLog) {
// 	task := mgr.PreTasks[len(mgr.PreTasks)-1]
// 	for i := range task.Context {
// 		task.Context[i].ToolLog = toolLog[task.Context[i].ID]
// 	}
// }

func (mgr *TaskMgr) refineContext(oldID int, newID int) error {
	// task := mgr.PreTasks[len(mgr.PreTasks)-1]
	// item, exist := task.Context[oldID]
	// if !exist {
	// 	return fmt.Errorf("contex ID %d not found", oldID)
	// }
	// item.ID = newID
	return nil
}

func (mgr *TaskMgr) createTask(task Task) error {
	if mgr.CurrentTask != nil {
		return fmt.Errorf("Current Task %s not finished, can not create new task", task.GetTask())
	}
	mgr.CurrentTask = task
	return nil
}

func (mgr *TaskMgr) finishTask() error {
	if mgr.CurrentTask == nil {
		return fmt.Errorf("There is no current task, can not finish task")
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
			builder.WriteString(task.FormatString())
		}
		builder.WriteByte('\n')
	}

	lastTask := mgr.PreTasks[length-1]
	builder.WriteString(lastTask.FormatString())
	builder.WriteString("Refine the context from the last task above\n")

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
			builder.WriteString(task.FormatString())
		}
		builder.WriteByte('\n')
	}

	if mgr.CurrentTask == nil {
		builder.WriteString("NO TASK IN PROGRESS\n")
	} else {
		builder.WriteString("Focus MAINLY on this 'Current Task', accomplish the 'Current Task'\n")
		builder.WriteString("** Current Tasks **\n")
		builder.WriteString(mgr.CurrentTask.FormatString())
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
type CreateExploreTaskArgs struct {
	Task         string
	ExpectOutput string
}

func (mgr *TaskMgr) CreateExploreTaskTool() ToolEndPoint {
	endpoint := CreateExploreTask()
	endpoint.Handler = func(args string) (string, error) {
		var para CreateExploreTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		task := &ExploreTask{
			Task:         para.Task,
			ExpectOutput: para.ExpectOutput,
			Context:      []ContextItem{},
		}
		err = mgr.createTask(task)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return endpoint
}
type CreateReasonTaskArgs struct {
	Task         string
	ExpectOutput string
}

func (mgr *TaskMgr) CreateReasonTaskTool() ToolEndPoint {
	endpoint := CreateReasonTask()
	endpoint.Handler = func(args string) (string, error) {
		var para CreateReasonTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		task := &ReasonTask{
			Task:         para.Task,
			ExpectOutput: para.ExpectOutput,
			Conclusion:   "",
		}
		err = mgr.createTask(task)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return endpoint
}
type CreateBuildTaskArgs struct {
	Task string
}

func (mgr *TaskMgr) CreateBuildTaskTool() ToolEndPoint {
	endpoint := CreateBuildTask()
	endpoint.Handler = func(args string) (string, error) {
		var para CreateBuildTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		task := &BuildTask{
			Task:      para.Task,
			ChangeLog: "",
			Context:   []ContextItem{},
		}
		err = mgr.createTask(task)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return endpoint
}
type CreateVerifyTaskArgs struct {
	Task string
}

func (mgr *TaskMgr) CreateVerifyTaskTool() ToolEndPoint {
	endpoint := CreateVerifyTask()
	endpoint.Handler = func(args string) (string, error) {
		var para CreateVerifyTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		task := &VerifyTask{
			Task:       para.Task,
			Conclusion: "",
			Context:    []ContextItem{},
		}
		err = mgr.createTask(task)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return endpoint
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

func (mgr *TaskMgr) FinishExploreTaskTool() ToolEndPoint {
	endpoint := FinishExploreTask()
	endpoint.Handler = func(args string) (string, error) {
		var para FinishExploreTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		// Cast to ExploreTask to update Context
		if exploreTask, ok := mgr.CurrentTask.(*ExploreTask); ok {
			exploreTask.Context = para.Context
		}
		err = mgr.finishTask()
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return endpoint
}

func (mgr *TaskMgr) FinishReasonTaskTool() ToolEndPoint {
	endpoint := FinishReasonTask()
	endpoint.Handler = func(args string) (string, error) {
		var para FinishReasonTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		// Cast to ReasonTask to update Conclusion
		if reasonTask, ok := mgr.CurrentTask.(*ReasonTask); ok {
			reasonTask.Conclusion = para.Conclusion
		}
		err = mgr.finishTask()
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return endpoint
}

func (mgr *TaskMgr) FinishBuildTaskTool() ToolEndPoint {
	endpoint := FinishBuildTask()
	endpoint.Handler = func(args string) (string, error) {
		var para FinishBuildTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		// Cast to BuildTask to update ChangeLog and Context
		if buildTask, ok := mgr.CurrentTask.(*BuildTask); ok {
			buildTask.ChangeLog = para.ChangeLog
			buildTask.Context = para.Context
		}
		err = mgr.finishTask()
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return endpoint
}

func (mgr *TaskMgr) FinishVerifyTaskTool() ToolEndPoint {
	endpoint := FinishVerifyTask()
	endpoint.Handler = func(args string) (string, error) {
		var para FinishVerifyTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		// Cast to VerifyTask to update Conclusion and Context
		if verifyTask, ok := mgr.CurrentTask.(*VerifyTask); ok {
			verifyTask.Conclusion = para.Conclusion
			verifyTask.Context = para.Context
		}
		err = mgr.finishTask()
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return endpoint
}
