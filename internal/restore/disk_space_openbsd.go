//go:build openbsd

package restore

import "golang.org/x/sys/unix"

func availableDiskSpace(path string) (uint64, error) {
	var statistics unix.Statfs_t
	if err := unix.Statfs(path, &statistics); err != nil {
		return 0, err
	}
	if statistics.F_bavail <= 0 {
		return 0, nil
	}
	return uint64(statistics.F_bavail) * uint64(statistics.F_bsize), nil
}
