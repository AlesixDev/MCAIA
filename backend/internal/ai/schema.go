package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/motion"
)

var motions = []string{"swing", "twist", "bob", "shift", "pulse", "pose", "reach"}

func ResponseSchema() map[string]any {
	move := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target":    map[string]any{"type": "string"},
			"motion":    map[string]any{"type": "string", "enum": motions},
			"axis":      map[string]any{"type": "string", "enum": []string{"x", "y", "z"}},
			"amplitude": map[string]any{"type": "number"},
			"cycles":    map[string]any{"type": "number"},
			"phase":     map[string]any{"type": "number"},
			"offset":    map[string]any{"type": "number"},
			"alternate": map[string]any{"type": "boolean"},
			"start":     map[string]any{"type": "number"},
			"end":       map[string]any{"type": "number"},
		},
		"required": []string{"target", "motion", "axis", "amplitude", "cycles"},
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":   map[string]any{"type": "string"},
			"length": map[string]any{"type": "number"},
			"loop":   map[string]any{"type": "string", "enum": []string{"once", "loop", "hold_on_last_frame"}},
			"moves":  map[string]any{"type": "array", "items": move},
		},
		"required": []string{"name", "length", "loop", "moves"},
	}
}

type draftMove struct {
	Target    string  `json:"target"`
	Motion    string  `json:"motion"`
	Axis      string  `json:"axis"`
	Amplitude float64 `json:"amplitude"`
	Cycles    float64 `json:"cycles"`
	Phase     float64 `json:"phase"`
	Offset    float64 `json:"offset"`
	Alternate bool    `json:"alternate"`
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
}

type draftPlan struct {
	Name   string      `json:"name"`
	Length float64     `json:"length"`
	Loop   string      `json:"loop"`
	Moves  []draftMove `json:"moves"`
}

func DecodePlan(payload string) (*motion.Plan, error) {
	trimmed := extractObject(payload)

	if trimmed == "" {
		return nil, fmt.Errorf("%w: no json object found", ErrInvalidResponse)
	}

	var parsed draftPlan

	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	if len(parsed.Moves) == 0 {
		return nil, fmt.Errorf("%w: the plan has no moves", ErrInvalidResponse)
	}

	plan := &motion.Plan{
		Name:   parsed.Name,
		Length: parsed.Length,
		Loop:   animation.LoopMode(parsed.Loop),
		Moves:  make([]motion.Move, 0, len(parsed.Moves)),
	}

	for _, item := range parsed.Moves {
		plan.Moves = append(plan.Moves, motion.Move{
			Target:    item.Target,
			Motion:    motion.Kind(strings.ToLower(strings.TrimSpace(item.Motion))),
			Axis:      item.Axis,
			Amplitude: item.Amplitude,
			Cycles:    item.Cycles,
			Phase:     item.Phase,
			Offset:    item.Offset,
			Alternate: item.Alternate,
			Start:     item.Start,
			End:       item.End,
		})
	}

	return plan, nil
}

func extractObject(payload string) string {
	start := strings.Index(payload, "{")
	end := strings.LastIndex(payload, "}")

	if start < 0 || end <= start {
		return ""
	}

	return payload[start : end+1]
}
