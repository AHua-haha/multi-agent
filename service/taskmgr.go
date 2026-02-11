package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type TaskItem struct {
	ID          int
	Completed   bool
	Goal        string
	AnswerSpec  string
	ContextSpec string

	Answer  string
	Context []ContextItem
}

type ContextItem struct {
	ID   int
	Desc string
}

func (item *TaskItem) formatString() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Task %d: %s\n", item.ID, item.Goal))
	if !item.Completed {
		builder.WriteString(fmt.Sprintf("Answer: %s\nContext: %s\n", item.AnswerSpec, item.ContextSpec))
	} else {
		if item.Answer != "" {
			builder.WriteString(fmt.Sprintf("Answer: %s\n", item.Answer))
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
	PreTasks    []TaskItem
	CurrentTask *TaskItem
}

func (mgr *TaskMgr) Reset(userGoal string) {
	mgr.UserGoal = userGoal
	mgr.PreTasks = nil
	mgr.CurrentTask = nil
}

func (mgr *TaskMgr) createTask(goal string, answerSpec string, contextSpec string) error {
	if mgr.CurrentTask != nil {
		return fmt.Errorf("Current Task %s not finished, can not create new task", mgr.CurrentTask.Goal)
	}
	mgr.CurrentTask = &TaskItem{
		ID:          len(mgr.PreTasks) + 1,
		Completed:   false,
		Goal:        goal,
		AnswerSpec:  answerSpec,
		ContextSpec: contextSpec,
	}
	return nil
}

func (mgr *TaskMgr) finishTask(answer string, contexts []ContextItem) error {
	if mgr.CurrentTask == nil {
		return fmt.Errorf("There is no current task, can not finish task")
	}

	mgr.CurrentTask.Completed = true
	mgr.CurrentTask.Answer = answer
	mgr.CurrentTask.Context = contexts
	mgr.PreTasks = append(mgr.PreTasks, *mgr.CurrentTask)

	mgr.CurrentTask = nil
	return nil
}

var DoingTaskInstruct = `
### INSTRUCTION
You are now working on a sub task. Focus on the currect sub task, use the available tools to accomplist the current sub task, after you accomplish the currect task, you MUST explicitly finish the task with the output IMMEDIATELY.

IMPORTANT: after you accomplish the current sub task, MUST IMMEDIATELY use the 'finish_task' tool to record the output of this current task.
IMPORTANT: do not continue the User Primary Goal unless you call 'finish_task' tool to finish the current task.
`

var CreateTaskInstruct = `
### INSTRUCTION
You are a reasoning agent that operates through Atomic Task Decomposition. You must NEVER attempt to solve a complex goal in a single step. Instead, you analyze the TASK_HISTORY, identify the immediate gap in knowledge, and define the smallest possible next step
Every task you generate must adhere to these constraints:
- ** Atomic Task **: each task should be atomic, each task should have only one goal, do not create a task with multiple goals, try to divide it to atomic task.
- ** Expected Output **: when create the atomic task, you MUST explicitly define the Expected Output. You can define two kinds of output, Answer and Context.
- ** Answer **: the task output can be short and concise facts and conclusions.
- ** Context **: the task output can be background context information like file content.

Now based on the User Primary Goal and Task History, create the smallest possible next task.
`

func (mgr *TaskMgr) GetTaskContextPrompt() string {
	intro := `
Below is the task history with the result of each task. The result of each task includes conclusion and context information.
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
	// if mgr.CurrentTask == nil {
	// 	builder.WriteString(CreateTaskInstruct)
	// } else {
	// 	builder.WriteString(DoingTaskInstruct)
	// }
	return builder.String()
}

type CreateTaskArgs struct {
	Goal        string
	AnswerSpec  string
	ContextSpec string
}
type FinishTaskArgs struct {
	Answer string
	// Context []ContextItem
}

func (mgr *TaskMgr) CreateTaskTool() ToolEndPoint {
	def := openai.FunctionDefinition{
		Name:        "create_task",
		Description: "Defines a structured task by outlining the objective, the specific target conclusion to be reached, and the background information that the agent want.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Goal": {
					Type:        jsonschema.String,
					Description: "The high-level objective of the task.",
				},
				"AnswerSpec": {
					Type:        jsonschema.String,
					Description: "A short and concise description of the specific target conclusion the agent is trying to find. Can be empty if agent only need the context infomation",
				},
				"ContextSpec": {
					Type:        jsonschema.String,
					Description: "A description of the background information and data points the agent needs. Can be empty if the agent only need the final conclusion",
				},
			},
			Required: []string{"Goal", "AnswerSpec", "ContextSpec"},
		},
	}
	Handler := func(args string) (string, error) {
		var para CreateTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		err = mgr.createTask(para.Goal, para.AnswerSpec, para.ContextSpec)
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
				"Answer": {
					Type:        jsonschema.String,
					Description: "The short and concise conclusions or answers which answer the AnswerSpec of the current in progress task.",
				},
				// "Context": {
				// 	Type:        jsonschema.String,
				// 	Description: "",
				// },
			},
			Required: []string{"Answer"},
		},
	}
	Handler := func(args string) (string, error) {
		var para FinishTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		err = mgr.finishTask(para.Answer, nil)
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
