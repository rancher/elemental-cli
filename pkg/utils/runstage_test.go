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

package utils_test

import (
	"bytes"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	conf "github.com/rancher-sandbox/elemental/pkg/config"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
	v1mock "github.com/rancher-sandbox/elemental/tests/mocks"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

func writeCmdline(s string, fs afero.Fs) error {
	if err := fs.Mkdir("/proc", os.ModePerm); err != nil {
		return err
	}
	return afero.WriteFile(fs, "/proc/cmdline", []byte(s), os.ModePerm)
}

var _ = Describe("run stage", Label("RunStage"), func() {
	var config *v1.RunConfig
	var runner *v1mock.FakeRunner
	var logger v1.Logger
	var syscall *v1mock.FakeSyscall
	var client *v1mock.FakeHTTPClient
	var mounter *v1mock.ErrorMounter
	var fs afero.Fs
	var memLog *bytes.Buffer

	BeforeEach(func() {
		runner = v1mock.NewFakeRunner()
		// Use a different config with a buffer for logger, so we can check the output
		// We also use the real fs
		memLog = &bytes.Buffer{}
		logger = v1.NewBufferLogger(memLog)
		fs = afero.NewMemMapFs()
		config = conf.NewRunConfig(
			v1.WithFs(fs),
			v1.WithRunner(runner),
			v1.WithLogger(logger),
			v1.WithMounter(mounter),
			v1.WithSyscall(syscall),
			v1.WithClient(client),
		)
	})

	It("fails if strict mode is enabled", Label("strict"), func() {
		writeCmdline("stages.c3po[0].datasource", fs)
		config.Logger.SetLevel(log.DebugLevel)
		config.Strict = true
		Expect(utils.RunStage("c3po", config)).ToNot(BeNil())
	})

	It("does not fail but prints errors by default", Label("strict"), func() {
		writeCmdline("stages.c3po[0].datasource", fs)

		config.Logger.SetLevel(log.DebugLevel)
		config.Strict = false
		out := utils.RunStage("c3po", config)
		Expect(out).To(BeNil())
		Expect(memLog.String()).To(ContainSubstring("Some errors found but were ignored"))
	})

	It("Goes over extra paths", func() {
		d, _ := afero.TempDir(fs, "", "elemental")
		_ = afero.WriteFile(fs, fmt.Sprintf("%s/test.yaml", d), []byte{}, os.ModePerm)
		defer os.RemoveAll(d)
		config.Logger.SetLevel(log.DebugLevel)
		config.CloudInitPaths = d
		Expect(utils.RunStage("luke", config)).To(BeNil())
		Expect(memLog).To(ContainSubstring(fmt.Sprintf("Adding extra paths: %s", d)))
		Expect(memLog).To(ContainSubstring("luke"))
		Expect(memLog).To(ContainSubstring("luke.before"))
		Expect(memLog).To(ContainSubstring("luke.after"))
	})

	It("parses cmdline uri", func() {
		d, _ := afero.TempDir(fs, "", "elemental")
		_ = afero.WriteFile(fs, fmt.Sprintf("%s/test.yaml", d), []byte{}, os.ModePerm)
		defer os.RemoveAll(d)
		writeCmdline(fmt.Sprintf("cos.setup=%s/test.yaml", d), fs)

		Expect(utils.RunStage("padme", config)).To(BeNil())
		Expect(memLog).To(ContainSubstring("padme"))
		Expect(memLog).To(ContainSubstring(fmt.Sprintf("%s/test.yaml", d)))
	})

	It("parses cmdline uri with dotnotation", func() {
		writeCmdline("stages.leia[0].commands[0]='echo beepboop'", fs)
		config.Logger.SetLevel(log.DebugLevel)
		Expect(utils.RunStage("leia", config)).To(BeNil())
		Expect(memLog).To(ContainSubstring("leia"))
		Expect(memLog).To(ContainSubstring("running command `echo beepboop`"))

		// try with a non-clean cmdline
		writeCmdline("BOOT=death-star single stages.leia[0].commands[0]='echo beepboop'", fs)
		Expect(utils.RunStage("leia", config)).To(BeNil())
		Expect(memLog).To(ContainSubstring("leia"))
		Expect(memLog).To(ContainSubstring("running command `echo beepboop`"))
	})

	// XXX: This currently fails as the fs in Runstage != from the fs of the cloudRunner
	// Runstage creates cloud-init paths if not present, otherwise yip would try to parse them as yaml
	// but the paths being created by RunStage are using aferofs, while yip is using vfs.
	// As aferofs doesn't comply to either the os.FS interface, or either simply returning the raw path, it's not possible to create
	// a direct wrapper between the two. (While otoh vfs has a wrapper to turn any afero into vfs)
	It("ignores YAML errors", func() {
		config.Logger.SetLevel(log.DebugLevel)
		writeCmdline("BOOT=death-star single stages.leia[0].commands[0]='echo beepboop'", fs)
		Expect(utils.RunStage("leia", config)).To(BeNil())
		Expect(memLog).To(ContainSubstring("leia"))
		Expect(memLog).To(ContainSubstring("running command `echo beepboop`"))
		fmt.Println(memLog.String())

		Expect(memLog.String()).ToNot(ContainSubstring("/proc/cmdline parsing returned errors while unmarshalling"))
		Expect(memLog.String()).ToNot(ContainSubstring("Some errors found but were ignored. Enable --strict mode to fail on those or --debug to see them in the log"))

		memLog.Reset()
		config.Logger.SetLevel(log.DebugLevel)

		writeCmdline("BOOT=death-star sing1!~@$%6^&**le /varlib stag_#var<Lib stages[0]='utterly broken by breaking schema'", fs)
		Expect(utils.RunStage("leia", config)).To(BeNil())

		fmt.Println(memLog.String())

		Expect(memLog.String()).To(ContainSubstring("/proc/cmdline parsing returned errors while unmarshalling"))
		Expect(memLog.String()).ToNot(ContainSubstring("Some errors found but were ignored. Enable --strict mode to fail on those or --debug to see them in the log"))

	})
})
