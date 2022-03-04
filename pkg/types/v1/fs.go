package v1

import (
	"io/fs"
	"os"
)

type FS interface {
	Open(name string) (*os.File, error)
	Chmod(name string, mode os.FileMode) error
	Create(name string) (*os.File, error)
	Mkdir(name string, perm os.FileMode) error
	Stat(name string) (os.FileInfo, error)
	RemoveAll(path string) error
	ReadFile(filename string) ([]byte, error)
	RawPath(name string) (string, error)
	Remove(name string) error
	OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
}
