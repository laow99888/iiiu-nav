//go:build !linux && !windows

package restore

import "math"

func availableDiskSpace(string) (uint64, error) { return math.MaxUint64, nil }
