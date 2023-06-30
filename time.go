package rest

import (
	"context"
	"time"

	"github.com/KarpelesLab/pjson"
)

type Time struct {
	time.Time
}

type timestampInternal struct {
	Unix   int64  `json:"unix"`                    // 1597242491
	Usec   int64  `json:"us"`                      // 747497
	TZ     string `json:"tz,omitempty"`            // Asia/Tokyo
	ISO    string `json:"iso,omitempty"`           // "2020-08-12 23:28:11.747497"
	Full   int64  `json:"full,omitempty,string"`   // "1597242491747497"
	UnixMS int64  `json:"unixms,omitempty,string"` // "1597242491747"
}

func (u *Time) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	var sd timestampInternal
	err := pjson.Unmarshal(data, &sd)
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
	sd.ISO = u.UTC().Format("2006-01-02 15:04:05")
	sd.Full = u.UnixMicro()
	sd.UnixMS = u.UnixMilli()

	return pjson.Marshal(sd)
}

func (u *Time) UnmarshalContextJSON(ctx context.Context, data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	var sd timestampInternal
	err := pjson.UnmarshalContext(ctx, data, &sd)
	if err != nil {
		return err
	}
	u.Time = time.Unix(sd.Unix, sd.Usec*1000) // *1000 means µs → ns
	return nil
}

func (u Time) MarshalContextJSON(ctx context.Context) ([]byte, error) {
	var sd timestampInternal
	sd.Unix = u.Unix()
	sd.Usec = int64(u.Nanosecond() / 1000)
	sd.TZ = u.Location().String()
	sd.ISO = u.UTC().Format("2006-01-02 15:04:05")
	sd.Full = u.UnixMicro()
	sd.UnixMS = u.UnixMilli()

	return pjson.MarshalContext(ctx, sd)
}
