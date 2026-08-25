package exporter

import (
	"strconv"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
)

type Timeline map[string]any

func BuildTimeline(frames []animation.Keyframe, useEasing bool) Timeline {
	if len(frames) == 0 {
		return nil
	}

	timeline := make(Timeline, len(frames))

	for _, frame := range frames {
		key := FormatTime(frame.Time)
		value := []float64{frame.Value[0], frame.Value[1], frame.Value[2]}

		switch frame.Interpolation {
		case animation.InterpolationCatmullRom:
			timeline[key] = map[string]any{"post": value, "lerp_mode": "catmullrom"}
		case animation.InterpolationStep:
			if useEasing {
				timeline[key] = map[string]any{"post": value, "easing": "step"}

				continue
			}

			timeline[key] = map[string]any{"pre": value, "post": value}
		default:
			timeline[key] = value
		}
	}

	return timeline
}

func FormatTime(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)

	if formatted == "0" {
		return "0.0"
	}

	return formatted
}

func BuildBones(a animation.Animation, useEasing bool) map[string]map[string]any {
	bones := make(map[string]map[string]any, len(a.Bones))

	for name, track := range a.Bones {
		entry := make(map[string]any, 3)

		if timeline := BuildTimeline(track.Rotation, useEasing); timeline != nil {
			entry["rotation"] = timeline
		}

		if timeline := BuildTimeline(track.Position, useEasing); timeline != nil {
			entry["position"] = timeline
		}

		if timeline := BuildTimeline(track.Scale, useEasing); timeline != nil {
			entry["scale"] = timeline
		}

		if len(entry) == 0 {
			continue
		}

		bones[name] = entry
	}

	return bones
}
