package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/database"
)

func service(t *testing.T) (*Service, string) {
	t.Helper()

	dir := t.TempDir()

	db, err := database.Open(dir)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	created, err := NewService(db, dir)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	return created, dir
}

func TestRegisterLoginAndResolve(t *testing.T) {
	accounts, _ := service(t)

	user, session, err := accounts.Register("Alex@Example.com ", "alesix", "supersecret")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if user.Email != "alex@example.com" {
		t.Fatalf("email should be normalized, got %q", user.Email)
	}

	if user.Password == "supersecret" {
		t.Fatalf("password must never be stored in clear text")
	}

	resolved, err := accounts.Resolve(session.Token)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if resolved.ID != user.ID {
		t.Fatalf("resolved the wrong user")
	}

	if _, _, err := accounts.Login("alex@example.com", "wrong"); !errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("expected a login failure, got %v", err)
	}

	if _, _, err := accounts.Login("ALEX@example.com", "supersecret"); err != nil {
		t.Fatalf("login: %v", err)
	}
}

func TestDuplicatesAndValidation(t *testing.T) {
	accounts, _ := service(t)

	if _, _, err := accounts.Register("alex@example.com", "alesix", "supersecret"); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, _, err := accounts.Register("alex@example.com", "other", "supersecret"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected duplicate email, got %v", err)
	}

	if _, _, err := accounts.Register("new@example.com", "alesix", "supersecret"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected duplicate username, got %v", err)
	}

	if _, _, err := accounts.Register("new@example.com", "ok", "supersecret"); !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("expected username rejection, got %v", err)
	}

	if _, _, err := accounts.Register("new@example.com", "valid", "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected weak password, got %v", err)
	}

	if _, _, err := accounts.Register("nope", "valid", "supersecret"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected invalid email, got %v", err)
	}
}

func TestSessionsSurviveRestart(t *testing.T) {
	accounts, dir := service(t)

	_, session, err := accounts.Register("alex@example.com", "alesix", "supersecret")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	db, err := database.Open(dir)
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}

	defer db.Close()

	reopened, err := NewService(db, dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	if _, err := reopened.Resolve(session.Token); err != nil {
		t.Fatalf("session should survive a restart: %v", err)
	}

	if err := reopened.Logout(session.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := reopened.Resolve(session.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected the session to be gone, got %v", err)
	}
}

func TestAvatarIsValidatedAndStored(t *testing.T) {
	accounts, dir := service(t)

	user, _, err := accounts.Register("alex@example.com", "alesix", "supersecret")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := accounts.SetAvatar(user.ID, []byte("not an image")); !errors.Is(err, ErrUnsupportedImage) {
		t.Fatalf("expected the image check to reject junk, got %v", err)
	}

	png := append([]byte{0x89}, []byte("PNG\r\n\x1a\n")...)

	updated, err := accounts.SetAvatar(user.ID, png)
	if err != nil {
		t.Fatalf("set avatar: %v", err)
	}

	if updated.Avatar != user.ID+".png" {
		t.Fatalf("unexpected avatar name %q", updated.Avatar)
	}

	if _, err := os.Stat(filepath.Join(dir, avatarFolder, updated.Avatar)); err != nil {
		t.Fatalf("avatar file missing: %v", err)
	}

	if _, err := accounts.AvatarPath("../users.json"); err == nil {
		t.Fatalf("avatar lookup must not escape its folder")
	}
}
