package auth

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	sessionTTL   = 30 * 24 * time.Hour
	maxAvatar    = 2 << 20
	avatarFolder = "avatars"
	timeLayout   = time.RFC3339Nano
)

type Service struct {
	db  *sql.DB
	dir string
}

func NewService(db *sql.DB, dir string) (*Service, error) {
	if err := os.MkdirAll(filepath.Join(dir, avatarFolder), 0o755); err != nil {
		return nil, err
	}

	return &Service{db: db, dir: dir}, nil
}

func (s *Service) Register(email, username, password string) (*User, *Session, error) {
	email = normalizeEmail(email)
	username = strings.TrimSpace(username)

	if err := validate(email, username, password); err != nil {
		return nil, nil, err
	}

	if taken, err := s.exists("email", email); err != nil {
		return nil, nil, err
	} else if taken {
		return nil, nil, ErrEmailTaken
	}

	if taken, err := s.exists("username", username); err != nil {
		return nil, nil, err
	} else if taken {
		return nil, nil, ErrUsernameTaken
	}

	hashed, err := hashPassword(password)
	if err != nil {
		return nil, nil, err
	}

	id, err := newID()
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()

	user := &User{
		ID:          id,
		Email:       email,
		Username:    username,
		DisplayName: username,
		Password:    hashed,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = s.db.Exec(
		`INSERT INTO users (id, email, username, display_name, avatar, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, '', ?, ?, ?)`,
		user.ID, user.Email, user.Username, user.DisplayName, user.Password,
		now.Format(timeLayout), now.Format(timeLayout),
	)

	if err != nil {
		return nil, nil, err
	}

	session, err := s.issue(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

func (s *Service) Login(email, password string) (*User, *Session, error) {
	user, err := s.byColumn("email", normalizeEmail(email))

	if err != nil || !verifyPassword(user.Password, password) {
		return nil, nil, ErrInvalidLogin
	}

	session, err := s.issue(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

func (s *Service) Logout(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)

	return err
}

func (s *Service) Resolve(token string) (*User, error) {
	var (
		userID  string
		expires string
	)

	err := s.db.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE token = ?`, token).
		Scan(&userID, &expires)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}

	if err != nil {
		return nil, err
	}

	deadline, err := time.Parse(timeLayout, expires)
	if err != nil || time.Now().After(deadline) {
		s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)

		return nil, ErrSessionNotFound
	}

	return s.byColumn("id", userID)
}

func (s *Service) UpdateProfile(userID, displayName, username string) (*User, error) {
	user, err := s.byColumn("id", userID)
	if err != nil {
		return nil, err
	}

	if username != "" && !strings.EqualFold(username, user.Username) {
		if !usernamePattern.MatchString(username) {
			return nil, ErrInvalidUsername
		}

		taken, err := s.exists("username", username)
		if err != nil {
			return nil, err
		}

		if taken {
			return nil, ErrUsernameTaken
		}

		user.Username = username
	}

	if trimmed := strings.TrimSpace(displayName); trimmed != "" {
		user.DisplayName = trimmed
	}

	user.UpdatedAt = time.Now().UTC()

	_, err = s.db.Exec(
		`UPDATE users SET username = ?, display_name = ?, updated_at = ? WHERE id = ?`,
		user.Username, user.DisplayName, user.UpdatedAt.Format(timeLayout), user.ID,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) SetAvatar(userID string, data []byte) (*User, error) {
	if len(data) > maxAvatar {
		return nil, ErrAvatarTooLarge
	}

	extension, err := imageExtension(data)
	if err != nil {
		return nil, err
	}

	user, err := s.byColumn("id", userID)
	if err != nil {
		return nil, err
	}

	filename := user.ID + extension

	if err := os.WriteFile(filepath.Join(s.dir, avatarFolder, filename), data, 0o644); err != nil {
		return nil, err
	}

	if user.Avatar != "" && user.Avatar != filename {
		os.Remove(filepath.Join(s.dir, avatarFolder, user.Avatar))
	}

	user.Avatar = filename
	user.UpdatedAt = time.Now().UTC()

	_, err = s.db.Exec(
		`UPDATE users SET avatar = ?, updated_at = ? WHERE id = ?`,
		user.Avatar, user.UpdatedAt.Format(timeLayout), user.ID,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) AvatarPath(filename string) (string, error) {
	clean := filepath.Base(filename)

	if clean == "." || clean == string(filepath.Separator) {
		return "", ErrUserNotFound
	}

	path := filepath.Join(s.dir, avatarFolder, clean)

	if _, err := os.Stat(path); err != nil {
		return "", ErrUserNotFound
	}

	return path, nil
}

func (s *Service) issue(userID string) (*Session, error) {
	token, err := newToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		Token:     token,
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(sessionTTL),
	}

	_, err = s.db.Exec(
		`INSERT INTO sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		session.Token, session.UserID,
		session.CreatedAt.Format(timeLayout), session.ExpiresAt.Format(timeLayout),
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Service) exists(column, value string) (bool, error) {
	var found int

	query := `SELECT COUNT(1) FROM users WHERE ` + column + ` = ?`

	if err := s.db.QueryRow(query, value).Scan(&found); err != nil {
		return false, err
	}

	return found > 0, nil
}

func (s *Service) byColumn(column, value string) (*User, error) {
	query := `SELECT id, email, username, display_name, avatar, password_hash, created_at, updated_at
	          FROM users WHERE ` + column + ` = ?`

	var (
		user    User
		created string
		updated string
	)

	err := s.db.QueryRow(query, value).Scan(
		&user.ID, &user.Email, &user.Username, &user.DisplayName,
		&user.Avatar, &user.Password, &created, &updated,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	user.CreatedAt, _ = time.Parse(timeLayout, created)
	user.UpdatedAt, _ = time.Parse(timeLayout, updated)

	return &user, nil
}
