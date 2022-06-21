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

	"github.com/rancher/elemental-cli/cmd/config"
	"github.com/rancher/elemental-cli/pkg/action"
	"github.com/rancher/elemental-cli/pkg/constants"
	v1 "github.com/rancher/elemental-cli/pkg/types/v1"
	"github.com/rancher/elemental-cli/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/mount-utils"
)

// NewBuildISO returns a new instance of the buid-iso subcommand and appends it to
// the root command. requireRoot is to initiate it with or without the CheckRoot
// pre-run check. This method is mostly used for testing purposes.
func NewBuildISO(root *cobra.Command, addCheckRoot bool) *cobra.Command {
	c := &cobra.Command{
		Use:   "build-iso SOURCE",
		Short: "builds bootable installation media ISOs",
		Long: "builds bootable installation media ISOs\n\n" +
			"SOURCE - should be provided as uri in following format <sourceType>:<sourceName>\n" +
			"    * <sourceType> - might be [\"dir\", \"file\", \"oci\", \"docker\", \"channel\"], as default is \"docker\"\n" +
			"    * <sourceName> - is path to file or directory, image name with tag version or channel name",
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
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

			cfg, err := config.ReadConfigBuild(viper.GetString("config-dir"), cmd.Flags(), mounter)
			if err != nil {
				cfg.Logger.Errorf("Error reading config: %s\n", err)
			}

			flags := cmd.Flags()
			err = validateCosignFlags(cfg.Logger, flags)
			if err != nil {
				return err
			}

			// Set this after parsing of the flags, so it fails on parsing and prints usage properly
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true // Do not propagate errors down the line, we control them
			spec, err := config.ReadBuildISO(cfg, flags)
			if err != nil {
				cfg.Logger.Errorf("invalid install command setup %v", err)
				return err
			}

			if len(args) == 1 {
				imgSource, err := v1.NewSrcFromURI(args[0])
				if err != nil {
					cfg.Logger.Errorf("not a valid rootfs source image argument: %s", args[0])
					return err
				}
				spec.RootFS = []*v1.ImageSource{imgSource}
			} else if len(spec.RootFS) == 0 {
				errmsg := "rootfs source image for building ISO was not provided"
				cfg.Logger.Errorf(errmsg)
				return fmt.Errorf(errmsg)
			}

			// Repos and overlays can't be unmarshaled directly as they require
			// to be merged on top and flags do not match any config value key
			oRootfs, _ := flags.GetString("overlay-rootfs")
			oUEFI, _ := flags.GetString("overlay-uefi")
			oISO, _ := flags.GetString("overlay-iso")
			repoURIs, _ := flags.GetStringArray("repo")

			if oRootfs != "" {
				if ok, err := utils.Exists(cfg.Fs, oRootfs); ok {
					spec.RootFS = append(spec.RootFS, v1.NewDirSrc(oRootfs))
				} else {
					cfg.Logger.Errorf("Invalid value for overlay-rootfs")
					return fmt.Errorf("Invalid path '%s': %v", oRootfs, err)
				}
			}
			if oUEFI != "" {
				if ok, err := utils.Exists(cfg.Fs, oUEFI); ok {
					spec.UEFI = append(spec.UEFI, v1.NewDirSrc(oUEFI))
				} else {
					cfg.Logger.Errorf("Invalid value for overlay-uefi")
					return fmt.Errorf("Invalid path '%s': %v", oUEFI, err)
				}
			}
			if oISO != "" {
				if ok, err := utils.Exists(cfg.Fs, oISO); ok {
					spec.Image = append(spec.Image, v1.NewDirSrc(oISO))
				} else {
					cfg.Logger.Errorf("Invalid value for overlay-iso")
					return fmt.Errorf("Invalid path '%s': %v", oISO, err)
				}
			}

			for _, u := range repoURIs {
				cfg.Repos = append(cfg.Repos, v1.Repository{URI: u, Priority: constants.LuetRepoMaxPrio, Arch: cfg.Arch})
			}

			buildISO := action.NewBuildISOAction(cfg, spec)
			err = buildISO.ISORun()
			if err != nil {
				cfg.Logger.Errorf(err.Error())
				return err
			}

			return nil
		},
	}
	root.AddCommand(c)
	c.Flags().StringP("name", "n", "", "Basename of the generated ISO file")
	c.Flags().StringP("output", "o", "", "Output directory (defaults to current directory)")
	c.Flags().Bool("date", false, "Adds a date suffix into the generated ISO file")
	c.Flags().String("overlay-rootfs", "", "Path of the overlayed rootfs data")
	c.Flags().String("overlay-uefi", "", "Path of the overlayed uefi data")
	c.Flags().String("overlay-iso", "", "Path of the overlayed iso data")
	c.Flags().String("label", "", "Label of the ISO volume")
	c.Flags().StringArray("repo", []string{}, "A repository URI for luet. Can be repeated to add more than one source.")
	addArchFlags(c)
	addCosignFlags(c)
	addSquashFsCompressionFlags(c)
	addLocalImageFlag(c)
	return c
}

// register the subcommand into rootCmd
var _ = NewBuildISO(rootCmd, true)
