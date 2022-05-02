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

package action_test

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/jaypipes/ghw/pkg/block"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/elemental/pkg/action"
	conf "github.com/rancher-sandbox/elemental/pkg/config"
	"github.com/rancher-sandbox/elemental/pkg/constants"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
	v1mock "github.com/rancher-sandbox/elemental/tests/mocks"
	"github.com/sirupsen/logrus"
	"github.com/twpayne/go-vfs"
	"github.com/twpayne/go-vfs/vfst"
	"k8s.io/mount-utils"
)

const printOutput = `BYT;
/dev/loop0:50593792s:loopback:512:512:gpt:Loopback device:;`
const partTmpl = `
%d:%ss:%ss:2048s:ext4::type=83;`

var _ = Describe("Runtime Actions", func() {
	var config *v1.RunConfig
	var runner *v1mock.FakeRunner
	var fs vfs.FS
	var logger v1.Logger
	var mounter *v1mock.ErrorMounter
	var syscall *v1mock.FakeSyscall
	var client *v1mock.FakeHTTPClient
	var cloudInit *v1mock.FakeCloudInitRunner
	var cleanup func()
	var memLog *bytes.Buffer
	var ghwTest v1mock.GhwMock

	BeforeEach(func() {
		runner = v1mock.NewFakeRunner()
		syscall = &v1mock.FakeSyscall{}
		mounter = v1mock.NewErrorMounter()
		client = &v1mock.FakeHTTPClient{}
		memLog = &bytes.Buffer{}
		logger = v1.NewBufferLogger(memLog)
		var err error
		fs, cleanup, err = vfst.NewTestFS(map[string]interface{}{})
		Expect(err).Should(BeNil())

		cloudInit = &v1mock.FakeCloudInitRunner{}
		config = conf.NewRunConfig(
			conf.WithFs(fs),
			conf.WithRunner(runner),
			conf.WithLogger(logger),
			conf.WithMounter(mounter),
			conf.WithSyscall(syscall),
			conf.WithClient(client),
			conf.WithCloudInitRunner(cloudInit),
		)
	})

	AfterEach(func() { cleanup() })

	Describe("Upgrade Action", Label("upgrade"), func() {
		var upgrade *action.UpgradeAction
		var memLog *bytes.Buffer
		var l v1.LuetInterface
		activeImg := fmt.Sprintf("%s/cOS/%s", constants.RunningStateDir, constants.ActiveImgFile)
		passiveImg := fmt.Sprintf("%s/cOS/%s", constants.RunningStateDir, constants.PassiveImgFile)
		recoveryImgSquash := fmt.Sprintf("%s/cOS/%s", constants.UpgradeRecoveryDir, constants.RecoverySquashFile)
		recoveryImg := fmt.Sprintf("%s/cOS/%s", constants.UpgradeRecoveryDir, constants.RecoveryImgFile)
		transitionImgSquash := fmt.Sprintf("%s/cOS/%s", constants.UpgradeRecoveryDir, constants.TransitionSquashFile)
		transitionImg := fmt.Sprintf("%s/cOS/%s", constants.RunningStateDir, constants.TransitionImgFile)
		transitionImgRecovery := fmt.Sprintf("%s/cOS/%s", constants.UpgradeRecoveryDir, constants.TransitionImgFile)

		BeforeEach(func() {
			memLog = &bytes.Buffer{}
			logger = v1.NewBufferLogger(memLog)
			config.Logger = logger
			logger.SetLevel(logrus.DebugLevel)
			l = &v1mock.FakeLuet{}
			config.Luet = l
			// These values are loaded from /etc/cos/config normally via CMD
			config.StateLabel = constants.StateLabel
			config.PassiveLabel = constants.PassiveLabel
			config.RecoveryLabel = constants.RecoveryLabel
			config.ActiveLabel = constants.ActiveLabel
			config.UpgradeImage = "system/cos-config"
			config.RecoveryImage = "system/cos-config"
			config.ImgSize = 10
			// Create fake /etc/os-release
			utils.MkdirAll(fs, filepath.Join(utils.GetUpgradeTempDir(config), "etc"), constants.DirPerm)

			err := config.Fs.WriteFile(filepath.Join(utils.GetUpgradeTempDir(config), "etc", "os-release"), []byte("GRUB_ENTRY_NAME=TESTOS"), constants.FilePerm)
			Expect(err).ShouldNot(HaveOccurred())

			// Create paths used by tests
			utils.MkdirAll(fs, fmt.Sprintf("%s/cOS", constants.RunningStateDir), constants.DirPerm)
			utils.MkdirAll(fs, fmt.Sprintf("%s/cOS", constants.UpgradeRecoveryDir), constants.DirPerm)

			mainDisk := block.Disk{
				Name: "device",
				Partitions: []*block.Partition{
					{
						Name:  "device1",
						Label: "COS_GRUB",
						Type:  "ext4",
					},
					{
						Name:       "device2",
						Label:      "COS_STATE",
						Type:       "ext4",
						MountPoint: constants.RunningStateDir,
					},
					{
						Name:  "device4",
						Label: "COS_ACTIVE",
						Type:  "ext4",
					},
					{
						Name:  "device5",
						Label: "COS_PASSIVE",
						Type:  "ext4",
					},
					{
						Name:  "device5",
						Label: "COS_RECOVERY",
						Type:  "ext4",
					},
					{
						Name:  "device6",
						Label: "COS_OEM",
						Type:  "ext4",
					},
				},
			}
			ghwTest = v1mock.GhwMock{}
			ghwTest.AddDisk(mainDisk)
			ghwTest.CreateDevices()
		})
		AfterEach(func() {
			ghwTest.Clean()
		})
		It("Fails if some hook fails and strict is set", func() {
			runner.SideEffect = func(command string, args ...string) ([]byte, error) {
				if command == "cat" && args[0] == "/proc/cmdline" {
					return []byte(constants.ActiveLabel), nil
				}
				return []byte{}, nil
			}
			config.Runner = runner
			config.DockerImg = "alpine"
			config.Strict = true
			cloudInit.Error = true
			upgrade = action.NewUpgradeAction(config)
			err := upgrade.Run()
			Expect(err).To(HaveOccurred())
			// Make sure is a cloud init error!
			Expect(err.Error()).To(ContainSubstring("cloud init"))
		})
		Describe(fmt.Sprintf("Booting from %s", constants.ActiveLabel), Label("active_label"), func() {
			BeforeEach(func() {
				runner.SideEffect = func(command string, args ...string) ([]byte, error) {
					if command == "cat" && args[0] == "/proc/cmdline" {
						return []byte(constants.ActiveLabel), nil
					}
					if command == "mv" && args[0] == "-f" && args[1] == activeImg && args[2] == passiveImg {
						// we doing backup, do the "move"
						source, _ := fs.ReadFile(activeImg)
						_ = fs.WriteFile(passiveImg, source, constants.FilePerm)
						_ = fs.RemoveAll(activeImg)
					}
					if command == "mv" && args[0] == "-f" && args[1] == transitionImg && args[2] == activeImg {
						// we doing the image substitution, do the "move"
						source, _ := fs.ReadFile(transitionImg)
						_ = fs.WriteFile(activeImg, source, constants.FilePerm)
						_ = fs.RemoveAll(transitionImg)
					}
					return []byte{}, nil
				}
				config.Runner = runner
				// Create fake active/passive files
				_ = fs.WriteFile(activeImg, []byte("active"), constants.FilePerm)
				_ = fs.WriteFile(passiveImg, []byte("passive"), constants.FilePerm)
			})
			AfterEach(func() {
				_ = fs.RemoveAll(activeImg)
				_ = fs.RemoveAll(passiveImg)
			})
			It("Successfully upgrades from docker image", Label("docker"), func() {
				config.DockerImg = "alpine"
				upgrade = action.NewUpgradeAction(config)
				err := upgrade.Run()
				Expect(err).ToNot(HaveOccurred())

				// Check that the rebrand worked with our os-release value
				Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

				// Expect cos-state to have been mounted with our fake lsblk values
				fakeMounted := mount.MountPoint{
					Device: "/dev/device2",
					Path:   "/run/initramfs/cos-state",
					Type:   "auto",
				}
				Expect(mounter.List()).To(ContainElement(fakeMounted))

				// This should be the new image
				info, err := fs.Stat(activeImg)
				Expect(err).ToNot(HaveOccurred())
				// Image size should be the config.ImgSize as its truncated from the upgrade
				Expect(info.Size()).To(BeNumerically("==", int64(config.ImgSize*1024*1024)))
				Expect(info.IsDir()).To(BeFalse())

				// Should have backed up active to passive
				info, err = fs.Stat(passiveImg)
				Expect(err).ToNot(HaveOccurred())
				// Should be a tiny image as it should only contain our text
				// As this was generated by us at the start test and moved by the upgrade from active.iomg
				Expect(info.Size()).To(BeNumerically(">", 0))
				Expect(info.Size()).To(BeNumerically("<", int64(config.ImgSize*1024*1024)))
				f, _ := fs.ReadFile(passiveImg)
				// This should be a backup so it should read active
				Expect(f).To(ContainSubstring("active"))

				// Expect transition image to be gone
				_, err = fs.Stat(transitionImg)
				Expect(err).To(HaveOccurred())
			})
			It("Successfully upgrades from directory", Label("directory"), func() {
				config.Directory, _ = utils.TempDir(fs, "", "elementalupgrade")
				// Create the dir on real os as rsync works on the real os
				defer fs.RemoveAll(config.Directory)
				// create a random file on it
				err := fs.WriteFile(fmt.Sprintf("%s/file.file", config.Directory), []byte("something"), constants.FilePerm)
				Expect(err).ToNot(HaveOccurred())

				upgrade = action.NewUpgradeAction(config)
				err = upgrade.Run()
				Expect(err).ToNot(HaveOccurred())

				// Check that the rebrand worked with our os-release value
				Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

				// Not much that we can create here as the dir copy was done on the real os, but we do the rest of the ops on a mem one
				// This should be the new image
				info, err := fs.Stat(activeImg)
				Expect(err).ToNot(HaveOccurred())
				// Image size should not be empty
				Expect(info.Size()).To(BeNumerically("==", int64(config.ImgSize*1024*1024)))
				Expect(info.IsDir()).To(BeFalse())

				// Should have backed up active to passive
				info, err = fs.Stat(passiveImg)
				Expect(err).ToNot(HaveOccurred())
				// Should be a tiny image as it should only contain our text
				// As this was generated by us at the start test and moved by the upgrade from active.img
				Expect(info.Size()).To(BeNumerically(">", 0))
				Expect(info.Size()).To(BeNumerically("<", int64(config.ImgSize*1024*1024)))
				f, _ := fs.ReadFile(passiveImg)
				// This should be a backup so it should read active
				Expect(f).To(ContainSubstring("active"))

				// Expect transition image to be gone
				_, err = fs.Stat(transitionImg)
				Expect(err).To(HaveOccurred())

			})
			It("Successfully upgrades from channel upgrade", Label("channel"), func() {
				config.ChannelUpgrades = true

				upgrade = action.NewUpgradeAction(config)
				err := upgrade.Run()
				Expect(err).ToNot(HaveOccurred())

				// Check that the rebrand worked with our os-release value
				Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

				// Not much that we can create here as the dir copy was done on the real os, but we do the rest of the ops on a mem one
				// This should be the new image
				// Should probably do well in mounting the image and checking contents to make sure everything worked
				info, err := fs.Stat(activeImg)
				Expect(err).ToNot(HaveOccurred())
				// Image size should not be empty
				Expect(info.Size()).To(BeNumerically("==", int64(config.ImgSize*1024*1024)))
				Expect(info.IsDir()).To(BeFalse())

				// Should have backed up active to passive
				info, err = fs.Stat(passiveImg)
				Expect(err).ToNot(HaveOccurred())
				// Should be an really small image as it should only contain our text
				// As this was generated by us at the start test and moved by the upgrade from active.iomg
				Expect(info.Size()).To(BeNumerically(">", 0))
				Expect(info.Size()).To(BeNumerically("<", int64(config.ImgSize*1024*1024)))
				f, _ := fs.ReadFile(passiveImg)
				// This should be a backup so it should read active
				Expect(f).To(ContainSubstring("active"))

				// Expect transition image to be gone
				_, err = fs.Stat(transitionImg)
				Expect(err).To(HaveOccurred())
			})
			It("Successfully upgrades with cosign", Pending, Label("channel", "cosign"), func() {})
			It("Successfully upgrades with mtree", Pending, Label("channel", "mtree"), func() {})
			It("Successfully upgrades with strict", Pending, Label("channel", "strict"), func() {})
		})
		Describe(fmt.Sprintf("Booting from %s", constants.PassiveLabel), Label("passive_label"), func() {
			BeforeEach(func() {
				runner.SideEffect = func(command string, args ...string) ([]byte, error) {
					if command == "cat" && args[0] == "/proc/cmdline" {
						return []byte(constants.PassiveLabel), nil
					}
					if command == "mv" && args[0] == "-f" && args[1] == transitionImg && args[2] == activeImg {
						// we doing the image substitution, do the "move"
						source, _ := fs.ReadFile(transitionImg)
						_ = fs.WriteFile(activeImg, source, constants.FilePerm)
						_ = fs.RemoveAll(transitionImg)
					}
					return []byte{}, nil
				}
				config.Runner = runner
				// Create fake active/passive files
				_ = fs.WriteFile(activeImg, []byte("active"), constants.FilePerm)
				_ = fs.WriteFile(passiveImg, []byte("passive"), constants.FilePerm)
			})
			AfterEach(func() {
				_ = fs.RemoveAll(activeImg)
				_ = fs.RemoveAll(passiveImg)
			})
			It("does not backup active img to passive", Label("docker"), func() {
				config.DockerImg = "alpine"
				upgrade = action.NewUpgradeAction(config)
				err := upgrade.Run()
				Expect(err).ToNot(HaveOccurred())
				// Check that the rebrand worked with our os-release value
				Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

				// Expect cos-state to have been mounted with our fake lsblk values
				fakeMounted := mount.MountPoint{
					Device: "/dev/device2",
					Path:   "/run/initramfs/cos-state",
					Type:   "auto",
				}
				Expect(mounter.List()).To(ContainElement(fakeMounted))

				// This should be the new image
				info, err := fs.Stat(activeImg)
				Expect(err).ToNot(HaveOccurred())
				// Image size should not be empty
				Expect(info.Size()).To(BeNumerically("==", int64(config.ImgSize*1024*1024)))
				Expect(info.IsDir()).To(BeFalse())

				// Passive should have not been touched
				info, err = fs.Stat(passiveImg)
				Expect(err).ToNot(HaveOccurred())
				// Should be a tiny image as it should only contain our text
				// As this was generated by us at the start test and moved by the upgrade from active.iomg
				Expect(info.Size()).To(BeNumerically(">", 0))
				Expect(info.Size()).To(BeNumerically("<", int64(config.ImgSize*1024*1024)))
				f, _ := fs.ReadFile(passiveImg)
				Expect(f).To(ContainSubstring("passive"))

				// Expect transition image to be gone
				_, err = fs.Stat(transitionImg)
				Expect(err).To(HaveOccurred())

			})
		})
		Describe(fmt.Sprintf("Booting from %s", constants.RecoveryLabel), Label("recovery_label"), func() {
			BeforeEach(func() {
				config.RecoveryUpgrade = true

				// Clean fake disks and generate a new one based on recovery boot,
				// i.e COS_RECOVERY mounted on its proper dir
				ghwTest.Clean()
				recoveryDisk := block.Disk{
					Name: "device",
					Partitions: []*block.Partition{
						{
							Name:  "device1",
							Label: "COS_GRUB",
							Type:  "ext4",
						},
						{
							Name:  "device2",
							Label: "COS_STATE",
							Type:  "ext4",
						},
						{
							Name:  "device4",
							Label: "COS_ACTIVE",
							Type:  "ext4",
						},
						{
							Name:  "device5",
							Label: "COS_PASSIVE",
							Type:  "ext4",
						},
						{
							Name:       "device5",
							Label:      "COS_RECOVERY",
							Type:       "ext4",
							MountPoint: constants.UpgradeRecoveryDir,
						},
						{
							Name:  "device6",
							Label: "COS_OEM",
							Type:  "ext4",
						},
					},
				}
				ghwTest.AddDisk(recoveryDisk)
				ghwTest.CreateDevices()
			})
			Describe("Using squashfs", Label("squashfs"), func() {
				BeforeEach(func() {
					runner.SideEffect = func(command string, args ...string) ([]byte, error) {
						if command == "cat" && args[0] == "/proc/cmdline" {
							return []byte(constants.RecoveryLabel), nil
						}
						if command == "mksquashfs" && args[0] == "/tmp/upgrade" && args[1] == "/run/initramfs/live/cOS/transition.squashfs" {
							// create the transition img for squash to fake it
							_, _ = fs.Create(transitionImgSquash)
						}
						if command == "mv" && args[0] == "-f" && args[1] == transitionImgSquash && args[2] == recoveryImgSquash {
							// fake "move"
							f, _ := fs.ReadFile(transitionImgSquash)
							_ = fs.WriteFile(recoveryImgSquash, f, constants.FilePerm)
							_ = fs.RemoveAll(transitionImgSquash)
						}
						return []byte{}, nil
					}
					config.Runner = runner
					// Create recoveryImgSquash so ti identifies that we are using squash recovery
					_ = fs.WriteFile(recoveryImgSquash, []byte("recovery"), constants.FilePerm)
				})
				It("Successfully upgrades recovery from docker image", Label("docker"), func() {
					// This should be the old image
					info, err := fs.Stat(recoveryImgSquash)
					Expect(err).ToNot(HaveOccurred())
					// Image size should be empty
					Expect(info.Size()).To(BeNumerically(">", 0))
					Expect(info.IsDir()).To(BeFalse())
					f, _ := fs.ReadFile(recoveryImgSquash)
					Expect(f).To(ContainSubstring("recovery"))

					config.DockerImg = "alpine"
					upgrade = action.NewUpgradeAction(config)
					err = upgrade.Run()
					Expect(err).ToNot(HaveOccurred())

					// Check that the rebrand worked with our os-release value
					Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

					// Expect cos-state to have been remounted back on RO
					fakeMounted := mount.MountPoint{
						Device: "/dev/device5",
						Path:   "/run/initramfs/live",
						Type:   "auto",
					}
					Expect(mounter.List()).To(ContainElement(fakeMounted))

					// This should be the new image
					info, err = fs.Stat(recoveryImgSquash)
					Expect(err).ToNot(HaveOccurred())
					// Image size should be empty
					Expect(info.Size()).To(BeNumerically("==", 0))
					Expect(info.IsDir()).To(BeFalse())
					f, _ = fs.ReadFile(recoveryImgSquash)
					Expect(f).ToNot(ContainSubstring("recovery"))

					// Transition squash should not exist
					info, err = fs.Stat(transitionImgSquash)
					Expect(err).To(HaveOccurred())

				})
				It("Successfully upgrades recovery from directory", Label("directory"), func() {
					config.Directory, _ = utils.TempDir(fs, "", "elemental")
					// create a random file on it
					_ = fs.WriteFile(fmt.Sprintf("%s/file.file", config.Directory), []byte("something"), constants.FilePerm)

					upgrade = action.NewUpgradeAction(config)
					err := upgrade.Run()
					Expect(err).ToNot(HaveOccurred())

					// Check that the rebrand worked with our os-release value
					Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

					// Expect cos-state to have been remounted back on RO
					fakeMounted := mount.MountPoint{
						Device: "/dev/device5",
						Path:   "/run/initramfs/live",
						Type:   "auto",
					}
					Expect(mounter.List()).To(ContainElement(fakeMounted))

					// This should be the new image
					info, err := fs.Stat(recoveryImgSquash)
					Expect(err).ToNot(HaveOccurred())
					// Image size should be empty
					Expect(info.Size()).To(BeNumerically("==", 0))
					Expect(info.IsDir()).To(BeFalse())

					// Transition squash should not exist
					info, err = fs.Stat(transitionImgSquash)
					Expect(err).To(HaveOccurred())

				})
				It("Successfully upgrades recovery from channel upgrade", Label("channel"), func() {
					// This should be the old image
					info, err := fs.Stat(recoveryImgSquash)
					Expect(err).ToNot(HaveOccurred())
					// Image size should be empty
					Expect(info.Size()).To(BeNumerically(">", 0))
					Expect(info.IsDir()).To(BeFalse())
					f, _ := fs.ReadFile(recoveryImgSquash)
					Expect(f).To(ContainSubstring("recovery"))

					config.ChannelUpgrades = true

					upgrade = action.NewUpgradeAction(config)
					err = upgrade.Run()
					Expect(err).ToNot(HaveOccurred())

					// Check that the rebrand worked with our os-release value
					Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

					// Expect cos-state to have been remounted back on RO
					fakeMounted := mount.MountPoint{
						Device: "/dev/device5",
						Path:   "/run/initramfs/live",
						Type:   "auto",
					}
					Expect(mounter.List()).To(ContainElement(fakeMounted))

					// This should be the new image
					info, err = fs.Stat(recoveryImgSquash)
					Expect(err).ToNot(HaveOccurred())
					// Image size should be empty
					Expect(info.Size()).To(BeNumerically("==", 0))
					Expect(info.IsDir()).To(BeFalse())
					f, _ = fs.ReadFile(recoveryImgSquash)
					Expect(f).ToNot(ContainSubstring("recovery"))

					// Transition squash should not exist
					info, err = fs.Stat(transitionImgSquash)
					Expect(err).To(HaveOccurred())
				})
			})
			Describe("Not using squashfs", Label("non-squashfs"), func() {
				BeforeEach(func() {
					runner.SideEffect = func(command string, args ...string) ([]byte, error) {
						if command == "cat" && args[0] == "/proc/cmdline" {
							return []byte(constants.RecoveryLabel), nil
						}
						if command == "mv" && args[0] == "-f" && args[1] == transitionImgRecovery && args[2] == recoveryImg {
							// fake "move"
							f, _ := fs.ReadFile(transitionImgRecovery)
							_ = fs.WriteFile(recoveryImg, f, constants.FilePerm)
							_ = fs.RemoveAll(transitionImgRecovery)
						}
						return []byte{}, nil
					}
					config.Runner = runner
					_ = fs.WriteFile(recoveryImg, []byte("recovery"), constants.FilePerm)

				})
				It("Successfully upgrades recovery from docker image", Label("docker"), func() {
					// This should be the old image
					info, err := fs.Stat(recoveryImg)
					Expect(err).ToNot(HaveOccurred())
					// Image size should not be empty
					Expect(info.Size()).To(BeNumerically(">", 0))
					Expect(info.Size()).To(BeNumerically("<", int64(config.ImgSize*1024*1024)))
					Expect(info.IsDir()).To(BeFalse())
					f, _ := fs.ReadFile(recoveryImg)
					Expect(f).To(ContainSubstring("recovery"))

					config.DockerImg = "alpine"
					config.Logger.SetLevel(logrus.DebugLevel)
					upgrade = action.NewUpgradeAction(config)
					err = upgrade.Run()
					Expect(err).ToNot(HaveOccurred())

					// Check that the rebrand worked with our os-release value
					Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

					// Expect cos-state to have been remounted back on RO
					fakeMounted := mount.MountPoint{
						Device: "/dev/device5",
						Path:   "/run/initramfs/live",
						Type:   "auto",
					}
					Expect(mounter.List()).To(ContainElement(fakeMounted))

					// Should have created recovery image
					info, err = fs.Stat(recoveryImg)
					Expect(err).ToNot(HaveOccurred())
					// Image size should be default size
					Expect(info.Size()).To(BeNumerically("==", int64(config.ImgSize*1024*1024)))

					// Expect the rest of the images to not be there
					for _, img := range []string{activeImg, passiveImg, recoveryImgSquash} {
						_, err := fs.Stat(img)
						Expect(err).To(HaveOccurred())
					}
					fmt.Printf(memLog.String())

				})
				It("Successfully upgrades recovery from directory", Label("directory"), func() {
					config.Directory, _ = utils.TempDir(fs, "", "elemental")
					// create a random file on it
					_ = fs.WriteFile(fmt.Sprintf("%s/file.file", config.Directory), []byte("something"), constants.FilePerm)

					upgrade = action.NewUpgradeAction(config)
					err := upgrade.Run()
					Expect(err).ToNot(HaveOccurred())

					// Check that the rebrand worked with our os-release value
					Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

					// Expect cos-state to have been remounted back on RO
					fakeMounted := mount.MountPoint{
						Device: "/dev/device5",
						Path:   "/run/initramfs/live",
						Type:   "auto",
					}
					Expect(mounter.List()).To(ContainElement(fakeMounted))

					// This should be the new image
					info, err := fs.Stat(recoveryImg)
					Expect(err).ToNot(HaveOccurred())
					// Image size should be default size
					Expect(info.Size()).To(BeNumerically("==", int64(config.ImgSize*1024*1024)))
					Expect(info.IsDir()).To(BeFalse())

					// Transition squash should not exist
					info, err = fs.Stat(transitionImgRecovery)
					Expect(err).To(HaveOccurred())
				})
				It("Successfully upgrades recovery from channel upgrade", Label("channel"), func() {
					// This should be the old image
					info, err := fs.Stat(recoveryImg)
					Expect(err).ToNot(HaveOccurred())
					// Image size should not be empty
					Expect(info.Size()).To(BeNumerically(">", 0))
					Expect(info.Size()).To(BeNumerically("<", int64(config.ImgSize*1024*1024)))
					Expect(info.IsDir()).To(BeFalse())
					f, _ := fs.ReadFile(recoveryImg)
					Expect(f).To(ContainSubstring("recovery"))

					config.ChannelUpgrades = true

					upgrade = action.NewUpgradeAction(config)
					err = upgrade.Run()
					Expect(err).ToNot(HaveOccurred())

					// Check that the rebrand worked with our os-release value
					Expect(memLog).To(ContainSubstring("default_menu_entry=TESTOS"))

					// Expect cos-state to have been remounted back on RO
					fakeMounted := mount.MountPoint{
						Device: "/dev/device5",
						Path:   "/run/initramfs/live",
						Type:   "auto",
					}
					Expect(mounter.List()).To(ContainElement(fakeMounted))

					// Should have created recovery image
					info, err = fs.Stat(recoveryImg)
					Expect(err).ToNot(HaveOccurred())
					// Should have default image size
					Expect(info.Size()).To(BeNumerically("==", int64(config.ImgSize*1024*1024)))

					// Expect the rest of the images to not be there
					for _, img := range []string{activeImg, passiveImg, recoveryImgSquash} {
						_, err := fs.Stat(img)
						Expect(err).To(HaveOccurred())
					}
				})
			})
		})
	})
})
