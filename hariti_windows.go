package hariti

import (
	"os"
)

func mklink(oldname string, newname string) error {
	return os.Link(oldname, newname)
}
