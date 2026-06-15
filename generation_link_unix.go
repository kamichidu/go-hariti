//go:build !windows
// +build !windows

package hariti

import "os"

func createGenerationLink(target, link string) error {
	return os.Symlink(target, link)
}
