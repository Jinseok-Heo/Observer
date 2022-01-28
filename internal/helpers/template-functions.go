package helpers

import "time"

func addTemplateFunctions() {
	views.AddGlobal("humanDate", func(t time.Time) string {
		return HumanDate(t)
	})

	views.AddGlobal("dateFromLayout", func(t time.Time, l string) string {
		return FormatDateWithLayout(t, l)
	})

	views.AddGlobal("dateAfterYearOne", func(t time.Time) bool {
		return DateAfterY1(t)
	})
}

// HumanDate formats a time in yyyy-MM-dd format
func HumanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// FormatDateWithLayout formats a time with provided (go compliant) format string, and returns it as string
func FormatDateWithLayout(t time.Time, format string) string {
	return t.Format(format)
}

// DateAfterY1 is used to verify that a date is after the year 1 (since go hates nulls)
func DateAfterY1(t time.Time) bool {
	yearOne := time.Date(0001, 11, 17, 20, 34, 58, 651387237, time.UTC)
	return t.After(yearOne)
}
