package agent

import (
	"fmt"
	"testing"
)

func TestWorkflow_SingleAgent(t *testing.T) {
	t.Run("test agent init", func(t *testing.T) {
		var w Workflow
		err := w.Init()
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		agent := NewAgent(w.client)
		endpoints, err := w.mcpclient.LoadAllTools()
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		agent.AddTools(endpoints)
		agent.toolDispatch.DebugTools()
	})
}
