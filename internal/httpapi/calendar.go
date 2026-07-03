package httpapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gaia-calendar/ent"
	"gaia-calendar/ent/calendarrequestlog"
	"gaia-calendar/ent/calendarsubscription"
	"gaia-calendar/ent/gaiacredential"
	"gaia-calendar/ent/scheduleentry"
	"gaia-calendar/ent/user"
	"gaia-calendar/internal/gaia"
	"gaia-calendar/internal/security"
)

func (s *Server) handleGetCalendarSubscription(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	token, err := s.ensureCalendarSubscriptionToken(r, u)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load calendar subscription")
		return
	}
	writeJSON(w, http.StatusOK, calendarSubscriptionPayload(s.cfg.BaseURL, token))
}

func (s *Server) handleRotateCalendarSubscription(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	token, err := security.RandomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create calendar token")
		return
	}
	encrypted, err := s.encryptor.Encrypt(token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to protect calendar token")
		return
	}
	existing, err := s.db.CalendarSubscription.Query().
		Where(calendarsubscription.HasUserWith(user.ID(u.ID))).
		Only(r.Context())
	if err == nil {
		if _, err := s.db.CalendarSubscription.UpdateOne(existing).
			SetTokenHash(security.HashToken(token)).
			SetEncryptedToken(encrypted).
			SetEnabled(true).
			Save(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to rotate calendar token")
			return
		}
	} else if _, err := s.db.CalendarSubscription.Create().
		SetUser(u).
		SetTokenHash(security.HashToken(token)).
		SetEncryptedToken(encrypted).
		SetEnabled(true).
		Save(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create calendar token")
		return
	}
	writeJSON(w, http.StatusOK, calendarSubscriptionPayload(s.cfg.BaseURL, token))
}

func (s *Server) ensureCalendarSubscriptionToken(r *http.Request, u *ent.User) (string, error) {
	existing, err := s.db.CalendarSubscription.Query().
		Where(calendarsubscription.HasUserWith(user.ID(u.ID))).
		Only(r.Context())
	if err == nil {
		token, err := s.encryptor.Decrypt(existing.EncryptedToken)
		if err == nil && token != "" {
			return token, nil
		}
	}
	token, err := security.RandomToken(32)
	if err != nil {
		return "", err
	}
	encrypted, err := s.encryptor.Encrypt(token)
	if err != nil {
		return "", err
	}
	if existing != nil {
		_, err = s.db.CalendarSubscription.UpdateOne(existing).
			SetTokenHash(security.HashToken(token)).
			SetEncryptedToken(encrypted).
			SetEnabled(true).
			Save(r.Context())
	} else {
		_, err = s.db.CalendarSubscription.Create().
			SetUser(u).
			SetTokenHash(security.HashToken(token)).
			SetEncryptedToken(encrypted).
			SetEnabled(true).
			Save(r.Context())
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

func calendarSubscriptionPayload(baseURL, token string) map[string]string {
	httpsURL := calendarFeedURL(baseURL, token)
	webcalURL := httpsURL
	if strings.HasPrefix(webcalURL, "https://") {
		webcalURL = "webcal://" + strings.TrimPrefix(webcalURL, "https://")
	} else if strings.HasPrefix(webcalURL, "http://") {
		webcalURL = "webcal://" + strings.TrimPrefix(webcalURL, "http://")
	}
	return map[string]string{"url": httpsURL, "webcalUrl": webcalURL}
}

func calendarFeedURL(baseURL, token string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "/calendar/" + token + ".ics"
	}
	parsed.Path = "/calendar/" + token + ".ics"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func (s *Server) handleCalendarFeed(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/calendar/"), ".ics")
	if token == "" || strings.Contains(token, "/") {
		writeError(w, http.StatusNotFound, "calendar not found")
		return
	}
	sub, err := s.db.CalendarSubscription.Query().
		Where(calendarsubscription.TokenHash(security.HashToken(token)), calendarsubscription.Enabled(true)).
		WithUser().
		Only(r.Context())
	if err != nil || sub.Edges.User == nil {
		writeError(w, http.StatusNotFound, "calendar not found")
		return
	}
	s.recordCalendarRequest(r, sub)
	entries, err := s.db.ScheduleEntry.Query().
		Where(scheduleentry.HasUserWith(user.ID(sub.Edges.User.ID))).
		Order(ent.Asc(scheduleentry.FieldShiftDate)).
		All(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load calendar")
		return
	}
	employee := ""
	if cred, err := s.db.GaiaCredential.Query().Where(gaiacredential.HasUserWith(user.ID(sub.Edges.User.ID))).Only(r.Context()); err == nil {
		employee = cred.EmployeeAccount
	}
	payload, _ := s.loadGaiaSessionPayload(r.Context(), sub.Edges.User)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(buildICS(sub.Edges.User.ID, calendarMetadata{
		EmployeeAccount: employee,
		EmployeeName:    payload.EmployeeName,
		LeaveBalances:   payload.LeaveBalances,
	}, entries)))
}

func (s *Server) recordCalendarRequest(r *http.Request, sub *ent.CalendarSubscription) {
	_, _ = s.db.CalendarRequestLog.Create().
		SetSubscription(sub).
		SetUserAgent(strings.TrimSpace(r.UserAgent())).
		SetRemoteAddr(clientIP(r)).
		SetPath(r.URL.Path).
		Save(r.Context())
}

func (s *Server) handleGetCalendarRequestLogs(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	logs, err := s.db.CalendarRequestLog.Query().
		Where(calendarrequestlog.HasSubscriptionWith(calendarsubscription.HasUserWith(user.ID(u.ID)))).
		Order(ent.Desc(calendarrequestlog.FieldRequestedAt)).
		Limit(20).
		All(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load calendar request logs")
		return
	}
	out := make([]calendarRequestLogResponse, 0, len(logs))
	for _, log := range logs {
		out = append(out, calendarRequestLogResponse{
			ID:          log.ID,
			RequestedAt: log.RequestedAt,
			UserAgent:   log.UserAgent,
			RemoteAddr:  log.RemoteAddr,
			Path:        log.Path,
		})
	}
	writeJSON(w, http.StatusOK, calendarRequestLogsResponse{Logs: out})
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		return strings.TrimSpace(first)
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := strings.Cut(r.RemoteAddr, ":")
	if err {
		return strings.TrimSpace(host)
	}
	return strings.TrimSpace(r.RemoteAddr)
}

type calendarMetadata struct {
	EmployeeAccount string
	EmployeeName    string
	LeaveBalances   []gaia.LeaveBalance
}

func buildICS(userID int, meta calendarMetadata, entries []*ent.ScheduleEntry) string {
	var b strings.Builder
	now := time.Now().UTC().Format("20060102T150405Z")
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Gaia Calendar//Schedule//ZH-HK\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	b.WriteString("X-WR-CALNAME:Gaia Schedule\r\n")
	b.WriteString("REFRESH-INTERVAL;VALUE=DURATION:PT30M\r\n")
	b.WriteString("X-PUBLISHED-TTL:PT30M\r\n")
	for _, total := range monthlyScheduleTotals(entries) {
		start := monthNoteTime(total.Month, 0)
		end := monthNoteTime(total.Month, 5)
		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString("UID:" + icsEscape(fmt.Sprintf("gaia-%d-%s-total@gaia-calendar", userID, total.Month.Format("200601"))) + "\r\n")
		b.WriteString("DTSTAMP:" + now + "\r\n")
		b.WriteString("DTSTART:" + start.Format("20060102T150405") + "\r\n")
		b.WriteString("DTEND:" + end.Format("20060102T150405") + "\r\n")
		b.WriteString("SUMMARY:" + icsEscape(total.Month.Format("2006-01")+" 預計總工時 "+formatHours(total.Hours)) + "\r\n")
		b.WriteString("END:VEVENT\r\n")
	}
	for _, event := range leaveBalanceEvents(entries, meta.LeaveBalances) {
		start := monthNoteTime(event.Month, 5)
		end := monthNoteTime(event.Month, 10)
		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString("UID:" + icsEscape(fmt.Sprintf("gaia-%d-%s-leave-balance@gaia-calendar", userID, event.Month.Format("200601"))) + "\r\n")
		b.WriteString("DTSTAMP:" + now + "\r\n")
		b.WriteString("DTSTART:" + start.Format("20060102T150405") + "\r\n")
		b.WriteString("DTEND:" + end.Format("20060102T150405") + "\r\n")
		b.WriteString("SUMMARY:" + icsEscape(event.Summary) + "\r\n")
		if event.Description != "" {
			b.WriteString("DESCRIPTION:" + icsEscape(event.Description) + "\r\n")
		}
		b.WriteString("END:VEVENT\r\n")
	}
	for _, entry := range entries {
		if shouldHideScheduleEntry(entry) {
			continue
		}
		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString("UID:" + icsEscape(fmt.Sprintf("gaia-%d-%s@gaia-calendar", userID, entry.ShiftDate.Format("20060102"))) + "\r\n")
		b.WriteString("DTSTAMP:" + now + "\r\n")
		if entry.StartTime != nil && entry.EndTime != nil {
			b.WriteString("DTSTART:" + entry.StartTime.Format("20060102T150405") + "\r\n")
			b.WriteString("DTEND:" + entry.EndTime.Format("20060102T150405") + "\r\n")
		} else {
			b.WriteString("DTSTART;VALUE=DATE:" + entry.ShiftDate.Format("20060102") + "\r\n")
			b.WriteString("DTEND;VALUE=DATE:" + entry.ShiftDate.AddDate(0, 0, 1).Format("20060102") + "\r\n")
		}
		b.WriteString("SUMMARY:" + icsEscape(scheduleSummary(entry)) + "\r\n")
		if description := scheduleDescription(meta, entry); description != "" {
			b.WriteString("DESCRIPTION:" + icsEscape(description) + "\r\n")
		}
		b.WriteString("END:VEVENT\r\n")
	}
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

type monthlyScheduleTotal struct {
	Month time.Time
	Hours float64
}

func monthlyScheduleTotals(entries []*ent.ScheduleEntry) []monthlyScheduleTotal {
	totals := map[string]monthlyScheduleTotal{}
	keys := make([]string, 0)
	for _, entry := range entries {
		if !countsTowardWorkHours(entry) {
			continue
		}
		month := time.Date(entry.ShiftDate.Year(), entry.ShiftDate.Month(), 1, 0, 0, 0, 0, entry.ShiftDate.Location())
		key := month.Format("2006-01")
		total, ok := totals[key]
		if !ok {
			total = monthlyScheduleTotal{Month: month}
			keys = append(keys, key)
		}
		total.Hours += *entry.Hours
		totals[key] = total
	}
	out := make([]monthlyScheduleTotal, 0, len(keys))
	for _, key := range keys {
		out = append(out, totals[key])
	}
	return out
}

func monthNoteTime(month time.Time, minute int) time.Time {
	return time.Date(month.Year(), month.Month(), 1, 0, minute, 0, 0, month.Location())
}

type leaveBalanceEvent struct {
	Month       time.Time
	Summary     string
	Description string
}

func leaveBalanceEvents(entries []*ent.ScheduleEntry, balances []gaia.LeaveBalance) []leaveBalanceEvent {
	if len(balances) == 0 {
		return nil
	}
	primary, ok := primaryLeaveBalance(balances)
	if !ok {
		return nil
	}
	months := scheduleMonths(entries)
	out := make([]leaveBalanceEvent, 0, len(months))
	for _, month := range months {
		out = append(out, leaveBalanceEvent{
			Month:       month,
			Summary:     leaveBalanceSummary(primary),
			Description: leaveBalanceDescription(balances),
		})
	}
	return out
}

func scheduleMonths(entries []*ent.ScheduleEntry) []time.Time {
	seen := map[string]bool{}
	months := make([]time.Time, 0)
	for _, entry := range entries {
		month := time.Date(entry.ShiftDate.Year(), entry.ShiftDate.Month(), 1, 0, 0, 0, 0, entry.ShiftDate.Location())
		key := month.Format("2006-01")
		if seen[key] {
			continue
		}
		seen[key] = true
		months = append(months, month)
	}
	return months
}

func primaryLeaveBalance(balances []gaia.LeaveBalance) (gaia.LeaveBalance, bool) {
	if len(balances) == 0 {
		return gaia.LeaveBalance{}, false
	}
	for _, balance := range balances {
		name := strings.ToLower(balance.Name + " " + balance.CreditID)
		if strings.Contains(name, "年假") || strings.Contains(name, "annual") || strings.Contains(name, "al") {
			return balance, true
		}
	}
	return balances[0], true
}

func leaveBalanceSummary(balance gaia.LeaveBalance) string {
	name := balance.Name
	if name == "" {
		name = balance.CreditID
	}
	if name == "" {
		name = "假期"
	}
	return name + " 已使用 " + formatBalanceAmount(balance.Used, balance.Unit) +
		" / 總數 " + formatBalanceAmount(balance.Total, balance.Unit) +
		" / 剩餘 " + formatBalanceAmount(balance.Remaining, balance.Unit)
}

func leaveBalanceDescription(balances []gaia.LeaveBalance) string {
	lines := make([]string, 0, len(balances))
	for _, balance := range balances {
		lines = append(lines, leaveBalanceSummary(balance))
	}
	return strings.Join(lines, "\n")
}

func formatBalanceAmount(value *float64, unit string) string {
	if value == nil {
		return "-"
	}
	return strings.TrimSuffix(formatHours(*value), "h") + unit
}

func scheduleSummary(entry *ent.ScheduleEntry) string {
	name := "排班"
	if entry.ShiftName != nil && *entry.ShiftName != "" {
		name = *entry.ShiftName
	} else if entry.ClassCode != nil && *entry.ClassCode != "" {
		name = *entry.ClassCode
	}
	return name
}

func scheduleDescription(meta calendarMetadata, entry *ent.ScheduleEntry) string {
	parts := []string{}
	if meta.EmployeeName != "" && meta.EmployeeAccount != "" {
		parts = append(parts, "Employee: "+meta.EmployeeName+" ("+meta.EmployeeAccount+")")
	} else if meta.EmployeeName != "" {
		parts = append(parts, "Employee: "+meta.EmployeeName)
	} else if meta.EmployeeAccount != "" {
		parts = append(parts, "Employee: "+meta.EmployeeAccount)
	}
	if entry.ClassCode != nil && *entry.ClassCode != "" {
		parts = append(parts, "Class: "+*entry.ClassCode)
	}
	if entry.StartTime != nil && entry.EndTime != nil {
		parts = append(parts, "Time: "+entry.StartTime.Format("15:04")+"-"+entry.EndTime.Format("15:04"))
	}
	if entry.Hours != nil {
		parts = append(parts, "Hours: "+formatHours(*entry.Hours))
	}
	return strings.Join(parts, "\n")
}

func formatHours(hours float64) string {
	if hours == float64(int(hours)) {
		return strconv.Itoa(int(hours)) + "h"
	}
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(hours, 'f', 2, 64), "0"), ".") + "h"
}

func icsEscape(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\r\n", "\\n")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, ";", "\\;")
	value = strings.ReplaceAll(value, ",", "\\,")
	return value
}
