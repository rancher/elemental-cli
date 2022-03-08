/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"errors"
	"os"
	"os/exec"
)

// CheckRoot is a helper to return on PreRunE, so we can add it to commands that require root
func CheckRoot() error {
	if os.Geteuid() != 0 {
		return errors.New("this command requires root privileges")
	}
	return nil
}

// CheckDeps is used by cmd to check a list of deps for a command and fail if they don't exists
func CheckDeps(deps []string) error {
	for _, dep := range deps {
		_, err := exec.LookPath(dep)
		if err != nil {
			return err
		}
	}
	return nil
}
