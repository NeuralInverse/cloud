package nicloud_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	entaudit "github.com/NeuralInverse/cloud/v2/enterprise/audit"
	"github.com/NeuralInverse/cloud/v2/enterprise/audit/backends"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

// TestAIProviderAuditDiff exercises the full HTTP -> enterprise auditor
// -> Postgres write path for AI provider updates. The mock auditor used
// elsewhere returns an empty diff, so this is the only place that
// proves changed properties land in the audit_logs row.
func TestAIProviderAuditDiff(t *testing.T) {
	t.Parallel()

	db, ps := dbtestutil.NewDB(t)
	auditor := entaudit.NewAuditor(
		db,
		entaudit.DefaultFilter,
		backends.NewPostgres(db, true),
	)

	ownerClient, _ := nicloudenttest.New(t, &nicloudenttest.Options{
		AuditLogging: true,
		Options: &nicloudtest.Options{
			Database: db,
			Pubsub:   ps,
			Auditor:  auditor,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureAuditLog: 1,
			},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitMedium)
	defer cancel()

	//nolint:gocritic // Owner role is the audience for this endpoint.
	provider, err := ownerClient.CreateAIProvider(ctx, nicloudsdk.CreateAIProviderRequest{
		Type:        nicloudsdk.AIProviderTypeOpenAI,
		Name:        "audit-target",
		DisplayName: "Audit Target",
		Enabled:     true,
		BaseURL:     "https://api.openai.com/v1",
	})
	require.NoError(t, err)

	newDisplay := "Renamed"
	newURL := "https://api.openai.com/v2"
	disabled := false
	_, err = ownerClient.UpdateAIProvider(ctx, provider.Name, nicloudsdk.UpdateAIProviderRequest{
		DisplayName: &newDisplay,
		BaseURL:     &newURL,
		Enabled:     &disabled,
	})
	require.NoError(t, err)

	rows, err := db.GetAuditLogsOffset(
		dbauthz.AsSystemRestricted(ctx),
		database.GetAuditLogsOffsetParams{
			ResourceType: string(database.ResourceTypeAIProvider),
			LimitOpt:     10,
		},
	)
	require.NoError(t, err)
	require.Len(t, rows, 2, "expected one create and one update audit row")

	// GetAuditLogsOffset returns entries sorted by time in descending order.
	updateLog := rows[0].AuditLog
	require.Equal(t, database.AuditActionWrite, updateLog.Action)

	var updateDiff audit.Map
	require.NoError(t, json.Unmarshal(updateLog.Diff, &updateDiff))

	if assert.Contains(t, updateDiff, "display_name", "display_name missing from diff") {
		assert.Equal(t, "Audit Target", updateDiff["display_name"].Old)
		assert.Equal(t, newDisplay, updateDiff["display_name"].New)
	}
	if assert.Contains(t, updateDiff, "base_url", "base_url missing from diff") {
		assert.Equal(t, "https://api.openai.com/v1", updateDiff["base_url"].Old)
		assert.Equal(t, newURL, updateDiff["base_url"].New)
	}
	if assert.Contains(t, updateDiff, "enabled", "enabled missing from diff") {
		assert.Equal(t, true, updateDiff["enabled"].Old)
		assert.Equal(t, false, updateDiff["enabled"].New)
	}
}
