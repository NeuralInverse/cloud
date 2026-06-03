package nicloud_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/x/skills"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestPatchUserSkill(t *testing.T) {
	t.Parallel()

	ownerRawClient := nicloudtest.New(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, ownerRawClient)
	memberRawClient, member := nicloudtest.CreateAnotherUser(t, ownerRawClient, firstUser.OrganizationID)
	memberClient := nicloudsdk.NewExperimentalClient(memberRawClient)
	auditorRawClient, _ := nicloudtest.CreateAnotherUser(t, ownerRawClient, firstUser.OrganizationID, rbac.RoleAuditor())
	auditorClient := nicloudsdk.NewExperimentalClient(auditorRawClient)
	ctx := testutil.Context(t, testutil.WaitMedium)

	_, err := memberClient.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
		Content: userSkillMarkdown("forbidden-skill", "Test skill", "Original body."),
	})
	require.NoError(t, err)

	_, err = auditorClient.UpdateUserSkill(ctx, member.ID.String(), "forbidden-skill", nicloudsdk.UpdateUserSkillRequest{
		Content: userSkillMarkdown("forbidden-skill", "Test skill", "Updated body."),
	})
	requireSDKErrorStatus(t, err, http.StatusForbidden)
}

func TestUserSkillsCRUD(t *testing.T) {
	t.Parallel()

	adminClient := nicloudtest.New(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	ownerClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	owner := nicloudsdk.NewExperimentalClient(ownerClient)
	ctx := testutil.Context(t, testutil.WaitMedium)

	emptyList, err := owner.UserSkills(ctx, nicloudsdk.Me)
	require.NoError(t, err)
	assert.NotNil(t, emptyList)
	assert.Empty(t, emptyList)

	emptyRes, err := owner.Request(ctx, http.MethodGet, "/api/experimental/users/me/skills", nil)
	require.NoError(t, err)
	defer emptyRes.Body.Close()
	require.Equal(t, http.StatusOK, emptyRes.StatusCode)
	var rawEmptyList []map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(emptyRes.Body).Decode(&rawEmptyList))
	assert.NotNil(t, rawEmptyList)
	assert.Empty(t, rawEmptyList)

	content := userSkillMarkdown("crud-skill", "Initial description", "Use this skill for CRUD tests.")
	created, err := owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{Content: content})
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, "crud-skill", created.Name)
	assert.Equal(t, "Initial description", created.Description)
	assert.Equal(t, content, created.Content)
	assert.NotZero(t, created.CreatedAt)
	assert.NotZero(t, created.UpdatedAt)

	list, err := owner.UserSkills(ctx, nicloudsdk.Me)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, created.ID, list[0].ID)
	assert.Equal(t, "crud-skill", list[0].Name)
	assert.Equal(t, "Initial description", list[0].Description)

	res, err := owner.Request(ctx, http.MethodGet, "/api/experimental/users/me/skills", nil)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	var rawList []map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(res.Body).Decode(&rawList))
	require.Len(t, rawList, 1)
	assert.NotContains(t, rawList[0], "content")

	got, err := owner.UserSkillByName(ctx, nicloudsdk.Me, "crud-skill")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, content, got.Content)

	updatedContent := userSkillMarkdown("crud-skill", "Updated description", "Updated body.")
	updated, err := owner.UpdateUserSkill(ctx, nicloudsdk.Me, "crud-skill", nicloudsdk.UpdateUserSkillRequest{Content: updatedContent})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, updatedContent, updated.Content)

	require.NoError(t, owner.DeleteUserSkill(ctx, nicloudsdk.Me, "crud-skill"))
	_, err = owner.UserSkillByName(ctx, nicloudsdk.Me, "crud-skill")
	requireSDKErrorStatus(t, err, http.StatusNotFound)
}

func TestUserSkillValidationAndConflicts(t *testing.T) {
	t.Parallel()

	adminClient := nicloudtest.New(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	ownerClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	otherClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	owner := nicloudsdk.NewExperimentalClient(ownerClient)
	other := nicloudsdk.NewExperimentalClient(otherClient)

	tests := []struct {
		name            string
		content         string
		expectedMessage string
	}{
		{
			name:            "MissingFrontmatterDelimiters",
			content:         "name: missing-frontmatter\n\nBody.",
			expectedMessage: "Invalid skill content.",
		},
		{
			name: "MissingName",
			content: "---\n" +
				"description: Missing name\n" +
				"---\n\nBody.",
			expectedMessage: "Invalid skill name.",
		},
		{
			name:            "NonKebabCaseName",
			content:         userSkillMarkdown("NotKebab", "Invalid", "Body."),
			expectedMessage: "Invalid skill name.",
		},
		{
			name:            "NameTooLong",
			content:         userSkillMarkdown(strings.Repeat("a", skills.MaxPersonalSkillNameBytes+1), "Invalid", "Body."),
			expectedMessage: "Invalid skill name.",
		},
		{
			name:            "EmptyBody",
			content:         userSkillMarkdown("empty-body", "Invalid", "   \n"),
			expectedMessage: "Skill body is required.",
		},
		{
			name:            "TooLarge",
			content:         strings.Repeat("a", skills.MaxPersonalSkillSizeBytes+1),
			expectedMessage: "Skill content is too large.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			subCtx := testutil.Context(t, testutil.WaitMedium)
			_, err := owner.CreateUserSkill(subCtx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{Content: tt.content})
			sdkErr := requireSDKErrorStatus(t, err, http.StatusBadRequest)
			assert.Equal(t, tt.expectedMessage, sdkErr.Message)
		})
	}

	t.Run("PatchEmptyBody", func(t *testing.T) {
		t.Parallel()

		subCtx := testutil.Context(t, testutil.WaitMedium)
		patchValidationContent := userSkillMarkdown("patch-validation", "Valid", "Body.")
		_, err := owner.CreateUserSkill(subCtx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{Content: patchValidationContent})
		require.NoError(t, err)
		_, err = owner.UpdateUserSkill(subCtx, nicloudsdk.Me, "patch-validation", nicloudsdk.UpdateUserSkillRequest{
			Content: userSkillMarkdown("patch-validation", "Invalid", "   \n"),
		})
		sdkErr := requireSDKErrorStatus(t, err, http.StatusBadRequest)
		assert.Equal(t, "Skill body is required.", sdkErr.Message)
	})

	t.Run("DuplicateNameConflict", func(t *testing.T) {
		t.Parallel()

		subCtx := testutil.Context(t, testutil.WaitMedium)
		sharedContent := userSkillMarkdown("shared-skill", "Shared", "Shared body.")
		_, err := owner.CreateUserSkill(subCtx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{Content: sharedContent})
		require.NoError(t, err)
		_, err = owner.CreateUserSkill(subCtx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{Content: sharedContent})
		requireSDKErrorStatus(t, err, http.StatusConflict)
	})

	t.Run("CrossUserSameNameAllowed", func(t *testing.T) {
		t.Parallel()

		subCtx := testutil.Context(t, testutil.WaitMedium)
		sharedContent := userSkillMarkdown("shared-skill", "Shared", "Shared body.")
		_, err := other.CreateUserSkill(subCtx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{Content: sharedContent})
		require.NoError(t, err)
	})
}

func TestUserSkillLimit(t *testing.T) {
	t.Parallel()

	adminClient := nicloudtest.New(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	ownerClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	owner := nicloudsdk.NewExperimentalClient(ownerClient)
	ctx := testutil.Context(t, testutil.WaitLong)

	for i := range skills.MaxPersonalSkillsPerUser {
		name := fmt.Sprintf("limit-skill-%03d", i)
		_, err := owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
			Content: userSkillMarkdown(name, "Limit", "Body."),
		})
		require.NoError(t, err)
	}

	_, err := owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
		Content: userSkillMarkdown("limit-skill-overflow", "Limit", "Body."),
	})
	sdkErr := requireSDKErrorStatus(t, err, http.StatusConflict)
	assert.Equal(t, "Personal skill limit reached.", sdkErr.Message)
	assert.Equal(t,
		fmt.Sprintf("Each user can have at most %d personal skills.", skills.MaxPersonalSkillsPerUser),
		sdkErr.Detail,
	)
}

func TestUserSkillLimitConcurrentCreates(t *testing.T) {
	t.Parallel()

	adminClient := nicloudtest.New(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	ownerClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	owner := nicloudsdk.NewExperimentalClient(ownerClient)
	ctx := testutil.Context(t, testutil.WaitLong)

	for i := range skills.MaxPersonalSkillsPerUser - 1 {
		name := fmt.Sprintf("concurrent-limit-skill-%03d", i)
		_, err := owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
			Content: userSkillMarkdown(name, "Limit", "Body."),
		})
		require.NoError(t, err)
	}

	const attempts = 8
	start := make(chan struct{})
	results := make(chan error, attempts)
	for i := range attempts {
		go func() {
			<-start
			name := fmt.Sprintf("concurrent-limit-overflow-%03d", i)
			_, err := owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
				Content: userSkillMarkdown(name, "Limit", "Body."),
			})
			results <- err
		}()
	}
	close(start)

	successes := 0
	for range attempts {
		err := <-results
		if err == nil {
			successes++
			continue
		}
		requireSDKErrorStatus(t, err, http.StatusConflict)
	}
	assert.Equal(t, 1, successes)

	list, err := owner.UserSkills(ctx, nicloudsdk.Me)
	require.NoError(t, err)
	assert.Len(t, list, skills.MaxPersonalSkillsPerUser)
}

func TestUserSkillRequestAllowsEscapedMaxSizeContent(t *testing.T) {
	t.Parallel()

	adminClient := nicloudtest.New(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	ownerClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	owner := nicloudsdk.NewExperimentalClient(ownerClient)
	ctx := testutil.Context(t, testutil.WaitMedium)

	prefix := "---\nname: escaped-limit-skill\ndescription: Escaped\n---\n\n"
	suffix := "\n"
	bodyLen := skills.MaxPersonalSkillSizeBytes - len(prefix) - len(suffix)
	require.Positive(t, bodyLen)
	content := prefix + strings.Repeat(`"`, bodyLen) + suffix
	require.Len(t, []byte(content), skills.MaxPersonalSkillSizeBytes)

	raw, err := json.Marshal(nicloudsdk.CreateUserSkillRequest{Content: content})
	require.NoError(t, err)
	require.Greater(t, len(raw), skills.MaxPersonalSkillSizeBytes+1024)

	created, err := owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
		Content: content,
	})
	require.NoError(t, err)
	assert.Equal(t, "escaped-limit-skill", created.Name)
}

func TestUserSkillMissingAndUpdateMismatch(t *testing.T) {
	t.Parallel()

	adminClient := nicloudtest.New(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	ownerClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	owner := nicloudsdk.NewExperimentalClient(ownerClient)
	ctx := testutil.Context(t, testutil.WaitMedium)

	_, err := owner.UserSkillByName(ctx, nicloudsdk.Me, "missing-skill")
	requireSDKErrorStatus(t, err, http.StatusNotFound)

	_, err = owner.UpdateUserSkill(ctx, nicloudsdk.Me, "missing-skill", nicloudsdk.UpdateUserSkillRequest{
		Content: userSkillMarkdown("missing-skill", "Missing", "Body."),
	})
	requireSDKErrorStatus(t, err, http.StatusNotFound)

	err = owner.DeleteUserSkill(ctx, nicloudsdk.Me, "missing-skill")
	requireSDKErrorStatus(t, err, http.StatusNotFound)

	_, err = owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
		Content: userSkillMarkdown("old-name", "Old", "Body."),
	})
	require.NoError(t, err)
	_, err = owner.UpdateUserSkill(ctx, nicloudsdk.Me, "old-name", nicloudsdk.UpdateUserSkillRequest{
		Content: userSkillMarkdown("new-name", "New", "Body."),
	})
	sdkErr := requireSDKErrorStatus(t, err, http.StatusBadRequest)
	assert.Equal(t, "Skill name in path does not match frontmatter name.", sdkErr.Message)
	assert.Equal(t, `path has "old-name", frontmatter has "new-name"`, sdkErr.Detail)
}

func TestUserSkillAuthorization(t *testing.T) {
	t.Parallel()

	adminClient := nicloudtest.New(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	ownerClient, ownerUser := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	otherClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	userAdminClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID, rbac.RoleUserAdmin())
	admin := nicloudsdk.NewExperimentalClient(adminClient)
	owner := nicloudsdk.NewExperimentalClient(ownerClient)
	other := nicloudsdk.NewExperimentalClient(otherClient)
	userAdmin := nicloudsdk.NewExperimentalClient(userAdminClient)
	ctx := testutil.Context(t, testutil.WaitMedium)
	targetUser := ownerUser.Username

	_, err := owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
		Content: userSkillMarkdown("auth-skill", "Auth", "Body."),
	})
	require.NoError(t, err)

	_, err = other.UserSkills(ctx, targetUser)
	requireSDKErrorStatus(t, err, http.StatusNotFound)
	_, err = other.UserSkillByName(ctx, targetUser, "auth-skill")
	requireSDKErrorStatus(t, err, http.StatusNotFound)
	_, err = other.CreateUserSkill(ctx, targetUser, nicloudsdk.CreateUserSkillRequest{
		Content: userSkillMarkdown("denied-create", "Denied", "Body."),
	})
	requireSDKErrorStatus(t, err, http.StatusNotFound)
	_, err = other.UpdateUserSkill(ctx, targetUser, "auth-skill", nicloudsdk.UpdateUserSkillRequest{
		Content: userSkillMarkdown("auth-skill", "Denied", "Body."),
	})
	requireSDKErrorStatus(t, err, http.StatusNotFound)
	err = other.DeleteUserSkill(ctx, targetUser, "auth-skill")
	requireSDKErrorStatus(t, err, http.StatusNotFound)

	_, err = userAdmin.UserSkills(ctx, targetUser)
	requireSDKErrorStatus(t, err, http.StatusNotFound)
	_, err = userAdmin.UserSkillByName(ctx, targetUser, "auth-skill")
	requireSDKErrorStatus(t, err, http.StatusNotFound)
	_, err = userAdmin.CreateUserSkill(ctx, targetUser, nicloudsdk.CreateUserSkillRequest{
		Content: userSkillMarkdown("denied-admin-create", "Denied", "Body."),
	})
	requireSDKErrorStatus(t, err, http.StatusForbidden)
	_, err = userAdmin.UpdateUserSkill(ctx, targetUser, "auth-skill", nicloudsdk.UpdateUserSkillRequest{
		Content: userSkillMarkdown("auth-skill", "Denied", "Body."),
	})
	requireSDKErrorStatus(t, err, http.StatusForbidden)
	err = userAdmin.DeleteUserSkill(ctx, targetUser, "auth-skill")
	requireSDKErrorStatus(t, err, http.StatusNotFound)

	_, err = admin.CreateUserSkill(ctx, targetUser, nicloudsdk.CreateUserSkillRequest{
		Content: userSkillMarkdown("admin-created", "Admin create", "Created by admin."),
	})
	requireSDKErrorStatus(t, err, http.StatusForbidden)

	list, err := admin.UserSkills(ctx, targetUser)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "auth-skill", list[0].Name)
	got, err := admin.UserSkillByName(ctx, targetUser, "auth-skill")
	require.NoError(t, err)
	assert.Equal(t, "auth-skill", got.Name)
	_, err = admin.UpdateUserSkill(ctx, targetUser, "auth-skill", nicloudsdk.UpdateUserSkillRequest{
		Content: userSkillMarkdown("auth-skill", "Admin update", "Updated by admin."),
	})
	requireSDKErrorStatus(t, err, http.StatusForbidden)
	require.NoError(t, admin.DeleteUserSkill(ctx, targetUser, "auth-skill"))
}

func TestUserSkillSoftDeleteCleanup(t *testing.T) {
	t.Parallel()

	adminClient, _, api := nicloudtest.NewWithAPI(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	ownerClient, ownerUser := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	owner := nicloudsdk.NewExperimentalClient(ownerClient)
	ctx := testutil.Context(t, testutil.WaitMedium)

	_, err := owner.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
		Content: userSkillMarkdown("soft-delete-skill", "Soft delete", "Body."),
	})
	require.NoError(t, err)

	require.NoError(t, adminClient.DeleteUser(ctx, ownerUser.ID))
	readAuthzCtx := dbauthz.AsSystemRestricted(ctx)
	_, err = api.Database.GetUserSkillByUserIDAndName(
		readAuthzCtx,
		database.GetUserSkillByUserIDAndNameParams{
			UserID: ownerUser.ID,
			Name:   "soft-delete-skill",
		},
	)
	require.ErrorIs(t, err, sql.ErrNoRows)

	createAuthzCtx := dbauthz.As(ctx, rbac.Subject{
		Type:  rbac.SubjectTypeUser,
		ID:    ownerUser.ID.String(),
		Roles: rbac.RoleIdentifiers{rbac.RoleMember()},
		Scope: rbac.ScopeAll,
	}.WithCachedASTValue())
	_, err = api.Database.InsertUserSkill(
		createAuthzCtx,
		database.InsertUserSkillParams{
			ID:          uuid.New(),
			UserID:      ownerUser.ID,
			Name:        "after-soft-delete",
			Description: "Soft delete",
			Content:     userSkillMarkdown("after-soft-delete", "Soft delete", "Body."),
		},
	)
	require.True(t, database.IsCheckViolation(err, database.CheckConstraint("user_skill_user_deleted")))
	require.ErrorContains(t, err, "Cannot create user_skill for deleted user")
}

func TestUserSkillDatabaseConstraints(t *testing.T) {
	t.Parallel()

	adminClient, _, api := nicloudtest.NewWithAPI(t, nil)
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	_, ownerUser := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	tests := []struct {
		name       string
		params     database.InsertUserSkillParams
		constraint database.CheckConstraint
	}{
		{
			name: "NameFormat",
			params: database.InsertUserSkillParams{
				ID:          uuid.New(),
				UserID:      ownerUser.ID,
				Name:        "not kebab",
				Description: "Invalid",
				Content:     userSkillMarkdown("not kebab", "Invalid", "Body."),
			},
			constraint: database.CheckUserSkillsNameFormat,
		},
		{
			name: "NameSize",
			params: database.InsertUserSkillParams{
				ID:          uuid.New(),
				UserID:      ownerUser.ID,
				Name:        strings.Repeat("a", skills.MaxPersonalSkillNameBytes+1),
				Description: "Invalid",
				Content:     userSkillMarkdown("too-long-name", "Invalid", "Body."),
			},
			constraint: database.CheckUserSkillsNameSize,
		},
		{
			name: "ContentSize",
			params: database.InsertUserSkillParams{
				ID:          uuid.New(),
				UserID:      ownerUser.ID,
				Name:        "content-too-large",
				Description: "Invalid",
				Content:     strings.Repeat("a", skills.MaxPersonalSkillSizeBytes+1),
			},
			constraint: database.CheckUserSkillsContentSize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t, testutil.WaitMedium)
			authzCtx := dbauthz.As(ctx, nicloudtest.AuthzUserSubject(ownerUser))

			_, err := api.Database.InsertUserSkill(authzCtx, tt.params)
			require.True(t, database.IsCheckViolation(err, tt.constraint), "expected %s, got %v", tt.constraint, err)
		})
	}
}

func TestUserSkillSchemaConstants(t *testing.T) {
	t.Parallel()

	_, _, sqlDB := dbtestutil.NewDBWithSQLDB(t)

	ctx := testutil.Context(t, testutil.WaitMedium)
	var triggerDef string
	require.NoError(t, sqlDB.QueryRowContext(
		ctx,
		`SELECT pg_get_functiondef('enforce_user_skills_per_user_limit'::regproc)`,
	).Scan(&triggerDef))
	assert.Contains(t, triggerDef, fmt.Sprintf("skill_limit constant int := %d;", skills.MaxPersonalSkillsPerUser))

	constraints := map[database.CheckConstraint]string{
		database.CheckUserSkillsNameSize:    fmt.Sprintf("octet_length(name) <= %d", skills.MaxPersonalSkillNameBytes),
		database.CheckUserSkillsNameFormat:  "name ~ '^[a-z0-9]+(-[a-z0-9]+)*$'::text",
		database.CheckUserSkillsContentSize: fmt.Sprintf("octet_length(content) <= %d", skills.MaxPersonalSkillSizeBytes),
	}
	for constraint, expected := range constraints {
		t.Run(string(constraint), func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t, testutil.WaitMedium)
			var constraintDef string
			require.NoError(t, sqlDB.QueryRowContext(
				ctx,
				`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = $1`,
				constraint,
			).Scan(&constraintDef))
			assert.Contains(t, constraintDef, expected)
		})
	}
}

//nolint:paralleltest,tparallel // Subtests share one auditor and run sequentially.
func TestUserSkillAudit(t *testing.T) {
	t.Parallel()

	auditor := audit.NewMock()
	adminClient := nicloudtest.New(t, &nicloudtest.Options{Auditor: auditor})
	firstUser := nicloudtest.CreateFirstUser(t, adminClient)
	memberClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, firstUser.OrganizationID)
	member := nicloudsdk.NewExperimentalClient(memberClient)
	ctx := testutil.Context(t, testutil.WaitMedium)
	auditor.ResetLogs()

	genName := func(t *testing.T) string {
		return strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	}

	t.Run("CreateEmitsLog", func(t *testing.T) {
		auditor.ResetLogs()
		name := genName(t)

		skill, err := member.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
			Content: userSkillMarkdown(name, "Audit", "Body."),
		})
		require.NoError(t, err)

		logs := auditor.AuditLogs()
		require.Len(t, logs, 1)
		assert.Equal(t, database.AuditActionCreate, logs[0].Action)
		assert.Equal(t, skill.ID, logs[0].ResourceID)
		assert.Equal(t, skill.Name, logs[0].ResourceTarget)
		assert.EqualValues(t, http.StatusCreated, logs[0].StatusCode)
	})

	t.Run("UpdateEmitsLog", func(t *testing.T) {
		auditor.ResetLogs()
		name := genName(t)

		skill, err := member.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
			Content: userSkillMarkdown(name, "Initial", "Body."),
		})
		require.NoError(t, err)
		_, err = member.UpdateUserSkill(ctx, nicloudsdk.Me, name, nicloudsdk.UpdateUserSkillRequest{
			Content: userSkillMarkdown(name, "Updated", "Updated body."),
		})
		require.NoError(t, err)

		logs := auditor.AuditLogs()
		require.Len(t, logs, 2)
		assert.Equal(t, database.AuditActionCreate, logs[0].Action)
		assert.Equal(t, database.AuditActionWrite, logs[1].Action)
		assert.Equal(t, skill.ID, logs[1].ResourceID)
		assert.Equal(t, skill.Name, logs[1].ResourceTarget)
		assert.EqualValues(t, http.StatusOK, logs[1].StatusCode)
	})

	t.Run("DeleteEmitsLog", func(t *testing.T) {
		auditor.ResetLogs()
		name := genName(t)

		skill, err := member.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
			Content: userSkillMarkdown(name, "Delete", "Body."),
		})
		require.NoError(t, err)
		require.NoError(t, member.DeleteUserSkill(ctx, nicloudsdk.Me, name))

		logs := auditor.AuditLogs()
		require.Len(t, logs, 2)
		assert.Equal(t, database.AuditActionCreate, logs[0].Action)
		assert.Equal(t, database.AuditActionDelete, logs[1].Action)
		assert.Equal(t, skill.ID, logs[1].ResourceID)
		assert.Equal(t, skill.Name, logs[1].ResourceTarget)
		assert.EqualValues(t, http.StatusNoContent, logs[1].StatusCode)
	})

	t.Run("ReadsDoNotEmitLogs", func(t *testing.T) {
		auditor.ResetLogs()
		name := genName(t)

		_, err := member.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
			Content: userSkillMarkdown(name, "Read", "Body."),
		})
		require.NoError(t, err)
		auditor.ResetLogs()

		_, err = member.UserSkills(ctx, nicloudsdk.Me)
		require.NoError(t, err)
		_, err = member.UserSkillByName(ctx, nicloudsdk.Me, name)
		require.NoError(t, err)
		assert.Empty(t, auditor.AuditLogs())
	})

	t.Run("ValidationFailureDoesNotEmitLog", func(t *testing.T) {
		auditor.ResetLogs()

		_, err := member.CreateUserSkill(ctx, nicloudsdk.Me, nicloudsdk.CreateUserSkillRequest{
			Content: userSkillMarkdown("bad-name", "Invalid", "   \n"),
		})
		requireSDKErrorStatus(t, err, http.StatusBadRequest)
		assert.Empty(t, auditor.AuditLogs())
	})

	t.Run("MissingSkillFailuresDoNotEmitLogs", func(t *testing.T) {
		auditor.ResetLogs()

		_, err := member.UpdateUserSkill(ctx, nicloudsdk.Me, "missing-audit-skill", nicloudsdk.UpdateUserSkillRequest{
			Content: userSkillMarkdown("missing-audit-skill", "Missing", "Body."),
		})
		requireSDKErrorStatus(t, err, http.StatusNotFound)
		err = member.DeleteUserSkill(ctx, nicloudsdk.Me, "missing-audit-skill")
		requireSDKErrorStatus(t, err, http.StatusNotFound)
		assert.Empty(t, auditor.AuditLogs())
	})
}

func userSkillMarkdown(name string, description string, body string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, description, body)
}

func requireSDKErrorStatus(t *testing.T, err error, status int, msgAndArgs ...any) *nicloudsdk.Error {
	t.Helper()
	require.Error(t, err, msgAndArgs...)
	sdkErr := nicloudtest.SDKError(t, err)
	require.Equal(t, status, sdkErr.StatusCode(), msgAndArgs...)
	return sdkErr
}
