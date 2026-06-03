package nicloud

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/notifications"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestInboxNotifications_ensureNotificationIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		icon         string
		templateID   uuid.UUID
		expectedIcon string
	}{
		{"WorkspaceCreated", "", notifications.TemplateWorkspaceCreated, nicloudsdk.InboxNotificationFallbackIconWorkspace},
		{"UserAccountCreated", "", notifications.TemplateUserAccountCreated, nicloudsdk.InboxNotificationFallbackIconAccount},
		{"TemplateDeleted", "", notifications.TemplateTemplateDeleted, nicloudsdk.InboxNotificationFallbackIconTemplate},
		{"TestNotification", "", notifications.TemplateTestNotification, nicloudsdk.InboxNotificationFallbackIconOther},
		{"TestExistingIcon", "https://cdn.cloud.neuralinverse.com/icon_notif.png", notifications.TemplateTemplateDeleted, "https://cdn.cloud.neuralinverse.com/icon_notif.png"},
		{"UnknownTemplate", "", uuid.New(), nicloudsdk.InboxNotificationFallbackIconOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			notif := nicloudsdk.InboxNotification{
				ID:         uuid.New(),
				UserID:     uuid.New(),
				TemplateID: tt.templateID,
				Title:      "notification title",
				Content:    "notification content",
				Icon:       tt.icon,
				CreatedAt:  time.Now(),
			}

			notif = ensureNotificationIcon(notif)
			require.Equal(t, tt.expectedIcon, notif.Icon)
		})
	}
}
