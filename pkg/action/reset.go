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
	cnst "github.com/rancher-sandbox/elemental/pkg/constants"
	"github.com/rancher-sandbox/elemental/pkg/elemental"
	"github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
	"os"
	"path/filepath"
)

func resetHook(config *v1.RunConfig, hook string, chroot bool) error {
	if chroot {
		extraMounts := map[string]string{}
		persistent := config.Partitions.GetByName(cnst.PersistentPartName)
		if persistent != nil {
			extraMounts[persistent.MountPoint] = "/usr/local"
		}
		oem := config.Partitions.GetByName(cnst.OEMPartName)
		if oem != nil {
			extraMounts[oem.MountPoint] = "/oem"
		}
		return ActionChrootHook(config, hook, config.ActiveImage.MountPoint, extraMounts)
	}
	return ActionHook(config, hook)
}

// ResetSetup will set installation parameters according to
// the given configuration flags
func ResetSetup(config *v1.RunConfig) error {
	if !utils.BootedFrom(config.Runner, cnst.RecoverySquashFile) &&
		!utils.BootedFrom(config.Runner, config.SystemLabel) {
		return errors.New("Reset can only be called from the recovery system")
	}

	SetupLuet(config)

	var rootTree string
	// TODO Properly include docker image source, luet package source and image source
	if config.Directory != "" {
		rootTree = config.Directory
	} else if config.DockerImg != "" {
		rootTree = config.DockerImg
	} else if utils.BootedFrom(config.Runner, cnst.RecoverySquashFile) {
		rootTree = cnst.IsoBaseTree
	}

	_, err := config.Fs.Stat(cnst.EfiDevice)
	efiExists := err == nil

	if efiExists {
		partEfi, err := utils.GetFullDeviceByLabel(config.Runner, cnst.EfiLabel, 1)
		if err != nil {
			return err
		}
		if partEfi.MountPoint == "" {
			partEfi.MountPoint = cnst.EfiDir
		}
		partEfi.Name = cnst.EfiPartName
		config.Partitions = append(config.Partitions, &partEfi)
	}

	// Only add it if it exists, not a hard requirement
	partOEM, err := utils.GetFullDeviceByLabel(config.Runner, cnst.OEMLabel, 1)
	if err == nil {
		if partOEM.MountPoint == "" {
			partOEM.MountPoint = cnst.OEMDir
		}
		partOEM.Name = cnst.OEMPartName
		config.Partitions = append(config.Partitions, &partOEM)
	} else {
		config.Logger.Warnf("No OEM partition found")
	}

	partState, err := utils.GetFullDeviceByLabel(config.Runner, cnst.StateLabel, 1)
	if err != nil {
		return err
	}
	if partState.MountPoint == "" {
		partState.MountPoint = cnst.StateDir
	}
	partState.Name = cnst.StatePartName
	config.Partitions = append(config.Partitions, &partState)
	config.Target = partState.Disk

	// Only add it if it exists, not a hard requirement
	partPersistent, err := utils.GetFullDeviceByLabel(config.Runner, cnst.PersistentLabel, 1)
	if err == nil {
		if partPersistent.MountPoint == "" {
			partPersistent.MountPoint = cnst.PersistentDir
		}
		partPersistent.Name = cnst.PersistentPartName
		config.Partitions = append(config.Partitions, &partPersistent)
	} else {
		config.Logger.Warnf("No Persistent partition found")
	}

	config.ActiveImage = v1.Image{
		Label:      config.ActiveLabel,
		Size:       cnst.ImgSize,
		File:       filepath.Join(partState.MountPoint, "cOS", cnst.ActiveImgFile),
		FS:         cnst.LinuxImgFs,
		RootTree:   rootTree,
		MountPoint: cnst.ActiveDir,
	}

	return nil
}

// ResetRun will reset the cos system to by following several steps
func ResetRun(config *v1.RunConfig) (err error) {
	ele := elemental.NewElemental(config)

	err = resetHook(config, cnst.BeforeResetHook, false)
	if err != nil {
		return err
	}

	// Reformat state partition
	state := config.Partitions.GetByName(cnst.StatePartName)
	err = ele.UnmountPartition(state)
	if err != nil {
		return err
	}
	err = ele.FormatPartition(state)
	if err != nil {
		return err
	}
	// Reformat persistent partitions
	if config.ResetPersistent {
		persistent := config.Partitions.GetByName(cnst.PersistentPartName)
		if persistent != nil {
			err = ele.UnmountPartition(persistent)
			if err != nil {
				return err
			}
			err = ele.FormatPartition(persistent)
			if err != nil {
				return err
			}
		}
		oem := config.Partitions.GetByName(cnst.OEMPartName)
		if oem != nil {
			err = ele.UnmountPartition(oem)
			if err != nil {
				return err
			}
			err = ele.FormatPartition(oem)
			if err != nil {
				return err
			}
		}
	}

	// Mount configured partitions
	err = ele.MountPartitions()
	if err != nil {
		return err
	}
	defer func() {
		if tmpErr := ele.UnmountPartitions(); tmpErr != nil && err == nil {
			err = tmpErr
		}
	}()

	// install Active
	// TODO all this logic should be part of the CopyImage(img *v1.Image) refactor
	if config.ActiveImage.RootTree != "" {
		// create active file system image
		err = ele.CreateFileSystemImage(config.ActiveImage)
		if err != nil {
			return err
		}

		//mount file system image
		err = ele.MountImage(&config.ActiveImage, "rw")
		if err != nil {
			return err
		}
		defer func() {
			if tmpErr := ele.UnmountImage(&config.ActiveImage); tmpErr != nil && err == nil {
				err = tmpErr
			}
		}()
		err = ele.CopyActive()
		if err != nil {
			return err
		}
	} else {
		srcImg := filepath.Join(cnst.RunningStateDir, "cOS", cnst.RecoveryImgFile)
		config.Logger.Infof("Copying Active image...")
		err := config.Fs.MkdirAll(filepath.Dir(config.ActiveImage.File), os.ModeDir)
		if err != nil {
			return err
		}
		err = utils.CopyFile(config.Fs, srcImg, config.ActiveImage.File)
		if err != nil {
			return err
		}
		_, err = config.Runner.Run("tune2fs", "-L", config.ActiveLabel, config.ActiveImage.File)
		if err != nil {
			config.Logger.Errorf("Failed to apply label %s to $s", config.ActiveLabel, config.ActiveImage.File)
			config.Fs.Remove(config.ActiveImage.File)
			return err
		}
		config.Logger.Infof("Finished copying Active...")
		err = ele.MountImage(&config.ActiveImage, "rw")
		if err != nil {
			return err
		}
		defer func() {
			if tmpErr := ele.UnmountImage(&config.ActiveImage); tmpErr != nil && err == nil {
				err = tmpErr
			}
		}()
	}

	// install grub
	grub := utils.NewGrub(config)
	err = grub.Install()
	if err != nil {
		return err
	}
	// Relabel SELinux
	_ = ele.SelinuxRelabel(cnst.ActiveDir, false)

	err = resetHook(config, cnst.AfterResetChrootHook, true)
	if err != nil {
		return err
	}

	// Unmount active image
	err = ele.UnmountImage(&config.ActiveImage)
	if err != nil {
		return err
	}

	// install Passive
	err = ele.CopyPassive()
	if err != nil {
		return err
	}

	err = resetHook(config, cnst.AfterResetHook, false)
	if err != nil {
		return err
	}

	// installation rebrand (only grub for now)
	err = ele.Rebrand()
	if err != nil {
		return err
	}

	// Reboot, poweroff or nothing
	if config.Reboot {
		config.Logger.Infof("Rebooting in 5 seconds")
		return utils.Reboot(config.Runner, 5)
	} else if config.PowerOff {
		config.Logger.Infof("Shutting down in 5 seconds")
		return utils.Shutdown(config.Runner, 5)
	}
	return err
}
