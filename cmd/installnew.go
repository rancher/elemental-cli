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
	"os/exec"

	"github.com/rancher-sandbox/elemental/cmd/config"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/mount-utils"
)

// NewInstallCmd returns a new instance of the install subcommand and appends it to
// the root command. requireRoot is to initiate it with or without the CheckRoot
// pre-run check. This method is mostly used for testing purposes.
func NewInstallNewCmd(root *cobra.Command, addCheckRoot bool) *cobra.Command {
	c := &cobra.Command{
		Use:   "installnew DEVICE",
		Short: "elemental installer",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			_ = viper.BindEnv("target", "ELEMENTAL_TARGET")
			if addCheckRoot {
				return CheckRoot()
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := exec.LookPath("mount")
			if err != nil {
				return err
			}
			mounter := mount.New(path)

			cfg, err := config.ReadConfigRunNew(viper.GetString("config-dir"), cmd, mounter)
			if err != nil {
				cfg.Logger.Errorf("Error reading config: %s\n", err)
			}

			if err := validateInstallUpgradeFlags(cfg.Logger); err != nil {
				return err
			}

			// Manage deprecated flags
			// Adapt 'docker-image' and 'directory'  deprecated flags to 'system' syntax
			// Adapt 'force-efi' and 'force-gpt' to 'firmware' and 'part-table'
			adaptDockerImageAndDirectoryFlagsToSystem()
			efi := viper.GetBool("force-efi")
			if efi {
				viper.Set("firmware", v1.EFI)
			}
			gpt := viper.GetBool("force-gpt")
			if gpt {
				viper.Set("part-table", v1.GPT)
			}

			// Maps flags or env vars to the sub install structure so viper
			//also unmarshals them
			keyRemap := map[string]string{
				"cloud-init":      "cloud-init",
				"iso":             "iso",
				"no-format":       "no-format",
				"part-table":      "part-table",
				"firmware":        "firmware",
				"tty":             "grub-tty",
				"force":           "force",
				"system":          "system.uri",
				"recovery-system": "recovery-system.uri",
			}

			cmd.SilenceUsage = true
			spec, err := config.ReadInstallSpec(cfg, keyRemap)
			if err != nil {
				cfg.Logger.Errorf("invalid reset command setup %v", err)
				return err
			}

			cfg.Logger.Infof("Loaded run config: %+v", cfg)
			cfg.Logger.Infof("Loaded spec config: %+v", spec)
			for _, part := range spec.Partitions {
				cfg.Logger.Infof("Loaded part: %+v", part)
			}
			cfg.Logger.Infof("Loaded system source: %+v", spec.ActiveImg.Source)
			cfg.Logger.Infof("Loaded recovery source: %+v", spec.RecoveryImg.Source)
			return nil
		},
	}
	firmType := newEnumFlag([]string{v1.EFI, v1.BIOS}, v1.EFI)
	pTableType := newEnumFlag([]string{v1.GPT, v1.MSDOS}, v1.GPT)

	root.AddCommand(c)
	c.Flags().StringP("cloud-init", "c", "", "Cloud-init config file")
	c.Flags().StringP("iso", "i", "", "Performs an installation from the ISO url")
	c.Flags().StringP("partition-layout", "p", "", "Partitioning layout file")
	c.Flags().MarkDeprecated("partition-layout", "'partition-layout' is deprecated and ignored please use a config file instead")
	c.Flags().BoolP("no-format", "", false, "Don’t format disks. It is implied that COS_STATE, COS_RECOVERY, COS_PERSISTENT, COS_OEM are already existing")

	c.Flags().BoolP("force-efi", "", false, "Forces an EFI installation")
	c.Flags().MarkDeprecated("force-efi", "'force-efi' is deprecated please use 'firmware' instead")
	c.Flags().Var(firmType, "firmware", "Firmware to install for ('esp' or 'bios_grub')")

	c.Flags().BoolP("force-gpt", "", false, "Forces a GPT partition table")
	c.Flags().MarkDeprecated("force-gpt", "'force-gpt' is deprecated please use 'part-table' instead")
	c.Flags().Var(pTableType, "part-table", "Partition table type to use")

	c.Flags().BoolP("tty", "", false, "Add named tty to grub")
	c.Flags().BoolP("force", "", false, "Force install")
	c.Flags().BoolP("eject-cd", "", false, "Try to eject the cd on reboot, only valid if booting from iso")
	addSharedInstallUpgradeFlags(c)
	return c
}

// register the subcommand into rootCmd
var _ = NewInstallNewCmd(rootCmd, false)
