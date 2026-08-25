package animation

import "math"

const defaultTolerance = 0.01

var channelTolerance = map[Channel]float64{
	ChannelRotation: 0.4,
	ChannelPosition: 0.03,
	ChannelScale:    0.004,
}

func Optimize(a *Animation, scale float64) int {
	if scale <= 0 {
		scale = 1
	}

	removed := 0

	for boneName, track := range a.Bones {
		for _, channel := range Channels() {
			frames := track.Channel(channel)

			tolerance := channelTolerance[channel] * scale

			if tolerance <= 0 {
				tolerance = defaultTolerance
			}

			reduced := reduce(frames, tolerance)
			removed += len(frames) - len(reduced)

			track.SetChannel(channel, reduced)
		}

		if track.IsEmpty() {
			delete(a.Bones, boneName)

			continue
		}

		a.Bones[boneName] = track
	}

	return removed
}

func reduce(frames []Keyframe, tolerance float64) []Keyframe {
	if len(frames) < 3 {
		return frames
	}

	result := []Keyframe{frames[0]}

	for index := 1; index < len(frames)-1; index++ {
		previous := result[len(result)-1]
		current := frames[index]
		next := frames[index+1]

		if current.Interpolation != previous.Interpolation {
			result = append(result, current)

			continue
		}

		if !isRedundant(previous, current, next, tolerance) {
			result = append(result, current)
		}
	}

	return append(result, frames[len(frames)-1])
}

func isRedundant(previous, current, next Keyframe, tolerance float64) bool {
	span := next.Time - previous.Time

	if span <= 0 {
		return false
	}

	ratio := (current.Time - previous.Time) / span

	for axis := 0; axis < 3; axis++ {
		expected := previous.Value[axis] + (next.Value[axis]-previous.Value[axis])*ratio

		if math.Abs(expected-current.Value[axis]) > tolerance {
			return false
		}
	}

	return true
}
