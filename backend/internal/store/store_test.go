package store

import (
	"errors"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/database"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

func skeleton() *rig.Rig {
	builder := rig.NewBuilder("golem", "bbmodel")
	builder.Add(rig.Bone{Name: "root", Origin: rig.Vec3{0, 0, 0}})

	return builder.Rig()
}

func sample(name string) *animation.Animation {
	return &animation.Animation{
		Name:   name,
		Length: 1,
		Loop:   animation.LoopCycle,
		Bones: map[string]animation.Track{
			"root": {Rotation: []animation.Keyframe{{Time: 0, Value: animation.Vec3{0, 0, 0}}}},
		},
	}
}

func TestProjectsAndAnimationsSurviveReopen(t *testing.T) {
	dir := t.TempDir()

	db, err := database.Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	projects := New(db)

	project, err := projects.Create("user-1", "golem", "bbmodel", skeleton(), []string{"note"}, []byte(`{"name":"golem"}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := projects.SaveAnimation(project.ID, sample("walk")); err != nil {
		t.Fatalf("save walk: %v", err)
	}

	if err := projects.SaveAnimation(project.ID, sample("idle")); err != nil {
		t.Fatalf("save idle: %v", err)
	}

	db.Close()

	reopened, err := database.Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	defer reopened.Close()

	loaded, err := New(reopened).Get(project.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if loaded.Name != "golem" || loaded.Format != "bbmodel" || len(loaded.Notes) != 1 {
		t.Fatalf("unexpected project %+v", loaded)
	}

	if !loaded.Rig.Has("root") {
		t.Fatalf("rig should round-trip through the database")
	}

	if got := loaded.Order; len(got) != 2 || got[0] != "walk" || got[1] != "idle" {
		t.Fatalf("animations lost their order: %v", got)
	}
}

func TestListOnlyReturnsTheOwnersProjects(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer db.Close()

	projects := New(db)

	if _, err := projects.Create("user-1", "mine", "bbmodel", skeleton(), nil, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := projects.Create("user-2", "theirs", "bbmodel", skeleton(), nil, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	mine, err := projects.List("user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(mine) != 1 || mine[0].Name != "mine" {
		t.Fatalf("owner scoping is broken: %+v", mine)
	}

	anonymous, err := projects.List("")
	if err != nil {
		t.Fatalf("list anonymous: %v", err)
	}

	if len(anonymous) != 0 {
		t.Fatalf("anonymous should not see anything, got %d", len(anonymous))
	}
}

func TestDeletingAProjectRemovesItsAnimations(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer db.Close()

	projects := New(db)

	project, err := projects.Create("user-1", "golem", "bbmodel", skeleton(), nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := projects.SaveAnimation(project.ID, sample("walk")); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := projects.DeleteAnimation(project.ID, "missing"); !errors.Is(err, ErrAnimationNotFound) {
		t.Fatalf("expected a missing animation error, got %v", err)
	}

	if err := projects.Delete(project.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := projects.Get(project.ID); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected the project to be gone, got %v", err)
	}

	var remaining int

	if err := db.QueryRow(`SELECT COUNT(1) FROM animations`).Scan(&remaining); err != nil {
		t.Fatalf("count: %v", err)
	}

	if remaining != 0 {
		t.Fatalf("animations should cascade, %d left", remaining)
	}
}
