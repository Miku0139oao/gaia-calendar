package httpapi

import (
	"strings"
	"testing"
	"time"

	"gaia-calendar/ent"
	"gaia-calendar/internal/gaia"
)

func TestBuildICSIncludesMonthlyTotalEvent(t *testing.T) {
	shiftName := "B"
	classCode := "TA002"
	leaveName := "年假(HK-AL)"
	offName := "OFF"
	offCode := "OFF"
	noScheduleCode := "no_schedule"
	hours := 8.5
	leaveHours := 1.0
	zeroHours := 0.0
	shiftDate := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, 7, 3, 13, 30, 0, 0, time.UTC)
	end := time.Date(2026, 7, 3, 23, 0, 0, 0, time.UTC)
	leaveStart := time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)
	leaveEnd := time.Date(2026, 7, 4, 18, 30, 0, 0, time.UTC)
	used := 3.5
	total := 12.0
	remaining := 8.5

	ics := buildICS(1, calendarMetadata{
		EmployeeAccount: "E123456",
		EmployeeName:    "Test Employee",
		LeaveBalances: []gaia.LeaveBalance{{
			CreditID:  "HK-AL",
			Name:      "年假",
			Used:      &used,
			Total:     &total,
			Remaining: &remaining,
			Unit:      "天",
		}},
	}, []*ent.ScheduleEntry{
		{ShiftDate: shiftDate, ShiftName: &shiftName, ClassCode: &classCode, StartTime: &start, EndTime: &end, Hours: &hours},
		{ShiftDate: time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC), ShiftName: &leaveName, StartTime: &leaveStart, EndTime: &leaveEnd, Hours: &leaveHours, RawJSON: `{"isEvent":"Y"}`},
		{ShiftDate: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC), ShiftName: &offName, ClassCode: &offCode, Hours: &zeroHours},
		{ShiftDate: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC), ClassCode: &noScheduleCode},
	})

	for _, want := range []string{
		"UID:gaia-1-202607-total@gaia-calendar",
		"DTSTART:20260701T000000",
		"DTEND:20260701T000500",
		"SUMMARY:2026-07 預計總工時 8.5h",
		"UID:gaia-1-202607-leave-balance@gaia-calendar",
		"DTSTART:20260701T000500",
		"DTEND:20260701T001000",
		"SUMMARY:年假 已使用 3.5天 / 總數 12天 / 剩餘 8.5天",
		"SUMMARY:B",
		"DESCRIPTION:Employee: Test Employee (E123456)\\nClass: TA002\\nTime: 13:30-23:00\\nHours: 8.5h",
		"SUMMARY:年假(HK-AL)",
	} {
		if !strings.Contains(ics, want) {
			t.Fatalf("ICS missing %q:\n%s", want, ics)
		}
	}
	if strings.Contains(ics, "SUMMARY:OFF") || strings.Contains(ics, "no_schedule") {
		t.Fatalf("ICS should not expose OFF or no_schedule entries:\n%s", ics)
	}
	if strings.Contains(ics, "VALUE=DATE:20260701") {
		t.Fatalf("monthly summary events should not be all-day:\n%s", ics)
	}
	if strings.Contains(ics, "SUMMARY:B 13:30-23:00") {
		t.Fatalf("shift event summaries should not repeat visible time ranges:\n%s", ics)
	}
}
