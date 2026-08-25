package exporter

import (
	"errors"
	"sort"
	"sync"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

var ErrUnknownFormat = errors.New("exporter: unknown format")

type Request struct {
	ModelName    string
	Namespace    string
	Rig          *rig.Rig
	Source       []byte
	SourceFormat string
	Animations   []animation.Animation
}

type Result struct {
	Filename    string
	ContentType string
	Data        []byte
}

type Exporter interface {
	ID() string
	Label() string
	Extension() string
	Export(Request) (*Result, error)
}

var (
	mu       sync.RWMutex
	registry = make(map[string]Exporter)
)

func Register(e Exporter) {
	mu.Lock()
	defer mu.Unlock()

	registry[e.ID()] = e
}

func Get(id string) (Exporter, error) {
	mu.RLock()
	defer mu.RUnlock()

	found, ok := registry[id]
	if !ok {
		return nil, ErrUnknownFormat
	}

	return found, nil
}

type Descriptor struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Extension string `json:"extension"`
}

func List() []Descriptor {
	mu.RLock()
	defer mu.RUnlock()

	items := make([]Descriptor, 0, len(registry))

	for _, item := range registry {
		items = append(items, Descriptor{ID: item.ID(), Label: item.Label(), Extension: item.Extension()})
	}

	sort.Slice(items, func(a, b int) bool {
		return items[a].ID < items[b].ID
	})

	return items
}
