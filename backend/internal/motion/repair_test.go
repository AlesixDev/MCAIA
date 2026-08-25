package motion

import (
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
)

func TestRepairFlipsForwardMotions(t *testing.T) {
	roles := Classify(humanoid())

	plan := &Plan{
		Name:   "punch",
		Length: 0.6,
		Loop:   animation.LoopNone,
		Moves: []Move{
			{Target: "right_arm", Motion: KindReach, Axis: "x", Amplitude: 55, Cycles: 1, Phase: 0.35},
		},
	}

	notes := Repair(plan, "punch forward with the right arm", roles)

	if plan.Moves[0].Amplitude != -55 {
		t.Fatalf("a forward punch must swing forward, got %.1f", plan.Moves[0].Amplitude)
	}

	if len(notes) == 0 {
		t.Fatalf("the fix should be reported")
	}
}

func TestRepairKeepsBackwardMotions(t *testing.T) {
	roles := Classify(humanoid())

	plan := &Plan{
		Loop:  animation.LoopNone,
		Moves: []Move{{Target: "right_arm", Motion: KindReach, Axis: "x", Amplitude: -40, Cycles: 1}},
	}

	Repair(plan, "swing the arm backwards", roles)

	if plan.Moves[0].Amplitude != 40 {
		t.Fatalf("a backward move must be positive, got %.1f", plan.Moves[0].Amplitude)
	}
}

func TestRepairNormalizesAliasesCyclesAndDuplicates(t *testing.T) {
	roles := Classify(humanoid())

	plan := &Plan{
		Loop: animation.LoopCycle,
		Moves: []Move{
			{Target: "head", Motion: Kind("rotate"), Axis: "x", Amplitude: 10, Cycles: 1.7, Phase: 2.25},
			{Target: "head", Motion: Kind("swing"), Axis: "x", Amplitude: 5, Cycles: 1},
		},
	}

	Repair(plan, "nod", roles)

	if len(plan.Moves) != 1 {
		t.Fatalf("the duplicate move should be dropped, got %d", len(plan.Moves))
	}

	move := plan.Moves[0]

	if move.Motion != KindSwing {
		t.Fatalf("the alias should map to swing, got %q", move.Motion)
	}

	if move.Cycles != 2 {
		t.Fatalf("looping cycles should be whole, got %.2f", move.Cycles)
	}

	if move.Phase < 0 || move.Phase >= 1 {
		t.Fatalf("phase should wrap into 0..1, got %.2f", move.Phase)
	}
}

func TestRepairAddsTheMissingGait(t *testing.T) {
	roles := Classify(humanoid())

	plan := &Plan{
		Loop:  animation.LoopCycle,
		Moves: []Move{{Target: "arms", Motion: KindSwing, Axis: "x", Amplitude: 20, Cycles: 1}},
	}

	Repair(plan, "a walking cycle", roles)

	found := false

	for _, move := range plan.Moves {
		if targetPart(move.Target, roles) == PartLeg && move.Alternate {
			found = true
		}
	}

	if !found {
		t.Fatalf("a walk without legs should get them: %+v", plan.Moves)
	}
}

func TestFallbackPicksAPresetFromThePrompt(t *testing.T) {
	roles := Classify(humanoid())

	plan := Fallback("please make it run fast", "run", 0.7, animation.LoopCycle, roles)

	if len(plan.Moves) != 3 {
		t.Fatalf("the run preset should have three moves, got %+v", plan.Moves)
	}

	result, _ := Synthesize(plan, humanoid())

	if len(result.Bones) < 5 {
		t.Fatalf("the preset should animate the whole body, got %d bones", len(result.Bones))
	}
}

func TestFallbackAlwaysProducesSomething(t *testing.T) {
	roles := Classify(humanoid())

	plan := Fallback("do something completely unheard of", "thing", 1, animation.LoopCycle, roles)

	result, _ := Synthesize(plan, humanoid())

	if len(result.Bones) == 0 {
		t.Fatalf("the generic fallback must still animate something")
	}
}
