package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/AlesixDev/MCAIA/backend/internal/ai"
	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter"
	"github.com/AlesixDev/MCAIA/backend/internal/motion"
	"github.com/AlesixDev/MCAIA/backend/internal/store"
)

const (
	defaultDuration = 1.0
	defaultAttempts = 2
	defaultFPS      = 20
)

type Pipeline struct {
	store    *store.Store
	engine   ai.Engine
	attempts int
}

func New(projects *store.Store, engine ai.Engine) *Pipeline {
	return &Pipeline{store: projects, engine: engine, attempts: defaultAttempts}
}

type GenerateInput struct {
	ProjectID string
	Prompt    string
	Name      string
	Duration  float64
	Loop      animation.LoopMode
	Style     string
	FPS       int
	Optimize  bool
}

type GenerateOutput struct {
	Animation *animation.Animation `json:"animation"`
	Warnings  []animation.Issue    `json:"warnings,omitempty"`
	Removed   int                  `json:"removed_keyframes"`
	Engine    string               `json:"engine"`
}

func (p *Pipeline) Generate(ctx context.Context, input GenerateInput) (*GenerateOutput, error) {
	project, err := p.store.Get(input.ProjectID)
	if err != nil {
		return nil, err
	}

	if input.Duration <= 0 {
		input.Duration = defaultDuration
	}

	if input.FPS <= 0 {
		input.FPS = defaultFPS
	}

	if input.Loop == "" {
		input.Loop = animation.LoopCycle
	}

	if input.Name == "" {
		input.Name = "generated"
	}

	request := ai.Request{
		Prompt:   input.Prompt,
		Name:     input.Name,
		Rig:      project.Rig,
		Duration: input.Duration,
		Loop:     input.Loop,
		Style:    input.Style,
		FPS:      input.FPS,
	}

	var lastErr error

	for attempt := 0; attempt < p.attempts; attempt++ {
		plan, err := p.engine.Generate(ctx, request)
		if err != nil {
			lastErr = err

			if errors.Is(err, ai.ErrEngineUnavailable) {
				break
			}

			continue
		}

		if plan.Name == "" {
			plan.Name = input.Name
		}

		if plan.Length <= 0 {
			plan.Length = input.Duration
		}

		if plan.Loop == "" {
			plan.Loop = input.Loop
		}

		roles := motion.Classify(project.Rig)
		repairs := motion.Repair(plan, input.Prompt, roles)

		draft, unresolved := motion.Synthesize(plan, project.Rig)

		warnings := make([]animation.Issue, 0)

		for _, note := range repairs {
			warnings = append(warnings, animation.Issue{Path: "plan", Message: note})
		}

		for _, target := range unresolved {
			warnings = append(warnings, animation.Issue{
				Path:    "moves." + target,
				Message: "the plan aimed at a bone or group this rig does not have",
			})
		}

		if err := animation.Normalize(draft, project.Rig); err != nil {
			var invalid *animation.ValidationError

			if errors.As(err, &invalid) {
				warnings = invalid.Issues
			}

			if len(draft.Bones) == 0 {
				lastErr = err

				continue
			}
		}

		if len(draft.Bones) == 0 {
			lastErr = fmt.Errorf("%w: the plan produced no usable tracks", ai.ErrInvalidResponse)

			continue
		}

		animation.Polish(draft, input.FPS)

		removed := 0

		if input.Optimize {
			removed = animation.Optimize(draft, 1)
		}

		draft.Name = uniqueName(draft.Name, project)

		if err := p.store.SaveAnimation(project.ID, draft); err != nil {
			return nil, err
		}

		return &GenerateOutput{Animation: draft, Warnings: warnings, Removed: removed, Engine: p.engine.ID()}, nil
	}

	return p.fallback(project, input, lastErr)
}

func uniqueName(name string, project *store.Project) string {
	if _, taken := project.Animations[name]; !taken {
		return name
	}

	for suffix := 2; ; suffix++ {
		candidate := name + "_" + strconv.Itoa(suffix)

		if _, taken := project.Animations[candidate]; !taken {
			return candidate
		}
	}
}

func (p *Pipeline) fallback(
	project *store.Project,
	input GenerateInput,
	lastErr error,
) (*GenerateOutput, error) {
	if errors.Is(lastErr, ai.ErrEngineUnavailable) {
		return nil, fmt.Errorf("pipeline: generation failed: %w", lastErr)
	}

	roles := motion.Classify(project.Rig)
	plan := motion.Fallback(input.Prompt, input.Name, input.Duration, input.Loop, roles)

	draft, _ := motion.Synthesize(plan, project.Rig)

	animation.Polish(draft, input.FPS)

	if err := animation.Normalize(draft, project.Rig); err != nil {
		var invalid *animation.ValidationError

		if !errors.As(err, &invalid) || len(draft.Bones) == 0 {
			if lastErr == nil {
				lastErr = ai.ErrInvalidResponse
			}

			return nil, fmt.Errorf("pipeline: generation failed: %w", lastErr)
		}
	}

	draft.Name = uniqueName(draft.Name, project)

	if err := p.store.SaveAnimation(project.ID, draft); err != nil {
		return nil, err
	}

	return &GenerateOutput{
		Animation: draft,
		Warnings: []animation.Issue{{
			Path:    "engine",
			Message: "the model did not return a usable plan, so this comes from a built-in preset",
		}},
		Engine: p.engine.ID(),
	}, nil
}

func (p *Pipeline) Save(projectID string, item *animation.Animation) ([]animation.Issue, error) {
	project, err := p.store.Get(projectID)
	if err != nil {
		return nil, err
	}

	warnings := make([]animation.Issue, 0)

	if err := animation.Normalize(item, project.Rig); err != nil {
		var invalid *animation.ValidationError

		if !errors.As(err, &invalid) {
			return nil, err
		}

		if len(item.Bones) == 0 {
			return invalid.Issues, err
		}

		warnings = invalid.Issues
	}

	if err := p.store.SaveAnimation(projectID, item); err != nil {
		return warnings, err
	}

	return warnings, nil
}

type ExportInput struct {
	ProjectID string
	Format    string
	Namespace string
	Names     []string
}

func (p *Pipeline) Export(input ExportInput) (*exporter.Result, error) {
	project, err := p.store.Get(input.ProjectID)
	if err != nil {
		return nil, err
	}

	target, err := exporter.Get(input.Format)
	if err != nil {
		return nil, err
	}

	items := project.Ordered()

	if len(input.Names) > 0 {
		wanted := make(map[string]struct{}, len(input.Names))

		for _, name := range input.Names {
			wanted[name] = struct{}{}
		}

		filtered := make([]animation.Animation, 0, len(input.Names))

		for _, item := range items {
			if _, ok := wanted[item.Name]; ok {
				filtered = append(filtered, item)
			}
		}

		items = filtered
	}

	if len(items) == 0 {
		return nil, errors.New("pipeline: nothing to export")
	}

	source, err := p.store.Source(project.ID)
	if err != nil {
		return nil, err
	}

	return target.Export(exporter.Request{
		ModelName:    project.Name,
		Namespace:    input.Namespace,
		Rig:          project.Rig,
		Source:       source,
		SourceFormat: project.Format,
		Animations:   items,
	})
}
