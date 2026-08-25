package motion

import (
	"math"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

func layeredRig() *rig.Rig {
	builder := rig.NewBuilder("skin", "test")

	builder.Add(rig.Bone{Name: "body", Origin: rig.Vec3{0, 12, 0}})
	builder.Add(rig.Bone{Name: "body_layer", Parent: "body", Origin: rig.Vec3{0, 12, 0}})
	builder.Add(rig.Bone{Name: "right_arm", Parent: "body", Origin: rig.Vec3{5, 22, 0}})
	builder.Add(rig.Bone{Name: "right_arm_layer", Parent: "right_arm", Origin: rig.Vec3{5, 22, 0}})
	builder.Add(rig.Bone{Name: "left_arm", Parent: "body", Origin: rig.Vec3{-5, 22, 0}})
	builder.Add(rig.Bone{Name: "left_arm_layer", Parent: "left_arm", Origin: rig.Vec3{-5, 22, 0}})
	builder.Add(rig.Bone{Name: "head", Parent: "body", Origin: rig.Vec3{0, 24, 0}})

	return builder.Rig()
}

func TestGroupMoveSkipsInheritedOverlays(t *testing.T) {
	plan := &Plan{
		Name:   "wave",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "arms", Motion: KindSwing, Axis: "z", Amplitude: 40, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, layeredRig())

	for _, overlay := range []string{"right_arm_layer", "left_arm_layer"} {
		if track, animated := result.Bones[overlay]; animated && !track.IsEmpty() {
			t.Errorf("%s was animated on its own, so it rotates twice", overlay)
		}
	}

	for _, limb := range []string{"right_arm", "left_arm"} {
		if track, animated := result.Bones[limb]; !animated || track.IsEmpty() {
			t.Errorf("%s should carry the move", limb)
		}
	}
}

func TestNamingAnOverlayDirectlyStillWorks(t *testing.T) {
	plan := &Plan{
		Name:   "twitch",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "right_arm_layer", Motion: KindSwing, Axis: "z", Amplitude: 20, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, layeredRig())

	if track, animated := result.Bones["right_arm_layer"]; !animated || track.IsEmpty() {
		t.Fatal("an explicitly named overlay should still animate")
	}
}

func TestOverlaysDoNotGetFollowThrough(t *testing.T) {
	plan := &Plan{
		Name:   "swing",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "right_arm", Motion: KindSwing, Axis: "z", Amplitude: 40, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, layeredRig())

	if track, animated := result.Bones["right_arm_layer"]; animated && !track.IsEmpty() {
		t.Fatal("the overlay was dragged behind its limb")
	}
}

func TestRealChildrenKeepFollowThrough(t *testing.T) {
	plan := &Plan{
		Name:   "twist",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "body", Motion: KindTwist, Axis: "y", Amplitude: 25, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, layeredRig())

	track, animated := result.Bones["head"]

	if !animated || track.IsEmpty() {
		t.Fatal("the head should drag behind the body")
	}

	peak := 0.0

	for _, frame := range track.Rotation {
		peak = math.Max(peak, math.Abs(frame.Value[1]))
	}

	if peak == 0 {
		t.Fatal("the head drag produced no motion")
	}
}
