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

package cloudinit

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/mudler/yip/pkg/executor"
	"github.com/mudler/yip/pkg/logger"
	"github.com/mudler/yip/pkg/plugins"
	"github.com/mudler/yip/pkg/schema"
	"github.com/rancher-sandbox/elemental/pkg/constants"
	"github.com/rancher-sandbox/elemental/pkg/partitioner"
	"github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/rancher-sandbox/elemental/pkg/utils"
	"github.com/twpayne/go-vfs"
	"os/exec"
	"strings"
)

type YipCloudInitRunner struct {
	exec    executor.Executor
	fs      vfs.FS
	console plugins.Console
}

// elementalCloudInitConsole represents a yip's Console implementations using
// the elemental v1.Runner interface.
type elementalCloudInitConsole struct {
	runner v1.Runner
	logger v1.Logger
}

// newElementalCloudInitConsole returns an instance of the elementalCloudInitConsole based on the
// given v1.Runner and v1.Logger.
func newElementalCloudInitConsole(l v1.Logger, r v1.Runner) *elementalCloudInitConsole {
	return &elementalCloudInitConsole{logger: l, runner: r}
}

// getRunner returns the internal runner used within this Console
func (c elementalCloudInitConsole) getRunner() v1.Runner {
	return c.runner
}

// Run runs a command using the v1.Runner internal instance
func (c elementalCloudInitConsole) Run(command string, opts ...func(cmd *exec.Cmd)) (string, error) {
	c.logger.Debugf("running command `%s`", command)
	cmd := c.runner.InitCmd("sh", "-c", command)
	for _, o := range opts {
		o(cmd)
	}
	out, err := c.runner.RunCmd(cmd)
	if err != nil {
		return string(out), fmt.Errorf("failed to run %s: %v", command, err)
	}

	return string(out), err
}

// Start runs a non blocking command using the v1.Runner internal instance
func (c elementalCloudInitConsole) Start(cmd *exec.Cmd, opts ...func(cmd *exec.Cmd)) error {
	c.logger.Debugf("running command `%s`", cmd)
	for _, o := range opts {
		o(cmd)
	}
	return cmd.Run()
}

// RunTemplate runs a sequence of non-blocking templated commands using the v1.Runner internal instance
func (c elementalCloudInitConsole) RunTemplate(st []string, template string) error {
	var errs error

	for _, svc := range st {
		out, err := c.Run(fmt.Sprintf(template, svc))
		if err != nil {
			c.logger.Error(out)
			c.logger.Error(err.Error())
			errs = multierror.Append(errs, err)
			continue
		}
	}
	return errs
}

// CloudInitRunner returns a default yip cloud init executor with the Elemental plugin set.
// It accepts a logger which is used inside the runner.
func NewYipCloudInitRunner(l v1.Logger, r v1.Runner) *YipCloudInitRunner {
	exec := executor.NewExecutor(
		executor.WithConditionals(
			plugins.NodeConditional,
			plugins.IfConditional,
		),
		executor.WithLogger(l),
		executor.WithPlugins(
			// Note, the plugin execution order depends on the order passed here
			plugins.DNS,
			plugins.Download,
			plugins.Git,
			plugins.Entities,
			plugins.EnsureDirectories,
			plugins.EnsureFiles,
			plugins.Commands,
			plugins.DeleteEntities,
			plugins.Hostname,
			plugins.Sysctl,
			plugins.User,
			plugins.SSH,
			plugins.LoadModules,
			plugins.Timesyncd,
			plugins.Systemctl,
			plugins.Environment,
			plugins.SystemdFirstboot,
			plugins.DataSources,
			layoutPlugin,
		),
	)
	return &YipCloudInitRunner{
		exec: exec, fs: vfs.OSFS,
		console: newElementalCloudInitConsole(l, r),
	}
}

func (ci YipCloudInitRunner) Run(stage string, args ...string) error {
	return ci.exec.Run(stage, ci.fs, ci.console, args...)
}

func (ci *YipCloudInitRunner) SetModifier(m schema.Modifier) {
	ci.exec.Modifier(m)
}

// Useful for testing purposes
func (ci *YipCloudInitRunner) SetFs(fs vfs.FS) {
	ci.fs = fs
}

// layoutPlugin is the elemental's implementation of Layout yip's plugin based
// on partitioner package
func layoutPlugin(l logger.Interface, s schema.Stage, fs vfs.FS, console plugins.Console) (err error) {
	if s.Layout.Device == nil {
		return nil
	}

	var dev *partitioner.Disk
	elemConsole, _ := console.(*elementalCloudInitConsole)
	runner := elemConsole.getRunner()
	log, _ := l.(v1.Logger)

	if len(strings.TrimSpace(s.Layout.Device.Label)) > 0 {
		partDevice, err := utils.GetFullDeviceByLabel(runner, s.Layout.Device.Label, 5)
		if err != nil {
			l.Errorf("Exiting, disk not found:\n %s", err.Error())
			return err
		}
		dev = partitioner.NewDisk(partDevice.Disk, partitioner.WithRunner(runner), partitioner.WithLogger(log))
	} else if len(strings.TrimSpace(s.Layout.Device.Path)) > 0 {
		dev = partitioner.NewDisk(s.Layout.Device.Path, partitioner.WithRunner(runner), partitioner.WithLogger(log))
	} else {
		l.Warnf("No target device defined, nothing to do")
		return nil
	}

	if !dev.Exists() {
		l.Errorf("Exiting, disk not found:\n %s", s.Layout.Device.Path)
		return errors.New("Target disk not found")
	}

	if s.Layout.Expand != nil {
		l.Infof("Extending last partition up to %d MiB", s.Layout.Expand.Size)
		out, err := dev.ExpandLastPartition(s.Layout.Expand.Size)
		if err != nil {
			l.Error(out)
			return err
		}
	}

	for _, part := range s.Layout.Parts {
		_, err := utils.GetFullDeviceByLabel(runner, part.FSLabel, 1)
		if err == nil {
			l.Warnf("Partition with FSLabel: %s already exists, ignoring", part.FSLabel)
			continue
		}

		// Set default filesystem
		if part.FileSystem == "" {
			part.FileSystem = constants.LinuxFs
		}

		l.Infof("Creating %s partition", part.FSLabel)
		partNum, err := dev.AddPartition(part.Size, part.FileSystem, part.PLabel)
		if err != nil {
			l.Error("Failed creating partition")
			return err
		}
		out, err := dev.FormatPartition(partNum, part.FileSystem, part.FSLabel)
		if err != nil {
			l.Errorf("Formatting partition failed: %s", out)
			return err
		}
	}
	return nil
}
