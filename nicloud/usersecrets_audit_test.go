package nicloud_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

//nolint:paralleltest,tparallel // Subtests share one nicloudtest.New server and run sequentially.
func TestUserSecretAudit(t *testing.T) {
	t.Parallel()

	auditor := audit.NewMock()
	client := nicloudtest.New(t, &nicloudtest.Options{Auditor: auditor})
	_ = nicloudtest.CreateFirstUser(t, client)
	ctx := testutil.Context(t, testutil.WaitMedium)

	genSecretName := func(t *testing.T) string {
		// Use test name derived secret names so subtests cannot
		// collide in the shared user's secret namespace.
		return strings.ReplaceAll(t.Name(), "/", "-")
	}

	t.Run("CreateEmitsLog", func(t *testing.T) {
		auditor.ResetLogs()

		secret, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  genSecretName(t),
			Value: "ghp_xxxxxxxxxxxx",
		})
		require.NoError(t, err)

		logs := auditor.AuditLogs()
		require.Len(t, logs, 1)
		assert.Equal(t, database.AuditActionCreate, logs[0].Action)
		assert.Equal(t, secret.ID, logs[0].ResourceID)
		assert.Equal(t, secret.Name, logs[0].ResourceTarget)
		assert.EqualValues(t, http.StatusCreated, logs[0].StatusCode)
	})

	t.Run("UpdateEmitsLog", func(t *testing.T) {
		auditor.ResetLogs()

		secret, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  genSecretName(t),
			Value: "old",
		})
		require.NoError(t, err)

		newDescription := "rotated"
		newValue := "new-value"
		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, secret.Name, nicloudsdk.UpdateUserSecretRequest{
			Description: &newDescription,
			Value:       &newValue,
		})
		require.NoError(t, err)

		logs := auditor.AuditLogs()
		require.Len(t, logs, 2)
		assert.Equal(t, database.AuditActionCreate, logs[0].Action)
		assert.Equal(t, database.AuditActionWrite, logs[1].Action)
		assert.Equal(t, secret.ID, logs[1].ResourceID)
		assert.Equal(t, secret.Name, logs[1].ResourceTarget)
		assert.EqualValues(t, http.StatusOK, logs[1].StatusCode)
	})

	t.Run("DeleteEmitsLog", func(t *testing.T) {
		auditor.ResetLogs()

		secret, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  genSecretName(t),
			Value: "value",
		})
		require.NoError(t, err)

		require.NoError(t, client.DeleteUserSecret(ctx, nicloudsdk.Me, secret.Name))

		logs := auditor.AuditLogs()
		require.Len(t, logs, 2)
		assert.Equal(t, database.AuditActionCreate, logs[0].Action)
		assert.Equal(t, database.AuditActionDelete, logs[1].Action)
		assert.Equal(t, secret.ID, logs[1].ResourceID)
		assert.Equal(t, secret.Name, logs[1].ResourceTarget)
		assert.EqualValues(t, http.StatusNoContent, logs[1].StatusCode)
	})

	t.Run("DeleteOfMissingWritesNoLog", func(t *testing.T) {
		auditor.ResetLogs()

		err := client.DeleteUserSecret(ctx, nicloudsdk.Me, "does-not-exist")
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusNotFound, sdkErr.StatusCode())

		require.Empty(t, auditor.AuditLogs())
	})

	t.Run("UpdateOfMissingWritesNoLog", func(t *testing.T) {
		auditor.ResetLogs()

		desc := "anything"
		_, err := client.UpdateUserSecret(ctx, nicloudsdk.Me, "does-not-exist", nicloudsdk.UpdateUserSecretRequest{
			Description: &desc,
		})
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusNotFound, sdkErr.StatusCode())

		require.Empty(t, auditor.AuditLogs())
	})

	t.Run("ValidationFailureWritesNoLog", func(t *testing.T) {
		auditor.ResetLogs()

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:    genSecretName(t),
			Value:   "value",
			EnvName: "1invalid",
		})
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusBadRequest, sdkErr.StatusCode())

		require.Empty(t, auditor.AuditLogs())
	})

	t.Run("EmptyUpdateWritesNoLog", func(t *testing.T) {
		auditor.ResetLogs()
		name := genSecretName(t)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  name,
			Value: "value",
		})
		require.NoError(t, err)
		// Reset to ignore the created log. We are only testing that the
		// no-op update does not add a new log.
		auditor.ResetLogs()

		_, err = client.UpdateUserSecret(ctx, nicloudsdk.Me, name, nicloudsdk.UpdateUserSecretRequest{})
		var sdkErr *nicloudsdk.Error
		require.ErrorAs(t, err, &sdkErr)
		assert.Equal(t, http.StatusBadRequest, sdkErr.StatusCode())

		require.Empty(t, auditor.AuditLogs())
	})

	t.Run("ReadsDoNotAudit", func(t *testing.T) {
		auditor.ResetLogs()
		secretName := genSecretName(t)

		_, err := client.CreateUserSecret(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSecretRequest{
			Name:  secretName,
			Value: "value",
		})
		require.NoError(t, err)
		// Discard the create log so the assertion below only sees audit entries
		// produced by later reads.
		auditor.ResetLogs()

		_, err = client.UserSecrets(ctx, nicloudsdk.Me)
		require.NoError(t, err)

		_, err = client.UserSecretByName(ctx, nicloudsdk.Me, secretName)
		require.NoError(t, err)

		require.Empty(t, auditor.AuditLogs())
	})
}
