package animation

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

const (
	MinLength      = 0.05
	MaxLength      = 120.0
	MaxKeyframes   = 512
	timeEpsilon    = 1e-4
	maxTranslation = 512.0
	maxScale       = 32.0
)

var namePattern = regexp.MustCompile(`^[a-z0-9_.]+$`)

type Issue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type ValidationError struct {
	Issues []Issue
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Issues))

	for _, issue := range e.Issues {
		parts = append(parts, issue.Path+": "+issue.Message)
	}

	return "animation: invalid (" + strings.Join(parts, "; ") + ")"
}

func Normalize(a *Animation, skeleton *rig.Rig) error {
	issues := make([]Issue, 0)

	a.Name = normalizeName(a.Name)

	if !namePattern.MatchString(a.Name) {
		issues = append(issues, Issue{Path: "name", Message: "must match [a-z0-9_.]+"})
	}

	switch a.Loop {
	case LoopNone, LoopCycle, LoopHold:
	case "":
		a.Loop = LoopNone
	default:
		issues = append(issues, Issue{Path: "loop", Message: "unknown loop mode " + string(a.Loop)})
	}

	if a.Bones == nil {
		a.Bones = make(map[string]Track)
	}

	longest := 0.0

	for boneName, track := range a.Bones {
		path := "bones." + boneName

		if skeleton != nil && !skeleton.Has(boneName) {
			issues = append(issues, Issue{Path: path, Message: "bone does not exist in the model rig"})

			delete(a.Bones, boneName)

			continue
		}

		for _, channel := range Channels() {
			frames := track.Channel(channel)

			if len(frames) == 0 {
				continue
			}

			cleaned, channelIssues := normalizeChannel(frames, channel, path+"."+string(channel))
			issues = append(issues, channelIssues...)

			track.SetChannel(channel, cleaned)

			if len(cleaned) > 0 {
				longest = math.Max(longest, cleaned[len(cleaned)-1].Time)
			}
		}

		if track.IsEmpty() {
			delete(a.Bones, boneName)

			continue
		}

		a.Bones[boneName] = track
	}

	if a.Length <= 0 {
		a.Length = longest
	}

	if a.Length < MinLength {
		a.Length = MinLength
	}

	if a.Length > MaxLength {
		issues = append(issues, Issue{Path: "length", Message: fmt.Sprintf("exceeds %.0fs", MaxLength)})
		a.Length = MaxLength
	}

	a.Length = round(a.Length)

	for boneName, track := range a.Bones {
		for _, channel := range Channels() {
			frames := track.Channel(channel)

			for index := range frames {
				if frames[index].Time > a.Length {
					frames[index].Time = a.Length
				}
			}

			track.SetChannel(channel, dedupe(frames))
		}

		a.Bones[boneName] = track
	}

	if len(a.Bones) == 0 {
		issues = append(issues, Issue{Path: "bones", Message: "animation has no usable tracks"})
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}

	return nil
}

func normalizeChannel(frames []Keyframe, channel Channel, path string) ([]Keyframe, []Issue) {
	issues := make([]Issue, 0)

	if len(frames) > MaxKeyframes {
		issues = append(issues, Issue{Path: path, Message: fmt.Sprintf("more than %d keyframes", MaxKeyframes)})
		frames = frames[:MaxKeyframes]
	}

	limit := maxTranslation

	if channel == ChannelScale {
		limit = maxScale
	}

	cleaned := make([]Keyframe, 0, len(frames))

	for index, frame := range frames {
		if math.IsNaN(frame.Time) || math.IsInf(frame.Time, 0) || frame.Time < 0 {
			issues = append(issues, Issue{Path: fmt.Sprintf("%s[%d].time", path, index), Message: "invalid timestamp"})

			continue
		}

		frame.Time = round(frame.Time)

		valid := true

		for axis := 0; axis < 3; axis++ {
			value := frame.Value[axis]

			if math.IsNaN(value) || math.IsInf(value, 0) {
				issues = append(issues, Issue{Path: fmt.Sprintf("%s[%d].value", path, index), Message: "invalid number"})
				valid = false

				break
			}

			if channel != ChannelRotation {
				frame.Value[axis] = math.Max(-limit, math.Min(limit, value))
			}

			frame.Value[axis] = round(frame.Value[axis])
		}

		if !valid {
			continue
		}

		switch frame.Interpolation {
		case InterpolationLinear, InterpolationCatmullRom, InterpolationStep:
		case "":
			frame.Interpolation = InterpolationLinear
		default:
			issues = append(issues, Issue{Path: fmt.Sprintf("%s[%d].interpolation", path, index), Message: "unknown interpolation"})
			frame.Interpolation = InterpolationLinear
		}

		cleaned = append(cleaned, frame)
	}

	sort.SliceStable(cleaned, func(a, b int) bool {
		return cleaned[a].Time < cleaned[b].Time
	})

	return dedupe(cleaned), issues
}

func dedupe(frames []Keyframe) []Keyframe {
	if len(frames) == 0 {
		return frames
	}

	result := frames[:1]

	for _, frame := range frames[1:] {
		if math.Abs(frame.Time-result[len(result)-1].Time) < timeEpsilon {
			result[len(result)-1] = frame

			continue
		}

		result = append(result, frame)
	}

	return result
}

func normalizeName(name string) string {
	lowered := strings.ToLower(strings.TrimSpace(name))
	lowered = strings.ReplaceAll(lowered, " ", "_")
	lowered = strings.ReplaceAll(lowered, "-", "_")

	if lowered == "" {
		return "untitled"
	}

	return lowered
}

func round(value float64) float64 {
	return math.Round(value*10000) / 10000
}
