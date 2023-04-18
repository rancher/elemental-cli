package mocks

import v1 "github.com/rancher/elemental-cli/pkg/types/v1"

type FakeImageExtractor struct {
	Logger     v1.Logger
	SideEffect func(imageRef, destination, platformRef string, local bool) error
}

var _ v1.ImageExtractor = FakeImageExtractor{}

func NewFakeImageExtractor(logger v1.Logger) *FakeImageExtractor {
	return &FakeImageExtractor{
		Logger: logger,
	}
}

func (f FakeImageExtractor) ExtractImage(imageRef, destination, platformRef string, local bool) error {
	f.Logger.Debugf("extracting %s to %s in platform %s", imageRef, destination, platformRef)
	if f.SideEffect != nil {
		f.Logger.Debugf("running sideeffect")
		return f.SideEffect(imageRef, destination, platformRef, local)
	}

	return nil
}
