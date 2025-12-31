package state

import (
	"cartero/internal/components"
	"cartero/internal/config"
	"cartero/internal/core"
)

type State struct {
	Config    *config.Config
	Registry  *components.Registry
	Pipeline  *core.Pipeline
	Platforms *components.PlatformComponent
}

func NewState(cfg *config.Config, registry *components.Registry, pipeline *core.Pipeline, platforms *components.PlatformComponent) *State {
	return &State{
		Config:    cfg,
		Registry:  registry,
		Pipeline:  pipeline,
		Platforms: platforms,
	}
}
