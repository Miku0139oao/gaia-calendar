package gaia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSyncScheduleUsesH5GatewayProxyForLogin(t *testing.T) {
	var loginSeen bool
	var encryptionSeen bool
	var parametersSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/h5-attendance/api/encryption/code":
			encryptionSeen = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": true,
				"data": map[string]any{
					"fragments": []string{"Rlc=", "dXQ=", "SXU=", "S1U="},
					"sequence":  "MiwwLDMsMQ==",
				},
			})
		case "/h5-attendance/api/appPunch/H5Standard/ValidateUserInfoByConfigure":
			if !encryptionSeen {
				t.Fatal("login request happened before encryption key lookup")
			}
			loginSeen = true
			if got := r.Header.Get("Url"); got != "/H5Standard/ValidateUserInfoByConfigure" {
				t.Fatalf("login Url header = %q", got)
			}
			if got := r.Header.Get("CompanyCode"); got != "ACMEHK" {
				t.Fatalf("login CompanyCode header = %q", got)
			}
			if got := r.Header.Get("X-Scene"); got != "login" {
				t.Fatalf("login X-SCENE header = %q", got)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode login payload: %v", err)
			}
			if got := payload["tokenValue"]; got == "" {
				t.Fatal("login tokenValue is empty")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": true,
				"data": map[string]any{
					"SessionId": "session-123",
					"PersonID":  "person-456",
					"UserID":    "user-789",
					"EmpID":     "emp-321",
				},
			})
		case "/h5-attendance/api/mobileCloud/v1/showNewConfigEncrypt":
			if !loginSeen {
				t.Fatal("cloud config request happened before login")
			}
			if got := r.URL.Query().Get("code"); got != "ACMEHK" {
				t.Fatalf("config code query = %q", got)
			}
			configJSON := `{
				"apiAddress": "https://api.example.test",
				"verNine": {"newShiftUrl": "https://shift.example.test"},
				"verEight": {"businessSystemVersion": "8"},
				"originConfig": {
					"javaAppPunchServerAddress": {"configValue": "https://punch.example.test"},
					"serverAttendanceJavaAdderss": {"configValue": "https://attendance.example.test"},
					"AttendanceJavaAdderss": {"configEnable": true}
				}
			}`
			encrypted, err := encryptDESECBPKCS7(configJSON, "IuFWKUut")
			if err != nil {
				t.Fatalf("encrypt config: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": true,
				"data":   encrypted,
			})
		case "/h5-attendance/api/wfm4integration/api/Schedule/GetScheduleParameter":
			if !loginSeen {
				t.Fatal("schedule parameter request happened before login")
			}
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"result": false})
		case "/h5-attendance/api/wfm4integration/Schedule/GetScheduleParameter",
			"/h5-attendance/api/netAttendance/Schedule/GetScheduleParameter",
			"/h5-attendance/api/javaAttendance/api/Schedule/GetScheduleParameter",
			"/h5-attendance/api/javaAttendance/Schedule/GetScheduleParameter",
			"/h5-attendance/api/appPunch/api/Schedule/GetScheduleParameter",
			"/h5-attendance/api/appPunch/Schedule/GetScheduleParameter",
			"/h5-attendance/api/BasicSchedule/api/Schedule/GetScheduleParameter",
			"/h5-attendance/api/BasicSchedule/Schedule/GetScheduleParameter",
			"/h5-attendance/api/interfaceCloud/api/Schedule/GetScheduleParameter":
			if !loginSeen {
				t.Fatal("schedule parameter request happened before login")
			}
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"result": false})
		case "/h5-attendance/api/netAttendance/api/Schedule/GetScheduleParameter":
			if !loginSeen {
				t.Fatal("schedule parameter request happened before login")
			}
			parametersSeen = true
			if got := r.Header.Get("Authorization"); got != "Bearer session-123" {
				t.Fatalf("parameter Authorization header = %q", got)
			}
			if got := r.Header.Get("Url"); got != "/api/Schedule/GetScheduleParameter" {
				t.Fatalf("parameter Url header = %q", got)
			}
			if got := r.URL.Query().Get("parameterCode"); got != "SS1034,SS1037,SS1082,SS1103,SS1142,SS1143,SS1154" {
				t.Fatalf("parameterCode query = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": true,
				"data": []map[string]any{
					{"parameterCode": "SS1037", "parameterValue": "show_ot"},
					{"parameterCode": "SS1103", "parameterValue": "show_meal"},
					{"parameterCode": "SS1082", "parameterValue": "show_job"},
				},
			})
		case "/h5-attendance/api/interfaceCloud/api/v2/schedule/getScheduleInfo":
			if !loginSeen {
				t.Fatal("schedule request happened before login")
			}
			if !parametersSeen {
				t.Fatal("schedule request happened before schedule parameter lookup")
			}
			if got := r.Header.Get("Authorization"); got != "Bearer session-123" {
				t.Fatalf("schedule Authorization header = %q", got)
			}
			if got := r.URL.Query().Get("sessionId"); got != "session-123" {
				t.Fatalf("schedule sessionId query = %q", got)
			}
			if got := r.URL.Query().Get("paramValue"); got != "show_ot" {
				t.Fatalf("schedule paramValue query = %q", got)
			}
			if got := r.URL.Query().Get("showMealConfig"); got != "show_meal" {
				t.Fatalf("schedule showMealConfig query = %q", got)
			}
			if got := r.URL.Query().Get("showJobParamValue"); got != "show_job" {
				t.Fatalf("schedule showJobParamValue query = %q", got)
			}
			if got := r.URL.Query().Get("appVersion"); got != "4.7.9" {
				t.Fatalf("schedule appVersion query = %q", got)
			}
			if got := r.URL.Query().Get("wfmFlag"); got != "8" {
				t.Fatalf("schedule wfmFlag query = %q", got)
			}
			if got := r.URL.Query().Get("apiAddress"); got == "" {
				t.Fatal("schedule apiAddress query is empty")
			}
			if got := r.URL.Query().Get("javaUrl"); got == "" {
				t.Fatal("schedule javaUrl query is empty")
			}
			if got := r.Header.Get("javaAppPunchServerAddress"); got == "" {
				t.Fatal("schedule javaAppPunchServerAddress header is empty")
			}
			if got := r.Header.Get("serverAttendanceJavaAdderss"); got == "" {
				t.Fatal("schedule serverAttendanceJavaAdderss header is empty")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": true,
				"data": []map[string]any{
					{"shiftDate": "2026-07-03", "shiftName": "Morning"},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := HTTPClient{BaseURL: server.URL, Client: server.Client()}
	entries, err := client.SyncSchedule(
		t.Context(),
		Credential{CompanyCode: "ACMEHK", EmployeeAccount: "E123456"},
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("SyncSchedule returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].ShiftName == nil || *entries[0].ShiftName != "Morning" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestParseScheduleKeepsGaiaEmptyCalendarPlaceholdersWithoutNegativeHours(t *testing.T) {
	body := []byte(`{
		"result": true,
		"data": [
			{"shiftDate":"2026-07-01","shiftName":null,"startTime":null,"endTime":null,"hours":-1,"taskList":null,"list":null},
			{"shiftDate":"2026-07-02","shiftName":"早班","startTime":"09:00","endTime":"18:00","hours":8}
		]
	}`)

	entries, err := parseSchedule(body)
	if err != nil {
		t.Fatalf("parseSchedule returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected calendar placeholders and real entries, got %#v", entries)
	}
	if entries[0].ShiftDate.Format("2006-01-02") != "2026-07-01" {
		t.Fatalf("unexpected entry date: %s", entries[0].ShiftDate.Format("2006-01-02"))
	}
	if entries[0].Hours != nil {
		t.Fatalf("placeholder should not keep negative hours: %#v", entries[0].Hours)
	}
	if entries[0].ClassCode == nil || *entries[0].ClassCode != "no_schedule" {
		t.Fatalf("placeholder class code = %#v", entries[0].ClassCode)
	}
}

func TestParseScheduleUsesAlternateAndNestedTimeFields(t *testing.T) {
	body := []byte(`{
		"result": true,
		"data": [
			{
				"shiftDate":"2026-07-03",
				"shiftName":null,
				"startTime":null,
				"endTime":null,
				"hours":-1,
				"taskList":[
					{"timeClassName":"開舖","startTime1":"08:30","endTime1":"12:30","hours":4},
					{"timeClassName":"收舖","beginTime":"13:30","endTime":"18:30","hours":5}
				]
			}
		]
	}`)

	entries, err := parseSchedule(body)
	if err != nil {
		t.Fatalf("parseSchedule returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one merged schedule entry, got %#v", entries)
	}
	entry := entries[0]
	if entry.ShiftName == nil || *entry.ShiftName != "開舖 / 收舖" {
		t.Fatalf("unexpected shift name: %#v", entry.ShiftName)
	}
	if entry.StartTime == nil || entry.StartTime.Format("15:04") != "08:30" {
		t.Fatalf("unexpected start time: %#v", entry.StartTime)
	}
	if entry.EndTime == nil || entry.EndTime.Format("15:04") != "18:30" {
		t.Fatalf("unexpected end time: %#v", entry.EndTime)
	}
	if entry.Hours == nil || *entry.Hours != 9 {
		t.Fatalf("unexpected hours: %#v", entry.Hours)
	}
}

func TestParseScheduleUsesLaborAccountNestedSchedule(t *testing.T) {
	body := []byte(`{
		"result": true,
		"data": [
			{
				"shiftDate":"2026-07-01",
				"schedule":[
					{
						"scheduleDate":"2026-07-01",
						"scheduleHour":8.5,
						"beginTime":"13:30:00",
						"endTime":"23:00:00",
						"timeClassCode":"TA002",
						"timeClassName":"B"
					}
				]
			}
		]
	}`)

	entries, err := parseSchedule(body)
	if err != nil {
		t.Fatalf("parseSchedule returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one schedule entry, got %#v", entries)
	}
	entry := entries[0]
	if entry.ShiftName == nil || *entry.ShiftName != "B" {
		t.Fatalf("unexpected shift name: %#v", entry.ShiftName)
	}
	if entry.StartTime == nil || entry.StartTime.Format("15:04") != "13:30" {
		t.Fatalf("unexpected start time: %#v", entry.StartTime)
	}
	if entry.EndTime == nil || entry.EndTime.Format("15:04") != "23:00" {
		t.Fatalf("unexpected end time: %#v", entry.EndTime)
	}
	if entry.Hours == nil || *entry.Hours != 8.5 {
		t.Fatalf("unexpected hours: %#v", entry.Hours)
	}
	if entry.ClassCode == nil || *entry.ClassCode != "TA002" {
		t.Fatalf("unexpected class code: %#v", entry.ClassCode)
	}
}

func TestParseScheduleTreatsLaborAccountOffDayAsNoTimedShift(t *testing.T) {
	body := []byte(`{
		"result": true,
		"data": [
			{
				"shiftDate":"2026-07-04",
				"schedule":[
					{
						"scheduleDate":"2026-07-04",
						"scheduleHour":0,
						"beginTime":"00:00:00",
						"endTime":"00:02:00",
						"timeClassCode":"OFF",
						"timeClassName":"OFF"
					}
				]
			}
		]
	}`)

	entries, err := parseSchedule(body)
	if err != nil {
		t.Fatalf("parseSchedule returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one schedule entry, got %#v", entries)
	}
	entry := entries[0]
	if entry.ShiftName == nil || *entry.ShiftName != "OFF" {
		t.Fatalf("unexpected shift name: %#v", entry.ShiftName)
	}
	if entry.StartTime != nil || entry.EndTime != nil {
		t.Fatalf("OFF day should not keep synthetic times: start=%#v end=%#v", entry.StartTime, entry.EndTime)
	}
	if entry.Hours == nil || *entry.Hours != 0 {
		t.Fatalf("unexpected hours: %#v", entry.Hours)
	}
}

func TestParseLeaveBalancesUsesGaiaCreditFields(t *testing.T) {
	body := []byte(`{
		"result": true,
		"data": [
			{"creaditId":"HK-AL","creaditName":"年假","creaditUsed":3.5,"creaditTotal":12,"creaditRemain":8.5,"creaditUnit":"0"}
		]
	}`)

	balances, err := parseLeaveBalances(body)
	if err != nil {
		t.Fatalf("parseLeaveBalances returned error: %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("expected one balance, got %#v", balances)
	}
	got := balances[0]
	if got.CreditID != "HK-AL" || got.Name != "年假" || got.Unit != "天" {
		t.Fatalf("unexpected balance identity: %#v", got)
	}
	if got.Used == nil || *got.Used != 3.5 {
		t.Fatalf("used = %#v", got.Used)
	}
	if got.Total == nil || *got.Total != 12 {
		t.Fatalf("total = %#v", got.Total)
	}
	if got.Remaining == nil || *got.Remaining != 8.5 {
		t.Fatalf("remaining = %#v", got.Remaining)
	}
}
