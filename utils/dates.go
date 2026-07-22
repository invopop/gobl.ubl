package utils

import (
	"time"

	"github.com/invopop/gobl/cal"
)

// FormatDate formats a GOBL date as an ISO 8601 date string (YYYY-MM-DD).
func FormatDate(date cal.Date) string {
	if date.IsZero() {
		return ""
	}
	t := date.Time()
	return t.Format("2006-01-02")
}

// ParseDate converts a date string to a cal.Date.
func ParseDate(date string) (cal.Date, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return cal.Date{}, err
	}

	return cal.MakeDate(t.Year(), t.Month(), t.Day()), nil
}
