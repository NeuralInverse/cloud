package nicloud_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestTokenCreation_ScopeValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		scope   nicloudsdk.APIKeyScope
		wantErr bool
	}{
		{name: "AllowsPublicLowLevelScope", scope: "workspace:read", wantErr: false},
		{name: "RejectsInternalOnlyScope", scope: "debug_info:read", wantErr: true},
		{name: "AllowsLegacyScopes", scope: "application_connect", wantErr: false},
		{name: "AllowsLegacyScopes2", scope: "all", wantErr: false},
		{name: "AllowsCanonicalSpecialScope", scope: "coder:all", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := nicloudtest.New(t, nil)
			_ = nicloudtest.CreateFirstUser(t, client)

			ctx, cancel := context.WithTimeout(t.Context(), testutil.WaitShort)
			defer cancel()

			resp, err := client.CreateToken(ctx, nicloudsdk.Me, nicloudsdk.CreateTokenRequest{Scope: tc.scope})
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, resp.Key)

			// Fetch and verify the stored scopes match expectation.
			keys, err := client.Tokens(ctx, nicloudsdk.Me, nicloudsdk.TokensFilter{})
			require.NoError(t, err)
			require.Len(t, keys, 1)

			// Normalize legacy singular scopes to canonical coder:* values.
			expected := tc.scope
			switch tc.scope {
			case nicloudsdk.APIKeyScopeAll:
				expected = nicloudsdk.APIKeyScopeNIAll
			case nicloudsdk.APIKeyScopeApplicationConnect:
				expected = nicloudsdk.APIKeyScopeNIApplicationConnect
			}

			require.Contains(t, keys[0].Scopes, expected)
		})
	}
}
