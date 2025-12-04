package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/dyike/CortexGo/config"
)

type EngineBuilder func(config.Config) (*Engine, error)

type Option func(*Runtime)

func WithBuilder(builder EngineBuilder) Option {
	return func(r *Runtime) {
		if builder != nil {
			r.builder = builder
		}
	}
}

func WithNotifier(fn func(topic, payload string)) Option {
	return func(r *Runtime) {
		r.notify = fn
	}
}

type Runtime struct {
	cfgMgr *config.Manager
	engine atomic.Pointer[Engine]

	builder EngineBuilder
	notify  func(string, string)
	cancel  context.CancelFunc
}

func NewRuntime(cfgMgr *config.Manager, opts ...Option) (*Runtime, error) {
	if cfgMgr == nil {
		return nil, fmt.Errorf("config manager is required")
	}

	rt := &Runtime{
		cfgMgr:  cfgMgr,
		builder: BuildEngine,
	}

	for _, opt := range opts {
		opt(rt)
	}

	if err := rt.reload(cfgMgr.Get()); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	rt.cancel = cancel
	if err := cfgMgr.Watch(ctx, func(cfg config.Config) {
		if err := rt.reload(cfg); err != nil && rt.notify == nil {
			log.Printf("engine reload failed: %v", err)
		}
	}); err != nil {
		cancel()
		return nil, err
	}

	return rt, nil
}

func (r *Runtime) Engine() *Engine {
	return r.engine.Load()
}

func (r *Runtime) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Runtime) UpdateConfigJSON(jsonStr string) error {
	return r.cfgMgr.UpdateFromJSON(jsonStr)
}

func (r *Runtime) reload(cfg config.Config) error {
	engine, err := r.builder(cfg)
	if err != nil {
		r.notifyFailure(err)
		return err
	}
	r.engine.Store(engine)
	r.notifySuccess(engine)
	return nil
}

func (r *Runtime) notifySuccess(engine *Engine) {
	if r.notify == nil {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"version":  engine.Version,
		"built_at": engine.BuiltAt.UTC().Format(time.RFC3339),
	})
	r.notify("engine.reloaded", string(payload))
}

func (r *Runtime) notifyFailure(err error) {
	if r.notify == nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"error": err.Error(),
	})
	r.notify("engine.reload_failed", string(payload))
}
