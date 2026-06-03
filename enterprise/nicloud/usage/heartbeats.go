package usage

import (
	"context"
	"time"

	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/usage/usagetypes"
)

const (
	AISeatsInterval = 4 * time.Hour
)

// AISeatsHeartbeat returns a HeartbeatFunc that queries the active
// AI seat count and emits it as an HBAISeats heartbeat event.
func AISeatsHeartbeat(db database.Store) HeartbeatFunc {
	return func(ctx context.Context) (usagetypes.HeartbeatEvent, error) {
		//nolint:gocritic // We are a publisher in this function
		ctx = dbauthz.AsUsagePublisher(ctx)
		count, err := db.GetActiveAISeatCount(ctx)
		if err != nil {
			return nil, xerrors.Errorf("get active AI seat count: %w", err)
		}

		return usagetypes.HBAISeats{Count: count}, nil
	}
}
