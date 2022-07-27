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

package action

import (
	_ "embed"
	"fmt"
	"html/template"
	"path/filepath"
	"time"

	"github.com/rancher/elemental-cli/pkg/constants"
	"github.com/rancher/elemental-cli/pkg/elemental"
	v1 "github.com/rancher/elemental-cli/pkg/types/v1"
	"github.com/rancher/elemental-cli/pkg/utils"
)

//go:embed ipxe.tmpl
var ipxeTemplate string

type BuildPXEAction struct {
	cfg  *v1.BuildConfig
	spec *v1.PXEConfig
	e    *elemental.Elemental
}

func NewBuildPXEAction(cfg *v1.BuildConfig, spec *v1.PXEConfig) *BuildPXEAction {
	return &BuildPXEAction{
		cfg:  cfg,
		e:    elemental.NewElemental(&cfg.Config),
		spec: spec,
	}
}

// BuildISORun will build an ISO from a given configuration
func (b *BuildPXEAction) Run() (err error) {
	cleanup := utils.NewCleanStack()
	defer func() { err = cleanup.Cleanup(err) }()

	pxeTmpDir, err := utils.TempDir(b.cfg.Fs, "", "elemental-pxe")
	if err != nil {
		return err
	}
	cleanup.Push(func() error { return b.cfg.Fs.RemoveAll(pxeTmpDir) })

	rootDir := filepath.Join(pxeTmpDir, "rootfs")
	err = utils.MkdirAll(b.cfg.Fs, rootDir, constants.DirPerm)
	if err != nil {
		return err
	}

	if b.cfg.OutDir != "" {
		err = utils.MkdirAll(b.cfg.Fs, b.cfg.OutDir, constants.DirPerm)
		if err != nil {
			b.cfg.Logger.Errorf("Failed creating output folder: %s", b.cfg.OutDir)
			return err
		}
	}

	b.cfg.Logger.Infof("Preparing squashfs root...")
	err = applySources(b.e, rootDir, b.spec.RootFS...)
	if err != nil {
		b.cfg.Logger.Errorf("Failed installing OS packages: %v", err)
		return err
	}
	err = utils.CreateDirStructure(b.cfg.Fs, rootDir)
	if err != nil {
		b.cfg.Logger.Errorf("Failed creating root directory structure: %v", err)
		return err
	}

	err = b.writeFiles(b.cfg.OutDir, rootDir)
	if err != nil {
		b.cfg.Logger.Errorf("Failed writing root tree: %v", err)
		return err
	}

	return err
}

func (b BuildPXEAction) writeFiles(outDir, rootDir string) error {
	var baseFileName string

	if b.cfg.Date {
		currTime := time.Now()
		baseFileName = fmt.Sprintf("%s.%s", b.cfg.Name, currTime.Format("20060102"))
	} else {
		baseFileName = b.cfg.Name
	}

	kernel, initrd, err := b.e.FindKernelInitrd(rootDir)
	if err != nil {
		b.cfg.Logger.Error("Could not find kernel and/or initrd", err)
		return err
	}

	b.cfg.Logger.Debugf("Copying Kernel file %s to root tree", kernel)
	err = utils.CopyFile(b.cfg.Fs, kernel, filepath.Join(outDir, fmt.Sprintf("%s%s", baseFileName, constants.PxeKernelSuffix)))
	if err != nil {
		b.cfg.Logger.Error("Could not copy kernel", err)
		return err
	}

	b.cfg.Logger.Debugf("Copying initrd file %s to root tree", initrd)
	err = utils.CopyFile(b.cfg.Fs, initrd, filepath.Join(outDir, fmt.Sprintf("%s%s", baseFileName, constants.PxeInitrdSuffix)))
	if err != nil {
		b.cfg.Logger.Error("Could not copy initrd", err)
		return err
	}

	b.cfg.Logger.Info("Creating squashfs...")
	squashOptions := append(constants.GetDefaultSquashfsOptions(), b.cfg.SquashFsCompressionConfig...)
	err = utils.CreateSquashFS(b.cfg.Runner, b.cfg.Logger, rootDir, filepath.Join(outDir, fmt.Sprintf("%s%s", baseFileName, constants.PxeRootfsSuffix)), squashOptions)
	if err != nil {
		b.cfg.Logger.Error("Could not create squashfs", err)
		return err
	}

	b.cfg.Logger.Info("Parsing iPXE template")
	ipxe, err := template.New("ipxe").Parse(ipxeTemplate)
	if err != nil {
		b.cfg.Logger.Error("Could not parse template", err)
		return err
	}

	ipxeFile, err := b.cfg.Fs.Create(filepath.Join(outDir, fmt.Sprintf("%s%s", baseFileName, constants.PxeConfigSuffix)))
	if err != nil {
		b.cfg.Logger.Error("Could not create iPXE config file", err)
		return err
	}

	b.cfg.Logger.Info("Writing iPXE config")
	err = ipxe.Execute(ipxeFile, struct {
		Name string
		URL  string
	}{
		Name: baseFileName,
		URL:  b.spec.PxeBootURL,
	})
	if err != nil {
		b.cfg.Logger.Error("Could not write config", err)
		return err
	}

	return nil
}
