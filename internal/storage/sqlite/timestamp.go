package sqlite

import "time"

func timestamp(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func timeFromTimestamp(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}
