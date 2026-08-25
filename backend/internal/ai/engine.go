package ai

import (
	"context"
	"errors"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/motion"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

var (
	ErrEngineUnavailable = errors.New("ai: engine unavailable")
	ErrInvalidResponse   = errors.New("ai: model returned an unusable response")
)

type Request struct {
	Prompt   string
	Name     string
	Rig      *rig.Rig
	Duration float64
	Loop     animation.LoopMode
	Style    string
	FPS      int
}

type Engine interface {
	ID() string
	Available(ctx context.Context) error
	Generate(ctx context.Context, request Request) (*motion.Plan, error)
}
