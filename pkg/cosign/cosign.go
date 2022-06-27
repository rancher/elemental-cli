/*
   Copyright © 2021 SUSE LLC

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

package cosign

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/sigstore/pkg/fulcioroots"
)

func VerifyKeyless(image string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	rekorClient, err := client.GetRekorClient("https://rekor.sigstore.dev")
	if err != nil {
		return false, fmt.Errorf("creating Rekor client: %w", err)
	}

	inter, err := fulcioroots.GetIntermediates()
	root, err := fulcioroots.Get()

	co := &cosign.CheckOpts{
		ClaimVerifier:     cosign.SimpleClaimVerifier,
		RootCerts:         root,
		IntermediateCerts: inter,
		RekorClient:       rekorClient,
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		return false, fmt.Errorf("parsing reference: %w", err)
	}

	_, bundleVerified, err := cosign.VerifyImageSignatures(ctx, ref, co)

	if err != nil {
		return false, err
	}

	return bundleVerified, nil
}
