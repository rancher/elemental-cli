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

package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/rancher-sandbox/elemental-cli/pkg/types/v1"
	v1mock "github.com/rancher-sandbox/elemental-cli/tests/mocks"
	"github.com/spf13/afero"
	"k8s.io/mount-utils"
	"testing"
)

func TestChrootSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Elemental chroot suite")
}

var _ = Describe("Chroot", func() {
	var config *v1.RunConfig
	var runner v1.Runner
	var logger v1.Logger
	var syscall v1.SyscallInterface
	var client v1.HTTPClient
	var mounter mount.Interface
	var fs afero.Fs

	BeforeEach(func() {
		runner = &v1mock.FakeRunner{}
		syscall = &v1mock.FakeSyscall{}
		mounter = v1mock.NewErrorMounter()
		client = &v1mock.FakeHttpClient{}
		logger = v1.NewNullLogger()
		fs = afero.NewMemMapFs()
		config = v1.NewRunConfig(
			v1.WithFs(fs),
			v1.WithRunner(runner),
			v1.WithLogger(logger),
			v1.WithMounter(mounter),
			v1.WithSyscall(syscall),
			v1.WithClient(client),
		)
	})
	Context("on success", func() {
		It("command should be called in the chroot", func() {
			syscallInterface := &v1mock.FakeSyscall{}
			config.Syscall = syscallInterface
			chroot := NewChroot(
				"/whatever",
				config,
			)
			defer chroot.Close()
			_, err := chroot.Run("chroot-command")
			Expect(err).To(BeNil())
			Expect(syscallInterface.WasChrootCalledWith("/whatever")).To(BeTrue())
		})

	})
	Context("on failure", func() {
		It("should return error if failed to chroot", func() {
			syscallInterface := &v1mock.FakeSyscall{ErrorOnChroot: true}
			config.Syscall = syscallInterface
			chroot := NewChroot(
				"/whatever",
				config,
			)
			defer chroot.Close()
			_, err := chroot.Run("chroot-command")
			Expect(err).ToNot(BeNil())
			Expect(syscallInterface.WasChrootCalledWith("/whatever")).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("chroot error"))
		})
		It("should return error if failed to mount on prepare", func() {
			mounter := v1mock.NewErrorMounter()
			mounter.ErrorOnMount = true
			config.Mounter = mounter

			chroot := NewChroot(
				"/whatever",
				config,
			)
			_, err := chroot.Run("chroot-command")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("mount error"))
		})
		It("should return error if failed to unmount on close", func() {
			mounter := v1mock.NewErrorMounter()
			mounter.ErrorOnUnmount = true
			config.Mounter = mounter
			
			chroot := NewChroot(
				"/whatever",
				config,
			)
			err := chroot.Close()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unmount error"))

		})
	})
})
