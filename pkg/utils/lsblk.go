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

package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	ghwUtil "github.com/jaypipes/ghw/pkg/util"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
)

// GhwPartitionToInternalPartition transforms a block.Partition from ghw lib to our v1.Partition type
func GhwPartitionToInternalPartition(partition *block.Partition) *v1.Partition {
	return &v1.Partition{
		Label:      partition.Label,
		Size:       uint(partition.SizeBytes / (1024 * 1024)), // Converts B to MB
		Name:       partition.Name,
		FS:         partition.Type,
		Flags:      nil,
		MountPoint: partition.MountPoint,
		Path:       filepath.Join("/dev", partition.Name),
		Disk:       filepath.Join("/dev", partition.Disk.Name),
	}
}

// GetAllPartitionsV2 returns all partitions in the system for all disks
func GetAllPartitionsV2() ([]*block.Partition, error) {
	var parts []*block.Partition
	blockDevices, err := block.New(ghw.WithDisableTools(), ghw.WithDisableWarnings())
	if err != nil {
		return nil, err
	}
	for _, d := range blockDevices.Disks {
		parts = append(parts, d.Partitions...)
	}
	return parts, nil
}

// GetDevicePartitionsV2 returns partitions under a given device
func GetDevicePartitionsV2(device string) ([]*block.Partition, error) {
	// We want to have the device always prefixed with a /dev
	if !strings.HasPrefix(device, "/dev") {
		device = filepath.Join("/dev", device)
	}
	blockDevices, err := block.New(ghw.WithDisableTools(), ghw.WithDisableWarnings())
	if err != nil {
		return nil, err
	}
	for _, disk := range blockDevices.Disks {
		if filepath.Join("/dev", disk.Name) == device {
			return disk.Partitions, nil
		}
	}
	return nil, fmt.Errorf("could not find disk %s", device)
}

func GetPartitionFSV2(partition string) (string, error) {
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
