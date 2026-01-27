package taskMgr

import (
	"fmt"
	"multi-agent/shared"
	"strings"
)

type TaskItem struct {
	ID          int
	Completed   bool
	Goal        string
	AnswerSpec  string
	ContextSpec string

	Answer  string
	Context []shared.ContextItem
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
	PreTasks    []TaskItem
	CurrentTask *TaskItem
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

func (mgr *TaskMgr) finishTask(anser string, contexts []shared.ContextItem) error {
	if mgr.CurrentTask == nil {
		return fmt.Errorf("There is no current task, can not finish task")
	}

	mgr.CurrentTask.Completed = true
	mgr.CurrentTask.Answer = anser
	mgr.CurrentTask.Context = contexts
	mgr.PreTasks = append(mgr.PreTasks, *mgr.CurrentTask)

	mgr.CurrentTask = nil
	return nil
}

func (mgr *TaskMgr) getTaskContextPrompt() string {
	var builder strings.Builder
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
