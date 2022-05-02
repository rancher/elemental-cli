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
	"github.com/rancher-sandbox/elemental/pkg/constants"
	cnst "github.com/rancher-sandbox/elemental/pkg/constants"
	"github.com/rancher-sandbox/elemental/pkg/elemental"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
)

func (i *ResetAction) resetHook(hook string, chroot bool) error {
	if chroot {
		extraMounts := map[string]string{}
		persistent, ok := i.spec.Partitions[cnst.PersistentPartName]
		if ok {
			extraMounts[persistent.MountPoint] = "/usr/local"
		}
		oem, ok := i.spec.Partitions[cnst.OEMPartName]
		if ok {
			extraMounts[oem.MountPoint] = "/oem"
		}
		return ChrootHook(i.cfg.Config, hook, i.cfg.Strict, i.spec.ActiveImg.MountPoint, extraMounts, i.cfg.CloudInitPaths...)
	}
	return Hook(i.cfg.Config, hook, i.cfg.Strict, i.cfg.CloudInitPaths...)
}

type ResetAction struct {
	cfg  *v1.RunConfigNew
	spec *v1.ResetSpec
}

func NewResetAction(cfg *v1.RunConfigNew, spec *v1.ResetSpec) *ResetAction {
	return &ResetAction{cfg: cfg, spec: spec}
}

// ResetRun will reset the cos system to by following several steps
func (r ResetAction) ResetRun(config *v1.RunConfig) (err error) { // nolint:gocyclo
	e := elemental.NewElemental(r.cfg.Config)
	cleanup := utils.NewCleanStack()
	defer func() { err = cleanup.Cleanup(err) }()

	err = r.resetHook(cnst.BeforeResetHook, false)
	if err != nil {
		return err
	}

	// Unmount partitions if any is already mounted before formatting
	err = e.UnmountPartitions(r.spec.Partitions.OrderedByMountPointPartitions(true))
	if err != nil {
		return err
	}

	// Reformat state partition
	err = e.FormatPartition(r.spec.Partitions[cnst.StatePartName])
	if err != nil {
		return err
	}
	// Reformat persistent partitions
	if r.spec.FormatPersistent {
		persistent, ok := r.spec.Partitions[cnst.PersistentPartName]
		if ok {
			err = e.FormatPartition(persistent)
			if err != nil {
				return err
			}
		}
		oem, ok := r.spec.Partitions[cnst.OEMPartName]
		if oem != nil {
			err = e.FormatPartition(oem)
			if err != nil {
				return err
			}
		}
	}

	// Mount configured partitions
	err = e.MountPartitions(r.spec.Partitions.OrderedByMountPointPartitions(false))
	if err != nil {
		return err
	}
	cleanup.Push(func() error {
		return e.UnmountPartitions(r.spec.Partitions.OrderedByMountPointPartitions(true))
	})

	// Deploy active image
	err = e.DeployImage(config.Images.GetActive(), true)
	if err != nil {
		return err
	}
	cleanup.Push(func() error { return e.UnmountImage(config.Images.GetActive()) })

	// install grub
	grub := utils.NewGrub(r.cfg.Config)
	err = grub.Install(
		r.spec.Target,
		r.spec.ActiveImg.MountPoint,
		r.spec.Partitions[constants.StatePartName].MountPoint,
		r.spec.GrubConf,
		r.spec.GrubTty,
		r.spec.Efi,
	)
	if err != nil {
		return err
	}
	// Relabel SELinux
	_ = e.SelinuxRelabel(cnst.ActiveDir, false)

	err = r.resetHook(cnst.AfterResetChrootHook, true)
	if err != nil {
		return err
	}

	// Unmount active image
	err = e.UnmountImage(&r.spec.ActiveImg)
	if err != nil {
		return err
	}

	// Install Passive
	err = e.DeployImage(&r.spec.PassiveImg, false)
	if err != nil {
		return err
	}

	err = r.resetHook(cnst.AfterResetHook, false)
	if err != nil {
		return err
	}

	// installation rebrand (only grub for now)

	if err != nil {
		return err
	}

	// Do not reboot/poweroff on cleanup errors
	err = cleanup.Cleanup(err)
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
