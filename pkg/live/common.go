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

package live

import (
	"github.com/rancher/elemental-cli/pkg/constants"
)

const (
	efiBootPath           = "/EFI/BOOT"
	isoLoaderPath         = "/boot/x86_64/loader"
	grubArm64Path         = grubPrefixDir + "/arm64-efi"
	grubEfiImagex86Dest   = efiBootPath + "/bootx64.efi"
	grubEfiImageArm64Dest = efiBootPath + "/bootaa64.efi"
	grubCfg               = "grub.cfg"
	grubPrefixDir         = "/boot/grub2"
	//TODO use some identifer known to be unique
	grubEfiCfg = "search --no-floppy --file --set=root " + constants.IsoKernelPath +
		"\nset prefix=($root)" + grubPrefixDir +
		"\nconfigfile $prefix/" + grubCfg

	// TODO not convinced having such a config here is the best idea
	grubCfgTemplate = `search --no-floppy --file --set=root /boot/kernel                               
	set default=0                                                                   
	set timeout=10                                                                  
	set timeout_style=menu                                                          
	set linux=linux                                                                 
	set initrd=initrd                                                               
	if [ "${grub_cpu}" = "x86_64" -o "${grub_cpu}" = "i386" -o "${grub_cpu}" = "arm64" ];then
		if [ "${grub_platform}" = "efi" ]; then                                     
			if [ "${grub_cpu}" != "arm64" ]; then                                   
				set linux=linuxefi                                                  
				set initrd=initrdefi                                                
			fi                                                                      
		fi                                                                          
	fi                                                                              
	if [ "${grub_platform}" = "efi" ]; then                                         
		echo "Please press 't' to show the boot menu on this console"               
	fi                                                                              
	set font=($root)/boot/${grub_cpu}/loader/grub2/fonts/unicode.pf2                
	if [ -f ${font} ];then                                                          
		loadfont ${font}                                                            
	fi                                                                              
	menuentry "%s" --class os --unrestricted {                                     
		echo Loading kernel...                                                      
		$linux ($root)/boot/kernel cdroot root=live:CDLABEL=%s rd.live.dir=/ rd.live.squashimg=rootfs.squashfs console=tty1 console=ttyS0 rd.cos.disable
		echo Loading initrd...                                                      
		$initrd ($root)/boot/initrd                                                 
	}                                                                               
																					
	if [ "${grub_platform}" = "efi" ]; then                                         
		hiddenentry "Text mode" --hotkey "t" {                                      
			set textmode=true                                                       
			terminal_output console                                                 
		}                                                                           
	fi`
)
