package ubl

import (
	"regexp"
	"time"

	"cloud.google.com/go/civil"
	"github.com/invopop/gobl/cal"
)

// timeZonePattern matches the time zone an xs:time may end with: "Z" or an
// offset such as "+04:00", within the XML Schema range of -14:00 to +14:00.
// Anything else is left on the time, so parsing fails.
var timeZonePattern = regexp.MustCompile(`(Z|[+-]((0\d|1[0-3]):[0-5]\d|14:00))$`)

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

// parseTime parses an xs:time, the type of cbc:IssueTime, such as "15:55:00",
// "15:55:00.5", "15:55:00Z" or "15:55:00+04:00". GOBL's time has no time zone,
// so the zone is dropped and the local time kept, the same way the issue date
// is a local date.
func parseTime(value string) (*cal.Time, error) {
	ct, err := civil.ParseTime(timeZonePattern.ReplaceAllString(value, ""))
	if err != nil {
		return nil, err
	}
	return &cal.Time{Time: ct}, nil
}

// formatDatePtr formats an optional date, returning an empty string when not
// set. cal.Period start and end dates are optional, only one is required.
func formatDatePtr(date *cal.Date) string {
	if date == nil {
		return ""
	}
	return formatDate(*date)
}
