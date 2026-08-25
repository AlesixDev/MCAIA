package importer

import (
	"bytes"
	"math"

	"github.com/AlesixDev/MCAIA/backend/internal/bbmodel"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

type Blockbench struct{}

func NewBlockbench() *Blockbench {
	return &Blockbench{}
}

func (b *Blockbench) ID() string {
	return "bbmodel"
}

func (b *Blockbench) Label() string {
	return "Blockbench (.bbmodel)"
}

func (b *Blockbench) Extensions() []string {
	return []string{".bbmodel"}
}

func (b *Blockbench) Detect(data []byte, filename string) bool {
	if HasExtension(filename, ".bbmodel") {
		return true
	}

	return bytes.Contains(data, []byte("\"outliner\"")) && bytes.Contains(data, []byte("\"elements\""))
}

func (b *Blockbench) Import(request Request) (*Result, error) {
	data, filename := request.Data, request.Filename

	project, err := bbmodel.ParseBytes(data)
	if err != nil {
		return nil, err
	}

	name := project.Name

	if name == "" || name == "model" {
		name = BaseName(filename, "model")
	}

	elements := make(map[string]bbmodel.Element, len(project.Elements))

	for _, element := range project.Elements {
		elements[element.UUID] = element
	}

	textures := make(map[string]int, len(project.Textures))

	for index, texture := range project.Textures {
		textures[texture.UUID] = index
	}

	builder := rig.NewBuilder(name, b.ID())

	for _, entry := range project.Outliner {
		if entry.Group != nil {
			walkGroup(builder, entry.Group, "", elements, textures, project.Meta.BoxUV)
		}
	}

	if builder.Empty() {
		return nil, ErrEmptyModel
	}

	skeleton := builder.Rig()
	skeleton.Resolution = [2]int{project.Resolution.Width, project.Resolution.Height}

	for _, texture := range project.Textures {
		skeleton.Textures = append(skeleton.Textures, rig.Texture{
			Name:   texture.Name,
			Source: texture.Source,
			Width:  texture.Width,
			Height: texture.Height,
		})
	}

	return &Result{Name: name, Format: b.ID(), Rig: skeleton}, nil
}

func faces(
	element bbmodel.Element,
	textures map[string]int,
	projectBoxUV bool,
) map[string]rig.Face {
	boxUV := projectBoxUV

	if element.BoxUV != nil {
		boxUV = *element.BoxUV
	}

	if boxUV {
		return boxFaces(element)
	}

	if len(element.Faces) == 0 {
		return nil
	}

	mapped := make(map[string]rig.Face, len(element.Faces))

	for side, face := range element.Faces {
		index := face.Texture.Index

		if index < 0 && face.Texture.UUID != "" {
			found, ok := textures[face.Texture.UUID]

			if !ok {
				continue
			}

			index = found
		}

		if index < 0 {
			continue
		}

		mapped[side] = rig.Face{UV: face.UV, Texture: index, Rotation: face.Rotation}
	}

	if len(mapped) == 0 {
		return nil
	}

	return mapped
}

func boxFaces(element bbmodel.Element) map[string]rig.Face {
	x := element.UVOffset[0]
	y := element.UVOffset[1]

	width := math.Round(element.To[0] - element.From[0])
	height := math.Round(element.To[1] - element.From[1])
	depth := math.Round(element.To[2] - element.From[2])

	if width == 0 && height == 0 && depth == 0 {
		return nil
	}

	return map[string]rig.Face{
		"up":    {UV: [4]float64{x + depth + width, y + depth, x + depth, y}},
		"down":  {UV: [4]float64{x + depth + 2*width, y + depth, x + depth + width, y}},
		"east":  {UV: [4]float64{x, y + depth, x + depth, y + depth + height}},
		"north": {UV: [4]float64{x + depth, y + depth, x + depth + width, y + depth + height}},
		"west":  {UV: [4]float64{x + depth + width, y + depth, x + 2*depth + width, y + depth + height}},
		"south": {UV: [4]float64{x + 2*depth + width, y + depth, x + 2*depth + 2*width, y + depth + height}},
	}
}

func walkGroup(
	builder *rig.Builder,
	node *bbmodel.OutlinerNode,
	parent string,
	elements map[string]bbmodel.Element,
	textures map[string]int,
	boxUV bool,
) {
	var box *rig.BoundingBox

	cubes := make([]rig.Cube, 0)

	for _, child := range node.Children {
		if child.Group != nil {
			continue
		}

		element, ok := elements[child.Ref]
		if !ok {
			continue
		}

		cubes = append(cubes, rig.Cube{
			Name:     element.Name,
			UUID:     element.UUID,
			From:     rig.Vec3(element.From),
			To:       rig.Vec3(element.To),
			Origin:   rig.Vec3(element.Origin),
			Rotation: rig.Vec3(element.Rotation),
			Inflate:  element.Inflate,
			Faces:    faces(element, textures, boxUV),
		})

		box = rig.Expand(box, rig.Vec3(element.From))
		box = rig.Expand(box, rig.Vec3(element.To))
	}

	name := builder.Add(rig.Bone{
		Name:     node.Name,
		UUID:     node.UUID,
		Parent:   parent,
		Origin:   rig.Vec3(node.Origin),
		Rotation: rig.Vec3(node.Rotation),
		Cubes:    cubes,
		Bounds:   box,
	})

	for _, child := range node.Children {
		if child.Group != nil {
			walkGroup(builder, child.Group, name, elements, textures, boxUV)
		}
	}
}
