package bbmodel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var (
	ErrEmptyDocument = errors.New("bbmodel: empty document")
	ErrNoOutliner    = errors.New("bbmodel: model has no outliner groups")
)

const maxDocumentSize = 64 << 20

func Parse(r io.Reader) (*Project, error) {
	raw, err := io.ReadAll(io.LimitReader(r, maxDocumentSize))
	if err != nil {
		return nil, fmt.Errorf("bbmodel: read: %w", err)
	}

	return ParseBytes(raw)
}

func ParseBytes(raw []byte) (*Project, error) {
	if len(raw) == 0 {
		return nil, ErrEmptyDocument
	}

	var project Project

	if err := json.Unmarshal(raw, &project); err != nil {
		return nil, fmt.Errorf("bbmodel: decode: %w", err)
	}

	var extra map[string]json.RawMessage

	if err := json.Unmarshal(raw, &extra); err != nil {
		return nil, fmt.Errorf("bbmodel: decode extra: %w", err)
	}

	project.Extra = extra

	if len(project.Outliner) == 0 {
		return nil, ErrNoOutliner
	}

	if project.Name == "" {
		project.Name = "model"
	}

	return &project, nil
}

func (p *Project) Marshal() ([]byte, error) {
	merged := make(map[string]json.RawMessage, len(p.Extra)+2)

	for key, value := range p.Extra {
		merged[key] = value
	}

	encoded, err := json.Marshal(*p)
	if err != nil {
		return nil, fmt.Errorf("bbmodel: encode: %w", err)
	}

	var typed map[string]json.RawMessage

	if err := json.Unmarshal(encoded, &typed); err != nil {
		return nil, fmt.Errorf("bbmodel: encode merge: %w", err)
	}

	for key, value := range typed {
		merged[key] = value
	}

	return json.MarshalIndent(merged, "", "  ")
}
