package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

var (
	ErrProjectNotFound   = errors.New("store: project not found")
	ErrAnimationNotFound = errors.New("store: animation not found")
)

const timeLayout = time.RFC3339Nano

type Project struct {
	ID         string                          `json:"id"`
	OwnerID    string                          `json:"owner_id,omitempty"`
	Name       string                          `json:"name"`
	Format     string                          `json:"format"`
	Notes      []string                        `json:"notes,omitempty"`
	CreatedAt  time.Time                       `json:"created_at"`
	UpdatedAt  time.Time                       `json:"updated_at"`
	Rig        *rig.Rig                        `json:"rig"`
	Animations map[string]*animation.Animation `json:"animations"`
	Order      []string                        `json:"order"`
}

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ownerID, name, format string, skeleton *rig.Rig, notes []string, source []byte) (*Project, error) {
	encodedRig, err := json.Marshal(skeleton)
	if err != nil {
		return nil, err
	}

	if notes == nil {
		notes = []string{}
	}

	encodedNotes, err := json.Marshal(notes)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	project := &Project{
		ID:         newID(),
		OwnerID:    ownerID,
		Name:       name,
		Format:     format,
		Notes:      notes,
		CreatedAt:  now,
		UpdatedAt:  now,
		Rig:        skeleton,
		Animations: make(map[string]*animation.Animation),
		Order:      []string{},
	}

	_, err = s.db.Exec(
		`INSERT INTO projects (id, owner_id, name, format, notes, rig, source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		project.ID, ownerID, name, format, string(encodedNotes), string(encodedRig), source,
		now.Format(timeLayout), now.Format(timeLayout),
	)

	if err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Store) Source(id string) ([]byte, error) {
	var source []byte

	err := s.db.QueryRow(`SELECT source FROM projects WHERE id = ?`, id).Scan(&source)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProjectNotFound
	}

	return source, err
}

func (s *Store) Get(id string) (*Project, error) {
	row := s.db.QueryRow(
		`SELECT id, owner_id, name, format, notes, rig, created_at, updated_at
		 FROM projects WHERE id = ?`, id,
	)

	project, err := scanProject(row)
	if err != nil {
		return nil, err
	}

	if err := s.loadAnimations(project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Store) List(ownerID string) ([]*Project, error) {
	rows, err := s.db.Query(
		`SELECT id, owner_id, name, format, notes, rig, created_at, updated_at
		 FROM projects WHERE owner_id = ? ORDER BY updated_at DESC`, ownerID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	items := make([]*Project, 0)

	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, project)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, project := range items {
		if err := s.loadAnimations(project); err != nil {
			return nil, err
		}
	}

	return items, nil
}

func (s *Store) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrProjectNotFound
	}

	return nil
}

func (s *Store) Rename(id, name string) error {
	result, err := s.db.Exec(
		`UPDATE projects SET name = ?, updated_at = ? WHERE id = ?`,
		name, time.Now().UTC(), id,
	)

	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrProjectNotFound
	}

	return nil
}

func (s *Store) SaveAnimation(id string, item *animation.Animation) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}

	var position int

	err = s.db.QueryRow(
		`SELECT COALESCE(MAX(position) + 1, 0) FROM animations WHERE project_id = ?`, id,
	).Scan(&position)

	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`INSERT INTO animations (project_id, name, position, payload)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (project_id, name) DO UPDATE SET payload = excluded.payload`,
		id, item.Name, position, string(payload),
	)

	if err != nil {
		return err
	}

	return s.touch(id)
}

func (s *Store) DeleteAnimation(id, name string) error {
	result, err := s.db.Exec(`DELETE FROM animations WHERE project_id = ? AND name = ?`, id, name)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrAnimationNotFound
	}

	return s.touch(id)
}

func (s *Store) touch(id string) error {
	_, err := s.db.Exec(
		`UPDATE projects SET updated_at = ? WHERE id = ?`,
		time.Now().UTC().Format(timeLayout), id,
	)

	return err
}

func (s *Store) loadAnimations(project *Project) error {
	rows, err := s.db.Query(
		`SELECT name, payload FROM animations WHERE project_id = ? ORDER BY position`, project.ID,
	)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var (
			name    string
			payload string
		)

		if err := rows.Scan(&name, &payload); err != nil {
			return err
		}

		var item animation.Animation

		if err := json.Unmarshal([]byte(payload), &item); err != nil {
			return err
		}

		project.Animations[name] = &item
		project.Order = append(project.Order, name)
	}

	return rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProject(row scanner) (*Project, error) {
	var (
		project    Project
		notes      string
		encodedRig string
		created    string
		updated    string
	)

	err := row.Scan(
		&project.ID, &project.OwnerID, &project.Name, &project.Format,
		&notes, &encodedRig, &created, &updated,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProjectNotFound
	}

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(notes), &project.Notes); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(encodedRig), &project.Rig); err != nil {
		return nil, err
	}

	project.CreatedAt, _ = time.Parse(timeLayout, created)
	project.UpdatedAt, _ = time.Parse(timeLayout, updated)
	project.Animations = make(map[string]*animation.Animation)
	project.Order = []string{}

	return &project, nil
}

func (p *Project) Ordered() []animation.Animation {
	items := make([]animation.Animation, 0, len(p.Order))

	for _, name := range p.Order {
		if item, ok := p.Animations[name]; ok {
			items = append(items, *item)
		}
	}

	return items
}

func (p *Project) OwnedBy(ownerID string) bool {
	return p.OwnerID == ownerID
}

func newID() string {
	buffer := make([]byte, 8)

	if _, err := rand.Read(buffer); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}

	return hex.EncodeToString(buffer)
}
