package importer

import (
	"encoding/binary"
	"testing"
)

const sampleGLTF = `{
  "asset": {"version": "2.0"},
  "scene": 0,
  "scenes": [{"nodes": [0]}],
  "meshes": [{"name": "body_mesh"}],
  "nodes": [
    {"name": "Root", "translation": [0, 0, 0], "children": [1, 2]},
    {"name": "Right Arm", "translation": [4, 12, 0], "mesh": 0},
    {"translation": [0, 12, 0], "rotation": [0, 0, 0, 1]}
  ]
}`

const sampleOBJ = `# exported
o Body
v -4 0 -2
v 4 12 2
o Right Arm
v 4 4 -2
v 8 12 2
f 1 2 3
`

func TestGLTFBuildsHierarchy(t *testing.T) {
	result, err := NewGLTF().Import(Request{Data: []byte(sampleGLTF), Filename: "golem.gltf"})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	if result.Rig.ModelName != "golem" {
		t.Fatalf("unexpected model name %q", result.Rig.ModelName)
	}

	arm, ok := result.Rig.Bones["right_arm"]
	if !ok {
		t.Fatalf("missing bone, got %v", result.Rig.Names())
	}

	if arm.Parent != "root" {
		t.Fatalf("expected right_arm parented to root, got %q", arm.Parent)
	}

	if arm.Origin != [3]float64{4, 12, 0} {
		t.Fatalf("unexpected pivot %v", arm.Origin)
	}

	if _, ok := result.Rig.Bones["node_2"]; !ok {
		t.Fatalf("unnamed node should get a fallback name, got %v", result.Rig.Names())
	}
}

func TestGLBContainerIsUnwrapped(t *testing.T) {
	payload := []byte(sampleGLTF)

	for len(payload)%4 != 0 {
		payload = append(payload, ' ')
	}

	glb := make([]byte, 0, glbHeader+glbChunkHead+len(payload))
	glb = binary.LittleEndian.AppendUint32(glb, glbMagic)
	glb = binary.LittleEndian.AppendUint32(glb, 2)
	glb = binary.LittleEndian.AppendUint32(glb, uint32(glbHeader+glbChunkHead+len(payload)))
	glb = binary.LittleEndian.AppendUint32(glb, uint32(len(payload)))
	glb = binary.LittleEndian.AppendUint32(glb, glbChunkJSON)
	glb = append(glb, payload...)

	result, err := NewGLTF().Import(Request{Data: glb, Filename: "golem.glb"})
	if err != nil {
		t.Fatalf("import glb: %v", err)
	}

	if len(result.Rig.Bones) != 3 {
		t.Fatalf("expected 3 bones, got %d", len(result.Rig.Bones))
	}
}

func TestOBJBuildsFlatRigWithCenteredPivots(t *testing.T) {
	result, err := NewOBJ().Import(Request{Data: []byte(sampleOBJ), Filename: "golem.obj"})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	if len(result.Rig.Roots) != 2 {
		t.Fatalf("expected every group to be a root, got %v", result.Rig.Roots)
	}

	arm, ok := result.Rig.Bones["right_arm"]
	if !ok {
		t.Fatalf("missing bone, got %v", result.Rig.Names())
	}

	if arm.Origin != [3]float64{6, 8, 0} {
		t.Fatalf("unexpected pivot %v", arm.Origin)
	}

	if len(result.Notes) == 0 {
		t.Fatalf("obj import should warn about the missing hierarchy")
	}
}

func TestDetectPicksTheRightImporter(t *testing.T) {
	Register(NewBlockbench())
	Register(NewGLTF())
	Register(NewOBJ())

	cases := []struct {
		filename string
		data     string
		want     string
	}{
		{"model.bbmodel", `{"outliner":[],"elements":[]}`, "bbmodel"},
		{"model.gltf", sampleGLTF, "gltf"},
		{"model.obj", sampleOBJ, "obj"},
		{"", sampleOBJ, "obj"},
		{"", sampleGLTF, "gltf"},
	}

	for _, item := range cases {
		found, err := Detect([]byte(item.data), item.filename)
		if err != nil {
			t.Fatalf("detect %q: %v", item.filename, err)
		}

		if found.ID() != item.want {
			t.Fatalf("detect %q: got %q, want %q", item.filename, found.ID(), item.want)
		}
	}

	if _, err := Detect([]byte("hello"), "notes.txt"); err == nil {
		t.Fatalf("expected an unknown format error")
	}
}
