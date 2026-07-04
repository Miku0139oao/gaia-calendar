package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"gaia-calendar/ent"
	"gaia-calendar/ent/user"
)

func (s *Server) admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := currentUser(r)
		if u == nil || u.Role != "admin" {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.User.Query().Order(ent.Asc(user.FieldID)).All(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
	}
	out := make([]adminUserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, adminUserDTO(u))
	}
	writeJSON(w, http.StatusOK, adminUsersResponse{Total: len(out), Users: out})
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	idText := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	id, err := strconv.Atoi(idText)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req adminUserUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	builder := s.db.User.UpdateOneID(id)
	if req.Role != nil {
		role := strings.TrimSpace(*req.Role)
		if role != "admin" && role != "user" {
			writeError(w, http.StatusBadRequest, "role must be admin or user")
			return
		}
		builder.SetRole(role)
	}
	if req.Nickname != nil {
		nickname := normalizeNickname(*req.Nickname)
		if nickname == "" {
			builder.ClearNickname()
		} else {
			if !validNickname(nickname) {
				writeError(w, http.StatusBadRequest, "nickname must be 3-32 characters and use letters, numbers, underscore, dot, or dash")
				return
			}
			builder.SetNickname(nickname)
		}
	}
	if req.EmailVerified != nil {
		builder.SetEmailVerified(*req.EmailVerified)
	}
	u, err := builder.Save(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": adminUserDTO(u)})
}

func adminUserDTO(u *ent.User) adminUserResponse {
	nickname := ""
	if u.Nickname != nil {
		nickname = *u.Nickname
	}
	return adminUserResponse{
		ID:            u.ID,
		Email:         u.Email,
		Nickname:      nickname,
		EmailVerified: u.EmailVerified,
		Role:          u.Role,
		CreatedAt:     u.CreatedAt,
		LastLoginAt:   u.LastLoginAt,
	}
}
