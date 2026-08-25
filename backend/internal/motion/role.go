package motion

import (
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

type Part string

const (
	PartHead    Part = "head"
	PartBody    Part = "body"
	PartArm     Part = "arm"
	PartLeg     Part = "leg"
	PartTail    Part = "tail"
	PartWing    Part = "wing"
	PartUnknown Part = "other"
)

type Side string

const (
	SideLeft   Side = "left"
	SideRight  Side = "right"
	SideCentre Side = "centre"
)

type Role struct {
	Bone string `json:"bone"`
	Part Part   `json:"part"`
	Side Side   `json:"side"`
}

var partWords = []struct {
	part  Part
	words []string
}{
	{PartHead, []string{"head", "skull", "neck", "jaw", "snout", "beak"}},
	{PartArm, []string{"arm", "hand", "claw", "shoulder", "fist", "paw"}},
	{PartLeg, []string{"leg", "foot", "feet", "thigh", "shin", "knee", "hoof"}},
	{PartTail, []string{"tail"}},
	{PartWing, []string{"wing", "fin"}},
	{PartBody, []string{"body", "torso", "chest", "waist", "hip", "spine", "root", "base"}},
}

var leftWords = []string{"left", "_l", "l_", ".l", "-l", "izquierda"}

var rightWords = []string{"right", "_r", "r_", ".r", "-r", "derecha"}

func Classify(skeleton *rig.Rig) map[string]Role {
	roles := make(map[string]Role, len(skeleton.Bones))

	for name, bone := range skeleton.Bones {
		roles[name] = Role{
			Bone: name,
			Part: detectPart(name, bone, skeleton),
			Side: detectSide(name, bone),
		}
	}

	return roles
}

func detectPart(name string, bone rig.Bone, skeleton *rig.Rig) Part {
	lowered := strings.ToLower(name)

	for _, entry := range partWords {
		for _, word := range entry.words {
			if strings.Contains(lowered, word) {
				return entry.part
			}
		}
	}

	if bone.Parent == "" {
		return PartBody
	}

	if bone.Bounds != nil {
		height := bone.Bounds.Max[1] - bone.Bounds.Min[1]
		width := bone.Bounds.Max[0] - bone.Bounds.Min[0]
		depth := bone.Bounds.Max[2] - bone.Bounds.Min[2]

		tallest := 0.0

		for _, other := range skeleton.Bones {
			if other.Bounds != nil {
				tallest = max(tallest, other.Bounds.Max[1])
			}
		}

		if height > width*1.6 && height > depth*1.6 {
			if bone.Origin[1] < tallest*0.45 {
				return PartLeg
			}

			return PartArm
		}
	}

	return PartUnknown
}

func detectSide(name string, bone rig.Bone) Side {
	lowered := strings.ToLower(name)

	for _, word := range rightWords {
		if strings.Contains(lowered, word) {
			return SideRight
		}
	}

	for _, word := range leftWords {
		if strings.Contains(lowered, word) {
			return SideLeft
		}
	}

	if bone.Origin[0] > 0.5 {
		return SideRight
	}

	if bone.Origin[0] < -0.5 {
		return SideLeft
	}

	return SideCentre
}

func Resolve(roles map[string]Role, target string) []string {
	lowered := strings.ToLower(strings.TrimSpace(target))

	if _, ok := roles[lowered]; ok {
		return []string{lowered}
	}

	part, side := groupTarget(lowered)

	if part == "" {
		return nil
	}

	matched := make([]string, 0)

	for name, role := range roles {
		if role.Part != part {
			continue
		}

		if side != "" && role.Side != side {
			continue
		}

		matched = append(matched, name)
	}

	return matched
}

func groupTarget(target string) (Part, Side) {
	side := Side("")

	switch {
	case strings.HasPrefix(target, "left_"), strings.HasPrefix(target, "left "):
		side = SideLeft
		target = target[5:]
	case strings.HasPrefix(target, "right_"), strings.HasPrefix(target, "right "):
		side = SideRight
		target = target[6:]
	}

	switch strings.TrimSuffix(target, "s") {
	case "head":
		return PartHead, side
	case "body", "torso", "chest":
		return PartBody, side
	case "arm":
		return PartArm, side
	case "leg":
		return PartLeg, side
	case "tail":
		return PartTail, side
	case "wing":
		return PartWing, side
	}

	return "", side
}
