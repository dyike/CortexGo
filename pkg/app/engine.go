package app

import (
	"sync/atomic"
	"time"

	"github.com/dyike/CortexGo/config"
)

type Engine struct {
	Config  config.Config
	BuiltAt time.Time
	Version uint64
}

var engineSeq atomic.Uint64

func BuildEngine(cfg config.Config) (*Engine, error) {
	return &Engine{
		Config:  cfg,
		BuiltAt: time.Now(),
		Version: engineSeq.Add(1),
	}, nil
}
