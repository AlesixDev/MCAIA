package motion

import (
	"math"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
)

func peakOf(frames []animation.Keyframe, axis int) float64 {
	peak := 0.0

	for _, frame := range frames {
		if math.Abs(frame.Value[axis]) > math.Abs(peak) {
			peak = frame.Value[axis]
		}
	}

	return peak
}

func TestChildrenDragBehindTheirParent(t *testing.T) {
	plan := &Plan{
		Name:   "turn",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "body", Motion: KindTwist, Axis: "y", Amplitude: 20, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, humanoid())

	head, ok := result.Bones["head"]
	if !ok {
		t.Fatalf("the head should trail the body it hangs from, got %v", result.Bones)
	}

	drag := peakOf(head.Rotation, 1)

	if drag == 0 {
		t.Fatalf("the drag should move the head")
	}

	if math.Abs(drag) > 20 {
		t.Fatalf("the drag must stay smaller than the motion it follows, got %.2f", drag)
	}

	leg := peakOf(result.Bones["left_leg"].Rotation, 1)

	if math.Abs(leg) >= math.Abs(drag) {
		t.Fatalf("a planted leg should drag less than a head: leg %.2f, head %.2f", leg, drag)
	}
}

func TestDragNeverAppearsWithoutParentRotation(t *testing.T) {
	plan := &Plan{
		Name:   "bob",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "body", Motion: KindBob, Axis: "y", Amplitude: 1, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, humanoid())

	if _, ok := result.Bones["head"]; ok {
		t.Fatalf("a parent that only translates should not drag its children")
	}
}

func TestReachAnticipatesAndOvershoots(t *testing.T) {
	plan := &Plan{
		Name:   "punch",
		Length: 1,
		Loop:   animation.LoopNone,
		Moves: []Move{
			{Target: "right_arm", Motion: KindReach, Axis: "x", Amplitude: -50, Phase: 0.5},
		},
	}

	result, _ := Synthesize(plan, humanoid())
	frames := result.Bones["right_arm"].Rotation

	wind := 0.0
	deepest := 0.0

	for _, frame := range frames {
		if frame.Time < 0.5 && frame.Value[0] > wind {
			wind = frame.Value[0]
		}

		if frame.Value[0] < deepest {
			deepest = frame.Value[0]
		}
	}

	if wind <= 0 {
		t.Fatalf("the arm should wind back before punching, got %.2f", wind)
	}

	if deepest > -50 {
		t.Fatalf("the punch should overshoot past its amplitude, got %.2f", deepest)
	}
}

func TestSpecificMovesBeatGroupMoves(t *testing.T) {
	plan := &Plan{
		Name:   "walk_and_wave",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "arms", Motion: KindSwing, Axis: "x", Amplitude: 25, Cycles: 1, Alternate: true},
			{Target: "right_arm", Motion: KindPose, Axis: "z", Amplitude: 90},
		},
	}

	result, _ := Synthesize(plan, humanoid())

	if peakOf(result.Bones["left_arm"].Rotation, 0) == 0 {
		t.Fatalf("the free arm should keep swinging with the group")
	}

	if peakOf(result.Bones["right_arm"].Rotation, 0) != 0 {
		t.Fatalf("the waving arm should drop the group swing on the same channel")
	}

	if peakOf(result.Bones["right_arm"].Rotation, 2) == 0 {
		t.Fatalf("the waving arm should keep its own pose")
	}
}

func TestFallbackLayersTwoPresets(t *testing.T) {
	roles := Classify(humanoid())

	plan := Fallback("walk and wave at the same time", "combo", 1, animation.LoopCycle, roles)

	result, _ := Synthesize(plan, humanoid())

	if peakOf(result.Bones["left_leg"].Rotation, 0) == 0 {
		t.Fatalf("the walk half of the combo is missing")
	}

	arm := result.Bones[firstArm(roles)].Rotation

	if peakOf(arm, 2) == 0 {
		t.Fatalf("the wave half of the combo is missing")
	}
}
