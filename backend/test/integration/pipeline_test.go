package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/ai"
	"github.com/AlesixDev/MCAIA/backend/internal/database"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter/bedrock"
	"github.com/AlesixDev/MCAIA/backend/internal/importer"
	"github.com/AlesixDev/MCAIA/backend/internal/motion"
	"github.com/AlesixDev/MCAIA/backend/internal/pipeline"
	"github.com/AlesixDev/MCAIA/backend/internal/store"
)

const sampleModel = `{
  "meta": {"format_version": "4.10", "model_format": "free", "box_uv": false},
  "name": "golem",
  "resolution": {"width": 64, "height": 64},
  "elements": [
    {"name": "body", "uuid": "cube-body", "from": [-4, 0, -2], "to": [4, 12, 2], "origin": [0, 0, 0]},
    {"name": "arm", "uuid": "cube-arm", "from": [4, 4, -2], "to": [8, 12, 2], "origin": [4, 12, 0]}
  ],
  "outliner": [
    {
      "name": "root",
      "uuid": "group-root",
      "origin": [0, 0, 0],
      "children": [
        "cube-body",
        {"name": "right arm", "uuid": "group-arm", "origin": [4, 12, 0], "children": ["cube-arm"]}
      ]
    }
  ]
}`

const draftResponse = `{
  "name": "Idle Wave",
  "length": 1.0,
  "loop": "loop",
  "moves": [
    {"target": "right_arm", "motion": "swing", "axis": "z", "amplitude": 30, "cycles": 1, "phase": 0},
    {"target": "ghost_bone", "motion": "swing", "axis": "x", "amplitude": 20, "cycles": 1}
  ]
}`

type stubEngine struct {
	payload string
}

func (s stubEngine) ID() string {
	return "stub"
}

func (s stubEngine) Available(ctx context.Context) error {
	return nil
}

func (s stubEngine) Generate(ctx context.Context, request ai.Request) (*motion.Plan, error) {
	return ai.DecodePlan(s.payload)
}

type brokenEngine struct{}

func (brokenEngine) ID() string {
	return "broken"
}

func (brokenEngine) Available(ctx context.Context) error {
	return nil
}

func (brokenEngine) Generate(ctx context.Context, request ai.Request) (*motion.Plan, error) {
	return nil, ai.ErrInvalidResponse
}

type workspace struct {
	flow     *pipeline.Pipeline
	projects *store.Store
	project  *store.Project
}

func setup(t *testing.T, engine ai.Engine) workspace {
	t.Helper()

	imported, err := importer.NewBlockbench().Import(importer.Request{
		Data:     []byte(sampleModel),
		Filename: "golem.bbmodel",
	})

	if err != nil {
		t.Fatalf("import model: %v", err)
	}

	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	projects := store.New(db)

	project, err := projects.Create("", imported.Name, imported.Format, imported.Rig, imported.Notes, []byte(sampleModel))
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	return workspace{flow: pipeline.New(projects, engine), projects: projects, project: project}
}

func working(t *testing.T) workspace {
	t.Helper()

	return setup(t, stubEngine{payload: draftResponse})
}

func TestRigBonesAreDerivedFromOutliner(t *testing.T) {
	space := working(t)

	if !space.project.Rig.Has("root") || !space.project.Rig.Has("right_arm") {
		t.Fatalf("unexpected bones: %v", space.project.Rig.Names())
	}

	if space.project.Rig.Bones["right_arm"].Parent != "root" {
		t.Fatalf("expected right_arm parented to root")
	}
}

func TestGenerateDropsUnknownBonesAndNormalizesName(t *testing.T) {
	space := working(t)

	output, err := space.flow.Generate(context.Background(), pipeline.GenerateInput{
		ProjectID: space.project.ID,
		Prompt:    "wave",
		Optimize:  true,
	})

	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if output.Animation.Name != "idle_wave" {
		t.Fatalf("expected normalized name, got %q", output.Animation.Name)
	}

	if _, ok := output.Animation.Bones["ghost_bone"]; ok {
		t.Fatalf("a move aimed at a missing bone must not create a track")
	}

	if _, ok := output.Animation.Bones["right_arm"]; !ok {
		t.Fatalf("expected the right arm to be animated, got %v", output.Animation.Bones)
	}
}

func TestAPresetRescuesAFailedModel(t *testing.T) {
	space := setup(t, brokenEngine{})

	output, err := space.flow.Generate(context.Background(), pipeline.GenerateInput{
		ProjectID: space.project.ID,
		Prompt:    "walk forward",
		Name:      "walk",
		Duration:  1,
		Loop:      "loop",
	})

	if err != nil {
		t.Fatalf("a failed model should still produce something: %v", err)
	}

	if len(output.Animation.Bones) == 0 {
		t.Fatalf("the preset produced nothing")
	}

	if len(output.Warnings) == 0 {
		t.Fatalf("the user must be told a preset was used")
	}
}

func TestGeneratingTwiceKeepsBothAnimations(t *testing.T) {
	space := working(t)

	first, err := space.flow.Generate(context.Background(), pipeline.GenerateInput{
		ProjectID: space.project.ID,
		Prompt:    "wave",
	})

	if err != nil {
		t.Fatalf("first generate: %v", err)
	}

	second, err := space.flow.Generate(context.Background(), pipeline.GenerateInput{
		ProjectID: space.project.ID,
		Prompt:    "wave",
	})

	if err != nil {
		t.Fatalf("second generate: %v", err)
	}

	if first.Animation.Name == second.Animation.Name {
		t.Fatalf("both animations were called %q, so the first was overwritten", first.Animation.Name)
	}

	reloaded, err := space.projects.Get(space.project.ID)
	if err != nil {
		t.Fatalf("reload project: %v", err)
	}

	if len(reloaded.Animations) != 2 {
		t.Fatalf("expected both animations to survive, found %d", len(reloaded.Animations))
	}

	for _, name := range []string{first.Animation.Name, second.Animation.Name} {
		if _, ok := reloaded.Animations[name]; !ok {
			t.Errorf("%q is missing from the project", name)
		}
	}
}

func TestExportPicksTheRequestedAnimation(t *testing.T) {
	exporter.Register(bedrock.New())

	space := working(t)

	first, err := space.flow.Generate(context.Background(), pipeline.GenerateInput{
		ProjectID: space.project.ID,
		Prompt:    "wave",
	})

	if err != nil {
		t.Fatalf("first generate: %v", err)
	}

	if _, err := space.flow.Generate(context.Background(), pipeline.GenerateInput{
		ProjectID: space.project.ID,
		Prompt:    "wave",
	}); err != nil {
		t.Fatalf("second generate: %v", err)
	}

	result, err := space.flow.Export(pipeline.ExportInput{
		ProjectID: space.project.ID,
		Format:    "bedrock",
		Names:     []string{first.Animation.Name},
	})

	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if !strings.Contains(string(result.Data), first.Animation.Name) {
		t.Fatalf("the export does not contain %q", first.Animation.Name)
	}
}

func TestExportProducesBedrockTimeline(t *testing.T) {
	exporter.Register(bedrock.New())

	space := working(t)

	if _, err := space.flow.Generate(context.Background(), pipeline.GenerateInput{
		ProjectID: space.project.ID,
		Prompt:    "wave",
	}); err != nil {
		t.Fatalf("generate: %v", err)
	}

	result, err := space.flow.Export(pipeline.ExportInput{
		ProjectID: space.project.ID,
		Format:    "bedrock",
	})

	if err != nil {
		t.Fatalf("export: %v", err)
	}

	var document struct {
		FormatVersion string `json:"format_version"`
		Animations    map[string]struct {
			Loop   any     `json:"loop"`
			Length float64 `json:"animation_length"`
			Bones  map[string]map[string]map[string]any
		} `json:"animations"`
	}

	if err := json.Unmarshal(result.Data, &document); err != nil {
		t.Fatalf("decode export: %v", err)
	}

	entry, ok := document.Animations["animation.golem.idle_wave"]
	if !ok {
		t.Fatalf("missing animation key in %s", string(result.Data))
	}

	if entry.Loop != true || entry.Length != 1 {
		t.Fatalf("unexpected header: %+v", entry)
	}

	timeline := entry.Bones["right_arm"]["rotation"]

	if len(timeline) < 4 {
		t.Fatalf("expected a sampled rotation timeline, got %v", timeline)
	}

	first, ok := timeline["0.0"].(map[string]any)
	if !ok || first["lerp_mode"] != "catmullrom" {
		t.Fatalf("keyframes should be exported smoothed, got %v", timeline["0.0"])
	}

	if !strings.HasSuffix(result.Filename, ".animation.json") {
		t.Fatalf("unexpected filename %q", result.Filename)
	}
}
