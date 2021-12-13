package utils

import (
	"github.com/rancher-sandbox/elemental-cli/pkg/types/v1"
	"github.com/spf13/afero"
	"k8s.io/mount-utils"
)

type ChrootOptions func(a *Chroot) error

// WithRunner allows to pass a v1.Runner interface to the chroot struct as an option in order to override the default runner
func WithRunner(runner v1.Runner) func(r *Chroot) error {
	return func(a *Chroot) error {
		a.runner = runner
		return nil
	}
}

// WithSyscall allows to pass a v1.SyscallInterface interface to the chroot struct as an option in order to override the default golang syscall
func WithSyscall(syscall v1.SyscallInterface) func(r *Chroot) error {
	return func(a *Chroot) error {
		a.syscall = syscall
		return nil
	}
}

// WithFS allows to pass a afero.Fs interface to the chroot struct as an option in order to override the default filesystem
func WithFS(fs afero.Fs) func(r *Chroot) error {
	return func(a *Chroot) error {
		a.fs = fs
		return nil
	}
}

// WithMounter allows to pass a mount.Interface interface to the chroot struct as an option in order to override the default mount command
func WithMounter(mounter mount.Interface) func(r *Chroot) error {
	return func(a *Chroot) error {
		a.mounter = mounter
		return nil
	}
}
