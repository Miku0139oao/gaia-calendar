package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gaia-calendar/ent"
	"gaia-calendar/ent/gaiacredential"
	"gaia-calendar/ent/scheduleentry"
	"gaia-calendar/ent/schedulesyncrun"
	"gaia-calendar/ent/user"
	"gaia-calendar/internal/gaia"
)

func (s *Server) handleGetGaiaCredential(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	cred, err := s.db.GaiaCredential.Query().Where(gaiacredential.HasUserWith(user.ID(u.ID))).Only(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"credential": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"credential": map[string]any{
		"companyCode":     cred.CompanyCode,
		"employeeAccount": cred.EmployeeAccount,
		"status":          cred.CredentialStatus,
		"lastLoginAt":     cred.LastLoginAt,
	}})
}

func (s *Server) handleSaveGaiaCredential(w http.ResponseWriter, r *http.Request) {
	var req gaiaCredentialRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.CompanyCode = strings.TrimSpace(req.CompanyCode)
	req.EmployeeAccount = strings.TrimSpace(req.EmployeeAccount)
	if req.CompanyCode == "" {
		req.CompanyCode = s.cfg.GaiaDefaultCompanyCode
	}
	if req.EmployeeAccount == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "company code, employee account, and password are required")
		return
	}
	encrypted, err := s.encryptor.Encrypt(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt Gaia password")
		return
	}
	u := currentUser(r)
	existing, err := s.db.GaiaCredential.Query().Where(gaiacredential.HasUserWith(user.ID(u.ID))).Only(r.Context())
	if err == nil {
		existing, err = s.db.GaiaCredential.UpdateOne(existing).
			SetCompanyCode(req.CompanyCode).
			SetEmployeeAccount(req.EmployeeAccount).
			SetEncryptedPassword(encrypted).
			SetCredentialStatus("saved").
			Save(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update Gaia credential")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "credential": existing.EmployeeAccount})
		return
	}
	created, err := s.db.GaiaCredential.Create().
		SetUser(u).
		SetCompanyCode(req.CompanyCode).
		SetEmployeeAccount(req.EmployeeAccount).
		SetEncryptedPassword(encrypted).
		SetCredentialStatus("saved").
		Save(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save Gaia credential")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "credential": created.EmployeeAccount})
}

func (s *Server) handleSyncSchedules(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	entryCount, err := s.syncSchedulesForUser(r.Context(), u)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "entryCount": entryCount})
}

func (s *Server) syncSchedulesForUser(ctx context.Context, u *ent.User) (int, error) {
	cred, err := s.db.GaiaCredential.Query().Where(gaiacredential.HasUserWith(user.ID(u.ID))).Only(ctx)
	if err != nil {
		return 0, err
	}
	password, err := s.encryptor.Decrypt(cred.EncryptedPassword)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -3, 0)
	end := start.AddDate(0, 10, -1)
	run, err := s.db.ScheduleSyncRun.Create().SetUser(u).SetRangeStart(start).SetRangeEnd(end).Save(ctx)
	if err != nil {
		return 0, err
	}
	credential := gaia.Credential{
		CompanyCode:     cred.CompanyCode,
		EmployeeAccount: cred.EmployeeAccount,
		Password:        password,
	}
	entries := make([]gaia.ScheduleEntry, 0)
	for monthStart := start; !monthStart.After(end); monthStart = monthStart.AddDate(0, 1, 0) {
		monthEnd := monthStart.AddDate(0, 1, -1)
		monthEntries, syncErr := s.gaia.SyncSchedule(ctx, credential, monthStart, monthEnd)
		if syncErr != nil {
			_, _ = s.db.ScheduleSyncRun.UpdateOne(run).SetStatus("failed").SetFinishedAt(time.Now()).SetErrorMessage(syncErr.Error()).Save(ctx)
			_, _ = s.db.GaiaCredential.UpdateOne(cred).SetCredentialStatus("error").Save(ctx)
			s.recordGaiaSyncFailure(ctx, u, cred.CompanyCode, syncErr)
			return 0, syncErr
		}
		entries = append(entries, monthEntries...)
	}
	if _, err := s.db.ScheduleEntry.Delete().
		Where(scheduleentry.HasUserWith(user.ID(u.ID)), scheduleentry.ShiftDateGTE(start), scheduleentry.ShiftDateLTE(end)).
		Exec(ctx); err != nil {
		_, _ = s.db.ScheduleSyncRun.UpdateOne(run).SetStatus("failed").SetFinishedAt(time.Now()).SetErrorMessage("failed to replace existing schedules").Save(ctx)
		return 0, err
	}
	for _, entry := range entries {
		upsertScheduleEntry(ctx, s, u, entry)
	}
	profile, balances, balanceErr := s.gaia.SyncLeaveBalances(ctx, credential)
	if balanceErr != nil {
		existingPayload, _ := s.loadGaiaSessionPayload(ctx, u)
		slog.Warn("Gaia leave balance sync failed", "userID", u.ID, "error", balanceErr)
		s.recordGaiaSyncSuccess(ctx, u, cred.CompanyCode, gaia.EmployeeProfile{Name: existingPayload.EmployeeName}, existingPayload.LeaveBalances)
	} else {
		s.recordGaiaSyncSuccess(ctx, u, cred.CompanyCode, profile, balances)
	}
	_, _ = s.db.GaiaCredential.UpdateOne(cred).SetCredentialStatus("ok").SetLastLoginAt(time.Now()).Save(ctx)
	_, _ = s.db.ScheduleSyncRun.UpdateOne(run).SetStatus("success").SetFinishedAt(time.Now()).SetEntryCount(len(entries)).Save(ctx)
	return len(entries), nil
}

func upsertScheduleEntry(ctx context.Context, s *Server, u *ent.User, entry gaia.ScheduleEntry) {
	existing, err := s.db.ScheduleEntry.Query().
		Where(scheduleentry.HasUserWith(user.ID(u.ID)), scheduleentry.ShiftDateEQ(entry.ShiftDate)).
		Only(ctx)
	builder := s.db.ScheduleEntry.Create().SetUser(u).SetShiftDate(entry.ShiftDate)
	if err == nil {
		update := s.db.ScheduleEntry.UpdateOne(existing)
		applyScheduleUpdate(update, entry)
		_, _ = update.Save(ctx)
		return
	}
	if entry.ShiftName != nil {
		builder.SetShiftName(*entry.ShiftName)
	}
	if entry.StartTime != nil {
		builder.SetStartTime(*entry.StartTime)
	}
	if entry.EndTime != nil {
		builder.SetEndTime(*entry.EndTime)
	}
	if entry.Hours != nil {
		builder.SetHours(*entry.Hours)
	}
	if entry.ClassCode != nil {
		builder.SetClassCode(*entry.ClassCode)
	}
	if entry.RawJSON != "" {
		builder.SetRawJSON(entry.RawJSON)
	}
	_, _ = builder.Save(ctx)
}

func applyScheduleUpdate(update *ent.ScheduleEntryUpdateOne, entry gaia.ScheduleEntry) {
	if entry.ShiftName != nil {
		update.SetShiftName(*entry.ShiftName)
	} else {
		update.ClearShiftName()
	}
	if entry.StartTime != nil {
		update.SetStartTime(*entry.StartTime)
	} else {
		update.ClearStartTime()
	}
	if entry.EndTime != nil {
		update.SetEndTime(*entry.EndTime)
	} else {
		update.ClearEndTime()
	}
	if entry.Hours != nil {
		update.SetHours(*entry.Hours)
	} else {
		update.ClearHours()
	}
	if entry.ClassCode != nil {
		update.SetClassCode(*entry.ClassCode)
	} else {
		update.ClearClassCode()
	}
	if entry.RawJSON != "" {
		update.SetRawJSON(entry.RawJSON)
	}
	update.SetSyncedAt(time.Now())
}

func (s *Server) handleSchedules(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	start, err := time.Parse("2006-01", month)
	if err != nil {
		writeError(w, http.StatusBadRequest, "month must use YYYY-MM")
		return
	}
	end := start.AddDate(0, 1, 0)
	entries, err := s.db.ScheduleEntry.Query().
		Where(scheduleentry.HasUserWith(user.ID(u.ID)), scheduleentry.ShiftDateGTE(start), scheduleentry.ShiftDateLT(end)).
		Order(ent.Asc(scheduleentry.FieldShiftDate)).
		All(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load schedules")
		return
	}
	out := make([]scheduleResponse, 0, len(entries))
	var totalHours float64
	for _, entry := range entries {
		if shouldHideScheduleEntry(entry) {
			continue
		}
		if countsTowardWorkHours(entry) {
			totalHours += *entry.Hours
		}
		out = append(out, scheduleResponse{
			ID:        entry.ID,
			ShiftDate: entry.ShiftDate.Format("2006-01-02"),
			ShiftName: entry.ShiftName,
			StartTime: entry.StartTime,
			EndTime:   entry.EndTime,
			Hours:     entry.Hours,
			ClassCode: entry.ClassCode,
			Segments:  scheduleSegmentsFromRaw(entry.RawJSON),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedules": out, "totalHours": totalHours})
}

func isNoScheduleEntry(classCode *string) bool {
	return classCode != nil && strings.EqualFold(strings.TrimSpace(*classCode), "no_schedule")
}

func shouldHideScheduleEntry(entry *ent.ScheduleEntry) bool {
	if isNoScheduleEntry(entry.ClassCode) {
		return true
	}
	if entry.ClassCode != nil && strings.EqualFold(strings.TrimSpace(*entry.ClassCode), "OFF") {
		return true
	}
	if entry.ShiftName != nil && strings.EqualFold(strings.TrimSpace(*entry.ShiftName), "OFF") {
		return true
	}
	return false
}

func countsTowardWorkHours(entry *ent.ScheduleEntry) bool {
	if shouldHideScheduleEntry(entry) || entry.Hours == nil || *entry.Hours <= 0 {
		return false
	}
	return !rawScheduleIsEvent(entry.RawJSON)
}

func rawScheduleIsEvent(raw string) bool {
	if raw == "" {
		return false
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(firstRawString(obj, "isEvent")), "Y")
}

func scheduleSegmentsFromRaw(raw string) []scheduleSegmentResponse {
	if raw == "" {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return nil
	}
	segments := make([]scheduleSegmentResponse, 0)
	for _, parentKey := range []string{"schedule", "shiftSegment", "taskList", "list", "details"} {
		for _, item := range rawArray(obj[parentKey]) {
			segment, ok := item.(map[string]any)
			if !ok {
				continue
			}
			segments = append(segments, segmentFromRaw(segment))
			for _, childKey := range []string{"shiftSegment", "segments", "details"} {
				for _, child := range rawArray(segment[childKey]) {
					childSegment, ok := child.(map[string]any)
					if ok {
						segments = append(segments, segmentFromRaw(childSegment))
					}
				}
			}
		}
	}
	out := segments[:0]
	seen := map[string]bool{}
	namedTimeSpans := map[string]bool{}
	for _, segment := range segments {
		if segment.Name == "" && segment.StartTime == "" && segment.EndTime == "" && segment.Hours == nil && segment.ClassCode == "" {
			continue
		}
		if strings.EqualFold(segment.ClassCode, "OFF") && segment.Hours != nil && *segment.Hours == 0 {
			continue
		}
		timeKey := segment.StartTime + "|" + segment.EndTime
		if segment.Name == "" && segment.Hours == nil && segment.ClassCode == "" && namedTimeSpans[timeKey] {
			continue
		}
		key := segment.Name + "|" + segment.StartTime + "|" + segment.EndTime + "|" + segment.ClassCode
		if segment.Hours != nil {
			key += "|" + strings.TrimRight(strings.TrimRight(strconv.FormatFloat(*segment.Hours, 'f', 2, 64), "0"), ".")
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		if segment.Name != "" || segment.Hours != nil || segment.ClassCode != "" {
			namedTimeSpans[timeKey] = true
		}
		out = append(out, segment)
	}
	return out
}

func segmentFromRaw(obj map[string]any) scheduleSegmentResponse {
	return scheduleSegmentResponse{
		Name:      firstRawString(obj, "shiftName", "timeClassName", "className", "name", "shortName", "segmentName"),
		StartTime: cleanClock(firstRawString(obj, "startTime", "startTime1", "beginTime", "shiftTimeFrom")),
		EndTime:   cleanClock(firstRawString(obj, "endTime", "endTime1", "finishTime", "shiftTimeTo")),
		Hours:     firstRawNonNegativeFloat(obj, "hours", "workHours", "scheduleHour"),
		ClassCode: firstRawString(obj, "classCode", "timeClassCode"),
	}
}

func rawArray(v any) []any {
	items, _ := v.([]any)
	return items
}

func firstRawString(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := obj[key].(string); ok {
			return value
		}
	}
	return ""
}

func firstRawNonNegativeFloat(obj map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		var value float64
		switch v := obj[key].(type) {
		case float64:
			value = v
		case int:
			value = float64(v)
		default:
			continue
		}
		if value >= 0 {
			return &value
		}
	}
	return nil
}

func cleanClock(value string) string {
	if len(value) >= len("15:04") && strings.Count(value, ":") >= 1 {
		return value[:5]
	}
	return value
}

func (s *Server) handleSyncRuns(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	limit := queryInt(r, "limit", 10, 1, 50)
	offset := queryInt(r, "offset", 0, 0, 10000)
	total, err := s.db.ScheduleSyncRun.Query().
		Where(schedulesyncrun.HasUserWith(user.ID(u.ID))).
		Count(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count sync runs")
		return
	}
	runs, err := s.db.ScheduleSyncRun.Query().
		Where(schedulesyncrun.HasUserWith(user.ID(u.ID))).
		Order(ent.Desc(schedulesyncrun.FieldStartedAt)).
		Limit(limit).
		Offset(offset).
		All(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load sync runs")
		return
	}
	type syncRun struct {
		ID           int        `json:"id"`
		StartedAt    time.Time  `json:"startedAt"`
		FinishedAt   *time.Time `json:"finishedAt,omitempty"`
		Status       string     `json:"status"`
		ErrorMessage *string    `json:"errorMessage,omitempty"`
		EntryCount   int        `json:"entryCount"`
		Marked       bool       `json:"marked"`
	}
	out := make([]syncRun, 0, len(runs))
	for _, run := range runs {
		out = append(out, syncRun{ID: run.ID, StartedAt: run.StartedAt, FinishedAt: run.FinishedAt, Status: run.Status, ErrorMessage: run.ErrorMessage, EntryCount: run.EntryCount, Marked: run.MarkedAt != nil})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"runs":    out,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"hasMore": offset+len(out) < total,
	})
}

func queryInt(r *http.Request, key string, fallback, min, max int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func (s *Server) handleUpdateSyncRun(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, ok := syncRunIDFromPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "sync run not found")
		return
	}
	var req syncRunUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	run, err := s.db.ScheduleSyncRun.Query().
		Where(schedulesyncrun.ID(id), schedulesyncrun.HasUserWith(user.ID(u.ID))).
		Only(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "sync run not found")
		return
	}
	update := s.db.ScheduleSyncRun.UpdateOne(run)
	if req.Marked {
		update.SetMarkedAt(time.Now())
	} else {
		update.ClearMarkedAt()
	}
	if _, err := update.Save(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update sync run")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteSyncRun(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, ok := syncRunIDFromPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "sync run not found")
		return
	}
	if _, err := s.db.ScheduleSyncRun.Delete().
		Where(schedulesyncrun.ID(id), schedulesyncrun.HasUserWith(user.ID(u.ID))).
		Exec(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete sync run")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func syncRunIDFromPath(path string) (int, bool) {
	value := strings.TrimPrefix(path, "/api/sync-runs/")
	if value == "" || strings.Contains(value, "/") {
		return 0, false
	}
	id, err := strconv.Atoi(value)
	return id, err == nil
}

func debugJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
