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
	"strings"

	"github.com/rancher/elemental-cli/pkg/constants"
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

// Install installs grub into the device, copy the config file and add any extra TTY to grub
func (g Grub) Install(target, rootDir, bootDir, grubConf, tty string, efi bool, stateLabel string, bootloaderName string) (err error) { // nolint:gocyclo
	var grubargs []string
	var grubdir, finalContent string
	// only install grub on non-efi systems
	if !efi {
		g.config.Logger.Info("Installing GRUB..")

		grubargs = append(
			grubargs,
			fmt.Sprintf("--root-directory=%s", rootDir),
			fmt.Sprintf("--boot-directory=%s", bootDir),
			"--target=i386-pc",
			target,
		)
		g.config.Logger.Debugf("Running grub with the following args: %s", grubargs)
		out, err := g.config.Runner.Run("grub2-install", grubargs...)
		if err != nil {
			g.config.Logger.Errorf(string(out))
			return err
		}
		g.config.Logger.Infof("Grub install to device %s complete", target)
	}

	// At this point the active mountpoint has all the data from the installation source, so we should be able to use
	// the grub.cfg bundled in there
	grubdir = filepath.Join(rootDir, grubConf)
	g.config.Logger.Infof("Using grub config dir %s", grubdir)

	grubCfg, err := g.config.Fs.ReadFile(grubdir)
	if err != nil {
		g.config.Logger.Errorf("Failed reading grub config file: %s", filepath.Join(rootDir, grubConf))
		return err
	}

	// Create Needed dir under state partition to store the grub.cfg and any needed modules
	err = MkdirAll(g.config.Fs, filepath.Join(bootDir, fmt.Sprintf("grub2/%s-efi", g.config.Arch)), cnst.DirPerm)
	if err != nil {
		return fmt.Errorf("error creating grub dir: %s", err)
	}

	grubConfTarget, err := g.config.Fs.Create(filepath.Join(bootDir, "grub2/grub.cfg"))
	if err != nil {
		return err
	}

	defer grubConfTarget.Close()

	if tty == "" {
		// Get current tty and remove /dev/ from its name
		out, err := g.config.Runner.Run("tty")
		tty = strings.TrimPrefix(strings.TrimSpace(string(out)), "/dev/")
		if err != nil {
			g.config.Logger.Warnf("failed to find current tty, leaving it unset")
			tty = ""
		}
	}

	ttyExists, _ := Exists(g.config.Fs, fmt.Sprintf("/dev/%s", tty))

	if ttyExists && tty != "" && tty != "console" && tty != constants.DefaultTty {
		// We need to add a tty to the grub file
		g.config.Logger.Infof("Adding extra tty (%s) to grub.cfg", tty)
		defConsole := fmt.Sprintf("console=%s", constants.DefaultTty)
		finalContent = strings.Replace(string(grubCfg), defConsole, fmt.Sprintf("%s console=%s", defConsole, tty), -1)
	} else {
		// We don't add anything, just read the file
		finalContent = string(grubCfg)
	}

	g.config.Logger.Infof("Copying grub contents from %s to %s", grubdir, filepath.Join(bootDir, "grub2/grub.cfg"))
	_, err = grubConfTarget.WriteString(finalContent)
	if err != nil {
		return err
	}

	if efi {
		// Copy required extra modules to boot dir under the state partition
		// otherwise if we insmod it will fail to find them
		// We no longer call grub-install here so the modules are not setup automatically in the state partition
		// as they were before. We now use the bundled grub.efi provided by the shim package
		for _, m := range []string{"loopback.mod", "squash4.mod"} {
			fileReadName := filepath.Join(rootDir, fmt.Sprintf("/usr/share/grub2/%s-efi/%s", g.config.Arch, m))
			fileWriteName := filepath.Join(bootDir, fmt.Sprintf("grub2/%s-efi/%s", g.config.Arch, m))
			g.config.Logger.Debugf("Copying %s to %s", fileReadName, fileWriteName)
			fileContent, err := g.config.Fs.ReadFile(fileReadName)
			if err != nil {
				return fmt.Errorf("error reading %s: %s", fileReadName, err)
			}
			err = g.config.Fs.WriteFile(fileWriteName, fileContent, cnst.FilePerm)
			if err != nil {
				return fmt.Errorf("error writing %s: %s", fileWriteName, err)
			}
		}

		if exists, _ := Exists(g.config.Fs, filepath.Join(rootDir, fmt.Sprintf("/usr/share/efi/%s/", g.config.Arch))); exists {
			g.config.Logger.Infof("Generating grub files for efi on %s", target)
			err = MkdirAll(g.config.Fs, filepath.Join(cnst.EfiDir, fmt.Sprintf("EFI/%s/", bootloaderName)), cnst.DirPerm)
			if err != nil {
				g.config.Logger.Errorf("Error creating dirs: %s", err)
				return err
			}

			// Copy needed files for efi boot
			for _, f := range []string{"shim.efi", "MokManager.efi", "grub.efi"} {
				fileReadName := filepath.Join(rootDir, fmt.Sprintf("/usr/share/efi/%s/%s", g.config.Arch, f))
				fileWriteName := filepath.Join(cnst.EfiDir, fmt.Sprintf("EFI/%s/%s", bootloaderName, f))

				g.config.Logger.Debugf("Copying %s to %s", fileReadName, fileWriteName)

				fileContent, err := g.config.Fs.ReadFile(fileReadName)
				if err != nil {
					return fmt.Errorf("error reading %s: %s", fileReadName, err)
				}
				err = g.config.Fs.WriteFile(fileWriteName, fileContent, cnst.FilePerm)
				if err != nil {
					return fmt.Errorf("error writing %s: %s", fileWriteName, err)
				}
			}

			// Add grub.cfg in EFI that chainloads the grub.cfg in recovery
			// Notice that we set the config to /grub2/grub.cfg which means the above we need to copy the file from
			// the installation source into that dir
			grubCfgContent := []byte(fmt.Sprintf("search --no-floppy --label --set=root %s\nset prefix=($root)/grub2\nconfigfile ($root)/grub2/grub.cfg", stateLabel))
			err = g.config.Fs.WriteFile(filepath.Join(cnst.EfiDir, fmt.Sprintf("EFI/%s/grub.cfg", bootloaderName)), grubCfgContent, cnst.FilePerm)
			if err != nil {
				return fmt.Errorf("error writing %s: %s", filepath.Join(cnst.EfiDir, fmt.Sprintf("EFI/%s/grub.cfg", bootloaderName)), err)
			}
			// Create entry in bootmanager
			// -c create
			// -d disk
			// -p partition containing loader
			// -w write unique sig to MBR if needed
			// -L label
			// -l loader (inverted slashes!)
			efibootmgrArgs := []string{"-c", "-d", target, "-p", "1", "-w", "-L", bootloaderName, "-l", fmt.Sprintf("\\EFI\\%s\\shim.efi", bootloaderName)}
			out, err := g.config.Runner.Run("efibootmgr", efibootmgrArgs...)
			if err != nil {
				g.config.Logger.Errorf("error running efibootmgr: %s", err)
				g.config.Logger.Debugf("error running efibootmgr: %s", out)
				return err
			}
		} else {
			return fmt.Errorf("shim files do not exist at %s", filepath.Join(rootDir, fmt.Sprintf("/usr/share/efi/%s", g.config.Arch)))
		}
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
