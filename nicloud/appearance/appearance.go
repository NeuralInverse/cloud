package appearance

import (
	"context"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type Fetcher interface {
	Fetch(ctx context.Context) (nicloudsdk.AppearanceConfig, error)
}

type AGPLFetcher struct {
	docsURL string
}

func (f AGPLFetcher) Fetch(context.Context) (nicloudsdk.AppearanceConfig, error) {
	return nicloudsdk.AppearanceConfig{
		AnnouncementBanners: []nicloudsdk.BannerConfig{},
		SupportLinks:        nicloudsdk.DefaultSupportLinks(f.docsURL),
		DocsURL:             f.docsURL,
	}, nil
}

func NewDefaultFetcher(docsURL string) Fetcher {
	if docsURL == "" {
		docsURL = nicloudsdk.DefaultDocsURL()
	}
	return &AGPLFetcher{
		docsURL: docsURL,
	}
}
