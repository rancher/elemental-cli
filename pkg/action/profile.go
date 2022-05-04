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

package action

import (
	"errors"
	"fmt"

	"github.com/rancher-sandbox/elemental/pkg/airgap"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
)

// Calls in luet with a profile list and creates multiple rootfs to convert in profiles
func CreateProfile(cfg *v1.BuildConfig, profileName, dst string) (err error) {
	if cfg.AssetSources[cfg.Arch] == nil {
		msg := fmt.Sprintf("no values in the config for arch %s", cfg.Arch)
		cfg.Logger.Error(msg)
		return errors.New(msg)
	}

	if len(cfg.AssetSources[cfg.Arch].Packages) == 0 {
		msg := fmt.Sprintf("no packages in the config for arch %s", cfg.Arch)
		cfg.Logger.Error(msg)
		return errors.New(msg)
	}

	if len(cfg.Config.Repos) == 0 {
		msg := fmt.Sprintf("no repositories configured for arch %s", cfg.Arch)
		cfg.Logger.Error(msg)
		return errors.New(msg)
	}

	cleanup := utils.NewCleanStack()
	defer func() { err = cleanup.Cleanup(err) }()

	// baseDir is where we are going install all packages
	baseDir, err := utils.TempDir(cfg.Fs, "", "elemental-build-disk-files")
	if err != nil {
		return err
	}
	cleanup.Push(func() error { return cfg.Fs.RemoveAll(baseDir) })

	// Extract required packages to basedir
	for _, pkg := range cfg.AssetSources[cfg.Arch].Packages {
		err = applySources(cfg.Config, baseDir, pkg.Name)
		if err != nil {
			cfg.Logger.Error(err)
		}
	}

	if err != nil {
		return
	}

	err = airgap.CreateProfileArchive(baseDir, dst, profileName)

	return
}
