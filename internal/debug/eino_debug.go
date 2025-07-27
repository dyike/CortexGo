package debug

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/devops"
	"github.com/dyike/CortexGo/internal/config"
)

type EinoDebugger struct {
	config *config.Config
	ctx    context.Context
}

func NewEinoDebugger(cfg *config.Config) *EinoDebugger {
	return &EinoDebugger{
		config: cfg,
		ctx:    context.Background(),
	}
}

func (d *EinoDebugger) Initialize() error {
	if !d.config.EinoDebugEnabled {
		return nil
	}

	if d.config.Debug {
		log.Printf("[EinoDebug] Initializing Eino visual debug plugin on port %d", d.config.EinoDebugPort)
	}

	err := devops.Init(d.ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize Eino debug plugin: %w", err)
	}

	if d.config.Debug {
		log.Printf("[EinoDebug] Successfully initialized debug server at http://localhost:%d", d.config.EinoDebugPort)
		log.Printf("[EinoDebug] You can now debug Eino orchestration artifacts through the web interface")
	}

	return nil
}

func (d *EinoDebugger) IsEnabled() bool {
	return d.config.EinoDebugEnabled
}

func (d *EinoDebugger) GetDebugURL() string {
	if !d.config.EinoDebugEnabled {
		return ""
	}
	return fmt.Sprintf("http://localhost:%d", d.config.EinoDebugPort)
}