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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/mount-utils"

	"github.com/rancher/elemental-cli/cmd/config"
	"github.com/rancher/elemental-cli/pkg/action"
	"github.com/rancher/elemental-cli/pkg/constants"
	v1 "github.com/rancher/elemental-cli/pkg/types/v1"
	"github.com/rancher/elemental-cli/pkg/utils"
)

// NewBuildPXE returns a new instance of the build-pxe subcommand and appends it to
// the passed command. requireRoot is to initiate it with or without the CheckRoot
// pre-run check.
func NewBuildPXE(root *cobra.Command, addCheckRoot bool) *cobra.Command {
	c := &cobra.Command{
		Use:   "pxe SOURCE",
		Short: "Build PXE bootable installation media files",
		Long: "build PXE bootable installation media files\n\n" +
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
			spec, err := config.ReadBuildPXE(cfg, flags)
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
			repoURIs, _ := flags.GetStringArray("repo")

			if oRootfs != "" {
				if ok, err := utils.Exists(cfg.Fs, oRootfs); ok {
					spec.RootFS = append(spec.RootFS, v1.NewDirSrc(oRootfs))
				} else {
					cfg.Logger.Errorf("Invalid value for overlay-rootfs")
					return fmt.Errorf("Invalid path '%s': %v", oRootfs, err)
				}
			}

			for _, u := range repoURIs {
				cfg.Repos = append(cfg.Repos, v1.Repository{URI: u, Priority: constants.LuetRepoMaxPrio, Arch: cfg.Arch})
			}

			builder := action.NewBuildPXEAction(cfg, spec)
			err = builder.Run()
			if err != nil {
				cfg.Logger.Errorf(err.Error())
				return err
			}

			return nil
		},
	}
	root.AddCommand(c)
	c.Flags().StringP("name", "n", "", "Basename of the generated pxe files")
	c.Flags().StringP("output", "o", "", "Output directory (defaults to current directory)")
	c.Flags().Bool("date", false, "Adds a date suffix into the generated file names")
	c.Flags().String("overlay-rootfs", "", "Path of the overlayed rootfs data")
	c.Flags().String("pxe-boot-url", "tftp://10.0.2.2/isos", "HTTP/TFTP URL used to generate PXE config")
	c.Flags().StringArray("repo", []string{}, "A repository URI for luet. Can be repeated to add more than one source.")
	addArchFlags(c)
	addCosignFlags(c)
	addSquashFsCompressionFlags(c)
	addLocalImageFlag(c)
	return c
}

var _ = NewBuildPXE(buildCmd, true)
