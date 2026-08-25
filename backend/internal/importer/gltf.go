package importer

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

const (
	glbMagic     = 0x46546C67
	glbChunkJSON = 0x4E4F534A
	glbChunkBIN  = 0x004E4942
	glbHeader    = 12
	glbChunkHead = 8
)

type gltfNode struct {
	Name        string       `json:"name"`
	Children    []int        `json:"children"`
	Translation *[3]float64  `json:"translation"`
	Rotation    *[4]float64  `json:"rotation"`
	Scale       *[3]float64  `json:"scale"`
	Matrix      *[16]float64 `json:"matrix"`
	Mesh        *int         `json:"mesh"`
	Skin        *int         `json:"skin"`
}

type gltfScene struct {
	Nodes []int `json:"nodes"`
}

type gltfAsset struct {
	Version   string `json:"version"`
	Generator string `json:"generator"`
}

type gltfPrimitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices"`
	Material   *int           `json:"material"`
	Mode       *int           `json:"mode"`
}

type gltfMesh struct {
	Name       string          `json:"name"`
	Primitives []gltfPrimitive `json:"primitives"`
}

type gltfAccessor struct {
	BufferView    *int   `json:"bufferView"`
	ByteOffset    int    `json:"byteOffset"`
	ComponentType int    `json:"componentType"`
	Normalized    bool   `json:"normalized"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
}

type gltfBufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
	ByteStride int `json:"byteStride"`
}

type gltfBuffer struct {
	URI        string `json:"uri"`
	ByteLength int    `json:"byteLength"`
}

type gltfImage struct {
	Name       string `json:"name"`
	URI        string `json:"uri"`
	MimeType   string `json:"mimeType"`
	BufferView *int   `json:"bufferView"`
}

type gltfTexture struct {
	Name   string `json:"name"`
	Source *int   `json:"source"`
}

type gltfMaterial struct {
	Name string `json:"name"`
	PBR  *struct {
		BaseColorTexture *struct {
			Index int `json:"index"`
		} `json:"baseColorTexture"`
	} `json:"pbrMetallicRoughness"`
}

type gltfDocument struct {
	Asset       gltfAsset        `json:"asset"`
	Scene       *int             `json:"scene"`
	Scenes      []gltfScene      `json:"scenes"`
	Nodes       []gltfNode       `json:"nodes"`
	Meshes      []gltfMesh       `json:"meshes"`
	Accessors   []gltfAccessor   `json:"accessors"`
	BufferViews []gltfBufferView `json:"bufferViews"`
	Buffers     []gltfBuffer     `json:"buffers"`
	Images      []gltfImage      `json:"images"`
	Textures    []gltfTexture    `json:"textures"`
	Materials   []gltfMaterial   `json:"materials"`
}

type GLTF struct{}

func NewGLTF() *GLTF {
	return &GLTF{}
}

func (g *GLTF) ID() string {
	return "gltf"
}

func (g *GLTF) Label() string {
	return "glTF / GLB (.gltf, .glb)"
}

func (g *GLTF) Extensions() []string {
	return []string{".gltf", ".glb"}
}

func (g *GLTF) Detect(data []byte, filename string) bool {
	if HasExtension(filename, ".gltf", ".glb") {
		return true
	}

	if len(data) >= 4 && binary.LittleEndian.Uint32(data[:4]) == glbMagic {
		return true
	}

	return bytes.Contains(data, []byte("\"asset\"")) && bytes.Contains(data, []byte("\"nodes\""))
}

func (g *GLTF) Import(request Request) (*Result, error) {
	data, filename := request.Data, request.Filename

	payload, err := jsonChunk(data)
	if err != nil {
		return nil, err
	}

	var document gltfDocument

	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("gltf: decode: %w", err)
	}

	if len(document.Nodes) == 0 {
		return nil, ErrEmptyModel
	}

	name := BaseName(filename, "model")
	builder := rig.NewBuilder(name, g.ID())

	reader := newMeshReader(&document, data, request)
	textures, materials, problems := reader.readTextures(request)

	walk := &nodeWalk{
		builder:  builder,
		document: &document,
		reader:   reader,
		textures: materials,
		visited:  make(map[int]bool, len(document.Nodes)),
		cubes:    make(map[string][]rig.Cube),
		bounds:   make(map[string]*rig.BoundingBox),
		pivots:   make(map[string]rig.Vec3),
	}

	if len(textures) > 0 {
		walk.width = float64(textures[0].Width)
		walk.height = float64(textures[0].Height)
	}

	for _, index := range rootNodes(&document) {
		walk.walkNode(index, "", rig.Vec3{})
	}

	if builder.Empty() {
		return nil, ErrEmptyModel
	}

	skeleton := builder.Rig()

	for boneName, cubes := range walk.cubes {
		bone, ok := skeleton.Bones[boneName]

		if !ok {
			continue
		}

		bone.Cubes = append(bone.Cubes, cubes...)
		bone.Bounds = walk.bounds[boneName]

		if pivot, corrected := walk.pivots[boneName]; corrected {
			bone.Origin = pivot
		}

		skeleton.Bones[boneName] = bone
	}

	notes := []string{
		"Pivots come from the glTF node translations, in the units of the file itself.",
	}

	if hasSkin(&document) {
		notes = append(notes, "This model carries a skin: the animatable bones are the skeleton nodes.")
	}

	if NormalizeScale(skeleton) {
		notes = append(notes, "The file was in block units, so it was scaled to Blockbench pixels (x16).")
	}

	skeleton.Textures = textures

	if len(textures) > 0 {
		skeleton.Resolution = [2]int{textures[0].Width, textures[0].Height}
		notes = append(notes, "Textures came embedded in the glTF.")
	}

	notes = append(notes, problems...)

	return &Result{Name: name, Format: g.ID(), Rig: skeleton, Notes: notes}, nil
}

func jsonChunk(data []byte) ([]byte, error) {
	if len(data) < 4 || binary.LittleEndian.Uint32(data[:4]) != glbMagic {
		return data, nil
	}

	if len(data) < glbHeader+glbChunkHead {
		return nil, fmt.Errorf("gltf: truncated glb container")
	}

	offset := glbHeader

	for offset+glbChunkHead <= len(data) {
		length := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		start := offset + glbChunkHead
		end := start + length

		if end > len(data) {
			return nil, fmt.Errorf("gltf: chunk out of bounds")
		}

		if kind == glbChunkJSON {
			return data[start:end], nil
		}

		offset = end
	}

	return nil, fmt.Errorf("gltf: no json chunk in glb")
}

func binChunk(data []byte) ([]byte, bool) {
	if len(data) < glbHeader+glbChunkHead || binary.LittleEndian.Uint32(data[:4]) != glbMagic {
		return nil, false
	}

	offset := glbHeader

	for offset+glbChunkHead <= len(data) {
		length := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		start := offset + glbChunkHead
		end := start + length

		if end > len(data) {
			return nil, false
		}

		if kind == glbChunkBIN {
			return data[start:end], true
		}

		offset = end
	}

	return nil, false
}

func rootNodes(document *gltfDocument) []int {
	index := 0

	if document.Scene != nil {
		index = *document.Scene
	}

	if index >= 0 && index < len(document.Scenes) && len(document.Scenes[index].Nodes) > 0 {
		return document.Scenes[index].Nodes
	}

	child := make(map[int]bool)

	for _, node := range document.Nodes {
		for _, reference := range node.Children {
			child[reference] = true
		}
	}

	roots := make([]int, 0)

	for position := range document.Nodes {
		if !child[position] {
			roots = append(roots, position)
		}
	}

	return roots
}

func hasSkin(document *gltfDocument) bool {
	for _, node := range document.Nodes {
		if node.Skin != nil {
			return true
		}
	}

	return false
}

type nodeWalk struct {
	builder  *rig.Builder
	document *gltfDocument
	reader   *meshReader
	textures map[int]int
	visited  map[int]bool
	cubes    map[string][]rig.Cube
	bounds   map[string]*rig.BoundingBox
	pivots   map[string]rig.Vec3
	width    float64
	height   float64
}

func (w *nodeWalk) walkNode(index int, parent string, offset rig.Vec3) {
	if index < 0 || index >= len(w.document.Nodes) || w.visited[index] {
		return
	}

	w.visited[index] = true

	node := w.document.Nodes[index]
	local := translation(node)

	origin := rig.Vec3{offset[0] + local[0], offset[1] + local[1], offset[2] + local[2]}
	label := nodeLabel(w.document, node, index)

	if node.Mesh != nil && len(node.Children) == 0 && parent != "" && boneName(label) == parent {
		w.attach(parent, *node.Mesh, origin, label, offset)

		return
	}

	name := w.builder.Add(rig.Bone{
		Name:     label,
		Parent:   parent,
		Origin:   origin,
		Rotation: euler(node.Rotation),
	})

	if node.Mesh != nil {
		w.attach(name, *node.Mesh, origin, label, offset)
	}

	for _, child := range node.Children {
		w.walkNode(child, name, origin)
	}
}

func (w *nodeWalk) attach(bone string, mesh int, offset rig.Vec3, label string, parentOrigin rig.Vec3) {
	if w.reader == nil {
		return
	}

	points, polygons := w.reader.geometry(mesh, offset, w.textures)

	if len(points) == 0 {
		return
	}

	box, oriented := Cuboid(points)

	if !oriented {
		bounds := (*rig.BoundingBox)(nil)

		for _, point := range points {
			bounds = rig.Expand(bounds, point)
		}

		box.From = bounds.Min
		box.To = bounds.Max
		box.Origin = bounds.Center()
	}

	cube := rig.Cube{
		Name:     label,
		From:     box.From,
		To:       box.To,
		Origin:   box.Origin,
		Rotation: box.Rotation,
	}

	if width, height, ok := w.resolution(); ok {
		cube.Faces = MapFaces(polygons, box.Axes, width, height, UVTopLeft)
	}

	w.cubes[bone] = append(w.cubes[bone], cube)

	for _, point := range points {
		w.bounds[bone] = rig.Expand(w.bounds[bone], point)
	}

	if isZero(offset) || !inside(offset, box.From, box.To) {
		w.pivots[bone] = parentOrigin
	}
}

func isZero(point rig.Vec3) bool {
	return point[0] == 0 && point[1] == 0 && point[2] == 0
}

func boneName(label string) string {
	trimmed := strings.TrimSpace(label)

	if trimmed == "" {
		return "bone"
	}

	replacer := strings.NewReplacer(" ", "_", ".", "_", "/", "_", "|", "_")

	return replacer.Replace(strings.ToLower(trimmed))
}

func inside(point, from, to rig.Vec3) bool {
	for axis := 0; axis < 3; axis++ {
		low := math.Min(from[axis], to[axis])
		high := math.Max(from[axis], to[axis])

		if point[axis] < low || point[axis] > high {
			return false
		}
	}

	return true
}

func (w *nodeWalk) resolution() (float64, float64, bool) {
	return w.width, w.height, w.width > 0 && w.height > 0
}

func nodeLabel(document *gltfDocument, node gltfNode, index int) string {
	if node.Name != "" {
		return node.Name
	}

	if node.Mesh != nil && *node.Mesh < len(document.Meshes) && document.Meshes[*node.Mesh].Name != "" {
		return document.Meshes[*node.Mesh].Name
	}

	return fmt.Sprintf("node_%d", index)
}

func translation(node gltfNode) rig.Vec3 {
	if node.Translation != nil {
		return rig.Vec3(*node.Translation)
	}

	if node.Matrix != nil {
		return rig.Vec3{node.Matrix[12], node.Matrix[13], node.Matrix[14]}
	}

	return rig.Vec3{}
}

func euler(quaternion *[4]float64) rig.Vec3 {
	if quaternion == nil {
		return rig.Vec3{}
	}

	x, y, z, w := quaternion[0], quaternion[1], quaternion[2], quaternion[3]

	sinPitch := 2 * (w*x + y*z)
	cosPitch := 1 - 2*(x*x+y*y)
	pitch := math.Atan2(sinPitch, cosPitch)

	sinYaw := 2 * (w*y - z*x)
	yaw := math.Asin(math.Max(-1, math.Min(1, sinYaw)))

	sinRoll := 2 * (w*z + x*y)
	cosRoll := 1 - 2*(y*y+z*z)
	roll := math.Atan2(sinRoll, cosRoll)

	return rig.Vec3{
		round(pitch * 180 / math.Pi),
		round(yaw * 180 / math.Pi),
		round(roll * 180 / math.Pi),
	}
}

func round(value float64) float64 {
	return math.Round(value*1000) / 1000
}
