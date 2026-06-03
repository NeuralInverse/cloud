package wsproxy

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/cryptokeys"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/wsproxy/wsproxysdk"
)

var _ cryptokeys.Fetcher = &ProxyFetcher{}

type ProxyFetcher struct {
	Client *wsproxysdk.Client
}

func (p *ProxyFetcher) Fetch(ctx context.Context, feature nicloudsdk.CryptoKeyFeature) ([]nicloudsdk.CryptoKey, error) {
	keys, err := p.Client.CryptoKeys(ctx, feature)
	if err != nil {
		return nil, xerrors.Errorf("crypto keys: %w", err)
	}
	return keys.CryptoKeys, nil
}
