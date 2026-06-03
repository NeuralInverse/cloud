package dispatch

import (
	"context"
	"encoding/json"
	"text/template"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"cdr.dev/slog/v3"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtime"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/pubsub"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications/types"
	nicloudpubsub "github.com/NeuralInverse/cloud/v2/nicloud/pubsub"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type InboxStore interface {
	InsertInboxNotification(ctx context.Context, arg database.InsertInboxNotificationParams) (database.InboxNotification, error)
}

// InboxHandler is responsible for dispatching notification messages to the Neural Inverse Cloud Inbox.
type InboxHandler struct {
	log    slog.Logger
	store  InboxStore
	pubsub pubsub.Pubsub
}

func NewInboxHandler(log slog.Logger, store InboxStore, ps pubsub.Pubsub) *InboxHandler {
	return &InboxHandler{log: log, store: store, pubsub: ps}
}

func (s *InboxHandler) Dispatcher(payload types.MessagePayload, titleTmpl, bodyTmpl string, _ template.FuncMap) (DeliveryFunc, error) {
	return s.dispatch(payload, titleTmpl, bodyTmpl), nil
}

func (s *InboxHandler) dispatch(payload types.MessagePayload, title, body string) DeliveryFunc {
	return func(ctx context.Context, msgID uuid.UUID) (bool, error) {
		userID, err := uuid.Parse(payload.UserID)
		if err != nil {
			return false, xerrors.Errorf("parse user ID: %w", err)
		}
		templateID, err := uuid.Parse(payload.NotificationTemplateID)
		if err != nil {
			return false, xerrors.Errorf("parse template ID: %w", err)
		}

		actions, err := json.Marshal(payload.Actions)
		if err != nil {
			return false, xerrors.Errorf("marshal actions: %w", err)
		}

		// nolint:exhaustruct
		insertedNotif, err := s.store.InsertInboxNotification(ctx, database.InsertInboxNotificationParams{
			ID:         msgID,
			UserID:     userID,
			TemplateID: templateID,
			Targets:    payload.Targets,
			Title:      title,
			Content:    body,
			Actions:    actions,
			CreatedAt:  dbtime.Now(),
		})
		if err != nil {
			return false, xerrors.Errorf("insert inbox notification: %w", err)
		}

		event := nicloudpubsub.InboxNotificationEvent{
			Kind: nicloudpubsub.InboxNotificationEventKindNew,
			InboxNotification: nicloudsdk.InboxNotification{
				ID:         msgID,
				UserID:     userID,
				TemplateID: templateID,
				Targets:    payload.Targets,
				Title:      title,
				Content:    body,
				Actions: func() []nicloudsdk.InboxNotificationAction {
					var actions []nicloudsdk.InboxNotificationAction
					err := json.Unmarshal(insertedNotif.Actions, &actions)
					if err != nil {
						return actions
					}
					return actions
				}(),
				ReadAt:    nil, // notification just has been inserted
				CreatedAt: insertedNotif.CreatedAt,
			},
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return false, xerrors.Errorf("marshal event: %w", err)
		}

		err = s.pubsub.Publish(nicloudpubsub.InboxNotificationForOwnerEventChannel(userID), payload)
		if err != nil {
			return false, xerrors.Errorf("publish event: %w", err)
		}

		return false, nil
	}
}
