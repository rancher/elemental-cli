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

package utils

import (
	"fmt"
	"path/filepath"

	cnst "github.com/rancher/elemental-cli/pkg/constants"
	v1 "github.com/rancher/elemental-cli/pkg/types/v1"
)

// Grub is the struct that will allow us to install grub to the target device
type Grub struct {
	config *v1.Config
}

func NewGrub(config *v1.Config) *Grub {
	g := &Grub{
		config: config,
	}

	return g
}

// Install installs grub into the device and copies the config file
func (g Grub) Install(target, rootDir, bootDir, grubConf string, efi bool) (err error) { // nolint:gocyclo
	var grubargs []string

	g.config.Logger.Info("Installing GRUB..")

	if efi {
		g.config.Logger.Infof("Installing grub efi for arch %s", g.config.Arch)
		grubargs = append(
			grubargs,
			fmt.Sprintf("--target=%s-efi", g.config.Arch),
			fmt.Sprintf("--efi-directory=%s", cnst.EfiDir),
		)
	} else {
		if g.config.Arch == "x86_64" {
			grubargs = append(grubargs, "--target=i386-pc")
		}
	}

	grubargs = append(
		grubargs,
		fmt.Sprintf("--root-directory=%s", rootDir),
		fmt.Sprintf("--boot-directory=%s", bootDir),
		"--removable", target,
	)

	g.config.Logger.Debugf("Running grub with the following args: %s", grubargs)
	out, err := g.config.Runner.Run("grub2-install", grubargs...)
	if err != nil {
		g.config.Logger.Errorf(string(out))
		return err
	}

	err = g.CopyConfigFile(rootDir, bootDir, grubConf)
	if err != nil {
		return err
	}

	g.config.Logger.Infof("Grub install to device %s complete", target)
	return nil
}

// CopyConfigFile copies the grub configuration file from the image root to the
// partition boot directory
func (g Grub) CopyConfigFile(rootDir, bootDir, grubConf string) error {
	var grubDir string

	// Select the proper dir for grub
	if ok, _ := IsDir(g.config.Fs, filepath.Join(bootDir, "grub")); ok {
		grubDir = filepath.Join(bootDir, "grub")
	} else if ok, _ := IsDir(g.config.Fs, filepath.Join(bootDir, "grub2")); ok {
		grubDir = filepath.Join(bootDir, "grub2")
	} else {
		return fmt.Errorf("no grub directory found in %s", bootDir)
	}

	g.config.Logger.Infof("Found grub config dir %s", grubDir)

	grubCfg, err := g.config.Fs.ReadFile(filepath.Join(rootDir, grubConf))
	if err != nil {
		g.config.Logger.Errorf("Failed reading grub config file: %s", filepath.Join(rootDir, grubConf))
		return err
	}

	grubConfTarget, err := g.config.Fs.Create(fmt.Sprintf("%s/grub.cfg", grubDir))
	if err != nil {
		return err
	}

	defer grubConfTarget.Close()

	g.config.Logger.Infof("Copying grub contents from %s to %s", grubConf, fmt.Sprintf("%s/grub.cfg", grubDir))
	_, err = grubConfTarget.WriteString(string(grubCfg))
	if err != nil {
		return err
	}
	return nil
}

// Sets the given key value pairs into as grub variables into the given file
func (g Grub) SetPersistentVariables(grubEnvFile string, vars map[string]string) error {
	for key, value := range vars {
		g.config.Logger.Debugf("Running grub2-editenv with params: %s set %s=%s", grubEnvFile, key, value)
		out, err := g.config.Runner.Run("grub2-editenv", grubEnvFile, "set", fmt.Sprintf("%s=%s", key, value))
		if err != nil {
			g.config.Logger.Errorf(fmt.Sprintf("Failed setting grub variables: %s", out))
			return err
		}
	}
	return nil
}

// SetDefaultEntry sets the default_meny_entry value in RunConfig.GrubOEMEnv file
func (g Grub) SetDefaultEntry(rootDir string, bootDir string, defaultEntry string) error {
	if defaultEntry == "" {
		osRelease, err := LoadEnvFile(g.config.Fs, filepath.Join(rootDir, "etc", "os-release"))
		if err != nil {
			g.config.Logger.Warnf("Could not load os-release file: %v", err)
			return nil
		}
		defaultEntry = osRelease["GRUB_ENTRY_NAME"]
		if defaultEntry == "" {
			g.config.Logger.Debug("unset grub default entry")
			return nil
		}
	}

	return g.SetPersistentVariables(
		filepath.Join(bootDir, cnst.GrubOEMEnv),
		map[string]string{"default_menu_entry": defaultEntry},
	)
}
