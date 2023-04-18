package v1

import (
	"context"
	"net/http"

	"github.com/containerd/containerd/archive"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type ImageExtractor interface {
	ExtractImage(imageRef, destination, platformRef string, local bool) error
}

type OCIImageExtractor struct{}

var _ ImageExtractor = OCIImageExtractor{}

func (e OCIImageExtractor) ExtractImage(imageRef, destination, platformRef string, local bool) error {
	platform, err := v1.ParsePlatform(platformRef)
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return err
	}

	image, err := image(ref, *platform, local)
	if err != nil {
		return err
	}

	reader := mutate.Extract(image)

	_, err = archive.Apply(context.Background(), destination, reader)
	return err
}

func image(ref name.Reference, platform v1.Platform, local bool) (v1.Image, error) {
	if local {
		return daemon.Image(ref)
	}

	return remote.Image(ref,
		remote.WithTransport(http.DefaultTransport),
		remote.WithPlatform(platform),
		remote.WithAuth(authn.Anonymous),
	)
}

func ParsePlatform(platform string) (os, arch, variant string, err error) {
	p, err := v1.ParsePlatform(platform)
	if err != nil {
		return "", "", "", err
	}

	return p.OS, p.Architecture, p.Variant, nil
}
