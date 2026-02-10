package main

import (
	"multi-agent/agent"
	_ "multi-agent/shared"

	"github.com/rs/zerolog/log"
)

func main() {
	workflow := agent.Workflow{}
	err := workflow.Init()
	if err != nil {
		log.Error().Err(err).Msg("workflow init failed")
		return
	}
	err = workflow.SingleAgent()
	if err != nil {
		log.Error().Err(err).Msg("run single agent failed")
		return
	}
}
