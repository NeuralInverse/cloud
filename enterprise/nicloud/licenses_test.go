package nicloud_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtime"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestPostLicense(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		respLic := nicloudenttest.AddLicense(t, client, nicloudenttest.LicenseOptions{
			AccountType: license.AccountTypeSalesforce,
			AccountID:   "testing",
			Features: license.Features{
				nicloudsdk.FeatureAuditLog: 1,
			},
		})
		assert.GreaterOrEqual(t, respLic.ID, int32(0))
		// just a couple spot checks for sanity
		assert.Equal(t, "testing", respLic.Claims["account_id"])
		features, err := respLic.FeaturesClaims()
		require.NoError(t, err)
		assert.EqualValues(t, 1, features[nicloudsdk.FeatureAuditLog])
	})

	t.Run("InvalidDeploymentID", func(t *testing.T) {
		t.Parallel()
		// The generated deployment will start out with a different deployment ID.
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		license := nicloudenttest.GenerateLicense(t, nicloudenttest.LicenseOptions{
			DeploymentIDs: []string{uuid.NewString()},
		})
		_, err := client.AddLicense(context.Background(), nicloudsdk.AddLicenseRequest{
			License: license,
		})
		errResp := &nicloudsdk.Error{}
		require.ErrorAs(t, err, &errResp)
		require.Equal(t, http.StatusBadRequest, errResp.StatusCode())
		require.Contains(t, errResp.Message, "License cannot be used on this deployment!")
	})

	t.Run("InvalidAccountID", func(t *testing.T) {
		t.Parallel()
		// The generated deployment will start out with a different deployment ID.
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		license := nicloudenttest.GenerateLicense(t, nicloudenttest.LicenseOptions{
			AllowEmpty: true,
			AccountID:  "",
		})
		_, err := client.AddLicense(context.Background(), nicloudsdk.AddLicenseRequest{
			License: license,
		})
		errResp := &nicloudsdk.Error{}
		require.ErrorAs(t, err, &errResp)
		require.Equal(t, http.StatusBadRequest, errResp.StatusCode())
		require.Contains(t, errResp.Message, "Invalid license")
	})

	t.Run("InvalidAccountType", func(t *testing.T) {
		t.Parallel()
		// The generated deployment will start out with a different deployment ID.
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		license := nicloudenttest.GenerateLicense(t, nicloudenttest.LicenseOptions{
			AllowEmpty:  true,
			AccountType: "",
		})
		_, err := client.AddLicense(context.Background(), nicloudsdk.AddLicenseRequest{
			License: license,
		})
		errResp := &nicloudsdk.Error{}
		require.ErrorAs(t, err, &errResp)
		require.Equal(t, http.StatusBadRequest, errResp.StatusCode())
		require.Contains(t, errResp.Message, "Invalid license")
	})

	t.Run("InvalidLicenseExpires", func(t *testing.T) {
		t.Parallel()
		// The generated deployment will start out with a different deployment ID.
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		license := nicloudenttest.GenerateLicense(t, nicloudenttest.LicenseOptions{
			GraceAt: time.Unix(99999999999, 0),
		})
		_, err := client.AddLicense(context.Background(), nicloudsdk.AddLicenseRequest{
			License: license,
		})
		errResp := &nicloudsdk.Error{}
		require.ErrorAs(t, err, &errResp)
		require.Equal(t, http.StatusBadRequest, errResp.StatusCode())
		require.Contains(t, errResp.Message, "Invalid license")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		client.SetSessionToken("")
		_, err := client.AddLicense(context.Background(), nicloudsdk.AddLicenseRequest{
			License: "content",
		})
		errResp := &nicloudsdk.Error{}
		if xerrors.As(err, &errResp) {
			assert.Equal(t, 401, errResp.StatusCode())
		} else {
			t.Error("expected to get error status 401")
		}
	})

	t.Run("Corrupted", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		nicloudenttest.AddLicense(t, client, nicloudenttest.LicenseOptions{})
		_, err := client.AddLicense(context.Background(), nicloudsdk.AddLicenseRequest{
			License: "invalid",
		})
		errResp := &nicloudsdk.Error{}
		if xerrors.As(err, &errResp) {
			assert.Equal(t, 400, errResp.StatusCode())
		} else {
			t.Error("expected to get error status 400")
		}
	})

	// Test a license that isn't yet valid, but will be in the future.  We should allow this so that
	// operators can upload a license ahead of time.
	t.Run("NotYet", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		respLic := nicloudenttest.AddLicense(t, client, nicloudenttest.LicenseOptions{
			AccountType: license.AccountTypeSalesforce,
			AccountID:   "testing",
			Features: license.Features{
				nicloudsdk.FeatureAuditLog: 1,
			},
			NotBefore: dbtime.Now().Add(time.Hour),
			GraceAt:   time.Now().Add(2 * time.Hour),
			ExpiresAt: time.Now().Add(3 * time.Hour),
		})
		assert.GreaterOrEqual(t, respLic.ID, int32(0))
		// just a couple spot checks for sanity
		assert.Equal(t, "testing", respLic.Claims["account_id"])
		features, err := respLic.FeaturesClaims()
		require.NoError(t, err)
		assert.EqualValues(t, 1, features[nicloudsdk.FeatureAuditLog])
	})

	// Test we still reject a license that isn't valid yet, but has other issues (e.g. expired
	// before it starts).
	t.Run("NotEver", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		lic := nicloudenttest.GenerateLicense(t, nicloudenttest.LicenseOptions{
			AccountType: license.AccountTypeSalesforce,
			AccountID:   "testing",
			Features: license.Features{
				nicloudsdk.FeatureAuditLog: 1,
			},
			NotBefore: dbtime.Now().Add(time.Hour),
			GraceAt:   time.Now().Add(2 * time.Hour),
			ExpiresAt: time.Now().Add(-time.Hour),
		})
		_, err := client.AddLicense(context.Background(), nicloudsdk.AddLicenseRequest{
			License: lic,
		})
		errResp := &nicloudsdk.Error{}
		require.ErrorAs(t, err, &errResp)
		require.Equal(t, http.StatusBadRequest, errResp.StatusCode())
		require.Contains(t, errResp.Detail, license.ErrMultipleIssues.Error())
	})
}

func TestGetLicense(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		nicloudenttest.AddLicense(t, client, nicloudenttest.LicenseOptions{
			AccountID: "testing",
			Features: license.Features{
				nicloudsdk.FeatureAuditLog:     1,
				nicloudsdk.FeatureSCIM:         1,
				nicloudsdk.FeatureBrowserOnly:  1,
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		})

		nicloudenttest.AddLicense(t, client, nicloudenttest.LicenseOptions{
			AccountID: "testing2",
			Features: license.Features{
				nicloudsdk.FeatureAuditLog:    1,
				nicloudsdk.FeatureSCIM:        1,
				nicloudsdk.FeatureBrowserOnly: 1,
				nicloudsdk.FeatureUserLimit:   200,
			},
			Trial: true,
		})

		licenses, err := client.Licenses(ctx)
		require.NoError(t, err)
		require.Len(t, licenses, 2)
		assert.Equal(t, int32(1), licenses[0].ID)
		assert.Equal(t, "testing", licenses[0].Claims["account_id"])

		features, err := licenses[0].FeaturesClaims()
		require.NoError(t, err)
		assert.Equal(t, map[nicloudsdk.FeatureName]int64{
			nicloudsdk.FeatureAuditLog:     1,
			nicloudsdk.FeatureSCIM:         1,
			nicloudsdk.FeatureBrowserOnly:  1,
			nicloudsdk.FeatureTemplateRBAC: 1,
		}, features)
		assert.Equal(t, int32(2), licenses[1].ID)
		assert.Equal(t, "testing2", licenses[1].Claims["account_id"])
		assert.Equal(t, true, licenses[1].Claims["trial"])

		features, err = licenses[1].FeaturesClaims()
		require.NoError(t, err)
		assert.Equal(t, map[nicloudsdk.FeatureName]int64{
			nicloudsdk.FeatureUserLimit:   200,
			nicloudsdk.FeatureAuditLog:    1,
			nicloudsdk.FeatureSCIM:        1,
			nicloudsdk.FeatureBrowserOnly: 1,
		}, features)
	})
}

func TestDeleteLicense(t *testing.T) {
	t.Parallel()
	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		err := client.DeleteLicense(ctx, 1)
		errResp := &nicloudsdk.Error{}
		if xerrors.As(err, &errResp) {
			assert.Equal(t, 404, errResp.StatusCode())
		} else {
			t.Error("expected to get error status 404")
		}
	})

	t.Run("BadID", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		//nolint:gocritic // RBAC is irrelevant here.
		resp, err := client.Request(ctx, http.MethodDelete, "/api/v2/licenses/drivers", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		nicloudenttest.AddLicense(t, client, nicloudenttest.LicenseOptions{
			AccountID: "testing",
			Features: license.Features{
				nicloudsdk.FeatureAuditLog: 1,
			},
		})
		nicloudenttest.AddLicense(t, client, nicloudenttest.LicenseOptions{
			AccountID: "testing2",
			Features: license.Features{
				nicloudsdk.FeatureAuditLog:  1,
				nicloudsdk.FeatureUserLimit: 200,
			},
		})

		licenses, err := client.Licenses(ctx)
		require.NoError(t, err)
		assert.Len(t, licenses, 2)
		for _, l := range licenses {
			err = client.DeleteLicense(ctx, l.ID)
			require.NoError(t, err)
		}
		licenses, err = client.Licenses(ctx)
		require.NoError(t, err)
		assert.Len(t, licenses, 0)
	})
}
