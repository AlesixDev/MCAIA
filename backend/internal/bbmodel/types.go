package bbmodel

import "encoding/json"

type Vec3 [3]float64

type Meta struct {
	FormatVersion string `json:"format_version"`
	ModelFormat   string `json:"model_format"`
	BoxUV         bool   `json:"box_uv"`
}

type Resolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type TextureRef struct {
	Index int
	UUID  string
}

func (t *TextureRef) UnmarshalJSON(data []byte) error {
	t.Index = -1
	t.UUID = ""

	if string(data) == "null" {
		return nil
	}

	var index int

	if err := json.Unmarshal(data, &index); err == nil {
		t.Index = index

		return nil
	}

	return json.Unmarshal(data, &t.UUID)
}

type Face struct {
	UV       [4]float64 `json:"uv"`
	Texture  TextureRef `json:"texture"`
	Rotation float64    `json:"rotation,omitempty"`
}

type Texture struct {
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	Source string `json:"source"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type Element struct {
	Name     string          `json:"name"`
	UUID     string          `json:"uuid"`
	Type     string          `json:"type,omitempty"`
	From     Vec3            `json:"from"`
	To       Vec3            `json:"to"`
	Origin   Vec3            `json:"origin"`
	Rotation Vec3            `json:"rotation,omitempty"`
	Inflate  float64         `json:"inflate,omitempty"`
	BoxUV    *bool           `json:"box_uv,omitempty"`
	UVOffset [2]float64      `json:"uv_offset,omitempty"`
	Faces    map[string]Face `json:"faces,omitempty"`
}

type OutlinerNode struct {
	Name     string          `json:"name"`
	UUID     string          `json:"uuid"`
	Origin   Vec3            `json:"origin"`
	Rotation Vec3            `json:"rotation,omitempty"`
	Export   *bool           `json:"export,omitempty"`
	Children []OutlinerEntry `json:"children,omitempty"`
}

type OutlinerEntry struct {
	Ref   string
	Group *OutlinerNode
}

func (e *OutlinerEntry) UnmarshalJSON(data []byte) error {
	var ref string

	if err := json.Unmarshal(data, &ref); err == nil {
		e.Ref = ref
		e.Group = nil

		return nil
	}

	var group OutlinerNode

	if err := json.Unmarshal(data, &group); err != nil {
		return err
	}

	e.Group = &group
	e.Ref = ""

	return nil
}

func (e OutlinerEntry) MarshalJSON() ([]byte, error) {
	if e.Group != nil {
		return json.Marshal(e.Group)
	}

	return json.Marshal(e.Ref)
}

type Project struct {
	Meta       Meta                       `json:"meta"`
	Name       string                     `json:"name"`
	ModelIdent string                     `json:"model_identifier,omitempty"`
	Resolution Resolution                 `json:"resolution"`
	Elements   []Element                  `json:"elements"`
	Outliner   []OutlinerEntry            `json:"outliner"`
	Textures   []Texture                  `json:"textures,omitempty"`
	Animations json.RawMessage            `json:"animations,omitempty"`
	Extra      map[string]json.RawMessage `json:"-"`
}
