package importer

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

const (
	componentByte          = 5120
	componentUnsignedByte  = 5121
	componentShort         = 5122
	componentUnsignedShort = 5123
	componentUnsignedInt   = 5125
	componentFloat         = 5126

	modeTriangles = 4
)

var componentSize = map[int]int{
	componentByte:          1,
	componentUnsignedByte:  1,
	componentShort:         2,
	componentUnsignedShort: 2,
	componentUnsignedInt:   4,
	componentFloat:         4,
}

var typeCount = map[string]int{
	"SCALAR": 1,
	"VEC2":   2,
	"VEC3":   3,
	"VEC4":   4,
	"MAT4":   16,
}

type meshReader struct {
	document *gltfDocument
	buffers  [][]byte
}

func newMeshReader(document *gltfDocument, data []byte, request Request) *meshReader {
	reader := &meshReader{document: document}

	binaryChunk, _ := binChunk(data)

	for _, buffer := range document.Buffers {
		reader.buffers = append(reader.buffers, resolveBuffer(buffer.URI, binaryChunk, request))
	}

	return reader
}

func resolveBuffer(uri string, binaryChunk []byte, request Request) []byte {
	if uri == "" {
		return binaryChunk
	}

	if decoded, ok := decodeDataURI(uri); ok {
		return decoded
	}

	if asset, ok := request.Asset(uri); ok {
		return asset
	}

	return nil
}

func decodeDataURI(uri string) ([]byte, bool) {
	if !strings.HasPrefix(uri, "data:") {
		return nil, false
	}

	comma := strings.Index(uri, ",")

	if comma < 0 {
		return nil, false
	}

	if !strings.Contains(uri[:comma], ";base64") {
		return nil, false
	}

	decoded, err := base64.StdEncoding.DecodeString(uri[comma+1:])

	if err != nil {
		return nil, false
	}

	return decoded, true
}

func (m *meshReader) read(index int) [][]float64 {
	if index < 0 || index >= len(m.document.Accessors) {
		return nil
	}

	accessor := m.document.Accessors[index]

	if accessor.BufferView == nil {
		return nil
	}

	view := *accessor.BufferView

	if view < 0 || view >= len(m.document.BufferViews) {
		return nil
	}

	bufferView := m.document.BufferViews[view]

	if bufferView.Buffer < 0 || bufferView.Buffer >= len(m.buffers) {
		return nil
	}

	blob := m.buffers[bufferView.Buffer]
	size, known := componentSize[accessor.ComponentType]
	count, shaped := typeCount[accessor.Type]

	if blob == nil || !known || !shaped {
		return nil
	}

	stride := bufferView.ByteStride

	if stride == 0 {
		stride = size * count
	}

	start := bufferView.ByteOffset + accessor.ByteOffset
	rows := make([][]float64, 0, accessor.Count)

	for element := 0; element < accessor.Count; element++ {
		base := start + element*stride

		if base+size*count > len(blob) {
			return rows
		}

		row := make([]float64, count)

		for component := 0; component < count; component++ {
			row[component] = number(blob, base+component*size, accessor.ComponentType, accessor.Normalized)
		}

		rows = append(rows, row)
	}

	return rows
}

func number(blob []byte, offset, component int, normalized bool) float64 {
	switch component {
	case componentFloat:
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(blob[offset:])))

	case componentUnsignedInt:
		return float64(binary.LittleEndian.Uint32(blob[offset:]))

	case componentUnsignedShort:
		value := float64(binary.LittleEndian.Uint16(blob[offset:]))

		if normalized {
			return value / 65535
		}

		return value

	case componentShort:
		value := float64(int16(binary.LittleEndian.Uint16(blob[offset:])))

		if normalized {
			return math.Max(value/32767, -1)
		}

		return value

	case componentUnsignedByte:
		value := float64(blob[offset])

		if normalized {
			return value / 255
		}

		return value

	case componentByte:
		value := float64(int8(blob[offset]))

		if normalized {
			return math.Max(value/127, -1)
		}

		return value
	}

	return 0
}

func (m *meshReader) geometry(mesh int, offset rig.Vec3, textures map[int]int) ([]rig.Vec3, []Polygon) {
	if mesh < 0 || mesh >= len(m.document.Meshes) {
		return nil, nil
	}

	points := make([]rig.Vec3, 0)
	polygons := make([]Polygon, 0)

	for _, primitive := range m.document.Meshes[mesh].Primitives {
		if primitive.Mode != nil && *primitive.Mode != modeTriangles {
			continue
		}

		positions := m.read(primitive.Attributes["POSITION"])

		if len(positions) == 0 {
			continue
		}

		corners := make([]rig.Vec3, 0, len(positions))

		for _, row := range positions {
			if len(row) < 3 {
				corners = append(corners, rig.Vec3{})

				continue
			}

			corners = append(corners, rig.Vec3{
				row[0] + offset[0],
				row[1] + offset[1],
				row[2] + offset[2],
			})
		}

		points = append(points, corners...)

		texture := -1

		if primitive.Material != nil {
			if found, ok := textures[*primitive.Material]; ok {
				texture = found
			}
		}

		coordinates := m.read(primitive.Attributes["TEXCOORD_0"])
		indices := m.indices(primitive, len(corners))

		for triangle := 0; triangle+2 < len(indices); triangle += 3 {
			polygon := Polygon{Texture: texture}

			for corner := 0; corner < 3; corner++ {
				at := indices[triangle+corner]

				if at < 0 || at >= len(corners) {
					polygon.Corners = nil

					break
				}

				polygon.Corners = append(polygon.Corners, corners[at])

				if at < len(coordinates) && len(coordinates[at]) >= 2 {
					polygon.UVs = append(polygon.UVs, [2]float64{coordinates[at][0], coordinates[at][1]})
				}
			}

			if len(polygon.Corners) < 3 {
				continue
			}

			polygons = append(polygons, polygon)
		}
	}

	return points, polygons
}

func (m *meshReader) indices(primitive gltfPrimitive, corners int) []int {
	if primitive.Indices == nil {
		sequence := make([]int, corners)

		for position := range sequence {
			sequence[position] = position
		}

		return sequence
	}

	rows := m.read(*primitive.Indices)
	indices := make([]int, 0, len(rows))

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}

		indices = append(indices, int(row[0]))
	}

	return indices
}

func (m *meshReader) readTextures(request Request) ([]rig.Texture, map[int]int, []string) {
	textures := make([]rig.Texture, 0, len(m.document.Images))
	byImage := make(map[int]int, len(m.document.Images))
	problems := make([]string, 0)

	for index, image := range m.document.Images {
		data, name, ok := m.imageBytes(image, index, request)

		if !ok {
			problems = append(problems, fmt.Sprintf("Image %d could not be read, so that material has no texture.", index))

			continue
		}

		texture, err := decodeTexture(name, data)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s could not be read: %s", name, err))

			continue
		}

		textures = append(textures, texture)
		byImage[index] = len(textures) - 1
	}

	materials := make(map[int]int, len(m.document.Materials))

	for index, material := range m.document.Materials {
		if material.PBR == nil || material.PBR.BaseColorTexture == nil {
			continue
		}

		reference := material.PBR.BaseColorTexture.Index

		if reference < 0 || reference >= len(m.document.Textures) {
			continue
		}

		source := m.document.Textures[reference].Source

		if source == nil {
			continue
		}

		if found, ok := byImage[*source]; ok {
			materials[index] = found
		}
	}

	return textures, materials, problems
}

func (m *meshReader) imageBytes(image gltfImage, index int, request Request) ([]byte, string, bool) {
	name := image.Name

	if name == "" {
		name = fmt.Sprintf("texture_%d.png", index)
	}

	if image.URI != "" {
		if decoded, ok := decodeDataURI(image.URI); ok {
			return decoded, name, true
		}

		if asset, ok := request.Asset(image.URI); ok {
			return asset, BaseFile(image.URI), true
		}

		return nil, name, false
	}

	if image.BufferView == nil {
		return nil, name, false
	}

	view := *image.BufferView

	if view < 0 || view >= len(m.document.BufferViews) {
		return nil, name, false
	}

	bufferView := m.document.BufferViews[view]

	if bufferView.Buffer < 0 || bufferView.Buffer >= len(m.buffers) {
		return nil, name, false
	}

	blob := m.buffers[bufferView.Buffer]
	end := bufferView.ByteOffset + bufferView.ByteLength

	if blob == nil || end > len(blob) {
		return nil, name, false
	}

	return blob[bufferView.ByteOffset:end], name, true
}
