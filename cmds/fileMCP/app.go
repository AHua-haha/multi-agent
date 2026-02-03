package main

import (
	mcpserver "multi-agent/mcp-server"
	_ "multi-agent/shared"

	"github.com/rs/zerolog/log"
)

func main() {
	s, err := mcpserver.NewServer("/root/multi-agent")
	if err != nil {
		log.Error().Err(err).Msg("Create server failed")
		return
	}
	err = s.Run()
	if err != nil {
		log.Error().Err(err).Msg("Run server failed")
		return
	}
	log.Info().Msg("Run server success")
}
