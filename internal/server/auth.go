package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/nairwolf/spacedchess/internal/store"
)

const sessionCookie = "spacedchess_session"

type ctxKey int

const userKey ctxKey = 0

func userFrom(r *http.Request) *store.User {
	u, _ := r.Context().Value(userKey).(*store.User)
	return u
}

// auth wraps a handler, requiring a valid session cookie.
func (s *Server) auth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || c.Value == "" {
			s.jsonError(w, http.StatusUnauthorized, "not logged in")
			return
		}
		u, err := s.store.UserBySession(r.Context(), c.Value)
		if errors.Is(err, store.ErrNotFound) {
			s.jsonError(w, http.StatusUnauthorized, "not logged in")
			return
		}
		if err != nil {
			s.storeError(w, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.allowRegistration {
		s.jsonError(w, http.StatusForbidden, "registration is disabled")
		return
	}
	var in credentials
	if !s.decode(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if len(in.Username) < 2 || len(in.Username) > 40 {
		s.jsonError(w, http.StatusBadRequest, "username must be 2–40 characters")
		return
	}
	if len(in.Password) < 8 {
		s.jsonError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		s.storeError(w, err)
		return
	}
	u, err := s.store.CreateUser(r.Context(), in.Username, string(hash))
	if errors.Is(err, store.ErrConflict) {
		s.jsonError(w, http.StatusConflict, "username is taken")
		return
	}
	if err != nil {
		s.storeError(w, err)
		return
	}
	token, err := s.store.CreateSession(r.Context(), u.ID)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.setSessionCookie(w, token, 30*24*3600)
	s.writeJSON(w, http.StatusCreated, u)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !s.decode(w, r, &in) {
		return
	}
	u, err := s.store.UserByUsername(r.Context(), strings.TrimSpace(in.Username))
	if errors.Is(err, store.ErrNotFound) {
		s.jsonError(w, http.StatusUnauthorized, "wrong username or password")
		return
	}
	if err != nil {
		s.storeError(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		s.jsonError(w, http.StatusUnauthorized, "wrong username or password")
		return
	}
	token, err := s.store.CreateSession(r.Context(), u.ID)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.setSessionCookie(w, token, 30*24*3600)
	s.writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		s.store.DeleteSession(r.Context(), c.Value)
	}
	s.setSessionCookie(w, "", -1)
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, userFrom(r))
}
