package nicloud_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestPostUserSecret(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, nil)
	_ = nicloudtest.CreateFirstUser(t, client)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		secret, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:        "github-token",
			Value:       "ghp_xxxxxxxxxxxx",
			Description: "Personal GitHub PAT",
			EnvName:     "GITHUB_TOKEN",
			FilePath:    "~/.github-token",
		})
		require.NoError(t, err)
		assert.Equal(t, "github-token", secret.Name)
		assert.Equal(t, "Personal GitHub PAT", secret.Description)
		assert.Equal(t, "GITHUB_TOKEN", secret.EnvName)
		assert.Equal(t, "~/.github-token", secret.FilePath)
		assert.NotZero(t, secret.ID)
		assert.NotZero(t, secret.CreatedAt)
	})

	t.Run("MissingName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Value: "some-value",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "name", "required")
	})

	t.Run("MissingValue", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name: "missing-value-secret",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "value", "required")
	})

	t.Run("InvalidName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "foo/bar",
			Value: "some-value",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "name", "must not contain")
	})

	t.Run("WhitespaceName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  " github",
			Value: "some-value",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "name", "whitespace")
	})

	t.Run("DuplicateName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "dup-secret",
			Value: "value1",
		})
		require.NoError(t, err)

		_, err = client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "dup-secret",
			Value: "value2",
		})
		requireSecretValidationEqualsError(t, err, http.StatusConflict, "name", "name already in use")
	})

	t.Run("DuplicateEnvName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "env-dup-1",
			Value:   "value1",
			EnvName: "DUPLICATE_ENV",
		})
		require.NoError(t, err)

		_, err = client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "env-dup-2",
			Value:   "value2",
			EnvName: "DUPLICATE_ENV",
		})
		requireSecretValidationEqualsError(t, err, http.StatusConflict, "env_name", "environment variable already in use")
	})

	t.Run("DuplicateFilePath", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:     "fp-dup-1",
			Value:    "value1",
			FilePath: "/tmp/dup-file",
		})
		require.NoError(t, err)

		_, err = client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:     "fp-dup-2",
			Value:    "value2",
			FilePath: "/tmp/dup-file",
		})
		requireSecretValidationEqualsError(t, err, http.StatusConflict, "file_path", "file path already in use")
	})

	t.Run("InvalidEnvName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "invalid-env-secret",
			Value:   "value",
			EnvName: "1INVALID",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "env_name", "must start")
	})

	t.Run("ReservedEnvName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "reserved-env-secret",
			Value:   "value",
			EnvName: "PATH",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "env_name", "reserved")
	})

	t.Run("CoderPrefixEnvName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "coder-prefix-secret",
			Value:   "value",
			EnvName: "NEURALINVERSE_AGENT_TOKEN",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "env_name", "NEURALINVERSE_")
	})

	t.Run("InvalidFilePath", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:     "bad-path-secret",
			Value:    "value",
			FilePath: "relative/path",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "file_path", "must start")
	})

	t.Run("NullByteInValue", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "null-byte-secret",
			Value: "before\x00after",
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "value", "null bytes")
	})

	t.Run("OversizedValue", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "oversized-secret",
			Value: strings.Repeat("a", nicloudsdk.MaxUserSecretValueBytes+1),
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "value", "must not exceed")
	})
}

func TestGetUserSecrets(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, nil)
	_ = nicloudtest.CreateFirstUser(t, client)

	// Verify no secrets exist on a fresh user.
	ctx := testutil.Context(t, testutil.WaitMedium)
	secrets, err := client.UserSecrets(ctx, nicloudsdk.Me)
	require.NoError(t, err)
	assert.Empty(t, secrets)

	t.Run("WithSecrets", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "list-secret-a",
			Value: "value-a",
		})
		require.NoError(t, err)

		_, err = client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "list-secret-b",
			Value: "value-b",
		})
		require.NoError(t, err)

		secrets, err := client.UserSecrets(ctx, nicloudsdk.Me)
		require.NoError(t, err)
		require.Len(t, secrets, 2)
		// Sorted by name.
		assert.Equal(t, "list-secret-a", secrets[0].Name)
		assert.Equal(t, "list-secret-b", secrets[1].Name)
	})
}

func TestGetUserSecret(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, nil)
	_ = nicloudtest.CreateFirstUser(t, client)

	t.Run("Found", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		created, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "get-found-secret",
			Value:   "my-value",
			EnvName: "GET_FOUND_SECRET",
		})
		require.NoError(t, err)

		got, err := client.UserSecretByName(ctx, nicloudsdk.Me, "get-found-secret")
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "get-found-secret", got.Name)
		assert.Equal(t, "GET_FOUND_SECRET", got.EnvName)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.UserSecretByName(ctx, nicloudsdk.Me, "nonexistent")
		require.Error(t, err)
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusNotFound, sdkErr.StatusCode())
	})
}

func TestPatchUserSecret(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, nil)
	_ = nicloudtest.CreateFirstUser(t, client)

	t.Run("UpdateDescription", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:        "patch-desc-secret",
			Value:       "my-value",
			Description: "original",
			EnvName:     "PATCH_DESC_ENV",
		})
		require.NoError(t, err)

		newDesc := "updated"
		updated, err := client.UpdateUserSecret(ctx, nicloudsdk.Me, "patch-desc-secret", nicloudsdk.UpdateUserSecretRequest{
			Description: &newDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, "updated", updated.Description)
		// Other fields unchanged.
		assert.Equal(t, "PATCH_DESC_ENV", updated.EnvName)
	})

	t.Run("NoFields", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "patch-nofields-secret",
			Value: "my-value",
		})
		require.NoError(t, err)

		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, "patch-nofields-secret", nicloudsdk.UpdateUserSecretRequest{})
		require.Error(t, err)
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusBadRequest, sdkErr.StatusCode())
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		newVal := "new-value"
		_, err := client.UpdateUserSecret(ctx, nicloudsdk.Me, "nonexistent", nicloudsdk.UpdateUserSecretRequest{
			Value: &newVal,
		})
		require.Error(t, err)
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusNotFound, sdkErr.StatusCode())
	})

	t.Run("ConflictEnvName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "conflict-env-1",
			Value:   "value1",
			EnvName: "CONFLICT_TAKEN_ENV",
		})
		require.NoError(t, err)

		_, err = client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "conflict-env-2",
			Value: "value2",
		})
		require.NoError(t, err)

		taken := "CONFLICT_TAKEN_ENV"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, "conflict-env-2", nicloudsdk.UpdateUserSecretRequest{
			EnvName: &taken,
		})
		requireSecretValidationEqualsError(t, err, http.StatusConflict, "env_name", "environment variable already in use")
	})

	t.Run("ConflictFilePath", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:     "conflict-fp-1",
			Value:    "value1",
			FilePath: "/tmp/conflict-taken",
		})
		require.NoError(t, err)

		_, err = client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "conflict-fp-2",
			Value: "value2",
		})
		require.NoError(t, err)

		taken := "/tmp/conflict-taken"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, "conflict-fp-2", nicloudsdk.UpdateUserSecretRequest{
			FilePath: &taken,
		})
		requireSecretValidationEqualsError(t, err, http.StatusConflict, "file_path", "file path already in use")
	})

	t.Run("InvalidEnvName", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "patch-invalid-env",
			Value: "good-value",
		})
		require.NoError(t, err)

		badEnvName := "1INVALID"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, "patch-invalid-env", nicloudsdk.UpdateUserSecretRequest{
			EnvName: &badEnvName,
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "env_name", "must start")
	})

	t.Run("InvalidFilePath", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "patch-invalid-file-path",
			Value: "good-value",
		})
		require.NoError(t, err)

		badFilePath := "relative/path"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, "patch-invalid-file-path", nicloudsdk.UpdateUserSecretRequest{
			FilePath: &badFilePath,
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "file_path", "must start")
	})

	t.Run("InvalidValue", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "patch-invalid-val",
			Value: "good-value",
		})
		require.NoError(t, err)

		badVal := "before\x00after"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, "patch-invalid-val", nicloudsdk.UpdateUserSecretRequest{
			Value: &badVal,
		})
		requireSecretValidationContainsError(t, err, http.StatusBadRequest, "value", "null bytes")
	})
}

func requireSecretValidationContainsError(t *testing.T, err error, status int, field string, detailContains string) {
	t.Helper()
	validation := requireSecretValidation(t, err, status, field)
	assert.Contains(t, validation.Detail, detailContains)
}

func requireSecretValidationEqualsError(t *testing.T, err error, status int, field string, detail string) {
	t.Helper()
	validation := requireSecretValidation(t, err, status, field)
	assert.Equal(t, detail, validation.Detail)
}

func requireSecretValidation(t *testing.T, err error, status int, field string) nicloudsdk.ValidationError {
	t.Helper()

	require.Error(t, err)
	var sdkErr *nicloudsdk.Error
	require.ErrorAs(t, err, &sdkErr)
	assert.Equal(t, status, sdkErr.StatusCode())
	for _, validation := range sdkErr.Validations {
		if validation.Field == field {
			return validation
		}
	}
	require.Failf(t, "missing validation", "field %q not found in %#v", field, sdkErr.Validations)
	return nicloudsdk.ValidationError{}
}

// TestUserSecretLimits exercises the per-user count and byte caps
// enforced by enforce_user_secrets_per_user_limits across both POST
// (creating a new secret) and PATCH (updating an existing one).
// Each subtest spins up its own server so it can burn the budget
// without affecting other tests.
//
// Each subtest checks three things per cap:
//
//   - POST past the cap is rejected with a 400.
//   - PATCH of an existing row at the cap is accepted; the trigger
//     uses FILTER (WHERE id IS DISTINCT FROM NEW.id) so an UPDATE
//     does not double-count its own row.
//   - A different user's budget is independent; the trigger groups
//     by user_id.
func TestUserSecretLimits(t *testing.T) {
	t.Parallel()

	t.Run("CountLimit", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitLong)

		client := nicloudtest.New(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		otherClient, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)

		// Fill the count budget exactly to the cap.
		var firstSecret nicloudsdk.UserSecret
		for i := 0; i < nicloudsdk.MaxUserSecretsPerUserCount; i++ {
			s, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
				Name:  fmt.Sprintf("count-limit-%03d", i),
				Value: "x",
			})
			require.NoError(t, err)
			if i == 0 {
				firstSecret = s
			}
		}

		// POST: the 51st secret is rejected.
		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "one-too-many",
			Value: "x",
		})
		requireSecretAPIError(t, err, http.StatusBadRequest, "at most")

		// PATCH at the cap: changing the description must succeed.
		// Without the FILTER clause the trigger would re-count
		// firstSecret and reject this UPDATE.
		newDescription := "renamed"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, firstSecret.Name, nicloudsdk.UpdateUserSecretRequest{
			Description: &newDescription,
		})
		require.NoError(t, err)

		// Other-user isolation: the second user's budget is independent.
		_, err = otherClient.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "other-user-secret",
			Value: "x",
		})
		require.NoError(t, err)
	})

	t.Run("TotalBytesLimit", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitLong)

		client := nicloudtest.New(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		otherClient, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)

		// Pre-fill the total-bytes budget exactly to the cap using
		// max-sized file-only secrets (which don't count against env
		// bytes).
		big := strings.Repeat("a", nicloudsdk.MaxUserSecretValueBytes)
		numBig := nicloudsdk.MaxUserSecretsTotalValueBytes / nicloudsdk.MaxUserSecretValueBytes
		remainder := nicloudsdk.MaxUserSecretsTotalValueBytes % nicloudsdk.MaxUserSecretValueBytes
		var firstSecret nicloudsdk.UserSecret
		for i := 0; i < numBig; i++ {
			s, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
				Name:     fmt.Sprintf("big-%03d", i),
				Value:    big,
				FilePath: fmt.Sprintf("/tmp/big-%03d", i),
			})
			require.NoError(t, err)
			if i == 0 {
				firstSecret = s
			}
		}
		if remainder > 0 {
			_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
				Name:     "big-pad",
				Value:    strings.Repeat("a", remainder),
				FilePath: "/tmp/big-pad",
			})
			require.NoError(t, err)
		}

		// POST: one more byte pushes past the total budget.
		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:     "overflow",
			Value:    "x",
			FilePath: "/tmp/overflow",
		})
		requireSecretAPIError(t, err, http.StatusBadRequest, "per-user budget")

		// PATCH at the cap: rewriting the existing row with a value
		// of the same size must succeed. The FILTER clause excludes
		// firstSecret's old bytes from the aggregate so the trigger
		// computes (cap - old) + new = cap, not cap + new.
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, firstSecret.Name, nicloudsdk.UpdateUserSecretRequest{
			Value: &big,
		})
		require.NoError(t, err)

		// Other-user isolation: a fresh user can fill their own
		// total-bytes budget without interference.
		for i := 0; i < numBig; i++ {
			_, err := otherClient.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
				Name:     fmt.Sprintf("other-big-%03d", i),
				Value:    big,
				FilePath: fmt.Sprintf("/tmp/other-big-%03d", i),
			})
			require.NoError(t, err)
		}
		if remainder > 0 {
			_, err := otherClient.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
				Name:     "other-big-pad",
				Value:    strings.Repeat("a", remainder),
				FilePath: "/tmp/other-big-pad",
			})
			require.NoError(t, err)
		}
	})

	t.Run("EnvBytesLimit", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitLong)

		client := nicloudtest.New(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		otherClient, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)

		// One env-injected secret consumes nearly the whole env budget.
		envBig, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "env-big",
			Value:   strings.Repeat("a", nicloudsdk.MaxUserSecretValueBytes-16),
			EnvName: "ENV_BIG",
		})
		require.NoError(t, err)

		// POST: another env-injected secret pushes us over the env budget.
		_, err = client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "env-overflow",
			Value:   strings.Repeat("a", 1024),
			EnvName: "ENV_OVERFLOW",
		})
		requireSecretAPIError(t, err, http.StatusBadRequest, "env_name")

		// A same-sized value used purely as a file is fine because
		// file_path secrets do not count against the env budget.
		fileOK, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:     "file-ok",
			Value:    strings.Repeat("a", 1024),
			FilePath: "/tmp/file-ok",
		})
		require.NoError(t, err)

		// PATCH at the cap: updating envBig's description must
		// succeed. Without FILTER, the trigger would re-add envBig's
		// 24 KiB to itself and reject the UPDATE.
		newDescription := "renamed"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, envBig.Name, nicloudsdk.UpdateUserSecretRequest{
			Description: &newDescription,
		})
		require.NoError(t, err)

		// PATCH a file_path secret to env mode: moves its 1 KiB into
		// the env budget, which already holds envBig's 24 KiB - 16.
		// new_env_bytes = 24560 + 1024 = 25584 > 24576, rejected.
		envName := "ENV_LATE"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, fileOK.Name, nicloudsdk.UpdateUserSecretRequest{
			EnvName: &envName,
		})
		requireSecretAPIError(t, err, http.StatusBadRequest, "env_name")

		// Other-user isolation: a fresh user can create their own
		// near-cap env secret.
		_, err = otherClient.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    "other-env-big",
			Value:   strings.Repeat("a", nicloudsdk.MaxUserSecretValueBytes-16),
			EnvName: "OTHER_ENV_BIG",
		})
		require.NoError(t, err)
	})
}

// requireSecretAPIError asserts a non-validation user-facing error.
// Used for trigger-driven failures (per-user limits) whose responses
// are plain nicloudsdk.Response without ValidationError entries.
func requireSecretAPIError(t *testing.T, err error, status int, detailContains string) {
	t.Helper()
	require.Error(t, err)
	var sdkErr *nicloudsdk.Error
	require.ErrorAs(t, err, &sdkErr)
	assert.Equal(t, status, sdkErr.StatusCode())
	combined := sdkErr.Message + " " + sdkErr.Response.Detail
	assert.Containsf(t, combined, detailContains,
		"expected response to contain %q; got Message=%q Detail=%q",
		detailContains, sdkErr.Message, sdkErr.Response.Detail)
}

func TestDeleteUserSecret(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, nil)
	_ = nicloudtest.CreateFirstUser(t, client)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  "delete-me-secret",
			Value: "my-value",
		})
		require.NoError(t, err)

		err = client.DeleteUserSecret(ctx, nicloudsdk.Me, "delete-me-secret")
		require.NoError(t, err)

		// Verify it's gone.
		_, err = client.UserSecretByName(ctx, nicloudsdk.Me, "delete-me-secret")
		require.Error(t, err)
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusNotFound, sdkErr.StatusCode())
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)

		err := client.DeleteUserSecret(ctx, nicloudsdk.Me, "nonexistent")
		require.Error(t, err)
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusNotFound, sdkErr.StatusCode())
	})
}
