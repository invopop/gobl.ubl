package gtou

import (
	"github.com/invopop/gobl/cal"
)

func formatDate(date cal.Date) string {
	if date.IsZero() {
		return ""
	}
	t := date.Time()
	return t.Format("2006-01-02")
}

func makePeriod(period *cal.Period) Period {
	return Period{
		StartDate: formatDate(period.Start),
		EndDate:   formatDate(period.End),
	}
}
