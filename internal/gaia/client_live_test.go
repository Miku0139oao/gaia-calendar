package gaia

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestLiveGaiaSchedule(t *testing.T) {
	companyCode := os.Getenv("GAIA_LIVE_COMPANY_CODE")
	account := os.Getenv("GAIA_LIVE_EMPLOYEE_ACCOUNT")
	password := os.Getenv("GAIA_LIVE_PASSWORD")
	if companyCode == "" || account == "" || password == "" {
		t.Skip("set GAIA_LIVE_COMPANY_CODE, GAIA_LIVE_EMPLOYEE_ACCOUNT, and GAIA_LIVE_PASSWORD to run live Gaia schedule probe")
	}

	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Timeout: 25 * time.Second, Jar: jar}
	client := HTTPClient{BaseURL: "https://gateway.gaiacloud.com", Client: httpClient}
	entries, err := client.SyncSchedule(
		t.Context(),
		Credential{CompanyCode: companyCode, EmployeeAccount: account, Password: password},
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local),
		time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local),
	)
	if err != nil {
		t.Fatalf("SyncSchedule returned error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("SyncSchedule returned no entries")
	}
	realEntries := 0
	for _, entry := range entries {
		if entry.StartTime != nil || entry.EndTime != nil || entry.Hours != nil {
			realEntries++
			t.Logf("%s shift=%s start=%s end=%s hours=%s class=%s",
				entry.ShiftDate.Format("2006-01-02"),
				stringPtrValue(entry.ShiftName),
				timePtrValue(entry.StartTime),
				timePtrValue(entry.EndTime),
				floatPtrValue(entry.Hours),
				stringPtrValue(entry.ClassCode),
			)
		}
	}
	if realEntries == 0 {
		probeLiveScheduleEndpoints(t, client, httpClient, Credential{CompanyCode: companyCode, EmployeeAccount: account, Password: password})
		limit := len(entries)
		if limit > 3 {
			limit = 3
		}
		for i := 0; i < limit; i++ {
			t.Logf("placeholder[%d] raw=%s", i, entries[i].RawJSON)
		}
		t.Fatalf("SyncSchedule returned %d placeholder entries but no dated shift times", len(entries))
	}
}

func probeLiveScheduleEndpoints(t *testing.T, client HTTPClient, httpClient *http.Client, cred Credential) {
	session, err := client.validateH5User(t.Context(), httpClient, cred)
	if err != nil {
		t.Logf("probe login failed: %v", err)
		return
	}
	cloud, err := client.fetchCloudConfig(t.Context(), httpClient, cred, session.EncryptionKey)
	if err != nil {
		t.Logf("probe cloud config failed: %v", err)
		return
	}
	params, err := client.fetchScheduleParameters(t.Context(), httpClient, cred, session)
	if err != nil {
		t.Logf("probe schedule parameters failed: %v", err)
		return
	}
	values := url.Values{}
	values.Set("beginDate", "2026-07-01")
	values.Set("endDate", "2026-07-31")
	values.Set("companyCode", cred.CompanyCode)
	values.Set("userId", firstNonEmpty(session.UserID, cred.EmployeeAccount))
	values.Set("personId", firstNonEmpty(session.PersonID, cred.EmployeeAccount))
	values.Set("sessionId", session.SessionID)
	values.Set("isNeedEvent", "true")
	values.Set("standardH5", "true")
	values.Set("isUsedAccrual", "false")
	values.Set("isUseNewCalculate", "false")
	values.Set("appVersion", "4.7.9")
	if cloud.WFMFlag != "" {
		values.Set("wfmFlag", cloud.WFMFlag)
	}
	if params.ParamValue != "" {
		values.Set("paramValue", params.ParamValue)
	}
	if params.ShowMealConfig != "" {
		values.Set("showMealConfig", params.ShowMealConfig)
	}
	if params.ShowJobParamValue != "" {
		values.Set("showJobParamValue", params.ShowJobParamValue)
	}
	if cloud.APIAddress != "" {
		encrypted, _ := encryptDESECBPKCS7(cloud.APIAddress, session.EncryptionKey)
		values.Set("apiAddress", encrypted)
	}
	if cloud.ShiftAddress != "" {
		encrypted, _ := encryptDESECBPKCS7(cloud.ShiftAddress, session.EncryptionKey)
		values.Set("javaUrl", encrypted)
		values.Set("forwardUrl", encrypted)
	}
	candidates := []struct {
		path string
		url  string
	}{
		{"/h5-attendance/api/interfaceCloud/api/v2/schedule/getScheduleInfo", "/api/v2/schedule/getScheduleInfo"},
		{"/h5-attendance/api/javaAttendance/Schedule/v2/getScheduleInfo", "/Schedule/v2/getScheduleInfo"},
		{"/h5-attendance/api/interfaceCloud/api/v2/schedule/getEmployeeStandardScheduleInfoNew", "/api/v2/schedule/getEmployeeStandardScheduleInfoNew"},
		{"/h5-attendance/api/javaAttendance/Schedule/v2/getEmployeeStandardScheduleInfoNew", "/Schedule/v2/getEmployeeStandardScheduleInfoNew"},
		{"/h5-attendance/api/interfaceCloud/api/v1/schedule/getEmployeeBranchInfo", "/api/v1/schedule/getEmployeeBranchInfo"},
		{"/h5-attendance/api/javaAttendance/Schedule/v1/getEmployeeBranchInfo", "/Schedule/v1/getEmployeeBranchInfo"},
		{"/h5-attendance/api/interfaceCloud/api/schedule/v2/getUserInfoOneDay", "/api/schedule/v2/getUserInfoOneDay"},
		{"/h5-attendance/api/javaAttendance/Schedule/V2/getUserInfoOneDay", "/Schedule/V2/getUserInfoOneDay"},
	}
	for _, candidate := range candidates {
		got, err := client.fetchScheduleFromPath(t.Context(), httpClient, cred, session, cloud, candidate.path, candidate.url, values)
		if err != nil {
			t.Logf("probe %s failed: %v", candidate.path, err)
			continue
		}
		t.Logf("probe %s entries=%d real=%t", candidate.path, len(got), hasDatedShiftTimes(got))
		if len(got) > 0 {
			t.Logf("probe %s first=%s", candidate.path, got[0].RawJSON)
		}
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func timePtrValue(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("15:04")
}

func floatPtrValue(value *float64) string {
	if value == nil {
		return ""
	}
	return time.Duration(*value * float64(time.Hour)).String()
}
