package nicloud

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"

	agpl "github.com/NeuralInverse/cloud/v2/nicloud/appearance"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Get appearance
// @ID get-appearance
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Enterprise
// @Success 200 {object} nicloudsdk.AppearanceConfig
// @Router /api/v2/appearance [get]
func (api *API) appearance(rw http.ResponseWriter, r *http.Request) {
	af := *api.AGPL.AppearanceFetcher.Load()
	cfg, err := af.Fetch(r.Context())
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to fetch appearance config.",
			Detail:  err.Error(),
		})
		return
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, cfg)
}

type appearanceFetcher struct {
	database     database.Store
	supportLinks []nicloudsdk.LinkConfig
	docsURL      string
	coderVersion string
}

func newAppearanceFetcher(store database.Store, links []nicloudsdk.LinkConfig, docsURL, coderVersion string) agpl.Fetcher {
	if docsURL == "" {
		docsURL = nicloudsdk.DefaultDocsURL()
	}
	return &appearanceFetcher{
		database:     store,
		supportLinks: links,
		docsURL:      docsURL,
		coderVersion: coderVersion,
	}
}

func (f *appearanceFetcher) Fetch(ctx context.Context) (nicloudsdk.AppearanceConfig, error) {
	var eg errgroup.Group
	var (
		applicationName         string
		logoURL                 string
		announcementBannersJSON string
	)
	eg.Go(func() (err error) {
		applicationName, err = f.database.GetApplicationName(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get application name: %w", err)
		}
		return nil
	})
	eg.Go(func() (err error) {
		logoURL, err = f.database.GetLogoURL(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get logo url: %w", err)
		}
		return nil
	})
	eg.Go(func() (err error) {
		announcementBannersJSON, err = f.database.GetAnnouncementBanners(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get notification banners: %w", err)
		}
		return nil
	})
	err := eg.Wait()
	if err != nil {
		return nicloudsdk.AppearanceConfig{}, err
	}

	cfg := nicloudsdk.AppearanceConfig{
		ApplicationName:     applicationName,
		LogoURL:             logoURL,
		AnnouncementBanners: []nicloudsdk.BannerConfig{},
		SupportLinks:        nicloudsdk.DefaultSupportLinks(f.docsURL),
		DocsURL:             f.docsURL,
	}

	if announcementBannersJSON != "" {
		err = json.Unmarshal([]byte(announcementBannersJSON), &cfg.AnnouncementBanners)
		if err != nil {
			return nicloudsdk.AppearanceConfig{}, xerrors.Errorf(
				"unmarshal announcement banners json: %w, raw: %s", err, announcementBannersJSON,
			)
		}

		// Redundant, but improves compatibility with slightly mismatched agent versions.
		// Maybe we can remove this after a grace period? -Kayla, May 6th 2024
		if len(cfg.AnnouncementBanners) > 0 {
			cfg.ServiceBanner = cfg.AnnouncementBanners[0]
		}
	}
	if len(f.supportLinks) > 0 {
		cfg.SupportLinks = f.supportLinks
	}

	return cfg, nil
}

func validateHexColor(color string) error {
	if len(color) != 7 {
		return xerrors.New("expected # prefix and 6 characters")
	}
	if color[0] != '#' {
		return xerrors.New("no # prefix")
	}
	_, err := hex.DecodeString(color[1:])
	return err
}

// @Summary Update appearance
// @ID update-appearance
// @Security Neural Inverse CloudSessionToken
// @Accept json
// @Produce json
// @Tags Enterprise
// @Param request body nicloudsdk.UpdateAppearanceConfig true "Update appearance request"
// @Success 200 {object} nicloudsdk.UpdateAppearanceConfig
// @Router /api/v2/appearance [put]
func (api *API) putAppearance(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !api.Authorize(r, policy.ActionUpdate, rbac.ResourceDeploymentConfig) {
		httpapi.Write(ctx, rw, http.StatusForbidden, nicloudsdk.Response{
			Message: "Insufficient permissions to update appearance",
		})
		return
	}

	var appearance nicloudsdk.UpdateAppearanceConfig
	if !httpapi.Read(ctx, rw, r, &appearance) {
		return
	}

	for _, banner := range appearance.AnnouncementBanners {
		if err := validateHexColor(banner.BackgroundColor); err != nil {
			httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
				Message: fmt.Sprintf("Invalid color format: %q", banner.BackgroundColor),
				Detail:  err.Error(),
			})
			return
		}
	}

	if appearance.AnnouncementBanners == nil {
		appearance.AnnouncementBanners = []nicloudsdk.BannerConfig{}
	}
	announcementBannersJSON, err := json.Marshal(appearance.AnnouncementBanners)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Unable to marshal announcement banners",
			Detail:  err.Error(),
		})
		return
	}

	err = api.Database.UpsertAnnouncementBanners(ctx, string(announcementBannersJSON))
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Unable to set announcement banners",
			Detail:  err.Error(),
		})
		return
	}

	err = api.Database.UpsertApplicationName(ctx, appearance.ApplicationName)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Unable to set application name",
			Detail:  err.Error(),
		})
		return
	}

	err = api.Database.UpsertLogoURL(ctx, appearance.LogoURL)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Unable to set logo URL",
			Detail:  err.Error(),
		})
		return
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, appearance)
}
