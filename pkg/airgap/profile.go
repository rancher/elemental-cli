package airgap

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/mholt/archiver"
	"github.com/otiai10/copy"
)

// To call it like:
// src=some/path
// dst=some/path/out
// profileName=baz
func CreateProfileArchive(changeDir, dst, profileName string) error {
	src := "."
	// takes a dir and creates a tar.xz of it. it places it under dst
	dst = filepath.Join(dst, fmt.Sprintf("%s.tar.xz", profileName))
	var cwd string
	if changeDir != "" {
		if !path.IsAbs(dst) {
			var err error
			cwd, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		os.Chdir(changeDir)
	}

	err := archiver.Archive([]string{src}, dst)
	if err != nil {
		return err
	}

	if changeDir != "" {
		os.Chdir(cwd)
	}

	// This is to preserve '.' inside the resulting archive
	if !path.IsAbs(dst) && changeDir != "" {
		output := path.Join(changeDir, path.Base(dst))
		defer os.RemoveAll(output)
		return copy.Copy(output, path.Join(cwd, dst))
	}
	return nil
}
