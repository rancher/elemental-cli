package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/twpayne/go-vfs"
)

// Random number state.
// We generate random temporary file names so that there's a good
// chance the file doesn't exist yet - keeps the number of tries in
// TempFile to a minimum.
var rand uint32
var randmu sync.Mutex

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextRandom() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

func TempDir(fs v1.FS, dir, prefix string) (name string, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		try := filepath.Join(dir, prefix+nextRandom())
		err = MkdirAll(fs, try, 0700)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		if err == nil {
			name = try
		}
		break
	}
	return
}

// Check if a file or directory exists.
func Exists(fs v1.FS, path string) (bool, error) {
	_, err := fs.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Check if the path is a dir
func IsDir(fs v1.FS, path string) (bool, error) {
	fi, err := fs.Stat(path)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

func MkdirAll(fs v1.FS, name string, mode os.FileMode) (err error) {
	if _, isReadOnly := fs.(*vfs.ReadOnlyFS); isReadOnly {
		return permError("mkdir", name)
	}
	if name, err = fs.RawPath(name); err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return os.MkdirAll(name, mode)
}

// permError returns an *os.PathError with Err syscall.EPERM.
func permError(op, path string) error {
	return &os.PathError{
		Op:   op,
		Path: path,
		Err:  syscall.EPERM,
	}
}
