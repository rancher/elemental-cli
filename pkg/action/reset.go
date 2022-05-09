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

func (r *ResetAction) resetHook(hook string, chroot bool) error {
	if chroot {
		extraMounts := map[string]string{}
		persistent, ok := r.spec.Partitions[cnst.PersistentPartName]
		if ok && persistent.MountPoint != "" {
			extraMounts[persistent.MountPoint] = "/usr/local" // nolint:goconst
		}
		oem, ok := r.spec.Partitions[cnst.OEMPartName]
		if ok && oem.MountPoint != "" {
			extraMounts[oem.MountPoint] = "/oem" // nolint:goconst
		}
		return ChrootHook(&r.cfg.Config, hook, r.cfg.Strict, r.spec.ActiveImg.MountPoint, extraMounts, r.cfg.CloudInitPaths...)
	}
	return Hook(&r.cfg.Config, hook, r.cfg.Strict, r.cfg.CloudInitPaths...)
}

type ResetAction struct {
	cfg  *v1.RunConfig
	spec *v1.ResetSpec
}

func NewResetAction(cfg *v1.RunConfig, spec *v1.ResetSpec) *ResetAction {
	return &ResetAction{cfg: cfg, spec: spec}
}

// ResetRun will reset the cos system to by following several steps
func (r ResetAction) Run() (err error) {
	e := elemental.NewElemental(&r.cfg.Config)
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
		if ok {
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
	err = e.DeployImage(&r.spec.ActiveImg, true)
	if err != nil {
		return err
	}
	cleanup.Push(func() error { return e.UnmountImage(&r.spec.ActiveImg) })

	// install grub
	grub := utils.NewGrub(&r.cfg.Config)
	err = grub.Install(
		r.spec.Target,
		r.spec.ActiveImg.MountPoint,
		r.spec.Partitions[constants.StatePartName].MountPoint,
		r.spec.GrubConf,
		r.spec.Tty,
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
	if r.cfg.Reboot {
		r.cfg.Logger.Infof("Rebooting in 5 seconds")
		return utils.Reboot(r.cfg.Runner, 5)
	} else if r.cfg.PowerOff {
		r.cfg.Logger.Infof("Shutting down in 5 seconds")
		return utils.Shutdown(r.cfg.Runner, 5)
	}
	return err
}
