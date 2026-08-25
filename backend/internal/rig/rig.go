package rig

import (
	"sort"
	"strconv"
	"strings"
)

type Vec3 [3]float64

type BoundingBox struct {
	Min Vec3 `json:"min"`
	Max Vec3 `json:"max"`
}

type Face struct {
	UV       [4]float64 `json:"uv"`
	Texture  int        `json:"texture"`
	Rotation float64    `json:"rotation,omitempty"`
}

type Texture struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type Cube struct {
	Name     string          `json:"name"`
	UUID     string          `json:"uuid,omitempty"`
	From     Vec3            `json:"from"`
	To       Vec3            `json:"to"`
	Origin   Vec3            `json:"origin"`
	Rotation Vec3            `json:"rotation"`
	Inflate  float64         `json:"inflate,omitempty"`
	Faces    map[string]Face `json:"faces,omitempty"`
}

type Bone struct {
	Name     string       `json:"name"`
	UUID     string       `json:"uuid,omitempty"`
	Parent   string       `json:"parent,omitempty"`
	Origin   Vec3         `json:"origin"`
	Rotation Vec3         `json:"rotation"`
	Depth    int          `json:"depth"`
	Children []string     `json:"children,omitempty"`
	Cubes    []Cube       `json:"cubes,omitempty"`
	Bounds   *BoundingBox `json:"bounds,omitempty"`
}

type Rig struct {
	ModelName  string          `json:"model_name"`
	Format     string          `json:"format"`
	Roots      []string        `json:"roots"`
	Bones      map[string]Bone `json:"bones"`
	Order      []string        `json:"order"`
	Textures   []Texture       `json:"textures,omitempty"`
	Resolution [2]int          `json:"resolution,omitempty"`
}

type Builder struct {
	rig *Rig
}

func NewBuilder(modelName, format string) *Builder {
	return &Builder{
		rig: &Rig{
			ModelName: modelName,
			Format:    format,
			Bones:     make(map[string]Bone),
		},
	}
}

func (b *Builder) Add(bone Bone) string {
	name := b.unique(normalize(bone.Name))

	bone.Name = name
	bone.Depth = 0

	if bone.Parent != "" {
		parent, ok := b.rig.Bones[bone.Parent]
		if !ok {
			bone.Parent = ""
		} else {
			bone.Depth = parent.Depth + 1

			parent.Children = append(parent.Children, name)
			b.rig.Bones[bone.Parent] = parent
		}
	}

	if bone.Parent == "" {
		b.rig.Roots = append(b.rig.Roots, name)
	}

	b.rig.Bones[name] = bone
	b.rig.Order = append(b.rig.Order, name)

	return name
}

func (b *Builder) Rig() *Rig {
	return b.rig
}

func (b *Builder) Empty() bool {
	return len(b.rig.Bones) == 0
}

func (b *Builder) unique(name string) string {
	if _, taken := b.rig.Bones[name]; !taken {
		return name
	}

	for index := 2; ; index++ {
		candidate := name + "_" + strconv.Itoa(index)

		if _, taken := b.rig.Bones[candidate]; !taken {
			return candidate
		}
	}
}

func normalize(name string) string {
	trimmed := strings.TrimSpace(name)

	if trimmed == "" {
		return "bone"
	}

	lowered := strings.ToLower(trimmed)
	replacer := strings.NewReplacer(" ", "_", ".", "_", "/", "_", "|", "_")

	return replacer.Replace(lowered)
}

func Expand(box *BoundingBox, point Vec3) *BoundingBox {
	if box == nil {
		return &BoundingBox{Min: point, Max: point}
	}

	for axis := 0; axis < 3; axis++ {
		box.Min[axis] = min(box.Min[axis], point[axis])
		box.Max[axis] = max(box.Max[axis], point[axis])
	}

	return box
}

func (b *BoundingBox) Center() Vec3 {
	return Vec3{
		(b.Min[0] + b.Max[0]) / 2,
		(b.Min[1] + b.Max[1]) / 2,
		(b.Min[2] + b.Max[2]) / 2,
	}
}

func (r *Rig) Has(name string) bool {
	_, ok := r.Bones[name]

	return ok
}

func (r *Rig) Names() []string {
	names := make([]string, 0, len(r.Bones))

	for name := range r.Bones {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
