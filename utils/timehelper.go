package utils

import (
	"time"
)

var DayMillisecs int64 = 86400000
var DayNanosecs int64 = 86400000000000

var LocTaipei, _ = time.LoadLocation("Asia/Taipei")
var TwTimezoneDiff, _ = time.ParseDuration("8h")

func MsAsTwTime(ms int64) time.Time {
	return time.Unix(0, ms*1000000).In(LocTaipei).Add(-TwTimezoneDiff)
}

func MsToTwTime(ms int64) time.Time {
	return time.Unix(0, ms*1000000).In(LocTaipei)
}

func NsAsTwTime(ns int64) time.Time {
	return time.Unix(0, ns).In(LocTaipei).Add(-TwTimezoneDiff)
}

func NsToTwTime(ns int64) time.Time {
	return time.Unix(0, ns).In(LocTaipei)
}

func AddDuration(t time.Time, s string) (*time.Time, error) {
	d, err := time.ParseDuration(s)
	// fmt.Println("d :", d)
	if err != nil {
		return nil, err
	}
	t2 := t.Add(d)
	return &t2, nil
}

func ParseMsDuration(s string) (int64, error) {
	d, err := time.ParseDuration(s)
	// fmt.Println("d :", d)
	if err != nil {
		return 0, err
	}
	return d.Nanoseconds() / 1000000, nil
}

func MsAddDuration(msTime int64, s string) (int64, error) {
	d, err := ParseMsDuration(s)
	// fmt.Println("d :", d)
	if err != nil {
		return 0, err
	}
	return msTime + d, nil
}

func ToMillisecond(t time.Time) int64 {
	return t.UnixNano() / 1000000
}

func ToTwTime(t time.Time) time.Time {
	return t.In(LocTaipei)
}

func AsTwTime(t time.Time) time.Time {
	return t.In(LocTaipei).Add(-TwTimezoneDiff)
}

func TwNow() time.Time {
	return ToTwTime(time.Now())
}

func TwDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, LocTaipei)
}

func TwStartOfDay(t time.Time) time.Time {
	year, month, day := ToTwTime(t).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, LocTaipei)
}

func TwStartOfMonth(t time.Time) time.Time {
	year, month, _ := ToTwTime(t).Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, LocTaipei)
}

func MsTwStartOfDay(ms int64) time.Time {
	year, month, day := MsToTwTime(ms).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, LocTaipei)
}

func MsTwStartOfMonth(ms int64) time.Time {
	year, month, _ := MsToTwTime(ms).Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, LocTaipei)
}

func NsTwStartOfDay(ns int64) time.Time {
	year, month, day := NsToTwTime(ns).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, LocTaipei)
}

func NsTwStartOfMonth(ns int64) time.Time {
	year, month, _ := NsToTwTime(ns).Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, LocTaipei)
}
