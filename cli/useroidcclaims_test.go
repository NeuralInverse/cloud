package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest/oidctest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestUserOIDCClaims(t *testing.T) {
	t.Parallel()

	newOIDCTest := func(t *testing.T) (*oidctest.FakeIDP, *nicloudsdk.Client) {
		t.Helper()

		fake := oidctest.NewFakeIDP(t,
			oidctest.WithServing(),
		)
		cfg := fake.OIDCConfig(t, nil, func(cfg *nicloud.OIDCConfig) {
			cfg.AllowSignups = true
		})
		ownerClient := nicloudtest.New(t, &nicloudtest.Options{
			OIDCConfig: cfg,
		})
		return fake, ownerClient
	}

	t.Run("OwnClaims", func(t *testing.T) {
		t.Parallel()

		fake, ownerClient := newOIDCTest(t)
		claims := jwt.MapClaims{
			"email":          "alice@cloud.neuralinverse.com",
			"email_verified": true,
			"sub":            uuid.NewString(),
			"groups":         []string{"admin", "eng"},
		}
		userClient, loginResp := fake.Login(t, ownerClient, claims)
		defer loginResp.Body.Close()

		inv, root := clitest.New(t, "users", "oidc-claims", "-o", "json")
		clitest.SetupConfig(t, userClient, root)

		buf := bytes.NewBuffer(nil)
		inv.Stdout = buf
		err := inv.WithContext(testutil.Context(t, testutil.WaitMedium)).Run()
		require.NoError(t, err)

		var resp nicloudsdk.OIDCClaimsResponse
		err = json.Unmarshal(buf.Bytes(), &resp)
		require.NoError(t, err, "unmarshal JSON output")
		require.NotEmpty(t, resp.Claims, "claims should not be empty")
		assert.Equal(t, "alice@cloud.neuralinverse.com", resp.Claims["email"])
	})

	t.Run("Table", func(t *testing.T) {
		t.Parallel()

		fake, ownerClient := newOIDCTest(t)
		claims := jwt.MapClaims{
			"email":          "bob@cloud.neuralinverse.com",
			"email_verified": true,
			"sub":            uuid.NewString(),
		}
		userClient, loginResp := fake.Login(t, ownerClient, claims)
		defer loginResp.Body.Close()

		inv, root := clitest.New(t, "users", "oidc-claims")
		clitest.SetupConfig(t, userClient, root)

		buf := bytes.NewBuffer(nil)
		inv.Stdout = buf
		err := inv.WithContext(testutil.Context(t, testutil.WaitMedium)).Run()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "email")
		require.Contains(t, output, "bob@cloud.neuralinverse.com")
	})

	t.Run("NotOIDCUser", func(t *testing.T) {
		t.Parallel()

		client := nicloudtest.New(t, nil)
		_ = nicloudtest.CreateFirstUser(t, client)

		inv, root := clitest.New(t, "users", "oidc-claims")
		clitest.SetupConfig(t, client, root)

		err := inv.WithContext(testutil.Context(t, testutil.WaitMedium)).Run()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not an OIDC user")
	})

	// Verify that two different OIDC users each only see their own
	// claims. The endpoint has no user parameter, so there is no way
	// to request another user's claims by design.
	t.Run("OnlyOwnClaims", func(t *testing.T) {
		t.Parallel()

		aliceFake, aliceOwnerClient := newOIDCTest(t)
		aliceClaims := jwt.MapClaims{
			"email":          "alice-isolation@cloud.neuralinverse.com",
			"email_verified": true,
			"sub":            uuid.NewString(),
		}
		aliceClient, aliceLoginResp := aliceFake.Login(t, aliceOwnerClient, aliceClaims)
		defer aliceLoginResp.Body.Close()

		bobFake, bobOwnerClient := newOIDCTest(t)
		bobClaims := jwt.MapClaims{
			"email":          "bob-isolation@cloud.neuralinverse.com",
			"email_verified": true,
			"sub":            uuid.NewString(),
		}
		bobClient, bobLoginResp := bobFake.Login(t, bobOwnerClient, bobClaims)
		defer bobLoginResp.Body.Close()

		ctx := testutil.Context(t, testutil.WaitMedium)

		// Alice sees her own claims.
		aliceResp, err := aliceClient.UserOIDCClaims(ctx)
		require.NoError(t, err)
		assert.Equal(t, "alice-isolation@cloud.neuralinverse.com", aliceResp.Claims["email"])

		// Bob sees his own claims.
		bobResp, err := bobClient.UserOIDCClaims(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bob-isolation@cloud.neuralinverse.com", bobResp.Claims["email"])
	})

	t.Run("ClaimsNeverNull", func(t *testing.T) {
		t.Parallel()

		fake, ownerClient := newOIDCTest(t)
		// Use minimal claims — just enough for OIDC login.
		claims := jwt.MapClaims{
			"email":          "minimal@cloud.neuralinverse.com",
			"email_verified": true,
			"sub":            uuid.NewString(),
		}
		userClient, loginResp := fake.Login(t, ownerClient, claims)
		defer loginResp.Body.Close()

		ctx := testutil.Context(t, testutil.WaitMedium)
		resp, err := userClient.UserOIDCClaims(ctx)
		require.NoError(t, err)
		require.NotNil(t, resp.Claims, "claims should never be nil, expected empty map")
	})
}
