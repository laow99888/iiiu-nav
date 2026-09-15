//go:build darwin

package restore

import "golang.org/x/sys/unix"

func availableDiskSpace(path string) (uint64, error) {
	var statistics unix.Statfs_t
	if err := unix.Statfs(path, &statistics); err != nil {
		return 0, err
	}
	return statistics.Bavail * uint64(statistics.Bsize), nil
}
