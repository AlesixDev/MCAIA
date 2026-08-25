package motion

import (
	"math"
	"sort"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

type Kind string

const (
	KindSwing Kind = "swing"
	KindTwist Kind = "twist"
	KindBob   Kind = "bob"
	KindShift Kind = "shift"
	KindPulse Kind = "pulse"
	KindPose  Kind = "pose"
	KindReach Kind = "reach"
)

type Move struct {
	Target    string  `json:"target"`
	Motion    Kind    `json:"motion"`
	Axis      string  `json:"axis"`
	Amplitude float64 `json:"amplitude"`
	Cycles    float64 `json:"cycles"`
	Phase     float64 `json:"phase"`
	Offset    float64 `json:"offset"`
	Alternate bool    `json:"alternate"`
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
}

type Plan struct {
	Name   string             `json:"name"`
	Length float64            `json:"length"`
	Loop   animation.LoopMode `json:"loop"`
	Moves  []Move             `json:"moves"`
}

const (
	maxSamplesPerChannel = 33
	minCycleSamples      = 4
	dragWindow           = 0.09
	minWindow            = 0.05
	blendFraction        = 0.2
	anticipation         = 0.16
	overshoot            = 1.08
)

var dragFactor = map[Part]float64{
	PartTail:    0.85,
	PartWing:    0.6,
	PartHead:    0.3,
	PartArm:     0.3,
	PartLeg:     0.12,
	PartBody:    0.18,
	PartUnknown: 0.25,
}

var limits = map[Part]struct {
	rotation    float64
	translation float64
}{
	PartHead:    {45, 1},
	PartBody:    {25, 3},
	PartArm:     {70, 1},
	PartLeg:     {60, 1},
	PartTail:    {50, 1},
	PartWing:    {80, 1},
	PartUnknown: {45, 1},
}

type contribution struct {
	channel animation.Channel
	axis    int
	times   []float64
	value   func(time float64) float64
	ceiling float64
}

func Synthesize(plan *Plan, skeleton *rig.Rig) (*animation.Animation, []string) {
	roles := Classify(skeleton)
	unresolved := make([]string, 0)

	length := plan.Length

	if length <= 0 {
		length = 1
	}

	result := &animation.Animation{
		Name:   plan.Name,
		Length: length,
		Loop:   plan.Loop,
		Bones:  make(map[string]animation.Track),
	}

	grouped := make(map[string][]contribution)
	claimed := explicit(plan, roles)

	for _, move := range plan.Moves {
		targets := Resolve(roles, move.Target)

		if len(targets) == 0 {
			unresolved = append(unresolved, move.Target)

			continue
		}

		sort.Strings(targets)

		group := isGroup(roles, move.Target)

		if group {
			targets = topmost(targets, skeleton)
		}

		for index, bone := range targets {
			if group && claimed[bone+"|"+string(channelOf(move.Motion))] {
				continue
			}

			phase := move.Phase

			if move.Alternate {
				phase += alternateOffset(roles[bone], index)
			}

			built := build(move, roles[bone], phase, length)

			if built == nil {
				continue
			}

			for _, carrier := range carriers(move, bone, roles, skeleton) {
				grouped[carrier] = append(grouped[carrier], *built)
			}
		}
	}

	for bone, drag := range follow(grouped, skeleton, roles) {
		grouped[bone] = append(grouped[bone], drag...)
	}

	for bone, contributions := range grouped {
		track := animation.Track{}

		for _, channel := range animation.Channels() {
			frames := combine(contributions, channel, length, ceiling(roles[bone], channel))

			if len(frames) > 0 {
				track.SetChannel(channel, frames)
			}
		}

		if !track.IsEmpty() {
			result.Bones[bone] = track
		}
	}

	return result, unresolved
}

func carriers(move Move, bone string, roles map[string]Role, skeleton *rig.Rig) []string {
	if channelOf(move.Motion) != animation.ChannelPosition || roles[bone].Part != PartBody {
		return []string{bone}
	}

	if len(skeleton.Roots) == 0 {
		return []string{bone}
	}

	return skeleton.Roots
}

func topmost(targets []string, skeleton *rig.Rig) []string {
	selected := make(map[string]bool, len(targets))

	for _, name := range targets {
		selected[name] = true
	}

	kept := make([]string, 0, len(targets))

	for _, name := range targets {
		if inherits(name, selected, skeleton) {
			continue
		}

		kept = append(kept, name)
	}

	if len(kept) == 0 {
		return targets
	}

	return kept
}

func inherits(name string, selected map[string]bool, skeleton *rig.Rig) bool {
	bone, ok := skeleton.Bones[name]

	for depth := 0; ok && bone.Parent != "" && depth < len(skeleton.Bones); depth++ {
		if selected[bone.Parent] {
			return true
		}

		bone, ok = skeleton.Bones[bone.Parent]
	}

	return false
}

func alternateOffset(role Role, index int) float64 {
	switch role.Side {
	case SideRight:
		return 0.5
	case SideLeft:
		return 0
	}

	return float64(index%2) * 0.5
}

func window(move Move) (float64, float64) {
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

func blend(start, end float64) func(float64) float64 {
	rampIn := start > 0
	rampOut := end < 1

	if !rampIn && !rampOut {
		return func(float64) float64 { return 1 }
	}

	ramp := math.Min(blendFraction, 0.5)

	return func(local float64) float64 {
		weight := 1.0

		if rampIn && local < ramp {
			weight = smoothstep(local / ramp)
		}

		if rampOut && local > 1-ramp {
			weight = math.Min(weight, smoothstep((1-local)/ramp))
		}

		return weight
	}
}

func smoothstep(value float64) float64 {
	value = math.Max(0, math.Min(1, value))

	return value * value * (3 - 2*value)
}

func build(move Move, role Role, phase, length float64) *contribution {
	channel, defaultAxis := channelFor(move.Motion)

	if channel == "" {
		return nil
	}

	axis := axisIndex(move.Axis, defaultAxis)
	limit := limits[role.Part]
	bound := limit.rotation

	if channel == animation.ChannelPosition {
		bound = limit.translation
	}

	if channel == animation.ChannelScale {
		bound = 0.6
	}

	amplitude := math.Max(-bound, math.Min(bound, move.Amplitude))

	cycles := move.Cycles

	if cycles <= 0 {
		cycles = 1
	}

	cycles = math.Min(cycles, 8)

	shape, marks := curve(move, amplitude, cycles, phase)

	if shape == nil {
		return nil
	}

	start, end := window(move)
	span := end - start
	weight := blend(start, end)

	times := make([]float64, 0, len(marks))

	for _, mark := range marks {
		times = append(times, length*(start+mark*span))
	}

	return &contribution{
		channel: channel,
		axis:    axis,
		times:   times,
		ceiling: ceiling(role, channel),
		value: func(time float64) float64 {
			ratio := time / math.Max(length, 0.001)

			if ratio < start || ratio > end {
				return 0
			}

			local := (ratio - start) / math.Max(span, 0.001)

			return weight(local) * shape(local)
		},
	}
}

func curve(move Move, amplitude, cycles, phase float64) (func(float64) float64, []float64) {
	switch move.Motion {
	case KindPose:
		return func(float64) float64 { return move.Offset + amplitude }, []float64{0, 1}

	case KindReach:
		peak := math.Max(0.1, math.Min(0.9, phase))
		wind := peak * (1 - anticipation*2)
		settle := peak + (1-peak)*0.45

		return func(local float64) float64 {
			switch {
			case local <= wind:
				return move.Offset - amplitude*anticipation*easeOut(local/math.Max(wind, 0.001))

			case local <= peak:
				inner := (local - wind) / math.Max(peak-wind, 0.001)

				return move.Offset + amplitude*(easeOut(inner)*(1+anticipation)-anticipation)

			case local <= settle:
				inner := (local - peak) / math.Max(settle-peak, 0.001)

				return move.Offset + amplitude*(overshoot-(overshoot-0.55)*easeIn(inner))
			}

			inner := (local - settle) / math.Max(1-settle, 0.001)

			return move.Offset + amplitude*0.55*(1-easeOut(inner))
		}, []float64{0, wind, peak, settle, (settle + 1) / 2, 1}
	}

	return func(local float64) float64 {
		return move.Offset + amplitude*math.Sin(2*math.Pi*(cycles*local+phase))
	}, evenFractions(cycles)
}

func evenFractions(cycles float64) []float64 {
	count := int(math.Ceil(cycles * minCycleSamples))

	if count < minCycleSamples {
		count = minCycleSamples
	}

	marks := make([]float64, 0, count+1)

	for index := 0; index <= count; index++ {
		marks = append(marks, float64(index)/float64(count))
	}

	return marks
}

func ceiling(role Role, channel animation.Channel) float64 {
	limit := limits[role.Part]

	switch channel {
	case animation.ChannelPosition:
		return limit.translation
	case animation.ChannelScale:
		return 0.6
	}

	return limit.rotation
}

func combine(
	contributions []contribution,
	channel animation.Channel,
	length float64,
	ceiling float64,
) []animation.Keyframe {
	relevant := make([]contribution, 0, len(contributions))
	moments := make(map[float64]bool)

	for _, item := range contributions {
		if item.channel != channel {
			continue
		}

		relevant = append(relevant, item)

		for _, time := range item.times {
			moments[round(math.Max(0, math.Min(length, time)))] = true
		}
	}

	if len(relevant) == 0 {
		return nil
	}

	times := make([]float64, 0, len(moments))

	for time := range moments {
		times = append(times, time)
	}

	sort.Float64s(times)

	if len(times) > maxSamplesPerChannel {
		times = thin(times, maxSamplesPerChannel)
	}

	base := 0.0

	if channel == animation.ChannelScale {
		base = 1
	}

	for _, item := range relevant {
		ceiling = math.Max(ceiling, item.ceiling)
	}

	frames := make([]animation.Keyframe, 0, len(times))

	for _, time := range times {
		value := animation.Vec3{base, base, base}

		for _, item := range relevant {
			value[item.axis] += item.value(time)
		}

		for axis := 0; axis < 3; axis++ {
			value[axis] = math.Max(base-ceiling, math.Min(base+ceiling, value[axis]))
		}

		frames = append(frames, animation.Keyframe{
			Time:          round(time),
			Value:         animation.Vec3{round(value[0]), round(value[1]), round(value[2])},
			Interpolation: animation.InterpolationCatmullRom,
		})
	}

	if len(frames) > 0 {
		frames[0].Interpolation = animation.InterpolationCatmullRom
		frames[len(frames)-1].Time = round(length)
	}

	return frames
}

func thin(times []float64, limit int) []float64 {
	result := make([]float64, 0, limit)
	step := float64(len(times)-1) / float64(limit-1)

	for index := 0; index < limit; index++ {
		result = append(result, times[int(math.Round(float64(index)*step))])
	}

	return result
}

func explicit(plan *Plan, roles map[string]Role) map[string]bool {
	claimed := make(map[string]bool)

	for _, move := range plan.Moves {
		if isGroup(roles, move.Target) {
			continue
		}

		for _, bone := range Resolve(roles, move.Target) {
			claimed[bone+"|"+string(channelOf(move.Motion))] = true
		}
	}

	return claimed
}

func isGroup(roles map[string]Role, target string) bool {
	_, exact := roles[strings.ToLower(strings.TrimSpace(target))]

	return !exact
}

func channelOf(kind Kind) animation.Channel {
	channel, _ := channelFor(kind)

	return channel
}

func follow(
	grouped map[string][]contribution,
	skeleton *rig.Rig,
	roles map[string]Role,
) map[string][]contribution {
	drags := make(map[string][]contribution)

	for name, contributions := range grouped {
		bone, ok := skeleton.Bones[name]

		if !ok || len(bone.Children) == 0 {
			continue
		}

		sources := make([]contribution, 0, len(contributions))

		for _, item := range contributions {
			if item.channel == animation.ChannelRotation {
				sources = append(sources, item)
			}
		}

		if len(sources) == 0 {
			continue
		}

		for _, child := range bone.Children {
			factor := dragFactor[roles[child].Part]

			if factor <= 0 {
				continue
			}

			if skeleton.Bones[child].Origin == bone.Origin {
				continue
			}

			for _, source := range sources {
				drags[child] = append(drags[child], lagged(source, factor))
			}
		}
	}

	return drags
}

func lagged(source contribution, factor float64) contribution {
	times := make([]float64, 0, len(source.times)*2)

	span := 0.0

	if len(source.times) > 1 {
		span = source.times[len(source.times)-1] * dragWindow
	}

	for _, time := range source.times {
		times = append(times, time, time+span)
	}

	value := source.value

	return contribution{
		channel: source.channel,
		axis:    source.axis,
		times:   times,
		ceiling: source.ceiling,
		value: func(time float64) float64 {
			return -factor * (value(time) - value(math.Max(0, time-span)))
		},
	}
}

func channelFor(kind Kind) (animation.Channel, int) {
	switch kind {
	case KindSwing, KindPose, KindReach:
		return animation.ChannelRotation, 0
	case KindTwist:
		return animation.ChannelRotation, 1
	case KindBob:
		return animation.ChannelPosition, 1
	case KindShift:
		return animation.ChannelPosition, 0
	case KindPulse:
		return animation.ChannelScale, 1
	}

	return "", 0
}

func axisIndex(axis string, fallback int) int {
	switch strings.ToLower(strings.TrimSpace(axis)) {
	case "x":
		return 0
	case "y":
		return 1
	case "z":
		return 2
	}

	return fallback
}

func easeIn(ratio float64) float64 {
	ratio = clamp01(ratio)

	return ratio * ratio
}

func easeOut(ratio float64) float64 {
	ratio = clamp01(ratio)

	return 1 - (1-ratio)*(1-ratio)
}

func clamp01(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}

func round(value float64) float64 {
	return math.Round(value*1000) / 1000
}
