package motion

import (
	"math"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

func waistRig() *rig.Rig {
	builder := rig.NewBuilder("player", "test")

	builder.Add(rig.Bone{Name: "waist", Origin: rig.Vec3{0, 12, 0}})
	builder.Add(rig.Bone{Name: "head", Parent: "waist", Origin: rig.Vec3{0, 24, 0}})
	builder.Add(rig.Bone{Name: "body", Parent: "waist", Origin: rig.Vec3{0, 24, 0}})
	builder.Add(rig.Bone{Name: "right_arm", Parent: "waist", Origin: rig.Vec3{5, 22, 0}})
	builder.Add(rig.Bone{Name: "left_arm", Parent: "waist", Origin: rig.Vec3{-5, 22, 0}})
	builder.Add(rig.Bone{Name: "right_leg", Origin: rig.Vec3{2, 12, 0}})
	builder.Add(rig.Bone{Name: "left_leg", Origin: rig.Vec3{-2, 12, 0}})

	return builder.Rig()
}

func offsetAt(clip *animation.Animation, skeleton *rig.Rig, bone string, at float64) [3]float64 {
	total := [3]float64{}
	current := bone

	for depth := 0; depth < len(skeleton.Bones)+1; depth++ {
		if track, ok := clip.Bones[current]; ok {
			value := valueOfChannel(track.Position, at)

			for axis := 0; axis < 3; axis++ {
				total[axis] += value[axis]
			}
		}

		parent := skeleton.Bones[current].Parent

		if parent == "" {
			break
		}

		current = parent
	}

	return total
}

func valueOfChannel(frames []animation.Keyframe, at float64) [3]float64 {
	if len(frames) == 0 {
		return [3]float64{}
	}

	best := frames[0]

	for _, frame := range frames {
		if math.Abs(frame.Time-at) < math.Abs(best.Time-at) {
			best = frame
		}
	}

	return [3]float64{best.Value[0], best.Value[1], best.Value[2]}
}

func TestBodyBobMovesTheWholeCharacter(t *testing.T) {
	skeleton := waistRig()

	plan := &Plan{
		Name:   "dance",
		Length: 2,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "body", Motion: KindBob, Axis: "y", Amplitude: 3, Cycles: 2},
		},
	}

	result, _ := Synthesize(plan, skeleton)

	for at := 0.0; at <= 2; at += 0.05 {
		reference := offsetAt(result, skeleton, "body", at)

		for _, bone := range skeleton.Names() {
			offset := offsetAt(result, skeleton, bone, at)

			for axis := 0; axis < 3; axis++ {
				if math.Abs(offset[axis]-reference[axis]) > 0.01 {
					t.Fatalf("at %.2fs %s is %.2f from the torso on axis %d",
						at, bone, offset[axis]-reference[axis], axis)
				}
			}
		}
	}

	moved := 0.0

	for _, frame := range result.Bones["right_leg"].Position {
		moved = math.Max(moved, math.Abs(frame.Value[1]))
	}

	if moved < 1 {
		t.Fatal("the legs never moved, so they were left behind by the bob")
	}
}

func TestLimbSlideIsHeldToASliver(t *testing.T) {
	plan := &Plan{
		Name:   "float",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "head", Motion: KindBob, Axis: "y", Amplitude: 6, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, waistRig())

	for _, frame := range result.Bones["head"].Position {
		for axis := 0; axis < 3; axis++ {
			if math.Abs(frame.Value[axis]) > limits[PartHead].translation+0.001 {
				t.Fatalf("at %.2fs the head slid %.2f off the neck", frame.Time, frame.Value[axis])
			}
		}
	}
}

func TestBodyRotationStaysOnTheBodyBone(t *testing.T) {
	plan := &Plan{
		Name:   "twist",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "body", Motion: KindTwist, Axis: "y", Amplitude: 20, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, waistRig())

	if track, ok := result.Bones["body"]; !ok || len(track.Rotation) == 0 {
		t.Fatal("a body twist should stay on the body bone")
	}
}

func TestOverlappingSectionsStayInsideTheLimit(t *testing.T) {
	skeleton := waistRig()

	plan := &Plan{
		Name:   "dance",
		Length: 2,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "waist", Motion: KindBob, Axis: "y", Amplitude: 3, Cycles: 2, Start: 0, End: 0.7},
			{Target: "waist", Motion: KindBob, Axis: "y", Amplitude: 3, Cycles: 3, Start: 0.3, End: 1},
		},
	}

	result, _ := Synthesize(plan, skeleton)
	limit := limits[PartBody].translation

	for _, frame := range result.Bones["waist"].Position {
		for axis := 0; axis < 3; axis++ {
			if math.Abs(frame.Value[axis]) > limit+0.001 {
				t.Fatalf("at %.2fs the trunk reached %.2f, past the %.2f limit",
					frame.Time, frame.Value[axis], limit)
			}
		}
	}
}

func TestRotationIsAlsoCappedOnceCombined(t *testing.T) {
	plan := &Plan{
		Name:   "flail",
		Length: 2,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "right_arm", Motion: KindSwing, Axis: "z", Amplitude: 70, Cycles: 1, Start: 0, End: 0.7},
			{Target: "right_arm", Motion: KindSwing, Axis: "z", Amplitude: 70, Cycles: 2, Start: 0.3, End: 1},
		},
	}

	result, _ := Synthesize(plan, waistRig())
	limit := limits[PartArm].rotation

	for _, frame := range result.Bones["right_arm"].Rotation {
		for axis := 0; axis < 3; axis++ {
			if math.Abs(frame.Value[axis]) > limit+0.001 {
				t.Fatalf("at %.2fs the arm reached %.2f degrees, past the %.2f limit",
					frame.Time, frame.Value[axis], limit)
			}
		}
	}
}
