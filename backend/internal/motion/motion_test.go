package motion

import (
	"math"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

func humanoid() *rig.Rig {
	builder := rig.NewBuilder("player", "bbmodel")

	builder.Add(rig.Bone{Name: "body", Origin: rig.Vec3{0, 12, 0}})
	builder.Add(rig.Bone{Name: "head", Parent: "body", Origin: rig.Vec3{0, 24, 0}})
	builder.Add(rig.Bone{Name: "left_arm", Parent: "body", Origin: rig.Vec3{-5, 22, 0}})
	builder.Add(rig.Bone{Name: "right_arm", Parent: "body", Origin: rig.Vec3{5, 22, 0}})
	builder.Add(rig.Bone{Name: "left_leg", Parent: "body", Origin: rig.Vec3{-2, 12, 0}})
	builder.Add(rig.Bone{Name: "right_leg", Parent: "body", Origin: rig.Vec3{2, 12, 0}})

	return builder.Rig()
}

func TestClassifyReadsPartsAndSides(t *testing.T) {
	roles := Classify(humanoid())

	cases := map[string]Role{
		"head":      {Part: PartHead, Side: SideCentre},
		"body":      {Part: PartBody, Side: SideCentre},
		"left_arm":  {Part: PartArm, Side: SideLeft},
		"right_leg": {Part: PartLeg, Side: SideRight},
	}

	for bone, want := range cases {
		got := roles[bone]

		if got.Part != want.Part || got.Side != want.Side {
			t.Fatalf("%s: got %s/%s, want %s/%s", bone, got.Part, got.Side, want.Part, want.Side)
		}
	}
}

func TestResolveExpandsGroups(t *testing.T) {
	roles := Classify(humanoid())

	if got := Resolve(roles, "arms"); len(got) != 2 {
		t.Fatalf("arms should resolve to both arms, got %v", got)
	}

	if got := Resolve(roles, "right_leg"); len(got) != 1 || got[0] != "right_leg" {
		t.Fatalf("an exact bone name should resolve to itself, got %v", got)
	}

	if got := Resolve(roles, "left_arm"); len(got) != 1 || got[0] != "left_arm" {
		t.Fatalf("a sided group should resolve to one bone, got %v", got)
	}

	if got := Resolve(roles, "tentacles"); len(got) != 0 {
		t.Fatalf("unknown targets resolve to nothing, got %v", got)
	}
}

func TestAlternateMovesLimbsInOpposition(t *testing.T) {
	plan := &Plan{
		Name:   "walk",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "legs", Motion: KindSwing, Axis: "x", Amplitude: 30, Cycles: 1, Alternate: true},
		},
	}

	result, _ := Synthesize(plan, humanoid())

	left := valueAt(t, result, "left_leg", animation.ChannelRotation, 0.25)
	right := valueAt(t, result, "right_leg", animation.ChannelRotation, 0.25)

	if math.Abs(left[0]+right[0]) > 0.01 {
		t.Fatalf("alternating legs should mirror: left %.2f, right %.2f", left[0], right[0])
	}

	if math.Abs(left[0]) < 20 {
		t.Fatalf("the swing should reach its amplitude at a quarter cycle, got %.2f", left[0])
	}
}

func TestLoopsCloseAndAmplitudesAreClamped(t *testing.T) {
	plan := &Plan{
		Name:   "wild",
		Length: 1,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "head", Motion: KindSwing, Axis: "y", Amplitude: 900, Cycles: 2},
		},
	}

	result, _ := Synthesize(plan, humanoid())
	animation.Polish(result, 20)

	frames := result.Bones["head"].Rotation

	if len(frames) < 3 {
		t.Fatalf("expected a sampled curve, got %d frames", len(frames))
	}

	first := frames[0]
	last := frames[len(frames)-1]

	if first.Value != last.Value {
		t.Fatalf("a looping animation must close: %v vs %v", first.Value, last.Value)
	}

	if math.Abs(last.Time-result.Length) > 0.001 {
		t.Fatalf("the last keyframe should sit on the length, got %.3f", last.Time)
	}

	for _, frame := range frames {
		if math.Abs(frame.Value[1]) > 45.001 {
			t.Fatalf("head rotation should be clamped to its limit, got %.2f", frame.Value[1])
		}
	}

	if frames[1].Interpolation != animation.InterpolationCatmullRom {
		t.Fatalf("curves should be smoothed, got %q", frames[1].Interpolation)
	}
}

func TestUnresolvedTargetsAreReported(t *testing.T) {
	plan := &Plan{
		Name:   "odd",
		Length: 1,
		Moves: []Move{
			{Target: "tentacles", Motion: KindSwing, Axis: "x", Amplitude: 20, Cycles: 1},
			{Target: "head", Motion: KindSwing, Axis: "x", Amplitude: 10, Cycles: 1},
		},
	}

	result, unresolved := Synthesize(plan, humanoid())

	if len(unresolved) != 1 || unresolved[0] != "tentacles" {
		t.Fatalf("expected the missing target to be reported, got %v", unresolved)
	}

	if _, ok := result.Bones["head"]; !ok {
		t.Fatalf("the valid move should still be synthesized")
	}
}

func TestReachPeaksOnceAndReturns(t *testing.T) {
	plan := &Plan{
		Name:   "punch",
		Length: 0.6,
		Loop:   animation.LoopNone,
		Moves: []Move{
			{Target: "right_arm", Motion: KindReach, Axis: "x", Amplitude: -60, Phase: 0.4},
		},
	}

	result, _ := Synthesize(plan, humanoid())

	start := valueAt(t, result, "right_arm", animation.ChannelRotation, 0)
	peak := valueAt(t, result, "right_arm", animation.ChannelRotation, 0.24)
	end := valueAt(t, result, "right_arm", animation.ChannelRotation, 0.6)

	if math.Abs(start[0]) > 0.01 || math.Abs(end[0]) > 0.01 {
		t.Fatalf("a reach starts and ends at rest, got %.2f and %.2f", start[0], end[0])
	}

	if peak[0] > -55 {
		t.Fatalf("the reach should hit its amplitude at the peak, got %.2f", peak[0])
	}
}

func TestPulseWritesScaleAroundOne(t *testing.T) {
	plan := &Plan{
		Name:   "breathe",
		Length: 2,
		Loop:   animation.LoopCycle,
		Moves: []Move{
			{Target: "body", Motion: KindPulse, Axis: "y", Amplitude: 0.08, Cycles: 1},
		},
	}

	result, _ := Synthesize(plan, humanoid())

	rest := valueAt(t, result, "body", animation.ChannelScale, 0)
	high := valueAt(t, result, "body", animation.ChannelScale, 0.5)

	if math.Abs(rest[1]-1) > 0.001 {
		t.Fatalf("scale should rest at 1, got %.3f", rest[1])
	}

	if math.Abs(high[1]-1.08) > 0.001 {
		t.Fatalf("scale should peak at 1 plus the amplitude, got %.3f", high[1])
	}
}

func valueAt(
	t *testing.T,
	a *animation.Animation,
	bone string,
	channel animation.Channel,
	time float64,
) animation.Vec3 {
	t.Helper()

	track, ok := a.Bones[bone]
	if !ok {
		t.Fatalf("%s was not animated", bone)
	}

	frames := track.Channel(channel)

	for _, frame := range frames {
		if math.Abs(frame.Time-time) < 0.001 {
			return frame.Value
		}
	}

	t.Fatalf("%s has no %s keyframe at %.2fs: %v", bone, channel, time, frames)

	return animation.Vec3{}
}
