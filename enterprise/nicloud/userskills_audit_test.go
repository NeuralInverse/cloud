package nicloud_test

import (
	"encoding/json"
	"fmt"
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

func TestUserSkillAuditDiffTracksContent(t *testing.T) {
	// User skill content is user-authored instruction text, not secret material.
	// The enterprise auditor needs to be used because it writes actual diffs.
	t.Parallel()

	db, ps := dbtestutil.NewDB(t)
	auditor := entaudit.NewAuditor(
		db,
		entaudit.DefaultFilter,
		backends.NewPostgres(db, true),
	)

	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
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
	memberClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)
	member := nicloudsdk.NewExperimentalClient(memberClient)
	ctx := testutil.Context(t, testutil.WaitMedium)

	initialContent := userSkillMarkdown("audit-tracking", "initial", "initial body")
	skill, err := member.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
		Content: initialContent,
	})
	require.NoError(t, err)

	newContent := userSkillMarkdown("audit-tracking", "after", "new body")
	_, err = member.UpdateUserSkill(ctx, nicloudsdk.Me, skill.Name, nicloudsdk.UpdateUserSkillRequest{
		Content: newContent,
	})
	require.NoError(t, err)

	rows, err := db.GetAuditLogsOffset(
		dbauthz.AsSystemRestricted(ctx),
		database.GetAuditLogsOffsetParams{
			ResourceType: string(database.ResourceTypeUserSkill),
			LimitOpt:     10,
		},
	)
	require.NoError(t, err)
	require.Len(t, rows, 2, "expected exactly two rows")
	createLog := rows[1].AuditLog
	updateLog := rows[0].AuditLog

	var createDiff audit.Map
	require.NoError(t, json.Unmarshal(createLog.Diff, &createDiff))
	if assert.Contains(t, createDiff, "description", "tracked field missing from create diff") {
		assert.Equal(t, "", createDiff["description"].Old)
		assert.Equal(t, "initial", createDiff["description"].New)
		assert.False(t, createDiff["description"].Secret)
	}
	if assert.Contains(t, createDiff, "content", "content field missing from create diff") {
		assert.False(t, createDiff["content"].Secret)
		assert.Equal(t, "", createDiff["content"].Old)
		assert.Equal(t, initialContent, createDiff["content"].New)
	}

	var updateDiff audit.Map
	require.NoError(t, json.Unmarshal(updateLog.Diff, &updateDiff))
	if assert.Contains(t, updateDiff, "description", "tracked field missing from update diff") {
		assert.Equal(t, "initial", updateDiff["description"].Old)
		assert.Equal(t, "after", updateDiff["description"].New)
		assert.False(t, updateDiff["description"].Secret)
	}
	if assert.Contains(t, updateDiff, "content", "content field missing from update diff") {
		assert.False(t, updateDiff["content"].Secret)
		assert.Equal(t, initialContent, updateDiff["content"].Old)
		assert.Equal(t, newContent, updateDiff["content"].New)
	}
	assert.NotContains(t, updateDiff, "created_at")
	assert.NotContains(t, updateDiff, "updated_at")
}

func userSkillMarkdown(name string, description string, body string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, description, body)
}
