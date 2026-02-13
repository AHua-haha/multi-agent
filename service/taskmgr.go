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
	Context    []ContextItem
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
	}
	return nil
}

func (mgr *TaskMgr) finishTask(answer string, contexts []ContextItem) error {
	if mgr.CurrentTask == nil {
		return fmt.Errorf("There is no current task, can not finish task")
	}

	mgr.CurrentTask.Completed = true
	mgr.CurrentTask.Conclusion = answer
	mgr.CurrentTask.Context = contexts
	mgr.PreTasks = append(mgr.PreTasks, mgr.CurrentTask)

	mgr.CurrentTask = nil
	return nil
}

var DoingTaskInstruct = `
### INSTRUCTION
You are now working on a sub task. Focus on the currect sub task, use the available tools to accomplist the current sub task, after you accomplish the currect task, you MUST explicitly finish the task with the output IMMEDIATELY.
Focus on the 'Conclusion Requirements' and the 'Background Context Requirements', make best effort to meet these requirements.

IMPORTANT: after you accomplish the current sub task, MUST IMMEDIATELY use the 'finish_task' tool to record the output of this current task.
IMPORTANT: do not continue the User Primary Goal unless you call 'finish_task' tool to finish the current task.
`

var CreateTaskInstruct = `
Analyze the 'Task History' against the 'User Primary Goal'. You must choose one of two paths:

#### PATH A: GOAL NOT COMPLETED (DECOMPOSE & CREATE TASK)
If information is missing or a multi-step process is still underway:
1. **Decompose**: Identify the immediate next logical task.
2. **Create Atomic Task**: Define a single, focused task, the task should have only one goal, the task MUST be atomic, NEVER create too complex task with multiple goals.
3. **Structured Expected Output Requirements**: Define the expected "Primary Conclusions" and "Background Context" for the task.
   - NEVER define too complex 'Expected Output', the 'Expected Output' should be simple and focused, 2-3 most essential items is the best.
   - ** Conclusions Requirements **: These are the direct facts and conclusions to extract as the output of the task.
   - ** Background Context Requirements **: background context to observe and record,

#### PATH B: GOAL COMPLETED (FINALIZE)
If all the 'Task History' provide a full answer:
1. **Synthesize**: Combine all facts into a coherent response.
2. **Nuance**: Add relevant "Background Context" to provide helpful background or warnings.
3. **Finalize**: Deliver the final response to the user directly without using the 'finish_task' tool.

IMPORTANT: never assign a task that is too complex and have multiple goals, decompose complex goals into the smallest possible units of task.
IMPORTANT: The 'Eexpected Ooutput Requirements' must be highly focused. Do not ask for "everything." Ask for the 1-2 most critical facts.

`

func (mgr *TaskMgr) GetTaskContextPrompt() string {
	intro := `
Below is the 'Task History' with the result of each task. The result of each task includes conclusion and context information.
`
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("** USER PRIMARY GOAL **: %s\n", mgr.UserGoal))
	builder.WriteString(intro)
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
