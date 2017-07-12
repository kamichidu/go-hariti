// +build !windows

package hariti

import (
	"fmt"
	"os"
	"path/filepath"
)

func mklink(oldname string, newname string) error {
	// create relative link
	relLink, err := filepath.Rel(filepath.Dir(newname), oldname)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(newname); err != nil {
		// there's no file, just create new one
		if err = os.Symlink(relLink, newname); err != nil {
			return err
		}
	} else if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		// there's a link, just check its state
		state, err := os.Readlink(newname)
		if err != nil {
			return err
		} else if state != relLink {
			return fmt.Errorf("%s should be point to %s, but %s", newname, relLink, state)
		}
	} else {
		// there's non-link file
		return fmt.Errorf("%s is already exists, ignored", newname)
	}
	return nil
}
