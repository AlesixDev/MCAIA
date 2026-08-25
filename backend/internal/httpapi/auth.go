package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/auth"
)

type contextKey string

const userKey contextKey = "mcaia.user"

const maxAvatarUpload = 4 << 20

func bearer(r *http.Request) string {
	header := r.Header.Get("Authorization")

	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)

	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func (s *Server) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r)

		if token != "" {
			if user, err := s.auth.Resolve(token); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), userKey, user))
			}
		}

		next.ServeHTTP(w, r)
	})
}

func currentUser(r *http.Request) *auth.User {
	user, _ := r.Context().Value(userKey).(*auth.User)

	return user
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) *auth.User {
	user := currentUser(r)

	if user == nil {
		writeError(w, http.StatusUnauthorized, "you need to sign in", nil)

		return nil
	}

	return user
}

func authStatus(err error) int {
	switch {
	case errors.Is(err, auth.ErrEmailTaken), errors.Is(err, auth.ErrUsernameTaken):
		return http.StatusConflict
	case errors.Is(err, auth.ErrInvalidLogin):
		return http.StatusUnauthorized
	case errors.Is(err, auth.ErrSessionNotFound), errors.Is(err, auth.ErrUserNotFound):
		return http.StatusUnauthorized
	case errors.Is(err, auth.ErrAvatarTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, auth.ErrWeakPassword),
		errors.Is(err, auth.ErrInvalidEmail),
		errors.Is(err, auth.ErrInvalidUsername),
		errors.Is(err, auth.ErrUnsupportedImage):
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}

type credentials struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body credentials

	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())

		return
	}

	user, session, err := s.auth.Register(body.Email, body.Username, body.Password)
	if err != nil {
		writeError(w, authStatus(err), err.Error(), nil)

		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user":       user.Profile(),
		"token":      session.Token,
		"expires_at": session.ExpiresAt,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body credentials

	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())

		return
	}

	user, session, err := s.auth.Login(body.Email, body.Password)
	if err != nil {
		writeError(w, authStatus(err), err.Error(), nil)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":       user.Profile(),
		"token":      session.Token,
		"expires_at": session.ExpiresAt,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if token := bearer(r); token != "" {
		s.auth.Logout(token)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := s.requireUser(w, r)

	if user == nil {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": user.Profile()})
}

type profileBody struct {
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	user := s.requireUser(w, r)

	if user == nil {
		return
	}

	var body profileBody

	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())

		return
	}

	updated, err := s.auth.UpdateProfile(user.ID, body.DisplayName, body.Username)
	if err != nil {
		writeError(w, authStatus(err), err.Error(), nil)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": updated.Profile()})
}

func (s *Server) handleAvatarUpload(w http.ResponseWriter, r *http.Request) {
	user := s.requireUser(w, r)

	if user == nil {
		return
	}

	defer r.Body.Close()

	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxAvatarUpload))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "avatar is too large", nil)

		return
	}

	updated, err := s.auth.SetAvatar(user.ID, data)
	if err != nil {
		writeError(w, authStatus(err), err.Error(), nil)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": updated.Profile()})
}

func (s *Server) handleAvatar(w http.ResponseWriter, r *http.Request) {
	path, err := s.auth.AvatarPath(r.PathValue("file"))
	if err != nil {
		writeError(w, http.StatusNotFound, "avatar not found", nil)

		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")

	http.ServeFile(w, r, path)
}
