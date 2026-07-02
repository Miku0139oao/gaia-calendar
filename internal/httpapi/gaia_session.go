package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gaia-calendar/ent"
	"gaia-calendar/ent/gaiasession"
	"gaia-calendar/ent/user"
	"gaia-calendar/internal/gaia"
)

type gaiaSessionPayload struct {
	EmployeeName        string              `json:"employeeName,omitempty"`
	LeaveBalances       []gaia.LeaveBalance `json:"leaveBalances,omitempty"`
	SyncedAt            time.Time           `json:"syncedAt,omitempty"`
	ConsecutiveFailures int                 `json:"consecutiveFailures,omitempty"`
	LastError           string              `json:"lastError,omitempty"`
	LastFailureEmailAt  *time.Time          `json:"lastFailureEmailAt,omitempty"`
}

func (s *Server) loadGaiaSessionPayload(ctx context.Context, u *ent.User) (gaiaSessionPayload, *ent.GaiaSession) {
	session, err := s.db.GaiaSession.Query().
		Where(gaiasession.HasUserWith(user.ID(u.ID))).
		Only(ctx)
	if err != nil {
		return gaiaSessionPayload{}, nil
	}
	decrypted, err := s.encryptor.Decrypt(session.EncryptedPayload)
	if err != nil {
		return gaiaSessionPayload{}, session
	}
	var payload gaiaSessionPayload
	if err := json.Unmarshal([]byte(decrypted), &payload); err != nil {
		return gaiaSessionPayload{}, session
	}
	return payload, session
}

func (s *Server) saveGaiaSessionPayload(ctx context.Context, u *ent.User, companyCode string, payload gaiaSessionPayload, existing *ent.GaiaSession) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	encrypted, err := s.encryptor.Encrypt(string(raw))
	if err != nil {
		return err
	}
	if existing != nil {
		update := s.db.GaiaSession.UpdateOne(existing).
			SetCompanyCode(companyCode).
			SetEncryptedPayload(encrypted)
		if payload.LastError == "" {
			update.ClearLastError()
		} else {
			update.SetLastError(payload.LastError)
		}
		_, err = update.Save(ctx)
		return err
	}
	create := s.db.GaiaSession.Create().
		SetUser(u).
		SetCompanyCode(companyCode).
		SetEncryptedPayload(encrypted)
	if payload.LastError != "" {
		create.SetLastError(payload.LastError)
	}
	_, err = create.Save(ctx)
	return err
}

func (s *Server) recordGaiaSyncSuccess(ctx context.Context, u *ent.User, companyCode string, profile gaia.EmployeeProfile, balances []gaia.LeaveBalance) {
	payload, existing := s.loadGaiaSessionPayload(ctx, u)
	if profile.Name != "" {
		payload.EmployeeName = profile.Name
	}
	payload.LeaveBalances = balances
	payload.SyncedAt = time.Now()
	payload.ConsecutiveFailures = 0
	payload.LastError = ""
	if err := s.saveGaiaSessionPayload(ctx, u, companyCode, payload, existing); err != nil {
		slog.Warn("failed to save Gaia session snapshot", "userID", u.ID, "error", err)
	}
}

func (s *Server) recordGaiaSyncFailure(ctx context.Context, u *ent.User, companyCode string, syncErr error) {
	payload, existing := s.loadGaiaSessionPayload(ctx, u)
	payload.ConsecutiveFailures++
	payload.LastError = syncErr.Error()
	now := time.Now()
	shouldEmail := payload.ConsecutiveFailures >= 3 &&
		(payload.LastFailureEmailAt == nil || now.Sub(*payload.LastFailureEmailAt) >= 24*time.Hour)
	if shouldEmail {
		if err := s.emailer.SendGaiaCredentialWarning(ctx, u.Email, syncErr.Error(), "zh-HK"); err != nil {
			slog.Warn("failed to send Gaia credential warning", "userID", u.ID, "error", err)
		} else {
			payload.LastFailureEmailAt = &now
		}
	}
	if err := s.saveGaiaSessionPayload(ctx, u, companyCode, payload, existing); err != nil {
		slog.Warn("failed to save Gaia sync failure snapshot", "userID", u.ID, "error", err)
	}
}
