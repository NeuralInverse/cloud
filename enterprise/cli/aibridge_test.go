package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbgen"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtime"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestAIBridgeListInterceptions(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		dv.AI.BridgeConfig.Enabled = true
		ownerClient, db, owner := nicloudenttest.NewWithDatabase(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAIBridge: 1,
				},
			},
		})
		_, member := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)
		now := dbtime.Now()
		interception1 := dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: member.ID,
			StartedAt:   now.Add(-time.Hour),
		}, &now)
		interception2EndedAt := now.Add(time.Minute)
		interception2 := dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: member.ID,
			StartedAt:   now,
		}, &interception2EndedAt)
		interception3EndedAt := now.Add(-time.Hour)
		interception3 := dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: owner.UserID,
			StartedAt:   now.Add(-2 * time.Hour),
		}, &interception3EndedAt)

		args := []string{
			"aibridge",
			"interceptions",
			"list",
		}
		inv, root := newCLI(t, args...)
		//nolint:gocritic // Owner can read all interceptions.
		clitest.SetupConfig(t, ownerClient, root)

		ctx := testutil.Context(t, testutil.WaitLong)

		out := bytes.NewBuffer(nil)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		// Owner sees all interceptions. Ordered by started_at DESC.
		requireHasInterceptions(t, out.Bytes(), []uuid.UUID{interception2.ID, interception1.ID, interception3.ID})
	})

	t.Run("Filter", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		dv.AI.BridgeConfig.Enabled = true
		ownerClient, db, owner := nicloudenttest.NewWithDatabase(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAIBridge: 1,
				},
			},
		})
		_, member := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

		now := dbtime.Now()

		// This interception should be returned since it matches all filters.
		goodInterceptionEndedAt := now.Add(time.Minute)
		goodInterception := dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: member.ID,
			Provider:    "real-provider",
			Model:       "real-model",
			StartedAt:   now,
		}, &goodInterceptionEndedAt)

		// These interceptions should not be returned since they don't match the
		// filters.
		_ = dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: owner.UserID,
			Provider:    goodInterception.Provider,
			Model:       goodInterception.Model,
			StartedAt:   goodInterception.StartedAt,
		}, nil)
		_ = dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: goodInterception.InitiatorID,
			Provider:    "bad-provider",
			Model:       goodInterception.Model,
			StartedAt:   goodInterception.StartedAt,
		}, nil)
		_ = dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: goodInterception.InitiatorID,
			Provider:    goodInterception.Provider,
			Model:       "bad-model",
			StartedAt:   goodInterception.StartedAt,
		}, nil)
		_ = dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: goodInterception.InitiatorID,
			Provider:    goodInterception.Provider,
			Model:       goodInterception.Model,
			// Violates the started after filter.
			StartedAt: now.Add(-2 * time.Hour),
		}, nil)
		_ = dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: goodInterception.InitiatorID,
			Provider:    goodInterception.Provider,
			Model:       goodInterception.Model,
			// Violates the started before filter.
			StartedAt: now.Add(2 * time.Hour),
		}, nil)

		args := []string{
			"aibridge",
			"interceptions",
			"list",
			"--started-after", now.Add(-time.Hour).Format(time.RFC3339),
			"--started-before", now.Add(time.Hour).Format(time.RFC3339),
			"--initiator", member.Username,
			"--provider", goodInterception.Provider,
			"--model", goodInterception.Model,
		}
		inv, root := newCLI(t, args...)
		//nolint:gocritic // Owner can read all interceptions.
		clitest.SetupConfig(t, ownerClient, root)

		ctx := testutil.Context(t, testutil.WaitLong)

		out := bytes.NewBuffer(nil)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		requireHasInterceptions(t, out.Bytes(), []uuid.UUID{goodInterception.ID})
	})

	t.Run("FilterByMe", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		dv.AI.BridgeConfig.Enabled = true
		ownerClient, db, owner := nicloudenttest.NewWithDatabase(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAIBridge: 1,
				},
			},
		})
		memberClient, member := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

		now := dbtime.Now()

		// Create an interception initiated by the member.
		_ = dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: member.ID,
			StartedAt:   now,
		}, nil)

		args := []string{
			"aibridge",
			"interceptions",
			"list",
			"--initiator", nicloudsdk.Me,
		}
		inv, root := newCLI(t, args...)
		clitest.SetupConfig(t, memberClient, root)

		ctx := testutil.Context(t, testutil.WaitLong)

		out := bytes.NewBuffer(nil)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		// Member cannot read their own interceptions.
		requireHasInterceptions(t, out.Bytes(), []uuid.UUID{})
	})

	t.Run("Pagination", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		dv.AI.BridgeConfig.Enabled = true
		ownerClient, db, owner := nicloudenttest.NewWithDatabase(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAIBridge: 1,
				},
			},
		})

		now := dbtime.Now()
		firstInterceptionEndedAt := now.Add(time.Minute)
		firstInterception := dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: owner.UserID,
			StartedAt:   now,
		}, &firstInterceptionEndedAt)
		returnedInterception := dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: owner.UserID,
			StartedAt:   now.Add(-time.Hour),
		}, &now)
		_ = dbgen.AIBridgeInterception(t, db, database.InsertAIBridgeInterceptionParams{
			InitiatorID: owner.UserID,
			StartedAt:   now.Add(-2 * time.Hour),
		}, nil)

		args := []string{
			"aibridge",
			"interceptions",
			"list",
			"--limit", "1",
			"--after-id", firstInterception.ID.String(),
		}
		inv, root := newCLI(t, args...)
		//nolint:gocritic // Owner can read all interceptions.
		clitest.SetupConfig(t, ownerClient, root)

		ctx := testutil.Context(t, testutil.WaitLong)

		out := bytes.NewBuffer(nil)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		// Only contains the second interception because after_id is the first
		// interception, and we set a limit of 1.
		requireHasInterceptions(t, out.Bytes(), []uuid.UUID{returnedInterception.ID})
	})
}

func requireHasInterceptions(t *testing.T, out []byte, ids []uuid.UUID) {
	t.Helper()

	var results []nicloudsdk.AIBridgeInterception
	require.NoError(t, json.Unmarshal(out, &results))
	require.Len(t, results, len(ids))
	for i, id := range ids {
		require.Equal(t, id, results[i].ID)
	}
}
