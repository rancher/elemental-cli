/*
Copyright © 2021 SUSE LLC

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
	"github.com/rancher-sandbox/elemental-cli/pkg/action"
	"github.com/rancher-sandbox/elemental-cli/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install DEVICE",
	Short: "elemental installer",
	Args: cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := utils.ReadConfigRun(viper.GetString("configDir"))
		if err != nil {
			fmt.Printf("Error reading config: %s\n", err)
		}
		// Should probably load whatever env vars we want to overload here and merge them into the viper configs
		// Note that vars with ELEMENTAL in front and that match entries in the config (only one level deep) are overwritten automatically
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Install called")
		install := action.NewInstallAction(args[0])
		err := install.Run()
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("configDir", "e", "/etc/elemental/", "dir where the elemental config resides")
	installCmd.Flags().StringP("dockerImage", "d", "", "Install a specified container image")
	installCmd.Flags().StringP("cloudInit", "c", "", "Cloud-init config file")
	installCmd.Flags().StringP("iso", "i", "", "Performs an installation from the ISO url")
	installCmd.Flags().StringP("partitionLayout", "p", "", "Partitioning layout file")
	installCmd.Flags().BoolP("noVerify", "", false, "Disable mtree checksum verification (requires images manifests generated with mtree separately)")
	installCmd.Flags().BoolP("noCosign", "", false, "Disable cosign verification (requires images with signatures)")
	installCmd.Flags().BoolP("noFormat", "", false, "Don’t format disks. It is implied that COS_STATE, COS_RECOVERY, COS_PERSISTENT, COS_OEM are already existing")
	installCmd.Flags().BoolP("forceEfi", "", false, "Forces an EFI installation")
	installCmd.Flags().BoolP("forceGpt", "", false, "Forces a GPT partition table")
	installCmd.Flags().BoolP("strict", "", false, "Enable strict check of hooks (They need to exit with 0)")
	installCmd.Flags().BoolP("debug", "", false, "Enables debugging information")
	installCmd.Flags().BoolP("poweroff", "", false, "Shutdown the system after install")

	viper.BindPFlags(installCmd.Flags())

}
