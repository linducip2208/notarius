// Package wib formats times for Asia/Jakarta display (UUJN operational).
// Database stays UTC; presentation converts to WIB.
package wib

import "time"

var loc *time.Location

func init() {
	l, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
		return
	}
	loc = l
}

// InWIB converts t to Asia/Jakarta.
func InWIB(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(loc)
}

// Format renders t in WIB as "02 Jan 2006 15:04 WIB".
func Format(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return InWIB(t).Format("02 Jan 2006 15:04 WIB")
}

// Date renders t in WIB as "02 Jan 2006".
func Date(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return InWIB(t).Format("02 Jan 2006")
}
