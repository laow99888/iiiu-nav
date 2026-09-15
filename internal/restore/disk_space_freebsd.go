//go:build freebsd

package restore

import "golang.org/x/sys/unix"

func availableDiskSpace(path string) (uint64, error) {
	var statistics unix.Statfs_t
	if err := unix.Statfs(path, &statistics); err != nil {
		return 0, err
	}
	if statistics.Bavail <= 0 {
		return 0, nil
	}
	return uint64(statistics.Bavail) * statistics.Bsize, nil
}
