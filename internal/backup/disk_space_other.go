//go:build !linux && !windows

package backup

import "math"

func availableDiskSpace(string) (uint64, error) { return math.MaxUint64, nil }
