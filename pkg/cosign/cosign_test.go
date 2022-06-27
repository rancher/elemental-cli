package cosign_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"

	"github.com/rancher/elemental-cli/pkg/cosign"
)

func TestCosignSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cosign test suite")
}

var _ = Describe("Cosign", func() {
	Describe("Verify", func() {
		It("Verifies correctly", func() {
			verified, err := cosign.Verify("quay.io/costoolkit/releases-teal:suc-integration-system-0.1-8", "")
			Expect(verified).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})
		It("Fails to parse an image", func() {
			verified, err := cosign.Verify("something-something%#@", "")
			Expect(verified).To(BeFalse())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not parse"))
		})
		It("Fails to load a non existing key", func() {
			verified, err := cosign.Verify("something-something%#@", "/tmp/key")
			Expect(verified).To(BeFalse())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not parse"))
		})
	})
})
