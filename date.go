package ubl

import (
	"strings"
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

// parseDate converts a date string to a cal.Date. UBL xsd:date permits an
// optional timezone (e.g. "...Z", "...+01:00"); NemHandel traffic is unzoned,
// so the zoned layout is only a fallback for conformant-but-zoned senders.
func parseDate(date string) (cal.Date, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		t, err = time.Parse("2006-01-02Z07:00", date)
		if err != nil {
			return cal.Date{}, err
		}
	}

	return cal.MakeDate(t.Year(), t.Month(), t.Day()), nil
}

// trimTimeZone strips an optional XSD timezone designator ("Z" or "±HH:MM")
// from a time-of-day value. civil.ParseTime rejects a zone, but UBL xsd:time
// permits one; a time value otherwise contains only digits, ':' and '.', so
// the first zone marker is unambiguous.
func trimTimeZone(t string) string {
	if i := strings.IndexAny(t, "Zz+-"); i >= 0 {
		return t[:i]
	}
	return t
}
