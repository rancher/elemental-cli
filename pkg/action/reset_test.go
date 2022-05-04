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
	"errors"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/elemental/pkg/action"
	conf "github.com/rancher-sandbox/elemental/pkg/config"
	"github.com/rancher-sandbox/elemental/pkg/constants"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
	v1mock "github.com/rancher-sandbox/elemental/tests/mocks"
	"github.com/twpayne/go-vfs"
	"github.com/twpayne/go-vfs/vfst"
)

var _ = Describe("Install action tests", func() {
	var config *v1.RunConfigNew
	var runner *v1mock.FakeRunner
	var fs vfs.FS
	var logger v1.Logger
	var mounter *v1mock.ErrorMounter
	var syscall *v1mock.FakeSyscall
	var client *v1mock.FakeHTTPClient
	var cloudInit *v1mock.FakeCloudInitRunner
	var cleanup func()
	var memLog *bytes.Buffer
	//var ghwTest v1mock.GhwMock

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
		config = conf.NewRunConfigNew(
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

	/*Describe("Reset Setup", Label("resetsetup"), func() {
		var bootedFrom, cmdFail string
		BeforeEach(func() {
			cmdFail = ""
			fs.Create(constants.EfiDevice)
			bootedFrom = constants.RecoverySquashFile
			runner.SideEffect = func(cmd string, args ...string) ([]byte, error) {
				if cmd == cmdFail {
					return []byte{}, errors.New("Command failed")
				}
				switch cmd {
				case "cat":
					return []byte(bootedFrom), nil
				default:
					return []byte{}, nil
				}
			}
			// This creates a fake disk for ghw to read from
			// we only need this 3 partitions for the reset tests
			mainDisk := block.Disk{
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
						Name:  "device3",
						Label: "COS_PERSISTENT",
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
		It("Configures reset command", func() {
			Expect(action.ResetSetup(config)).To(BeNil())
			Expect(config.Target).To(Equal("/dev/device"))
			Expect(config.Images.GetActive().Source.Value()).To(Equal(constants.IsoBaseTree))
			Expect(config.Images.GetActive().Source.IsDir()).To(BeTrue())
		})
		It("Configures reset command with --docker-image", func() {
			config.DockerImg = "some-image"
			Expect(action.ResetSetup(config)).To(BeNil())
			Expect(config.Target).To(Equal("/dev/device"))
			Expect(config.Images.GetActive().Source.Value()).To(Equal("some-image"))
			Expect(config.Images.GetActive().Source.IsDocker()).To(BeTrue())
		})
		It("Configures reset command with --directory", func() {
			config.Directory = "/some/local/dir"
			Expect(action.ResetSetup(config)).To(BeNil())
			Expect(config.Target).To(Equal("/dev/device"))
			Expect(config.Images.GetActive().Source.Value()).To(Equal("/some/local/dir"))
			Expect(config.Images.GetActive().Source.IsDir()).To(BeTrue())
		})
		It("Fails if not booting from recovery", func() {
			bootedFrom = ""
			Expect(action.ResetSetup(config)).NotTo(BeNil())
		})
		It("Fails if partitions are not found", func() {
			// remove the disk
			ghwTest.RemoveDisk("device")
			Expect(action.ResetSetup(config)).NotTo(BeNil())
		})
	})*/
	Describe("Reset Action", Label("reset"), func() {
		var statePart, persistentPart, oemPart *v1.Partition
		var spec *v1.ResetSpec
		var reset *action.ResetAction
		var cmdFail string
		var err error
		BeforeEach(func() {
			spec, err = conf.NewResetSpec(config.Config)
			Expect(err).ShouldNot(HaveOccurred())
			cmdFail = ""
			recoveryImg := filepath.Join(constants.RunningStateDir, "cOS", constants.RecoveryImgFile)
			err = utils.MkdirAll(fs, filepath.Dir(recoveryImg), constants.DirPerm)
			Expect(err).To(BeNil())
			_, err = fs.Create(recoveryImg)
			Expect(err).To(BeNil())

			statePart = &v1.Partition{
				Label:      constants.StateLabel,
				Path:       "/dev/device1",
				Disk:       "/dev/device",
				FS:         constants.LinuxFs,
				Name:       constants.StatePartName,
				MountPoint: constants.StateDir,
			}
			oemPart = &v1.Partition{
				Label:      constants.OEMLabel,
				Path:       "/dev/device2",
				Disk:       "/dev/device",
				FS:         constants.LinuxFs,
				Name:       constants.OEMPartName,
				MountPoint: constants.PersistentDir,
			}
			persistentPart = &v1.Partition{
				Label:      constants.PersistentLabel,
				Path:       "/dev/device3",
				Disk:       "/dev/device",
				FS:         constants.LinuxFs,
				Name:       constants.PersistentPartName,
				MountPoint: constants.OEMDir,
			}
			spec.Partitions[constants.PersistentPartName] = persistentPart
			spec.Partitions[constants.StatePartName] = statePart
			spec.Partitions[constants.OEMPartName] = oemPart

			spec.ActiveImg.Size = 16
			spec.Target = statePart.Disk

			grubCfg := filepath.Join(spec.ActiveImg.MountPoint, spec.GrubConf)
			err = utils.MkdirAll(fs, filepath.Dir(grubCfg), constants.DirPerm)
			Expect(err).To(BeNil())
			_, err = fs.Create(grubCfg)
			Expect(err).To(BeNil())

			runner.SideEffect = func(cmd string, args ...string) ([]byte, error) {
				if cmdFail == cmd {
					return []byte{}, errors.New("Command failed")
				}
				return []byte{}, nil
			}
			reset = action.NewResetAction(config, spec)
		})
		It("Successfully resets on non-squashfs recovery", func() {
			config.Reboot = true
			Expect(reset.ResetRun()).To(BeNil())
		})
		It("Successfully resets on non-squashfs recovery including persistent data", func() {
			spec.FormatPersistent = true
			Expect(reset.ResetRun()).To(BeNil())
		})
		It("Successfully resets on squashfs recovery", Label("squashfs"), func() {
			config.PowerOff = true
			Expect(reset.ResetRun()).To(BeNil())
		})
		It("Successfully resets despite having errors on hooks", func() {
			cloudInit.Error = true
			Expect(reset.ResetRun()).To(BeNil())
		})
		It("Successfully resets from a docker image", Label("docker"), func() {
			spec.ActiveImg.Source = v1.NewDockerSrc("my/image:latest")
			luet := v1mock.NewFakeLuet()
			config.Luet = luet
			Expect(reset.ResetRun()).To(BeNil())
			Expect(luet.UnpackCalled()).To(BeTrue())
		})
		It("Fails installing grub", func() {
			cmdFail = "grub2-install"
			Expect(reset.ResetRun()).NotTo(BeNil())
		})
		It("Fails formatting state partition", func() {
			cmdFail = "mkfs.ext4"
			Expect(reset.ResetRun()).NotTo(BeNil())
		})
		It("Fails setting the active label on non-squashfs recovery", func() {
			cmdFail = "tune2fs"
			Expect(reset.ResetRun()).NotTo(BeNil())
		})
		It("Fails setting the passive label on squashfs recovery", func() {
			cmdFail = "tune2fs"
			Expect(reset.ResetRun()).NotTo(BeNil())
		})
		It("Fails mounting partitions", func() {
			mounter.ErrorOnMount = true
			Expect(reset.ResetRun()).NotTo(BeNil())
		})
		It("Fails unmounting partitions", func() {
			mounter.ErrorOnUnmount = true
			Expect(reset.ResetRun()).NotTo(BeNil())
		})
		It("Fails unpacking docker image ", func() {
			spec.ActiveImg.Source = v1.NewDockerSrc("my/image:latest")
			luet := v1mock.NewFakeLuet()
			luet.OnUnpackError = true
			config.Luet = luet
			Expect(reset.ResetRun()).NotTo(BeNil())
			Expect(luet.UnpackCalled()).To(BeTrue())
		})
	})
})
