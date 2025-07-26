package agents

import (
	"context"

	"github.com/dyike/CortexGo/pkg/config"
	"github.com/dyike/CortexGo/pkg/models"
)

type Agent interface {
	Name() string
	Process(ctx context.Context, state *models.AgentState) (*models.AgentState, error)
	GetConfig() *config.Config
}

type BaseAgent struct {
	name   string
	config *config.Config
}

func NewBaseAgent(name string, config *config.Config) *BaseAgent {
	return &BaseAgent{
		name:   name,
		config: config,
	}
}

func (b *BaseAgent) Name() string {
	return b.name
}

func (b *BaseAgent) GetConfig() *config.Config {
	return b.config
}
