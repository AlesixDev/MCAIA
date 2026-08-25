package importer

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

var (
	ErrUnknownFormat = errors.New("importer: unsupported model format")
	ErrEmptyModel    = errors.New("importer: the model has no groups or nodes to animate")
)

const MaxSize = 64 << 20

type Request struct {
	Data     []byte
	Filename string
	Assets   map[string][]byte
}

func (r Request) Asset(name string) ([]byte, bool) {
	if r.Assets == nil {
		return nil, false
	}

	data, ok := r.Assets[strings.ToLower(BaseFile(name))]

	return data, ok
}

type Result struct {
	Name   string
	Format string
	Rig    *rig.Rig
	Notes  []string
}

const (
	blockUnit      = 16.0
	blockUnitLimit = 4.0
)

func NormalizeScale(skeleton *rig.Rig) bool {
	var box *rig.BoundingBox

	for _, bone := range skeleton.Bones {
		for _, cube := range bone.Cubes {
			box = rig.Expand(box, cube.From)
			box = rig.Expand(box, cube.To)
		}
	}

	if box == nil {
		return false
	}

	span := 0.0

	for axis := 0; axis < 3; axis++ {
		span = max(span, box.Max[axis]-box.Min[axis])
	}

	if span == 0 || span > blockUnitLimit {
		return false
	}

	for name, bone := range skeleton.Bones {
		bone.Origin = scaleVec(bone.Origin)

		for index, cube := range bone.Cubes {
			cube.From = scaleVec(cube.From)
			cube.To = scaleVec(cube.To)
			cube.Origin = scaleVec(cube.Origin)
			bone.Cubes[index] = cube
		}

		if bone.Bounds != nil {
			bone.Bounds.Min = scaleVec(bone.Bounds.Min)
			bone.Bounds.Max = scaleVec(bone.Bounds.Max)
		}

		skeleton.Bones[name] = bone
	}

	return true
}

func scaleVec(value rig.Vec3) rig.Vec3 {
	return rig.Vec3{value[0] * blockUnit, value[1] * blockUnit, value[2] * blockUnit}
}

type Importer interface {
	ID() string
	Label() string
	Extensions() []string
	Detect(data []byte, filename string) bool
	Import(request Request) (*Result, error)
}

var (
	mu       sync.RWMutex
	registry []Importer
)

func Register(item Importer) {
	mu.Lock()
	defer mu.Unlock()

	registry = append(registry, item)
}

func Detect(data []byte, filename string) (Importer, error) {
	mu.RLock()
	defer mu.RUnlock()

	for _, item := range registry {
		if item.Detect(data, filename) {
			return item, nil
		}
	}

	return nil, ErrUnknownFormat
}

func Import(request Request) (*Result, error) {
	item, err := Detect(request.Data, request.Filename)
	if err != nil {
		return nil, err
	}

	return item.Import(request)
}

type Descriptor struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Extensions []string `json:"extensions"`
}

func List() []Descriptor {
	mu.RLock()
	defer mu.RUnlock()

	items := make([]Descriptor, 0, len(registry))

	for _, item := range registry {
		items = append(items, Descriptor{
			ID:         item.ID(),
			Label:      item.Label(),
			Extensions: item.Extensions(),
		})
	}

	sort.Slice(items, func(a, b int) bool {
		return items[a].ID < items[b].ID
	})

	return items
}

func BaseFile(name string) string {
	trimmed := strings.TrimSpace(name)

	if index := strings.LastIndexAny(trimmed, "/\\"); index >= 0 {
		trimmed = trimmed[index+1:]
	}

	return trimmed
}

func HasExtension(filename string, extensions ...string) bool {
	lowered := strings.ToLower(filename)

	for _, extension := range extensions {
		if strings.HasSuffix(lowered, extension) {
			return true
		}
	}

	return false
}

func BaseName(filename, fallback string) string {
	trimmed := strings.TrimSpace(filename)

	if trimmed == "" {
		return fallback
	}

	if index := strings.LastIndexAny(trimmed, "/\\"); index >= 0 {
		trimmed = trimmed[index+1:]
	}

	if index := strings.Index(trimmed, "."); index > 0 {
		trimmed = trimmed[:index]
	}

	if trimmed == "" {
		return fallback
	}

	return trimmed
}
