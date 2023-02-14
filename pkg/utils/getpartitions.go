/*
Copyright © 2022 - 2023 SUSE LLC

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
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	ghwUtil "github.com/jaypipes/ghw/pkg/util"
	v1 "github.com/rancher/elemental-cli/pkg/types/v1"
)

const loopType = "loop"
const cryptType = "crypto_LUKS"

type BlockDevice struct {
	Label      string `json:"label"`
	PartLabel  string `json:"partlabel"`
	MountPoint string `json:"mountpoint"`
	Path       string `json:"path"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Size       string `json:"fssize"`
	FS         string `json:"fstype"`
	PartFlags  string `json:"partflags"`
	PKName     string `json:"pkname"`
	Kname      string `json:"kname"`
}

type LsblkResponse struct {
	BlockDevices []BlockDevice `json:"blockdevices"`
}

// GetAllPartitions returns all partitions in the system for all disks
func GetAllPartitions() (v1.PartitionList, error) {
	buf := bytes.NewBuffer(make([]byte, 0))
	ebuff := bytes.NewBuffer(make([]byte, 0))

	cmd := exec.Command("lsblk", "-aJOl")
	cmd.Stdout = buf
	cmd.Stderr = ebuff

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s:  %s", err, ebuff.String())
	}

	var blocks LsblkResponse
	var parts []*v1.Partition

	if err := json.NewDecoder(buf).Decode(&blocks); err != nil {
		return nil, err
	}

	// map devices by kernel name for determining root later
	deviceMap := make(map[string]BlockDevice, 0)
	for _, device := range blocks.BlockDevices {
		deviceMap[device.Kname] = device
	}

	for _, device := range blocks.BlockDevices {
		// we explicitly ignore crypt and loop devices as they are likely backed by the actual device or partition
		// we want to interact with.
		if device.Type == loopType || device.FS == cryptType {
			continue
		}

		// if a device is not the root kernel device walk the device tree to find the root
		rootDisk := device
		for rootDisk.PKName != "" {
			rootDisk = deviceMap[rootDisk.PKName]
		}

		size, _ := strconv.Atoi(device.Size)

		parts = append(parts, &v1.Partition{
			Name:            device.PartLabel,
			FilesystemLabel: device.Label,
			Size:            uint(size / (1024 * 1024)),
			FS:              device.FS,
			Flags:           strings.Split(device.PartFlags, " "),
			MountPoint:      device.MountPoint,
			Path:            device.Path,
			Disk:            rootDisk.Path,
		})
	}

	return parts, nil
}

// GetPartitionFS gets the FS of a partition given
func GetPartitionFS(partition string) (string, error) {
	// We want to have the device always prefixed with a /dev
	if !strings.HasPrefix(partition, "/dev") {
		partition = filepath.Join("/dev", partition)
	}
	blockDevices, err := block.New(ghw.WithDisableTools(), ghw.WithDisableWarnings())
	if err != nil {
		return "", err
	}

	for _, disk := range blockDevices.Disks {
		for _, part := range disk.Partitions {
			if filepath.Join("/dev", part.Name) == partition {
				if part.Type == ghwUtil.UNKNOWN {
					return "", fmt.Errorf("could not find filesystem for partition %s", partition)
				}
				return part.Type, nil
			}
		}
	}
	return "", fmt.Errorf("could not find filesystem for partition %s", partition)
}
