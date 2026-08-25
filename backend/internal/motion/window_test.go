package motion

import (
	"math"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

func windowRig() *rig.Rig {
	builder := rig.NewBuilder("dancer", "test")

	builder.Add(rig.Bone{Name: "body", Origin: rig.Vec3{0, 12, 0}})
	builder.Add(rig.Bone{Name: "right_arm", Parent: "body", Origin: rig.Vec3{5, 22, 0}})
	builder.Add(rig.Bone{Name: "left_arm", Parent: "body", Origin: rig.Vec3{-5, 22, 0}})

	return builder.Rig()
}

func TestFullClipMoveIsUnchanged(t *testing.T) {
	plan := &Plan{
		Name:   "wave",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "right_arm", Motion: KindSwing, Axis: "z", Amplitude: 30, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, windowRig())
	frames := result.Bones["right_arm"].Rotation

	if len(frames) == 0 {
		t.Fatal("no rotation keyframes")
	}

	peak := 0.0

	for _, frame := range frames {
		peak = math.Max(peak, math.Abs(frame.Value[2]))
	}

	if peak < 25 {
		t.Fatalf("expected the swing to reach its amplitude, peaked at %.2f", peak)
	}

	if math.Abs(frames[0].Value[2]) > 0.01 {
		t.Fatalf("a full clip sine should start at rest, got %.3f", frames[0].Value[2])
	}
}

func TestWindowedMoveOnlyActsInsideItsSection(t *testing.T) {
	plan := &Plan{
		Name:   "dance",
		Length: 2,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "right_arm", Motion: KindPose, Axis: "z", Amplitude: 90, Start: 0.5, End: 1},
		},
	}

	result, _ := Synthesize(plan, windowRig())
	frames := result.Bones["right_arm"].Rotation

	if len(frames) == 0 {
		t.Fatal("no rotation keyframes")
	}

	for _, frame := range frames {
		lifted := math.Abs(frame.Value[2]) > 1

		if frame.Time < 0.9 && lifted {
			t.Fatalf("arm moved at %.2fs, before its section starts", frame.Time)
		}
	}

	peak := 0.0

	for _, frame := range frames {
		if frame.Time >= 1 {
			peak = math.Max(peak, math.Abs(frame.Value[2]))
		}
	}

	if peak < 60 {
		t.Fatalf("expected the pose to be held inside its section, peaked at %.2f", peak)
	}
}

func TestSectionsOnTheSameBoneSurviveRepair(t *testing.T) {
	plan := &Plan{
		Name:   "dance",
		Length: 2,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "right_arm", Motion: KindSwing, Axis: "z", Amplitude: 30, Cycles: 1, Start: 0, End: 0.5},
			{Target: "right_arm", Motion: KindSwing, Axis: "z", Amplitude: 50, Cycles: 2, Start: 0.5, End: 1},
		},
	}

	Repair(plan, "dance", Classify(windowRig()))

	if len(plan.Moves) != 2 {
		t.Fatalf("expected both sections to survive, got %d", len(plan.Moves))
	}
}

func TestRepairStillDropsRealDuplicates(t *testing.T) {
	plan := &Plan{
		Name:   "walk",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "right_arm", Motion: KindSwing, Axis: "z", Amplitude: 30, Cycles: 1},
			{Target: "right_arm", Motion: KindSwing, Axis: "z", Amplitude: 40, Cycles: 1},
		},
	}

	Repair(plan, "walk", Classify(windowRig()))

	if len(plan.Moves) != 1 {
		t.Fatalf("expected the duplicate to be dropped, got %d", len(plan.Moves))
	}
}

func TestDegenerateWindowFallsBackToFullClip(t *testing.T) {
	for _, item := range []Move{
		{Target: "body", Motion: KindSwing, Axis: "x", Amplitude: 20, Cycles: 1, Start: 0.5, End: 0.51},
		{Target: "body", Motion: KindSwing, Axis: "x", Amplitude: 20, Cycles: 1, Start: 0.8, End: 0.2},
	} {
		start, end := window(item)

		if start != 0 || end != 1 {
			t.Fatalf("expected a fallback to the full clip, got %.2f..%.2f", start, end)
		}
	}
}

func TestSectionEasesInsteadOfSnapping(t *testing.T) {
	plan := &Plan{
		Name:   "dance",
		Length: 2,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "body", Motion: KindPose, Axis: "y", Amplitude: 40, Start: 0.4, End: 0.9},
		},
	}

	result, _ := Synthesize(plan, windowRig())
	frames := result.Bones["body"].Rotation

	biggest := 0.0

	for index := 1; index < len(frames); index++ {
		step := math.Abs(frames[index].Value[1] - frames[index-1].Value[1])
		biggest = math.Max(biggest, step)
	}

	if biggest >= 40 {
		t.Fatalf("the pose snapped in one step of %.2f degrees", biggest)
	}
}
