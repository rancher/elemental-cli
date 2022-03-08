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

package constants

func InstallDeps() []string {
	return []string{
		"rsync", "losetup",
		"wipefs", "tune2fs",
		"blkid", "lsblk",
		"e2fsck", "resize2fs",
		"mount", "umount",
		"xfs_growfs", "udevadm",
		"mksquashfs", "grub2-install",
		"grub2-editenv",
	}
}

func UpgradeDeps() []string {
	return InstallDeps()
}
