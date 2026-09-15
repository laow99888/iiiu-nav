//go:build !linux && !windows && !darwin && !freebsd && !openbsd && !dragonfly

package restore

import "math"

func availableDiskSpace(string) (uint64, error) { return math.MaxUint64, nil }
