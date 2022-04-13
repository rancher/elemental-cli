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
	"fmt"
	"os/exec"

	"github.com/rancher-sandbox/elemental/cmd/config"
	"github.com/rancher-sandbox/elemental/pkg/action"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/mount-utils"
)

// buildISO represents the build-iso command
var buildISO = &cobra.Command{
	Use:   "build-iso IMAGE",
	Short: "elemental build-iso IMAGE",
	Args:  cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return CheckRoot()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := exec.LookPath("mount")
		if err != nil {
			return err
		}
		mounter := mount.New(path)

		cfg, err := config.ReadConfigBuild(viper.GetString("config-dir"), mounter)
		if err != nil {
			cfg.Logger.Errorf("Error reading config: %s\n", err)
		}

		if len(args) == 1 {
			cfg.ISO.RootFS = []string{args[0]}
		}

		err = validateCosignFlags(cfg.Logger)
		if err != nil {
			return err
		}

		//TODO validate there is, at least some source for rootfs, uefi and isoimage

		// Set this after parsing of the flags, so it fails on parsing and prints usage properly
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true // Do not propagate errors down the line, we control them

		oRootfs, _ := cmd.Flags().GetString("overlay-rootfs")
		oUEFI, _ := cmd.Flags().GetString("overlay-uefi")
		oISO, _ := cmd.Flags().GetString("overlay-iso")
		repoURIs, _ := cmd.Flags().GetStringArray("repo")

		if oRootfs != "" {
			if ok, err := utils.Exists(cfg.Fs, oRootfs); ok {
				cfg.ISO.RootFS = append(cfg.ISO.RootFS, oRootfs)
			} else {
				cfg.Logger.Errorf("Invalid value for overlay-rootfs")
				return fmt.Errorf("Invalid path '%s': %v", oRootfs, err)
			}
		}
		if oUEFI != "" {
			if ok, err := utils.Exists(cfg.Fs, oUEFI); ok {
				cfg.ISO.UEFI = append(cfg.ISO.UEFI, oUEFI)
			} else {
				cfg.Logger.Errorf("Invalid value for overlay-uefi")
				return fmt.Errorf("Invalid path '%s': %v", oUEFI, err)
			}
		}
		if oISO != "" {
			if ok, err := utils.Exists(cfg.Fs, oISO); ok {
				cfg.ISO.Image = append(cfg.ISO.Image, oISO)
			} else {
				cfg.Logger.Errorf("Invalid value for overlay-iso")
				return fmt.Errorf("Invalid path '%s': %v", oISO, err)
			}
		}

		repos := []v1.Repository{}
		for _, u := range repoURIs {
			repos = append(repos, v1.Repository{URI: u})
		}
		cfg.Repos = repos

		err = action.BuildISORun(cfg)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildISO)
	buildISO.Flags().String("label", "", "Label of the ISO volume")
	buildISO.Flags().StringP("name", "n", "", "Basename of the generated ISO file")
	buildISO.Flags().Bool("date", true, "Adds a date suffix into the generated ISO file")
	buildISO.Flags().String("overlay-rootfs", "", "Path of the overlayed rootfs data")
	buildISO.Flags().String("overlay-uefi", "", "Path of the overlayed uefi data")
	buildISO.Flags().String("overlay-iso", "", "Path of the overlayed iso data")
	buildISO.Flags().StringArray("isoimage", []string{}, "A source for the ISO image. Can be repeated to add more than one source.")
	buildISO.Flags().StringArray("uefi", []string{}, "A source for the UEFI image. Can be repeated to add more than one source.")
	buildISO.Flags().StringArray("repo", []string{}, "A repository URI for luet. Can be repeated to add more than one source.")
	addCosignFlags(buildISO)
}
