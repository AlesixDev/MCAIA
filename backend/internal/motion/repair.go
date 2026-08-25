package motion

import (
	"fmt"
	"math"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
)

var kindAliases = map[string]Kind{
	"swing":     KindSwing,
	"rotate":    KindSwing,
	"rotation":  KindSwing,
	"wave":      KindSwing,
	"sway":      KindSwing,
	"twist":     KindTwist,
	"turn":      KindTwist,
	"yaw":       KindTwist,
	"bob":       KindBob,
	"bounce":    KindBob,
	"bobbing":   KindBob,
	"shift":     KindShift,
	"slide":     KindShift,
	"move":      KindShift,
	"translate": KindShift,
	"pulse":     KindPulse,
	"scale":     KindPulse,
	"breathe":   KindPulse,
	"pose":      KindPose,
	"hold":      KindPose,
	"static":    KindPose,
	"reach":     KindReach,
	"punch":     KindReach,
	"strike":    KindReach,
	"stab":      KindReach,
}

var forwardWords = []string{
	"forward", "forwards", "punch", "kick", "stab", "lunge", "thrust", "charge", "attack", "adelante",
}

var backwardWords = []string{"backward", "backwards", "lean back", "recoil", "retreat", "atras", "atrás"}

var upWords = []string{"jump", "hop", "leap", "rise", "stand up", "saltar", "salto"}

var downWords = []string{"crouch", "duck", "sneak", "sink", "kneel", "agachar"}

func trimWindow(move Move) (float64, float64) {
	start := math.Max(0, math.Min(1, move.Start))
	end := move.End

	if end <= 0 {
		end = 1
	}

	end = math.Max(0, math.Min(1, end))

	if end-start < minWindow {
		return 0, 1
	}

	return start, end
}

func Repair(plan *Plan, prompt string, roles map[string]Role) []string {
	lowered := strings.ToLower(prompt)
	notes := make([]string, 0)

	forward := containsAny(lowered, forwardWords)
	backward := containsAny(lowered, backwardWords)
	up := containsAny(lowered, upWords)
	down := containsAny(lowered, downWords)

	seen := make(map[string]bool)
	repaired := make([]Move, 0, len(plan.Moves))

	for _, move := range plan.Moves {
		if kind, ok := kindAliases[strings.ToLower(string(move.Motion))]; ok {
			move.Motion = kind
		}

		if _, known := channelFor(move.Motion); known == 0 && move.Motion == "" {
			continue
		}

		move.Start, move.End = trimWindow(move)

		key := fmt.Sprintf("%s|%s|%s|%.3f|%.3f",
			strings.ToLower(move.Target), move.Motion, strings.ToLower(move.Axis),
			move.Start, move.End)

		if seen[key] {
			notes = append(notes, "dropped a duplicate move on "+move.Target)

			continue
		}

		seen[key] = true

		if plan.Loop == animation.LoopCycle && move.Cycles > 0 && move.Start <= 0 && move.End >= 1 {
			rounded := math.Max(1, math.Round(move.Cycles))

			if math.Abs(rounded-move.Cycles) > 0.01 {
				notes = append(notes, "rounded the cycles of "+move.Target+" so the loop closes")
				move.Cycles = rounded
			}
		}

		move.Phase = move.Phase - math.Floor(move.Phase)

		if fixed, changed := correct(move, roles, forward, backward, up, down); changed {
			notes = append(notes, "flipped the direction of the "+string(move.Motion)+" on "+move.Target)
			move = fixed
		}

		repaired = append(repaired, move)
	}

	plan.Moves = repaired

	if added := ensureGait(plan, lowered, roles); added {
		notes = append(notes, "added the missing alternating legs of a gait")
	}

	return notes
}

func correct(move Move, roles map[string]Role, forward, backward, up, down bool) (Move, bool) {
	part := targetPart(move.Target, roles)

	limb := part == PartArm || part == PartLeg

	if limb && strings.EqualFold(move.Axis, "x") && move.Motion == KindReach {
		if forward && move.Amplitude > 0 {
			move.Amplitude = -move.Amplitude

			return move, true
		}

		if backward && move.Amplitude < 0 {
			move.Amplitude = -move.Amplitude

			return move, true
		}
	}

	if move.Motion == KindBob {
		if up && move.Amplitude < 0 {
			move.Amplitude = -move.Amplitude

			return move, true
		}

		if down && move.Amplitude > 0 {
			move.Amplitude = -move.Amplitude

			return move, true
		}
	}

	return move, false
}

func ensureGait(plan *Plan, prompt string, roles map[string]Role) bool {
	if !strings.Contains(prompt, "walk") && !strings.Contains(prompt, "run") &&
		!strings.Contains(prompt, "camin") && !strings.Contains(prompt, "corr") {
		return false
	}

	if len(Resolve(roles, "legs")) == 0 {
		return false
	}

	for index, move := range plan.Moves {
		if targetPart(move.Target, roles) != PartLeg {
			continue
		}

		if !move.Alternate {
			plan.Moves[index].Alternate = true

			return true
		}

		return false
	}

	amplitude := 30.0

	if strings.Contains(prompt, "run") || strings.Contains(prompt, "corr") {
		amplitude = 45
	}

	plan.Moves = append(plan.Moves, Move{
		Target:    "legs",
		Motion:    KindSwing,
		Axis:      "x",
		Amplitude: amplitude,
		Cycles:    1,
		Alternate: true,
	})

	return true
}

func targetPart(target string, roles map[string]Role) Part {
	matched := Resolve(roles, target)

	if len(matched) == 0 {
		return PartUnknown
	}

	return roles[matched[0]].Part
}

func containsAny(text string, words []string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}

	return false
}
