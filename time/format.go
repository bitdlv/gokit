package timex

import (
	"time"
)

func ToLocalString(t time.Time) string {
	return t.Local().Format(layout)
}
