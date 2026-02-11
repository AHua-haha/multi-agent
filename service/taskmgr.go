package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type TaskItem struct {
	ID             int
	Completed      bool
	Goal           string
	ConclusionSpec string
	ContextSpec    string

	Conclusion string
	Context    []ContextItem
}

type ContextItem struct {
	ID   int
	Desc string
}

func (item *TaskItem) formatString() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Task %d: %s\n", item.ID, item.Goal))
	if !item.Completed {
		builder.WriteString(fmt.Sprintf("Conclusions Instruction: %s\nBackground Context Instruction: %s\n", item.ConclusionSpec, item.ContextSpec))
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
		ID:             len(mgr.PreTasks) + 1,
		Completed:      false,
		Goal:           goal,
		ConclusionSpec: answerSpec,
		ContextSpec:    contextSpec,
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
You are the **Task Orchestrator Agent**. Your goal is to translate high-level user objectives into precise, atomic, and structured task definitions for a Worker Agent.
You do not execute actions yourself. Instead, you generate a structured **Task Definition**.
The most critical part of your definition is the **Expected Output Structure**. You must strictly instruct the Worker Agent to report its findings in two distinct categories.

#### The "Expected Output" Definition
When defining what the task should return, you must mandate two specific distinct fields:
1. **Conclusion Instruction (Facts & Direct Results):** - These are the direct facts and conclusions to extract as the output of the task.
2. **Context Instruction (Background & Meta-Data):** - meta-data or environmental context to observe and record,

#### ** Task Decomposition **
- You must NEVER attempt to solve a complex goal in a single step. Instead, you analyze the TASK_HISTORY, identify the immediate gap in knowledge, and define the smallest possible next task.
- each task should be atomic, each task should have only one goal, do not create a task with multiple goals, try to divide it to atomic task.

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
	Goal           string
	ConclusionSpec string
	ContextSpec    string
}
type FinishTaskArgs struct {
	Conclusion string
	// Context []ContextItem
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
				"ConclusionSpec": {
					Type:        jsonschema.String,
					Description: "Specific instructions on what direct facts to extract as the output of the task, can be empty if do not need",
				},
				"ContextSpec": {
					Type:        jsonschema.String,
					Description: "Specific instructions on what meta-data or environmental context to observe and record. can be empty if do not need",
				},
			},
			Required: []string{"Goal", "ConclusionSpec", "ContextSpec"},
		},
	}
	Handler := func(args string) (string, error) {
		var para CreateTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		err = mgr.createTask(para.Goal, para.ConclusionSpec, para.ContextSpec)
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
					Description: "The short and concise conclusions and facts required by the task conclusion instruction.",
				},
				// "Context": {
				// 	Type:        jsonschema.String,
				// 	Description: "",
				// },
			},
			Required: []string{"Conclusion"},
		},
	}
	Handler := func(args string) (string, error) {
		var para FinishTaskArgs
		err := json.Unmarshal([]byte(args), &para)
		if err != nil {
			return "", err
		}
		err = mgr.finishTask(para.Conclusion, nil)
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
