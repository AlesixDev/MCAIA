package importer

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

type OBJ struct{}

func NewOBJ() *OBJ {
	return &OBJ{}
}

func (o *OBJ) ID() string {
	return "obj"
}

func (o *OBJ) Label() string {
	return "Wavefront OBJ (.obj)"
}

func (o *OBJ) Extensions() []string {
	return []string{".obj"}
}

func (o *OBJ) Detect(data []byte, filename string) bool {
	if HasExtension(filename, ".obj") {
		return true
	}

	head := data

	if len(head) > 4096 {
		head = head[:4096]
	}

	return bytes.Contains(head, []byte("\nv ")) || bytes.HasPrefix(head, []byte("v "))
}

type objFace struct {
	corners  []int
	uvs      []int
	material string
}

type objGroup struct {
	name   string
	bounds *rig.BoundingBox
	points []rig.Vec3
	faces  []objFace
}

type objDocument struct {
	vertices  []rig.Vec3
	texcoords [][2]float64
	groups    []*objGroup
	libraries []string
}

func (o *OBJ) Import(request Request) (*Result, error) {
	document, err := parseOBJ(request.Data)
	if err != nil {
		return nil, err
	}

	name := BaseName(request.Filename, "model")
	builder := rig.NewBuilder(name, o.ID())

	materials, problems := readMaterials(document.libraries, request)

	for _, group := range document.groups {
		origin := rig.Vec3{}

		if group.bounds != nil {
			origin = group.bounds.Center()
		}

		cubes := make([]rig.Cube, 0, 1)

		if group.bounds != nil {
			cube := rig.Cube{
				Name:   group.name,
				From:   group.bounds.Min,
				To:     group.bounds.Max,
				Origin: origin,
			}

			box, oriented := Cuboid(group.points)

			if oriented {
				cube.From = box.From
				cube.To = box.To
				cube.Origin = box.Origin
				cube.Rotation = box.Rotation
				origin = box.Origin
			}

			if len(materials.textures) > 0 {
				cube.Faces = MapFaces(
					polygons(group, document, materials),
					box.Axes,
					float64(materials.textures[0].Width),
					float64(materials.textures[0].Height),
					UVBottomLeft,
				)
			}

			cubes = append(cubes, cube)
		}

		builder.Add(rig.Bone{
			Name:   group.name,
			Origin: origin,
			Cubes:  cubes,
			Bounds: group.bounds,
		})
	}

	if builder.Empty() {
		return nil, ErrEmptyModel
	}

	notes := []string{
		"OBJ stores no hierarchy or pivots: every group becomes a loose bone pivoted at the centre of its geometry.",
	}

	skeleton := builder.Rig()

	if NormalizeScale(skeleton) {
		notes = append(notes, "The file was in block units, so it was scaled to Blockbench pixels (x16).")
	}

	skeleton.Textures = materials.textures

	if len(materials.textures) > 0 {
		skeleton.Resolution = [2]int{materials.textures[0].Width, materials.textures[0].Height}
		notes = append(notes, "Textures came from the material library next to the model.")
	}

	notes = append(notes, problems...)

	return &Result{Name: name, Format: o.ID(), Rig: skeleton, Notes: notes}, nil
}

func parseOBJ(data []byte) (*objDocument, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)

	document := &objDocument{}
	current := (*objGroup)(nil)
	material := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)

		switch fields[0] {
		case "mtllib":
			if len(fields) > 1 {
				document.libraries = append(document.libraries, strings.Join(fields[1:], " "))
			}

		case "usemtl":
			if len(fields) > 1 {
				material = fields[1]
			}

		case "o", "g":
			label := "group"

			if len(fields) > 1 {
				label = strings.Join(fields[1:], "_")
			}

			current = &objGroup{name: label}
			document.groups = append(document.groups, current)

		case "v":
			if len(fields) < 4 {
				continue
			}

			point, ok := vertex(fields[1:4])
			if !ok {
				continue
			}

			document.vertices = append(document.vertices, point)

			if current == nil {
				continue
			}

			current.bounds = rig.Expand(current.bounds, point)
			current.points = append(current.points, point)

		case "vt":
			if len(fields) < 3 {
				continue
			}

			u, uErr := strconv.ParseFloat(fields[1], 64)
			v, vErr := strconv.ParseFloat(fields[2], 64)

			if uErr != nil || vErr != nil {
				continue
			}

			document.texcoords = append(document.texcoords, [2]float64{u, v})

		case "f":
			if current == nil || len(fields) < 4 {
				continue
			}

			face := objFace{material: material}

			for _, field := range fields[1:] {
				corner, uv := reference(field, len(document.vertices), len(document.texcoords))

				if corner < 0 {
					face.corners = nil

					break
				}

				face.corners = append(face.corners, corner)
				face.uvs = append(face.uvs, uv)
			}

			if len(face.corners) >= 3 {
				current.faces = append(current.faces, face)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return document, nil
}

func reference(field string, vertices, texcoords int) (int, int) {
	parts := strings.Split(field, "/")

	resolve := func(raw string, total int) int {
		if raw == "" {
			return -1
		}

		value, err := strconv.Atoi(raw)
		if err != nil || value == 0 {
			return -1
		}

		if value < 0 {
			value = total + value + 1
		}

		if value < 1 || value > total {
			return -1
		}

		return value - 1
	}

	corner := resolve(parts[0], vertices)
	uv := -1

	if len(parts) > 1 {
		uv = resolve(parts[1], texcoords)
	}

	return corner, uv
}

func polygons(group *objGroup, document *objDocument, materials *materialLibrary) []Polygon {
	result := make([]Polygon, 0, len(group.faces))

	for _, face := range group.faces {
		texture := materials.index(face.material)

		if texture < 0 {
			continue
		}

		polygon := Polygon{Texture: texture}

		for index, corner := range face.corners {
			polygon.Corners = append(polygon.Corners, document.vertices[corner])

			uv := face.uvs[index]

			if uv < 0 || uv >= len(document.texcoords) {
				continue
			}

			polygon.UVs = append(polygon.UVs, document.texcoords[uv])
		}

		if len(polygon.Corners) < 3 || len(polygon.UVs) == 0 {
			continue
		}

		result = append(result, polygon)
	}

	return result
}

func vertex(fields []string) (rig.Vec3, bool) {
	point := rig.Vec3{}

	for index := 0; index < 3; index++ {
		value, err := strconv.ParseFloat(fields[index], 64)
		if err != nil {
			return point, false
		}

		point[index] = value
	}

	return point, true
}
