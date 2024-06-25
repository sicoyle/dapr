/*
Copyright 2023 The Dapr Authors
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

package sentry

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc"

	sentryv1pb "github.com/dapr/dapr/pkg/proto/sentry/v1"
	"github.com/dapr/dapr/pkg/security"
	securitytoken "github.com/dapr/dapr/pkg/security/token"
)

// Options contains the configuration options for connecting and requesting
// certificates from sentry.
type Options struct {
	SentryAddress string
	SentryID      spiffeid.ID
	Security      security.Handler
}

// Requester is used to request certificates from the sentry service for any
// daprd identity.
type Requester struct {
	sentryAddress  string
	sentryID       spiffeid.ID
	sec            security.Handler
	kubernetesMode bool
	sentryConn     *grpc.ClientConn
}

// New returns a new instance of the Requester.
func New(ctx context.Context, opts Options) (*Requester, error) {
	_, kubeMode := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	r := &Requester{
		sentryAddress:  opts.SentryAddress,
		sentryID:       opts.SentryID,
		sec:            opts.Security,
		kubernetesMode: kubeMode,
	}

	return r, r.dialSentryConnection(ctx)
}

// dialSentryConnection creates the gRPC connection to the Sentry service and blocks for 1 minute.
func (r *Requester) dialSentryConnection(ctx context.Context) error {
	connCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	conn, err := grpc.DialContext(connCtx, r.sentryAddress, r.sec.GRPCDialOptionMTLS(r.sentryID), grpc.WithBlock()) //nolint:staticcheck
	if err != nil {
		return fmt.Errorf("error establishing connection to sentry: %w", err)
	}
	r.sentryConn = conn

	return nil
}

// RequestCertificateFromSentry requests a certificate from sentry for a
// generic daprd identity in a namespace.
// Returns the signed certificate chain and leaf private key as a PEM encoded
// byte slice.
func (r *Requester) RequestCertificateFromSentry(ctx context.Context, namespace string) ([]byte, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: "_unknown"},
		DNSNames: []string{"_unknown"},
	}, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create sidecar csr: %w", err)
	}

	token, tokenValidator, err := securitytoken.GetSentryToken(r.kubernetesMode)
	if err != nil {
		return nil, nil, fmt.Errorf("error obtaining token: %w", err)
	}

	resp, err := sentryv1pb.NewCAClient(r.sentryConn).SignCertificate(ctx,
		&sentryv1pb.SignCertificateRequest{
			CertificateSigningRequest: pem.EncodeToMemory(&pem.Block{
				Type: "CERTIFICATE REQUEST", Bytes: csrDER,
			}),
			Id:             "_unknown",
			Token:          token,
			Namespace:      namespace,
			TokenValidator: tokenValidator,
		}, grpcRetry.WithMax(10), grpcRetry.WithPerRetryTimeout(time.Second*3))
	if err != nil {
		return nil, nil, fmt.Errorf("error from sentry SignCertificate: %w", err)
	}

	keyCS8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	return resp.GetWorkloadCertificate(), pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: keyCS8,
	}), nil
}
