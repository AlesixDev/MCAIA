package animation

import "math"

func Polish(a *Animation, fps int) {
	if fps > 0 {
		snap(a, fps)
	}

	smooth(a)

	if a.Loop == LoopCycle {
		closeLoop(a)
	}
}

func snap(a *Animation, fps int) {
	step := 1 / float64(fps)

	for name, track := range a.Bones {
		for _, channel := range Channels() {
			frames := track.Channel(channel)

			for index := range frames {
				frames[index].Time = round(math.Round(frames[index].Time/step) * step)
			}

			track.SetChannel(channel, dedupe(frames))
		}

		a.Bones[name] = track
	}
}

func smooth(a *Animation) {
	for name, track := range a.Bones {
		for _, channel := range Channels() {
			frames := track.Channel(channel)

			if len(frames) < 3 {
				continue
			}

			for index := range frames {
				if frames[index].Interpolation == InterpolationStep {
					continue
				}

				frames[index].Interpolation = InterpolationCatmullRom
			}

			track.SetChannel(channel, frames)
		}

		a.Bones[name] = track
	}
}

func closeLoop(a *Animation) {
	for name, track := range a.Bones {
		for _, channel := range Channels() {
			frames := track.Channel(channel)

			if len(frames) < 2 {
				continue
			}

			first := frames[0]

			if first.Time > 0 {
				frames = append([]Keyframe{{
					Time:          0,
					Value:         first.Value,
					Interpolation: first.Interpolation,
				}}, frames...)
			}

			last := len(frames) - 1

			if math.Abs(frames[last].Time-a.Length) > timeEpsilon {
				frames = append(frames, Keyframe{
					Time:          a.Length,
					Value:         frames[0].Value,
					Interpolation: frames[0].Interpolation,
				})

				last = len(frames) - 1
			}

			frames[last].Value = frames[0].Value

			track.SetChannel(channel, frames)
		}

		a.Bones[name] = track
	}
}
