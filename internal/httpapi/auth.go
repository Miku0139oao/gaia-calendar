package httpapi

import (
	"net/http"
	"net/url"
	"time"

	"gaia-calendar/ent"
	"gaia-calendar/ent/appsession"
	"gaia-calendar/ent/emailverificationcode"
	"gaia-calendar/ent/passwordresettoken"
	"gaia-calendar/ent/user"
	"gaia-calendar/internal/security"
)

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	email := normalizeEmail(req.Email)
	if validationError := validateRegisterRequest(email, req); validationError != "" {
		writeError(w, http.StatusBadRequest, validationError)
		return
	}
	nickname := normalizeNickname(req.Nickname)
	if req.Nickname != "" && !validNickname(nickname) {
		writeError(w, http.StatusBadRequest, "nickname must be 3-32 characters and use letters, numbers, underscore, dot, or dash")
		return
	}
	exists, err := s.db.User.Query().Where(user.Email(email)).Exist(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check user")
		return
	}
	if exists {
		writeError(w, http.StatusConflict, "email is already registered")
		return
	}
	userCount, err := s.db.User.Query().Count(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count users")
		return
	}
	role := "user"
	if userCount == 0 {
		role = "admin"
	}
	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	u, err := s.db.User.Create().
		SetEmail(email).
		SetPasswordHash(passwordHash).
		SetEmailVerified(!s.cfg.EmailVerificationRequired).
		SetRole(role).
		Save(r.Context())
	if err == nil && nickname != "" {
		u, err = s.db.User.UpdateOne(u).SetNickname(nickname).Save(r.Context())
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	if s.cfg.EmailVerificationRequired {
		if err := s.createAndSendVerificationCode(r, u, email, req.Locale); err != nil {
			logHTTPError(err)
			writeError(w, http.StatusBadGateway, "failed to send verification email")
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "user": userDTO(u)})
}

func validateRegisterRequest(email string, req registerRequest) string {
	if email == "" || !validPassword(req.Password) {
		return "email and password with at least 8 characters are required"
	}
	if req.Password != req.ConfirmPassword {
		return "password confirmation does not match"
	}
	return ""
}

func (s *Server) createAndSendVerificationCode(r *http.Request, u *ent.User, email, locale string) error {
	code, err := security.VerificationCode()
	if err != nil {
		return err
	}
	hash := security.HashToken(code)
	_, err = s.db.EmailVerificationCode.Create().
		SetUser(u).
		SetEmail(email).
		SetCodeHash(hash).
		SetExpiresAt(time.Now().Add(10 * time.Minute)).
		Save(r.Context())
	if err != nil {
		return err
	}
	return s.emailer.SendVerificationCode(r.Context(), email, code, locale)
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	email := normalizeEmail(req.Email)
	codeHash := security.HashToken(req.Code)
	verification, err := s.db.EmailVerificationCode.Query().
		Where(
			emailverificationcode.Email(email),
			emailverificationcode.CodeHash(codeHash),
			emailverificationcode.ConsumedAtIsNil(),
			emailverificationcode.ExpiresAtGT(time.Now()),
		).
		WithUser().
		Order(ent.Desc(emailverificationcode.FieldCreatedAt)).
		First(r.Context())
	if err != nil || verification.Edges.User == nil {
		_, _ = s.db.EmailVerificationCode.Update().
			Where(emailverificationcode.Email(email), emailverificationcode.ConsumedAtIsNil()).
			AddAttemptCount(1).
			Save(r.Context())
		writeError(w, http.StatusBadRequest, "invalid or expired verification code")
		return
	}
	_, err = s.db.User.UpdateOne(verification.Edges.User).SetEmailVerified(true).Save(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify user")
		return
	}
	_, _ = s.db.EmailVerificationCode.UpdateOne(verification).SetConsumedAt(time.Now()).Save(r.Context())
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	identifier := normalizeEmail(req.Email)
	if identifier == "" {
		identifier = normalizeEmail(req.Nickname)
	}
	nickname := normalizeNickname(identifier)
	u, err := s.db.User.Query().
		Where(user.Or(user.Email(identifier), user.Nickname(nickname))).
		Only(r.Context())
	if err != nil || !security.CheckPassword(u.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if !u.EmailVerified {
		writeError(w, http.StatusForbidden, "email is not verified")
		return
	}
	token, err := security.RandomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	expires := time.Now().Add(30 * 24 * time.Hour)
	_, err = s.db.AppSession.Create().SetUser(u).SetTokenHash(security.HashToken(token)).SetExpiresAt(expires).Save(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save session")
		return
	}
	_, _ = s.db.User.UpdateOne(u).SetLastLoginAt(time.Now()).Save(r.Context())
	s.setSessionCookie(w, token, expires)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": userDTO(u)})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("gaia_calendar_session"); err == nil {
		_, _ = s.db.AppSession.Update().
			Where(appsession.TokenHash(security.HashToken(cookie.Value))).
			SetRevokedAt(time.Now()).
			Save(r.Context())
	}
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleRequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	email := normalizeEmail(req.Email)
	if email == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	u, err := s.db.User.Query().Where(user.Email(email)).Only(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	token, err := security.RandomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create reset token")
		return
	}
	resetURL, err := buildPasswordResetURL(s.cfg.BaseURL, token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build reset URL")
		return
	}
	_, _ = s.db.PasswordResetToken.Update().
		Where(passwordresettoken.Email(email), passwordresettoken.ConsumedAtIsNil()).
		SetConsumedAt(time.Now()).
		Save(r.Context())
	_, err = s.db.PasswordResetToken.Create().
		SetUser(u).
		SetEmail(email).
		SetTokenHash(security.HashToken(token)).
		SetExpiresAt(time.Now().Add(30 * time.Minute)).
		Save(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save reset token")
		return
	}
	if err := s.emailer.SendPasswordReset(r.Context(), email, resetURL, req.Locale); err != nil {
		logHTTPError(err)
		writeError(w, http.StatusBadGateway, "failed to send reset email")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirmRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Token == "" || !validPassword(req.NewPassword) {
		writeError(w, http.StatusBadRequest, "token and password with at least 8 characters are required")
		return
	}
	reset, err := s.db.PasswordResetToken.Query().
		Where(
			passwordresettoken.TokenHash(security.HashToken(req.Token)),
			passwordresettoken.ConsumedAtIsNil(),
			passwordresettoken.ExpiresAtGT(time.Now()),
		).
		WithUser().
		Only(r.Context())
	if err != nil || reset.Edges.User == nil {
		writeError(w, http.StatusBadRequest, "invalid or expired reset link")
		return
	}
	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	if _, err := s.db.User.UpdateOne(reset.Edges.User).SetPasswordHash(hash).Save(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}
	_, _ = s.db.PasswordResetToken.UpdateOne(reset).SetConsumedAt(time.Now()).Save(r.Context())
	_, _ = s.db.AppSession.Update().Where(appsession.HasUserWith(user.ID(reset.Edges.User.ID))).SetRevokedAt(time.Now()).Save(r.Context())
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"user": userDTO(currentUser(r))})
}

func buildPasswordResetURL(baseURL, token string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	parsed.Path = "/reset-password"
	parsed.RawQuery = ""
	q := parsed.Query()
	q.Set("token", token)
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}
