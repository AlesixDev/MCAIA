package motion

import (
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
)

type preset struct {
	name  string
	words []string
	build func(roles map[string]Role) []Move
}

var presets = []preset{
	{
		name:  "run",
		words: []string{"run", "sprint", "corr"},
		build: func(roles map[string]Role) []Move {
			return gait(45, 34, 1.2)
		},
	},
	{
		name:  "walk",
		words: []string{"walk", "camin", "step", "march"},
		build: func(roles map[string]Role) []Move {
			return gait(30, 22, 0.6)
		},
	},
	{
		name:  "attack",
		words: []string{"punch", "attack", "hit", "strike", "stab", "golpe", "atac"},
		build: func(roles map[string]Role) []Move {
			arm := firstArm(roles)

			if arm == "" {
				return nil
			}

			return []Move{
				{Target: arm, Motion: KindReach, Axis: "x", Amplitude: -55, Cycles: 1, Phase: 0.35},
				{Target: "body", Motion: KindTwist, Axis: "y", Amplitude: 10, Cycles: 1, Phase: 0.35},
			}
		},
	},
	{
		name:  "wave",
		words: []string{"wave", "hello", "greet", "salud"},
		build: func(roles map[string]Role) []Move {
			arm := firstArm(roles)

			if arm == "" {
				return nil
			}

			return []Move{
				{Target: arm, Motion: KindPose, Axis: "z", Amplitude: 95, Cycles: 1},
				{Target: arm, Motion: KindSwing, Axis: "z", Amplitude: 18, Cycles: 3},
			}
		},
	},
	{
		name:  "jump",
		words: []string{"jump", "hop", "leap", "salt"},
		build: func(roles map[string]Role) []Move {
			return []Move{
				{Target: "body", Motion: KindBob, Axis: "y", Amplitude: 3, Cycles: 1},
				{Target: "legs", Motion: KindSwing, Axis: "x", Amplitude: 20, Cycles: 1},
				{Target: "arms", Motion: KindSwing, Axis: "x", Amplitude: -25, Cycles: 1},
			}
		},
	},
	{
		name:  "nod",
		words: []string{"nod", "yes", "asent"},
		build: func(roles map[string]Role) []Move {
			return []Move{{Target: "head", Motion: KindSwing, Axis: "x", Amplitude: 12, Cycles: 2}}
		},
	},
	{
		name:  "shake",
		words: []string{"shake", "no ", "deny", "nieg"},
		build: func(roles map[string]Role) []Move {
			return []Move{{Target: "head", Motion: KindTwist, Axis: "y", Amplitude: 22, Cycles: 2}}
		},
	},
	{
		name:  "idle",
		words: []string{"idle", "breath", "respir", "quiet", "stand"},
		build: func(roles map[string]Role) []Move {
			return []Move{
				{Target: "body", Motion: KindBob, Axis: "y", Amplitude: 0.3, Cycles: 1},
				{Target: "head", Motion: KindSwing, Axis: "x", Amplitude: 3, Cycles: 1},
			}
		},
	},
}

func Fallback(prompt, name string, length float64, loop animation.LoopMode, roles map[string]Role) *Plan {
	lowered := strings.ToLower(prompt)

	layered := make([]Move, 0)

	for _, item := range presets {
		if !containsAny(lowered, item.words) {
			continue
		}

		layered = layer(layered, usable(item.build(roles), roles), roles)
	}

	if len(layered) > 0 {
		return &Plan{Name: name, Length: length, Loop: loop, Moves: layered}
	}

	generic := usable([]Move{
		{Target: "body", Motion: KindBob, Axis: "y", Amplitude: 0.3, Cycles: 1},
		{Target: "head", Motion: KindSwing, Axis: "x", Amplitude: 4, Cycles: 1},
		{Target: "arms", Motion: KindSwing, Axis: "x", Amplitude: 6, Cycles: 1, Alternate: true},
	}, roles)

	if len(generic) == 0 {
		for bone := range roles {
			generic = []Move{{Target: bone, Motion: KindSwing, Axis: "x", Amplitude: 8, Cycles: 1}}

			break
		}
	}

	return &Plan{Name: name, Length: length, Loop: loop, Moves: generic}
}

func layer(base, extra []Move, roles map[string]Role) []Move {
	claimed := make(map[string]bool)

	for _, move := range extra {
		if isGroup(roles, move.Target) {
			continue
		}

		for _, bone := range Resolve(roles, move.Target) {
			claimed[bone+"|"+string(channelOf(move.Motion))] = true
		}
	}

	result := make([]Move, 0, len(base)+len(extra))

	for _, move := range base {
		if !isGroup(roles, move.Target) {
			result = append(result, move)

			continue
		}

		kept := false

		for _, bone := range Resolve(roles, move.Target) {
			if !claimed[bone+"|"+string(channelOf(move.Motion))] {
				kept = true
			}
		}

		if kept {
			result = append(result, move)
		}
	}

	return append(result, extra...)
}

func gait(legs, arms, bob float64) []Move {
	return []Move{
		{Target: "legs", Motion: KindSwing, Axis: "x", Amplitude: legs, Cycles: 1, Alternate: true},
		{Target: "arms", Motion: KindSwing, Axis: "x", Amplitude: arms, Cycles: 1, Phase: 0.5, Alternate: true},
		{Target: "body", Motion: KindBob, Axis: "y", Amplitude: bob, Cycles: 2},
	}
}

func firstArm(roles map[string]Role) string {
	if matched := Resolve(roles, "right_arm"); len(matched) > 0 {
		return matched[0]
	}

	if matched := Resolve(roles, "arms"); len(matched) > 0 {
		return matched[0]
	}

	return ""
}

func usable(moves []Move, roles map[string]Role) []Move {
	result := make([]Move, 0, len(moves))

	for _, move := range moves {
		if len(Resolve(roles, move.Target)) > 0 {
			result = append(result, move)
		}
	}

	return result
}
