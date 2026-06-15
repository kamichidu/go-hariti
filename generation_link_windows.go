//go:build windows
// +build windows

package hariti

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
)

func createGenerationLink(target, link string) error {
	var absTarget string
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(link), target)
	} else {
		absTarget = target
	}

	absTarget, err := filepath.Abs(absTarget)
	if err != nil {
		return err
	}

	stderr := new(bytes.Buffer)
	cmd := exec.Command("cmd.exe", "/C", "mklink", "/J", link, absTarget)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return errors.New(stderr.String() + err.Error())
	}
	return nil
}
