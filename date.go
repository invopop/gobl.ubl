package ubl

import (
	"time"

	"github.com/invopop/gobl/cal"
)

func formatDate(date cal.Date) string {
	if date.IsZero() {
		return ""
	}
	t := date.Time()
	return t.Format("2006-01-02")
}

// parseDate converts a date string to a cal.Date.
func parseDate(date string) (cal.Date, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return cal.Date{}, err
	}

	return cal.MakeDate(t.Year(), t.Month(), t.Day()), nil
}

// formatDatePtr formats an optional date, returning an empty string when not
// set. cal.Period start and end dates are optional, only one is required.
func formatDatePtr(date *cal.Date) string {
	if date == nil {
		return ""
	}
	return formatDate(*date)
}
