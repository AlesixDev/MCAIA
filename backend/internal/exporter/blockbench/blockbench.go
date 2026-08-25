package blockbench

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

const (
	formatVersion = "4.10"
	uuidNamespace = "mcaia"
)

type Exporter struct{}

func New() *Exporter {
	return &Exporter{}
}

func (e *Exporter) ID() string {
	return "bbmodel"
}

func (e *Exporter) Label() string {
	return "Blockbench project (.bbmodel)"
}

func (e *Exporter) Extension() string {
	return ".bbmodel"
}

func (e *Exporter) Export(request exporter.Request) (*exporter.Result, error) {
	document, err := baseDocument(request)
	if err != nil {
		return nil, err
	}

	animations := make([]any, 0, len(request.Animations))

	for _, item := range request.Animations {
		animations = append(animations, encode(item, request.Rig))
	}

	document["animations"] = animations

	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("bbmodel: encode: %w", err)
	}

	return &exporter.Result{
		Filename:    slug(request.ModelName) + e.Extension(),
		ContentType: "application/json",
		Data:        data,
	}, nil
}

func baseDocument(request exporter.Request) (map[string]any, error) {
	if request.SourceFormat == "bbmodel" && len(request.Source) > 0 {
		var document map[string]any

		if err := json.Unmarshal(request.Source, &document); err != nil {
			return nil, fmt.Errorf("bbmodel: reuse source: %w", err)
		}

		animatable(document)

		return document, nil
	}

	return synthesize(request.ModelName, request.Rig), nil
}

func synthesize(modelName string, skeleton *rig.Rig) map[string]any {
	elements := make([]any, 0)
	outliner := make([]any, 0, len(skeleton.Roots))

	var build func(name string) any

	build = func(name string) any {
		bone, ok := skeleton.Bones[name]
		if !ok {
			return nil
		}

		children := make([]any, 0, len(bone.Children)+len(bone.Cubes))

		for index, cube := range bone.Cubes {
			id := identifier(name, index)

			elements = append(elements, map[string]any{
				"name":     fallback(cube.Name, name),
				"uuid":     id,
				"type":     "cube",
				"from":     cube.From,
				"to":       cube.To,
				"origin":   cube.Origin,
				"rotation": cube.Rotation,
				"faces":    faces(cube),
			})

			children = append(children, id)
		}

		for _, child := range bone.Children {
			if entry := build(child); entry != nil {
				children = append(children, entry)
			}
		}

		return map[string]any{
			"name":     bone.Name,
			"uuid":     identifier(name, -1),
			"origin":   bone.Origin,
			"rotation": bone.Rotation,
			"children": children,
		}
	}

	for _, root := range skeleton.Roots {
		if entry := build(root); entry != nil {
			outliner = append(outliner, entry)
		}
	}

	width, height := 16, 16

	if skeleton.Resolution[0] > 0 && skeleton.Resolution[1] > 0 {
		width, height = skeleton.Resolution[0], skeleton.Resolution[1]
	}

	textures := make([]any, 0, len(skeleton.Textures))

	for index, texture := range skeleton.Textures {
		textures = append(textures, map[string]any{
			"id":          strconv.Itoa(index),
			"uuid":        identifier("texture-"+texture.Name, index),
			"name":        texture.Name,
			"folder":      "",
			"source":      texture.Source,
			"mode":        "bitmap",
			"width":       texture.Width,
			"height":      texture.Height,
			"render_mode": "default",
		})
	}

	return map[string]any{
		"meta": map[string]any{
			"format_version": formatVersion,
			"model_format":   "free",
			"box_uv":         false,
		},
		"name":       modelName,
		"resolution": map[string]any{"width": width, "height": height},
		"elements":   elements,
		"outliner":   outliner,
		"textures":   textures,
	}
}

func faces(cube rig.Cube) map[string]any {
	if len(cube.Faces) == 0 {
		return blankFaces()
	}

	result := make(map[string]any, len(cube.Faces))

	for _, side := range []string{"north", "east", "south", "west", "up", "down"} {
		face, ok := cube.Faces[side]

		if !ok {
			result[side] = map[string]any{"uv": []float64{0, 0, 0, 0}, "texture": nil}

			continue
		}

		entry := map[string]any{
			"uv":      []float64{face.UV[0], face.UV[1], face.UV[2], face.UV[3]},
			"texture": face.Texture,
		}

		if face.Rotation != 0 {
			entry["rotation"] = face.Rotation
		}

		result[side] = entry
	}

	return result
}

func animatable(document map[string]any) {
	animated := map[string]bool{
		"free":          true,
		"bedrock":       true,
		"bedrock_old":   true,
		"modded_entity": true,
		"animated_java": true,
	}

	meta, ok := document["meta"].(map[string]any)
	if !ok {
		meta = make(map[string]any, 2)
		document["meta"] = meta
	}

	format, _ := meta["model_format"].(string)

	if animated[format] {
		return
	}

	meta["model_format"] = "free"
}

func encode(item animation.Animation, skeleton *rig.Rig) map[string]any {
	entry := map[string]any{
		"uuid":             identifier("animation-"+item.Name, -1),
		"name":             "animation." + item.Name,
		"loop":             loopMode(item.Loop),
		"override":         false,
		"length":           item.Length,
		"snapping":         24,
		"selected":         false,
		"anim_time_update": "",
		"blend_weight":     "",
		"start_delay":      "",
		"loop_delay":       "",
		"animators":        animators(item, skeleton),
	}

	return entry
}

func animators(item animation.Animation, skeleton *rig.Rig) map[string]any {
	result := make(map[string]any, len(item.Bones))

	for bone, track := range item.Bones {
		keyframes := make([]any, 0)

		for _, channel := range animation.Channels() {
			for index, frame := range track.Channel(channel) {
				keyframes = append(keyframes, map[string]any{
					"channel":       string(channel),
					"data_points":   []any{point(frame)},
					"uuid":          identifier(bone+string(channel), index),
					"time":          frame.Time,
					"color":         -1,
					"interpolation": interpolation(frame.Interpolation),
				})
			}
		}

		result[groupID(bone, skeleton)] = map[string]any{
			"name":      bone,
			"type":      "bone",
			"keyframes": keyframes,
		}
	}

	return result
}

func groupID(bone string, skeleton *rig.Rig) string {
	if skeleton != nil {
		if entry, ok := skeleton.Bones[bone]; ok && entry.UUID != "" {
			return entry.UUID
		}
	}

	return identifier(bone, -1)
}

func point(frame animation.Keyframe) map[string]any {
	return map[string]any{
		"x": formatNumber(frame.Value[0]),
		"y": formatNumber(frame.Value[1]),
		"z": formatNumber(frame.Value[2]),
	}
}

func formatNumber(value float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.4f", value), "0"), ".")
}

func interpolation(mode animation.Interpolation) string {
	switch mode {
	case animation.InterpolationCatmullRom:
		return "catmullrom"
	case animation.InterpolationStep:
		return "step"
	}

	return "linear"
}

func loopMode(mode animation.LoopMode) string {
	switch mode {
	case animation.LoopCycle:
		return "loop"
	case animation.LoopHold:
		return "hold"
	}

	return "once"
}

func blankFaces() map[string]any {
	faces := make(map[string]any, 6)

	for _, side := range []string{"north", "east", "south", "west", "up", "down"} {
		faces[side] = map[string]any{"uv": []float64{0, 0, 16, 16}, "texture": nil}
	}

	return faces
}

func identifier(seed string, index int) string {
	hash := uint64(1469598103934665603)

	full := uuidNamespace + seed + fmt.Sprint(index)

	for position := 0; position < len(full); position++ {
		hash ^= uint64(full[position])
		hash *= 1099511628211
	}

	first := fmt.Sprintf("%016x", hash)

	hash ^= uint64(len(full))
	hash *= 1099511628211

	second := fmt.Sprintf("%016x", hash)

	return fmt.Sprintf("%s-%s-%s-%s-%s",
		first[0:8], first[8:12], first[12:16], second[0:4], second[4:16])
}

func fallback(value, other string) string {
	if strings.TrimSpace(value) == "" {
		return other
	}

	return value
}

func slug(value string) string {
	lowered := strings.ToLower(strings.TrimSpace(value))
	lowered = strings.ReplaceAll(lowered, " ", "_")

	if lowered == "" {
		return "model"
	}

	return lowered
}
