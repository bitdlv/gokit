package timex

import (
	"time"
)

const layout = "2006-01-02 15:04:05"

func ParseEast8(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.ParseInLocation(layout, s, time.FixedZone("UTC+8", 8*60*60))
}

func ParseLocal(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(layout, s)
}

func ParseEast8UnixOrZero(format string, s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.ParseInLocation(format, s, time.FixedZone("UTC+8", 8*60*60))
	if err != nil {
		return 0
	}
	return t.Unix()
}
