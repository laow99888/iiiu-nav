//go:build !production

package webui

import "io/fs"

func Assets() fs.FS {
	return nil
}
