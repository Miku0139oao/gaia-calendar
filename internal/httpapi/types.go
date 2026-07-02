package httpapi

import "time"

type registerRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
	Locale          string `json:"locale,omitempty"`
}

type verifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type passwordResetRequest struct {
	Email  string `json:"email"`
	Locale string `json:"locale,omitempty"`
}

type passwordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type gaiaCredentialRequest struct {
	CompanyCode     string `json:"companyCode"`
	EmployeeAccount string `json:"employeeAccount"`
	Password        string `json:"password"`
}

type syncRunUpdateRequest struct {
	Marked bool `json:"marked"`
}

type userResponse struct {
	ID            int    `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	Role          string `json:"role"`
}

type scheduleResponse struct {
	ID        int                       `json:"id"`
	ShiftDate string                    `json:"shiftDate"`
	ShiftName *string                   `json:"shiftName,omitempty"`
	StartTime *time.Time                `json:"startTime,omitempty"`
	EndTime   *time.Time                `json:"endTime,omitempty"`
	Hours     *float64                  `json:"hours,omitempty"`
	ClassCode *string                   `json:"classCode,omitempty"`
	Segments  []scheduleSegmentResponse `json:"segments,omitempty"`
}

type scheduleSegmentResponse struct {
	Name      string   `json:"name,omitempty"`
	StartTime string   `json:"startTime,omitempty"`
	EndTime   string   `json:"endTime,omitempty"`
	Hours     *float64 `json:"hours,omitempty"`
	ClassCode string   `json:"classCode,omitempty"`
}
