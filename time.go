package rest

import (
	"encoding/json"
	"time"
)

type Time struct {
	time.Time
}

type timestampInternal struct {
	Unix int64  `json:"unix"`         // 1597242491
	Usec int64  `json:"us"`           // 747497
	TZ   string `json:"tz,omitempty"` // Asia/Tokyo
	// iso="2020-08-12 23:28:11.747497"
	// full="1597242491747497"
	// unixms="1597242491747"
}

func (u *Time) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	var sd timestampInternal
	err := json.Unmarshal(data, &sd)
	if err != nil {
		return err
	}
	u.Time = time.Unix(sd.Unix, sd.Usec*1000) // *1000 means µs → ns
	return nil
}

func (u Time) MarshalJSON() ([]byte, error) {
	var sd timestampInternal
	sd.Unix = u.Unix()
	sd.Usec = int64(u.Nanosecond() / 1000)
	sd.TZ = u.Location().String()

	return json.Marshal(sd)
}
