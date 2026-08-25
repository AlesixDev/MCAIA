package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	iterations = 600000
	keyLength  = 32
	saltLength = 16
)

var (
	ErrEmailTaken       = errors.New("auth: email already registered")
	ErrUsernameTaken    = errors.New("auth: username already taken")
	ErrInvalidLogin     = errors.New("auth: wrong email or password")
	ErrUserNotFound     = errors.New("auth: user not found")
	ErrSessionNotFound  = errors.New("auth: session expired or invalid")
	ErrWeakPassword     = errors.New("auth: password must be at least 8 characters")
	ErrInvalidEmail     = errors.New("auth: invalid email address")
	ErrInvalidUsername  = errors.New("auth: username must be 3-24 characters, letters, digits, _ or -")
	ErrUnsupportedImage = errors.New("auth: avatar must be a png, jpeg, webp or gif image")
	ErrAvatarTooLarge   = errors.New("auth: avatar is larger than 2 MB")
)

var (
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s.]+\.[^@\s]+$`)
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,24}$`)
)

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Avatar      string    `json:"avatar,omitempty"`
	Password    string    `json:"password_hash"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Profile struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Avatar      string    `json:"avatar,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (u *User) Profile() Profile {
	return Profile{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Avatar:      u.Avatar,
		CreatedAt:   u.CreatedAt,
	}
}

type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)

	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, keyLength)
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		"pbkdf2-sha256",
		strconv.Itoa(iterations),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	}, "$"), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")

	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}

	rounds, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}

	key, err := pbkdf2.Key(sha256.New, password, salt, rounds, len(expected))
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(key, expected) == 1
}

func newToken() (string, error) {
	buffer := make([]byte, 32)

	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}

func newID() (string, error) {
	buffer := make([]byte, 8)

	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validate(email, username, password string) error {
	if !emailPattern.MatchString(email) {
		return ErrInvalidEmail
	}

	if !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}

	if len(password) < 8 {
		return ErrWeakPassword
	}

	return nil
}

func imageExtension(data []byte) (string, error) {
	switch {
	case len(data) >= 8 && string(data[1:4]) == "PNG":
		return ".png", nil
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8:
		return ".jpg", nil
	case len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return ".webp", nil
	case len(data) >= 6 && string(data[0:3]) == "GIF":
		return ".gif", nil
	}

	return "", fmt.Errorf("%w", ErrUnsupportedImage)
}
