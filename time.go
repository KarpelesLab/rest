package rest

import (
	"strconv"
	"time"
)

type Time struct {
	Full string `json:"full"`
	TZ   string `json:"tz"`
	// ignored fields: unix, us, iso
}

func (t *Time) Time() time.Time {
	// need to generate time object
	v, err := strconv.ParseInt(t.Full, 10, 64)
	if err != nil {
		// failed
		return time.Time{}
	}

	unix := v / 1000000
	unix_us := v - (unix * 1000000)

	return time.Unix(unix, unix_us*1000)
}
