package bedrock

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter"
)

const formatVersion = "1.8.0"

type Exporter struct{}

func New() *Exporter {
	return &Exporter{}
}

func (e *Exporter) ID() string {
	return "bedrock"
}

func (e *Exporter) Label() string {
	return "Blockbench / Bedrock animation"
}

func (e *Exporter) Extension() string {
	return ".animation.json"
}

func (e *Exporter) Export(request exporter.Request) (*exporter.Result, error) {
	namespace := request.Namespace

	if namespace == "" {
		namespace = slug(request.ModelName)
	}

	animations := make(map[string]any, len(request.Animations))

	for _, item := range request.Animations {
		animations[key(namespace, item.Name)] = encode(item)
	}

	document := map[string]any{
		"format_version": formatVersion,
		"animations":     animations,
	}

	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("bedrock: encode: %w", err)
	}

	return &exporter.Result{
		Filename:    slug(request.ModelName) + e.Extension(),
		ContentType: "application/json",
		Data:        data,
	}, nil
}

func encode(item animation.Animation) map[string]any {
	entry := map[string]any{
		"animation_length": item.Length,
		"bones":            exporter.BuildBones(item, false),
	}

	switch item.Loop {
	case animation.LoopCycle:
		entry["loop"] = true
	case animation.LoopHold:
		entry["loop"] = "hold_on_last_frame"
	}

	return entry
}

func key(namespace, name string) string {
	return "animation." + namespace + "." + name
}

func slug(value string) string {
	lowered := strings.ToLower(strings.TrimSpace(value))
	lowered = strings.ReplaceAll(lowered, " ", "_")

	if lowered == "" {
		return "model"
	}

	return lowered
}
