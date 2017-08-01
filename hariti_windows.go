package hariti

import (
	"bytes"
	"errors"
	"os/exec"
)

func mklink(oldname string, newname string) error {
	stderr := new(bytes.Buffer)
	cmd := exec.Command("cmd.exe", "/C", "mklink", "/J", newname, oldname)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return errors.New(stderr.String() + err.Error())
	}
	return nil
}
