package gaia

import (
	"bytes"
	"context"
	"crypto/des"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Credential struct {
	CompanyCode     string
	EmployeeAccount string
	Password        string
}

type ScheduleEntry struct {
	ShiftDate time.Time
	ShiftName *string
	StartTime *time.Time
	EndTime   *time.Time
	Hours     *float64
	ClassCode *string
	RawJSON   string
}

type EmployeeProfile struct {
	Name string `json:"name,omitempty"`
}

type LeaveBalance struct {
	CreditID  string   `json:"creditId,omitempty"`
	Name      string   `json:"name,omitempty"`
	Used      *float64 `json:"used,omitempty"`
	Total     *float64 `json:"total,omitempty"`
	Remaining *float64 `json:"remaining,omitempty"`
	Unit      string   `json:"unit,omitempty"`
	RawJSON   string   `json:"rawJson,omitempty"`
}

type sessionInfo struct {
	SessionID     string
	PersonID      string
	UserID        string
	EmpID         string
	EmployeeName  string
	EncryptionKey string
}

type scheduleParameters struct {
	ParamValue        string
	ShowMealConfig    string
	ShowJobParamValue string
}

type cloudConfig struct {
	APIAddress                         string
	ShiftAddress                       string
	WFMFlag                            string
	JavaAppPunchServerAddress          string
	ServerAttendanceJavaAddress        string
	ServerAttendanceJavaAddressEnabled bool
}

type Client interface {
	SyncSchedule(ctx context.Context, cred Credential, start, end time.Time) ([]ScheduleEntry, error)
	SyncLeaveBalances(ctx context.Context, cred Credential) (EmployeeProfile, []LeaveBalance, error)
}

type HTTPClient struct {
	BaseURL string
	Client  *http.Client
}

func NewClient() Client {
	jar, _ := cookiejar.New(nil)
	return HTTPClient{
		BaseURL: "https://gateway.gaiacloud.com",
		Client:  &http.Client{Timeout: 25 * time.Second, Jar: jar},
	}
}

func (c HTTPClient) SyncSchedule(ctx context.Context, cred Credential, start, end time.Time) ([]ScheduleEntry, error) {
	if cred.CompanyCode == "" || cred.EmployeeAccount == "" {
		return nil, fmt.Errorf("Gaia company code and employee account are required")
	}
	client := c.Client
	if client == nil {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{Timeout: 25 * time.Second, Jar: jar}
	}
	session, err := c.validateH5User(ctx, client, cred)
	if err != nil {
		return nil, err
	}
	cloud, err := c.fetchCloudConfig(ctx, client, cred, session.EncryptionKey)
	if err != nil {
		return nil, err
	}
	parameters, err := c.fetchScheduleParameters(ctx, client, cred, session)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("beginDate", start.Format("2006-01-02"))
	values.Set("endDate", end.Format("2006-01-02"))
	values.Set("companyCode", cred.CompanyCode)
	values.Set("userId", firstNonEmpty(session.UserID, cred.EmployeeAccount))
	values.Set("personId", firstNonEmpty(session.PersonID, cred.EmployeeAccount))
	values.Set("sessionId", session.SessionID)
	values.Set("isNeedEvent", "true")
	values.Set("standardH5", "true")
	values.Set("isUsedAccrual", "false")
	values.Set("isUseNewCalculate", "false")
	values.Set("appVersion", "4.7.9")
	if cloud.APIAddress != "" {
		encrypted, err := encryptDESECBPKCS7(cloud.APIAddress, session.EncryptionKey)
		if err != nil {
			return nil, err
		}
		values.Set("apiAddress", encrypted)
	}
	if cloud.ShiftAddress != "" {
		encrypted, err := encryptDESECBPKCS7(cloud.ShiftAddress, session.EncryptionKey)
		if err != nil {
			return nil, err
		}
		values.Set("javaUrl", encrypted)
	}
	if cloud.WFMFlag != "" {
		values.Set("wfmFlag", cloud.WFMFlag)
	}
	if parameters.ParamValue != "" {
		values.Set("paramValue", parameters.ParamValue)
	}
	if parameters.ShowMealConfig != "" {
		values.Set("showMealConfig", parameters.ShowMealConfig)
	}
	if parameters.ShowJobParamValue != "" {
		values.Set("showJobParamValue", parameters.ShowJobParamValue)
	}
	entries, err := c.fetchScheduleFromPath(ctx, client, cred, session, cloud, "/h5-attendance/api/interfaceCloud/api/v2/schedule/getScheduleInfo", "/api/v2/schedule/getScheduleInfo", values)
	if err != nil {
		return nil, err
	}
	if hasDatedShiftTimes(entries) {
		return entries, nil
	}
	directEntries, directErr := c.fetchScheduleFromPath(ctx, client, cred, session, cloud, "/h5-attendance/api/javaAttendance/Schedule/v2/getScheduleInfo", "/Schedule/v2/getScheduleInfo", values)
	if directErr == nil && (hasDatedShiftTimes(directEntries) || len(entries) == 0) {
		return directEntries, nil
	}
	laborValues := cloneValues(values)
	laborValues.Set("appVersion", "4.1.0")
	laborValues.Set("paramValue", firstNonEmpty(parameters.ParamValue, "0,1,2,3"))
	laborEntries, laborErr := c.fetchScheduleFromPath(ctx, client, cred, session, cloud, "/h5-laboraccount-attendance/api/Schedule/la/getScheduleInfo", "/Schedule/la/getScheduleInfo", laborValues)
	if laborErr == nil && (hasDatedShiftTimes(laborEntries) || len(entries) == 0) {
		return laborEntries, nil
	}
	return entries, nil
}

func (c HTTPClient) SyncLeaveBalances(ctx context.Context, cred Credential) (EmployeeProfile, []LeaveBalance, error) {
	if cred.CompanyCode == "" || cred.EmployeeAccount == "" {
		return EmployeeProfile{}, nil, fmt.Errorf("Gaia company code and employee account are required")
	}
	client := c.Client
	if client == nil {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{Timeout: 25 * time.Second, Jar: jar}
	}
	session, err := c.validateH5User(ctx, client, cred)
	if err != nil {
		return EmployeeProfile{}, nil, err
	}
	profile := EmployeeProfile{Name: session.EmployeeName}
	balances, err := c.fetchLeaveBalancesFromPath(ctx, client, cred, session, "/h5-attendance/api/javaAttendance/Leave/GetEmployeeLeaveBalanceNew")
	if err == nil {
		return profile, balances, nil
	}
	fallback, fallbackErr := c.fetchLeaveBalancesFromPath(ctx, client, cred, session, "/h5-attendance/api/netAttendance/Leave/GetEmployeeLeaveBalanceNew")
	if fallbackErr == nil {
		return profile, fallback, nil
	}
	return profile, nil, err
}

func (c HTTPClient) fetchLeaveBalancesFromPath(ctx context.Context, client *http.Client, cred Credential, session sessionInfo, path string) ([]LeaveBalance, error) {
	values := url.Values{}
	if session.PersonID != "" {
		values.Set("personId", session.PersonID)
	}
	reqURL := c.BaseURL + path
	if encoded := values.Encode(); encoded != "" {
		reqURL += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	setH5Headers(req, cred.CompanyCode, "/Leave/GetEmployeeLeaveBalanceNew", session.SessionID, "")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-HK,zh-Hant;q=0.9,en-HK;q=0.8,en;q=0.7")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Gaia/4.1.0")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Gaia leave balance API returned %s: %s", res.Status, trimBody(body))
	}
	return parseLeaveBalances(body)
}

func (c HTTPClient) fetchScheduleFromPath(ctx context.Context, client *http.Client, cred Credential, session sessionInfo, cloud cloudConfig, path, upstreamURL string, values url.Values) ([]ScheduleEntry, error) {
	reqURL := c.BaseURL + path + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	setH5Headers(req, cred.CompanyCode, upstreamURL, session.SessionID, "")
	if strings.Contains(path, "/h5-laboraccount-attendance/") {
		setLaborAccountHeaders(req, cred.CompanyCode)
	}
	if cloud.ServerAttendanceJavaAddressEnabled {
		if cloud.JavaAppPunchServerAddress != "" {
			encrypted, err := encryptDESECBPKCS7(cloud.JavaAppPunchServerAddress, session.EncryptionKey)
			if err != nil {
				return nil, err
			}
			req.Header.Set("javaAppPunchServerAddress", encrypted)
		}
		if cloud.ServerAttendanceJavaAddress != "" {
			encrypted, err := encryptDESECBPKCS7(cloud.ServerAttendanceJavaAddress, session.EncryptionKey)
			if err != nil {
				return nil, err
			}
			req.Header.Set("serverAttendanceJavaAdderss", encrypted)
		}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Gaia schedule API returned %s: %s", res.Status, trimBody(body))
	}
	return parseSchedule(body)
}

func (c HTTPClient) fetchCloudConfig(ctx context.Context, client *http.Client, cred Credential, key string) (cloudConfig, error) {
	values := url.Values{}
	values.Set("code", cred.CompanyCode)
	values.Set("language", "ZH-HK")
	values.Set("appSystem", "android")
	values.Set("forH5", "true")
	values.Set("currentAppVersion", "4.7.9")
	reqURL := c.BaseURL + "/h5-attendance/api/mobileCloud/v1/showNewConfigEncrypt?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return cloudConfig{}, err
	}
	setH5Headers(req, cred.CompanyCode, "/v1/showNewConfigEncrypt", "", "login")
	res, err := client.Do(req)
	if err != nil {
		return cloudConfig{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return cloudConfig{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return cloudConfig{}, fmt.Errorf("Gaia cloud config API returned %s: %s", res.Status, trimBody(body))
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return cloudConfig{}, err
	}
	if result, ok := parsed["result"].(bool); ok && !result {
		return cloudConfig{}, fmt.Errorf("Gaia cloud config failed: %s", trimBody(body))
	}
	encrypted := firstString(parsed, "data", "resultData", "value")
	if encrypted == "" {
		return cloudConfig{}, fmt.Errorf("Gaia cloud config response missing encrypted data")
	}
	decrypted, err := decryptDESECBPKCS7(encrypted, key)
	if err != nil {
		return cloudConfig{}, err
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(decrypted), &config); err != nil {
		return cloudConfig{}, fmt.Errorf("decode Gaia cloud config: %w", err)
	}
	return cloudConfig{
		APIAddress:                         stringPath(config, "apiAddress"),
		ShiftAddress:                       stringPath(config, "verNine", "newShiftUrl"),
		WFMFlag:                            stringPath(config, "verEight", "businessSystemVersion"),
		JavaAppPunchServerAddress:          stringPath(config, "originConfig", "javaAppPunchServerAddress", "configValue"),
		ServerAttendanceJavaAddress:        stringPath(config, "originConfig", "serverAttendanceJavaAdderss", "configValue"),
		ServerAttendanceJavaAddressEnabled: boolPath(config, "originConfig", "AttendanceJavaAdderss", "configEnable"),
	}, nil
}

func (c HTTPClient) fetchScheduleParameters(ctx context.Context, client *http.Client, cred Credential, session sessionInfo) (scheduleParameters, error) {
	values := url.Values{}
	values.Set("parameterCode", "SS1034,SS1037,SS1082,SS1103,SS1142,SS1143,SS1154")
	paths := []string{
		"/h5-attendance/api/wfm4integration/api/Schedule/GetScheduleParameter",
		"/h5-attendance/api/wfm4integration/Schedule/GetScheduleParameter",
		"/h5-attendance/api/netAttendance/api/Schedule/GetScheduleParameter",
		"/h5-attendance/api/netAttendance/Schedule/GetScheduleParameter",
		"/h5-attendance/api/javaAttendance/api/Schedule/GetScheduleParameter",
		"/h5-attendance/api/javaAttendance/Schedule/GetScheduleParameter",
		"/h5-attendance/api/appPunch/api/Schedule/GetScheduleParameter",
		"/h5-attendance/api/appPunch/Schedule/GetScheduleParameter",
		"/h5-attendance/api/BasicSchedule/api/Schedule/GetScheduleParameter",
		"/h5-attendance/api/BasicSchedule/Schedule/GetScheduleParameter",
		"/h5-attendance/api/interfaceCloud/api/Schedule/GetScheduleParameter",
	}
	var lastErr error
	for _, path := range paths {
		params, retry, err := c.fetchScheduleParametersFromPath(ctx, client, cred, session, path, values)
		if err == nil {
			return params, nil
		}
		lastErr = err
		if !retry {
			break
		}
	}
	return scheduleParameters{}, lastErr
}

func (c HTTPClient) fetchScheduleParametersFromPath(ctx context.Context, client *http.Client, cred Credential, session sessionInfo, path string, values url.Values) (scheduleParameters, bool, error) {
	reqURL := c.BaseURL + path + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return scheduleParameters{}, false, err
	}
	setH5Headers(req, cred.CompanyCode, "/api/Schedule/GetScheduleParameter", session.SessionID, "")
	res, err := client.Do(req)
	if err != nil {
		return scheduleParameters{}, false, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return scheduleParameters{}, false, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return scheduleParameters{}, res.StatusCode == http.StatusNotFound, fmt.Errorf("Gaia schedule parameter API returned %s: %s", res.Status, trimBody(body))
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return scheduleParameters{}, false, err
	}
	params := scheduleParameters{}
	for _, item := range findScheduleArray(decoded) {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		switch firstString(obj, "parameterCode", "code") {
		case "SS1037":
			params.ParamValue = firstString(obj, "parameterValue", "value")
		case "SS1103":
			params.ShowMealConfig = firstString(obj, "parameterValue", "value")
		case "SS1082":
			params.ShowJobParamValue = firstString(obj, "parameterValue", "value")
		}
	}
	return params, false, nil
}

func (c HTTPClient) validateH5User(ctx context.Context, client *http.Client, cred Credential) (sessionInfo, error) {
	key, err := c.fetchEncryptionKey(ctx, client)
	if err != nil {
		return sessionInfo{}, err
	}
	tokenValue, err := encryptDESECBPKCS7(cred.EmployeeAccount, key)
	if err != nil {
		return sessionInfo{}, err
	}
	payload := map[string]any{
		"isGaiaSso":  true,
		"tokenkey":   "PSNACCOUNT|EMPLOYEEID",
		"tokenValue": tokenValue,
		"encodeType": "1",
		"lan":        "ZH-HK",
		"tenant":     cred.CompanyCode,
	}
	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/h5-attendance/api/appPunch/H5Standard/ValidateUserInfoByConfigure", bytes.NewReader(data))
	if err != nil {
		return sessionInfo{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	setH5Headers(req, cred.CompanyCode, "/H5Standard/ValidateUserInfoByConfigure", "", "login")
	res, err := client.Do(req)
	if err != nil {
		return sessionInfo{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return sessionInfo{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return sessionInfo{}, fmt.Errorf("Gaia login returned %s: %s", res.Status, trimBody(body))
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return sessionInfo{}, err
	}
	if result, ok := parsed["result"].(bool); ok && !result {
		return sessionInfo{}, fmt.Errorf("Gaia login failed: %s", trimBody(body))
	}
	dataObj, _ := parsed["data"].(map[string]any)
	return sessionInfo{
		SessionID:     firstStringIn(parsed, dataObj, "SessionId", "sessionId"),
		PersonID:      firstStringIn(parsed, dataObj, "PersonID", "personId"),
		UserID:        firstStringIn(parsed, dataObj, "UserID", "userId"),
		EmpID:         firstStringIn(parsed, dataObj, "EmpID", "empId"),
		EmployeeName:  firstStringIn(parsed, dataObj, "EmpName", "empName", "EmployeeName", "employeeName", "Name", "name", "DisplayName", "displayName"),
		EncryptionKey: key,
	}, nil
}

func (c HTTPClient) fetchEncryptionKey(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/h5-attendance/api/encryption/code", nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("Gaia encryption key API returned %s: %s", res.Status, trimBody(body))
	}
	type encryptionCode struct {
		Fragments []string `json:"fragments"`
		Sequence  string   `json:"sequence"`
	}
	var parsed struct {
		Fragments []string       `json:"fragments"`
		Sequence  string         `json:"sequence"`
		Data      encryptionCode `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	code := encryptionCode{Fragments: parsed.Fragments, Sequence: parsed.Sequence}
	if code.Sequence == "" && parsed.Data.Sequence != "" {
		code = parsed.Data
	}
	sequenceBytes, err := base64.StdEncoding.DecodeString(code.Sequence)
	if err != nil {
		return "", fmt.Errorf("decode Gaia encryption sequence: %w", err)
	}
	var key string
	for _, part := range bytes.Split(sequenceBytes, []byte(",")) {
		var index int
		if _, err := fmt.Sscanf(string(part), "%d", &index); err != nil {
			return "", fmt.Errorf("parse Gaia encryption sequence: %w", err)
		}
		if index < 0 || index >= len(code.Fragments) {
			return "", fmt.Errorf("Gaia encryption sequence index %d out of range", index)
		}
		fragment, err := base64.StdEncoding.DecodeString(code.Fragments[index])
		if err != nil {
			return "", fmt.Errorf("decode Gaia encryption fragment: %w", err)
		}
		key += string(fragment)
	}
	if len(key) != 8 {
		return "", fmt.Errorf("Gaia encryption key length = %d", len(key))
	}
	return key, nil
}

func encryptDESECBPKCS7(plain, key string) (string, error) {
	block, err := des.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	data := []byte(plain)
	padding := blockSize - len(data)%blockSize
	for i := 0; i < padding; i++ {
		data = append(data, byte(padding))
	}
	encrypted := make([]byte, len(data))
	for start := 0; start < len(data); start += blockSize {
		block.Encrypt(encrypted[start:start+blockSize], data[start:start+blockSize])
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptDESECBPKCS7(cipherText, key string) (string, error) {
	block, err := des.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	if len(decoded) == 0 || len(decoded)%blockSize != 0 {
		return "", fmt.Errorf("Gaia encrypted payload length = %d", len(decoded))
	}
	decrypted := make([]byte, len(decoded))
	for start := 0; start < len(decoded); start += blockSize {
		block.Decrypt(decrypted[start:start+blockSize], decoded[start:start+blockSize])
	}
	padding := int(decrypted[len(decrypted)-1])
	if padding == 0 || padding > blockSize || padding > len(decrypted) {
		return "", fmt.Errorf("Gaia encrypted payload has invalid padding")
	}
	for _, value := range decrypted[len(decrypted)-padding:] {
		if int(value) != padding {
			return "", fmt.Errorf("Gaia encrypted payload has invalid padding")
		}
	}
	return string(decrypted[:len(decrypted)-padding]), nil
}

func setH5Headers(req *http.Request, companyCode, upstreamURL, sessionID, scene string) {
	req.Header.Set("CompanyCode", companyCode)
	req.Header.Set("Url", upstreamURL)
	req.Header.Set("Gaia_language", "ZH-HK")
	req.Header.Set("source", "mobile")
	req.Header.Set("tenant", companyCode)
	req.Header.Set("gaiaLanguage", "ZH-HK")
	req.Header.Set("h5Language", "ZH-HK")
	req.Header.Set("X-SCENE", scene)
	req.Header.Set("Authorization", "Bearer "+sessionID)
}

func setLaborAccountHeaders(req *http.Request, companyCode string) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-HK,zh-Hant;q=0.9,en-HK;q=0.8,en;q=0.7")
	req.Header.Set("CompanyCode", strings.ToLower(companyCode))
	req.Header.Set("Origin", "https://gateway.gaiacloud.com")
	req.Header.Set("Referer", "https://gateway.gaiacloud.com/h5-laboraccount-attendance")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Gaia/4.1.0")
	req.Header.Set("source", "mobile")
	req.Header.Set("tenant", strings.ToLower(companyCode))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseSchedule(body []byte) ([]ScheduleEntry, error) {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	items := findScheduleArray(decoded)
	if items == nil {
		return []ScheduleEntry{}, nil
	}
	out := make([]ScheduleEntry, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if entry, ok := parseScheduleObject(obj); ok {
			out = append(out, entry)
		}
	}
	return out, nil
}

func parseLeaveBalances(body []byte) ([]LeaveBalance, error) {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	items := findScheduleArray(decoded)
	if items == nil {
		return []LeaveBalance{}, nil
	}
	out := make([]LeaveBalance, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		balance := LeaveBalance{
			CreditID: firstString(obj, "creaditId", "creditId", "leaveCode", "code"),
			Name:     firstString(obj, "creaditName", "creditName", "leaveName", "name"),
			Unit:     leaveBalanceUnit(firstString(obj, "creaditUnit", "creditUnit", "unit")),
		}
		if value, ok := firstFloat(obj, "creaditUsed", "creditUsed", "used", "usedAmount"); ok {
			balance.Used = &value
		}
		if value, ok := firstFloat(obj, "creaditTotal", "creditTotal", "total", "totalAmount"); ok {
			balance.Total = &value
		}
		if value, ok := firstFloat(obj, "creaditRemain", "creditRemain", "remain", "remaining", "remainAmount"); ok {
			balance.Remaining = &value
		}
		raw, _ := json.Marshal(obj)
		balance.RawJSON = string(raw)
		if balance.Name == "" && balance.CreditID == "" {
			continue
		}
		out = append(out, balance)
	}
	return out, nil
}

func leaveBalanceUnit(value string) string {
	switch strings.TrimSpace(value) {
	case "0":
		return "天"
	case "1":
		return "小時"
	default:
		return value
	}
}

func parseScheduleObject(obj map[string]any) (ScheduleEntry, bool) {
	segments := scheduleSegments(obj)
	shiftDate := firstString(obj, "shiftDate", "date", "scheduleDate")
	if shiftDate == "" {
		for _, segment := range segments {
			if shiftDate = firstString(segment, "shiftDate", "date", "scheduleDate"); shiftDate != "" {
				break
			}
		}
	}
	date, err := time.Parse("2006-01-02", shiftDate)
	if err != nil {
		return ScheduleEntry{}, false
	}
	raw, _ := json.Marshal(obj)
	entry := ScheduleEntry{ShiftDate: date, RawJSON: string(raw)}

	names := make([]string, 0, len(segments))
	var totalHours float64
	var hasHours bool
	for _, segment := range segments {
		if v := firstString(segment, "shiftName", "className", "name", "timeClassName", "shortName"); v != "" && !containsString(names, v) {
			names = append(names, v)
		}
		if entry.ClassCode == nil {
			if v := firstString(segment, "classCode", "timeClassCode"); v != "" {
				entry.ClassCode = &v
			}
		}
		if v, ok := nonNegativeFloat(segment, "hours", "workHours", "scheduleHour"); ok {
			totalHours += v
			hasHours = true
		}
		if start := parseDateTime(date, firstString(segment, "startTime", "startTime1", "beginTime", "shiftTimeFrom")); start != nil {
			if entry.StartTime == nil || start.Before(*entry.StartTime) {
				entry.StartTime = start
			}
		}
		if end := parseDateTime(date, firstString(segment, "endTime", "endTime1", "finishTime", "shiftTimeTo")); end != nil {
			if entry.EndTime == nil || end.After(*entry.EndTime) {
				entry.EndTime = end
			}
		}
	}
	if len(names) > 0 {
		name := strings.Join(names, " / ")
		entry.ShiftName = &name
	}
	if hasHours {
		entry.Hours = &totalHours
	} else if entry.StartTime != nil && entry.EndTime != nil {
		hours := entry.EndTime.Sub(*entry.StartTime).Hours()
		if hours < 0 {
			hours += 24
		}
		entry.Hours = &hours
	}
	if isOffDay(entry) {
		entry.StartTime = nil
		entry.EndTime = nil
	}
	if entry.ShiftName == nil && entry.StartTime == nil && entry.EndTime == nil && entry.Hours == nil && entry.ClassCode == nil {
		if hours, ok := firstFloat(obj, "hours", "workHours"); ok && hours < 0 {
			classCode := "no_schedule"
			entry.ClassCode = &classCode
			return entry, true
		}
		return ScheduleEntry{}, false
	}
	return entry, true
}

func isOffDay(entry ScheduleEntry) bool {
	if entry.Hours == nil || *entry.Hours != 0 {
		return false
	}
	for _, value := range []*string{entry.ShiftName, entry.ClassCode} {
		if value != nil && strings.EqualFold(strings.TrimSpace(*value), "OFF") {
			return true
		}
	}
	return false
}

func scheduleSegments(obj map[string]any) []map[string]any {
	segments := []map[string]any{obj}
	for _, key := range []string{"list", "taskList", "schedule", "scheduleList", "shiftList", "details"} {
		for _, item := range arrayValues(obj[key]) {
			if segment, ok := item.(map[string]any); ok {
				segments = append(segments, segment)
			}
		}
	}
	return segments
}

func arrayValues(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case map[string]any:
		return findScheduleArray(x)
	default:
		return nil
	}
}

func findScheduleArray(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case map[string]any:
		for _, key := range []string{"data", "result", "list", "rows", "scheduleList"} {
			if found := findScheduleArray(x[key]); found != nil {
				return found
			}
		}
	}
	return nil
}

func firstString(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := obj[key].(string); ok {
			return v
		}
	}
	return ""
}

func firstStringIn(primary, secondary map[string]any, keys ...string) string {
	if value := firstString(primary, keys...); value != "" {
		return value
	}
	return firstString(secondary, keys...)
}

func firstFloat(obj map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		switch v := obj[key].(type) {
		case float64:
			return v, true
		case int:
			return float64(v), true
		}
	}
	return 0, false
}

func nonNegativeFloat(obj map[string]any, keys ...string) (float64, bool) {
	value, ok := firstFloat(obj, keys...)
	if !ok || value < 0 {
		return 0, false
	}
	return value, true
}

func cloneValues(values url.Values) url.Values {
	cloned := url.Values{}
	for key, items := range values {
		for _, item := range items {
			cloned.Add(key, item)
		}
	}
	return cloned
}

func hasDatedShiftTimes(entries []ScheduleEntry) bool {
	for _, entry := range entries {
		if entry.StartTime != nil || entry.EndTime != nil || entry.Hours != nil {
			return true
		}
		if entry.ShiftName != nil && (entry.ClassCode == nil || *entry.ClassCode != "no_schedule") {
			return true
		}
	}
	return false
}

func stringPath(obj map[string]any, keys ...string) string {
	var current any = obj
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = next[key]
	}
	value, _ := current.(string)
	return value
}

func boolPath(obj map[string]any, keys ...string) bool {
	var current any = obj
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current = next[key]
	}
	value, _ := current.(bool)
	return value
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func parseDateTime(date time.Time, value string) *time.Time {
	if value == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "15:04:05", "15:04"} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			if layout == "15:04:05" || layout == "15:04" {
				parsed = time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.Local)
			}
			return &parsed
		}
	}
	return nil
}

func trimBody(body []byte) string {
	if len(body) > 500 {
		body = body[:500]
	}
	return string(body)
}
