package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbgen"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications/notificationstest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func createOpts(t *testing.T) *nicloudtest.Options {
	t.Helper()

	dt := nicloudtest.DeploymentValues(t)
	return &nicloudtest.Options{
		DeploymentValues: dt,
	}
}

func TestNotifications(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		command      string
		expectPaused bool
	}{
		{
			name:         "PauseNotifications",
			command:      "pause",
			expectPaused: true,
		},
		{
			name:         "ResumeNotifications",
			command:      "resume",
			expectPaused: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			ownerClient, db := nicloudtest.NewWithDatabase(t, createOpts(t))
			_ = nicloudtest.CreateFirstUser(t, ownerClient)

			// when
			inv, root := clitest.New(t, "notifications", tt.command)
			clitest.SetupConfig(t, ownerClient, root)

			var buf bytes.Buffer
			inv.Stdout = &buf
			err := inv.Run()
			require.NoError(t, err)

			// then
			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
			t.Cleanup(cancel)
			settingsJSON, err := db.GetNotificationsSettings(ctx)
			require.NoError(t, err)

			var settings nicloudsdk.NotificationsSettings
			err = json.Unmarshal([]byte(settingsJSON), &settings)
			require.NoError(t, err)
			require.Equal(t, tt.expectPaused, settings.NotifierPaused)
		})
	}
}

func TestPauseNotifications_RegularUser(t *testing.T) {
	t.Parallel()

	// given
	ownerClient, db := nicloudtest.NewWithDatabase(t, createOpts(t))
	owner := nicloudtest.CreateFirstUser(t, ownerClient)
	anotherClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

	// when
	inv, root := clitest.New(t, "notifications", "pause")
	clitest.SetupConfig(t, anotherClient, root)

	var buf bytes.Buffer
	inv.Stdout = &buf
	err := inv.Run()
	var sdkError *nicloudsdk.Error
	require.Error(t, err)
	require.ErrorAsf(t, err, &sdkError, "error should be of type *nicloudsdk.Error")
	assert.Equal(t, http.StatusForbidden, sdkError.StatusCode())
	assert.Contains(t, sdkError.Message, "Forbidden.")

	// then
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
	t.Cleanup(cancel)
	settingsJSON, err := db.GetNotificationsSettings(ctx)
	require.NoError(t, err)

	var settings nicloudsdk.NotificationsSettings
	err = json.Unmarshal([]byte(settingsJSON), &settings)
	require.NoError(t, err)
	require.False(t, settings.NotifierPaused) // still running
}

func TestNotificationsTest(t *testing.T) {
	t.Parallel()

	t.Run("OwnerCanSendTestNotification", func(t *testing.T) {
		t.Parallel()

		notifyEnq := &notificationstest.FakeEnqueuer{}

		// Given: An owner user.
		ownerClient := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues:      nicloudtest.DeploymentValues(t),
			NotificationsEnqueuer: notifyEnq,
		})
		_ = nicloudtest.CreateFirstUser(t, ownerClient)

		// When: The owner user attempts to send the test notification.
		inv, root := clitest.New(t, "notifications", "test")
		clitest.SetupConfig(t, ownerClient, root)

		// Then: we expect a notification to be sent.
		err := inv.Run()
		require.NoError(t, err)

		sent := notifyEnq.Sent(notificationstest.WithTemplateID(notifications.TemplateTestNotification))
		require.Len(t, sent, 1)
	})

	t.Run("MemberCannotSendTestNotification", func(t *testing.T) {
		t.Parallel()

		notifyEnq := &notificationstest.FakeEnqueuer{}

		// Given: A member user.
		ownerClient := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues:      nicloudtest.DeploymentValues(t),
			NotificationsEnqueuer: notifyEnq,
		})
		ownerUser := nicloudtest.CreateFirstUser(t, ownerClient)
		memberClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, ownerUser.OrganizationID)

		// When: The member user attempts to send the test notification.
		inv, root := clitest.New(t, "notifications", "test")
		clitest.SetupConfig(t, memberClient, root)

		// Then: we expect an error and no notifications to be sent.
		err := inv.Run()
		var sdkError *nicloudsdk.Error
		require.Error(t, err)
		require.ErrorAsf(t, err, &sdkError, "error should be of type *nicloudsdk.Error")
		assert.Equal(t, http.StatusForbidden, sdkError.StatusCode())

		sent := notifyEnq.Sent(notificationstest.WithTemplateID(notifications.TemplateTestNotification))
		require.Len(t, sent, 0)
	})
}

func TestCustomNotifications(t *testing.T) {
	t.Parallel()

	t.Run("BadRequest", func(t *testing.T) {
		t.Parallel()

		notifyEnq := &notificationstest.FakeEnqueuer{}

		ownerClient := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues:      nicloudtest.DeploymentValues(t),
			NotificationsEnqueuer: notifyEnq,
		})

		// Given: A member user
		ownerUser := nicloudtest.CreateFirstUser(t, ownerClient)
		memberClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, ownerUser.OrganizationID)

		// When: The member user attempts to send a custom notification with empty title and message
		inv, root := clitest.New(t, "notifications", "custom", "", "")
		clitest.SetupConfig(t, memberClient, root)

		// Then: an error is expected with no notifications sent
		err := inv.Run()
		var sdkError *nicloudsdk.Error
		require.Error(t, err)
		require.ErrorAsf(t, err, &sdkError, "error should be of type *nicloudsdk.Error")
		require.Equal(t, http.StatusBadRequest, sdkError.StatusCode())
		require.Equal(t, "Invalid request body", sdkError.Message)

		sent := notifyEnq.Sent(notificationstest.WithTemplateID(notifications.TemplateTestNotification))
		require.Len(t, sent, 0)
	})

	t.Run("SystemUserNotAllowed", func(t *testing.T) {
		t.Parallel()

		notifyEnq := &notificationstest.FakeEnqueuer{}

		ownerClient, db := nicloudtest.NewWithDatabase(t, &nicloudtest.Options{
			DeploymentValues:      nicloudtest.DeploymentValues(t),
			NotificationsEnqueuer: notifyEnq,
		})

		// Given: A system user (prebuilds system user)
		_, token := dbgen.APIKey(t, db, database.APIKey{
			UserID:    database.PrebuildsSystemUserID,
			LoginType: database.LoginTypeNone,
		})
		systemUserClient := nicloudsdk.New(ownerClient.URL)
		systemUserClient.SetSessionToken(token)

		// When: The system user attempts to send a custom notification
		inv, root := clitest.New(t, "notifications", "custom", "Custom Title", "Custom Message")
		clitest.SetupConfig(t, systemUserClient, root)

		// Then: an error is expected with no notifications sent
		err := inv.Run()
		var sdkError *nicloudsdk.Error
		require.Error(t, err)
		require.ErrorAsf(t, err, &sdkError, "error should be of type *nicloudsdk.Error")
		require.Equal(t, http.StatusForbidden, sdkError.StatusCode())
		require.Equal(t, "Forbidden", sdkError.Message)

		sent := notifyEnq.Sent(notificationstest.WithTemplateID(notifications.TemplateTestNotification))
		require.Len(t, sent, 0)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		notifyEnq := &notificationstest.FakeEnqueuer{}

		ownerClient := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues:      nicloudtest.DeploymentValues(t),
			NotificationsEnqueuer: notifyEnq,
		})

		// Given: A member user
		ownerUser := nicloudtest.CreateFirstUser(t, ownerClient)
		memberClient, memberUser := nicloudtest.CreateAnotherUser(t, ownerClient, ownerUser.OrganizationID)

		// When: The member user attempts to send a custom notification
		inv, root := clitest.New(t, "notifications", "custom", "Custom Title", "Custom Message")
		clitest.SetupConfig(t, memberClient, root)

		// Then: we expect a custom notification to be sent to the member user
		err := inv.Run()
		require.NoError(t, err)

		sent := notifyEnq.Sent(notificationstest.WithTemplateID(notifications.TemplateCustomNotification))
		require.Len(t, sent, 1)
		require.Equal(t, memberUser.ID, sent[0].UserID)
		require.Len(t, sent[0].Labels, 2)
		require.Equal(t, "Custom Title", sent[0].Labels["custom_title"])
		require.Equal(t, "Custom Message", sent[0].Labels["custom_message"])
		require.Equal(t, memberUser.ID.String(), sent[0].CreatedBy)
	})
}
