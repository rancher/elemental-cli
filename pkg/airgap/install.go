package airgap

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
)

// TODO: go generate tags to pack things with elemental profile create
var assets embed.FS

func Extract(profile, dst string) error {
	source := fmt.Sprintf("%s.tar.xz", profile)
	f, err := assets.Open(source)
	if err != nil {
		return err
	}

	tempdir, err := os.MkdirTemp("", "elemental")
	defer os.RemoveAll(tempdir)

	if err := copyFileContents(f, tempdir); err != nil {
		return err
	}

	uaIface, err := archiver.ByExtension(source)
	if err != nil {
		return err
	}

	un, ok := uaIface.(archiver.Unarchiver)
	if !ok {
		return fmt.Errorf("format specified by source filename is not an archive format: %s (%T)", source, uaIface)
	}

	mytar := &archiver.Tar{
		OverwriteExisting:      true,
		MkdirAll:               true,
		ImplicitTopLevelFolder: false,
		ContinueOnError:        false,
	}

	switch v := uaIface.(type) {
	case *archiver.Tar:
		uaIface = mytar
	case *archiver.TarBrotli:
		v.Tar = mytar
	case *archiver.TarBz2:
		v.Tar = mytar
	case *archiver.TarGz:
		v.Tar = mytar
	case *archiver.TarLz4:
		v.Tar = mytar
	case *archiver.TarSz:
		v.Tar = mytar
	case *archiver.TarXz:
		v.Tar = mytar
	case *archiver.TarZstd:
		v.Tar = mytar
	}
	return un.Unarchive(filepath.Join(tempdir, source), dst)
}

func copyFileContents(in fs.File, dst string) (err error) {
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()

	os.Chmod(dst, 0755)
	return
}
