package cli_test

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbgen"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestProvisioners_Golden(t *testing.T) {
	t.Parallel()

	// Replace UUIDs with predictable values for golden files.
	replace := make(map[string]string)
	updateReplaceUUIDs := func(nicloudAPI *nicloud.API) {
		systemCtx := dbauthz.AsSystemRestricted(context.Background())
		provisioners, err := nicloudAPI.Database.GetProvisionerDaemons(systemCtx)
		require.NoError(t, err)
		slices.SortFunc(provisioners, func(a, b database.ProvisionerDaemon) int {
			return cmp.Or(
				a.CreatedAt.Compare(b.CreatedAt),
				bytes.Compare(a.ID[:], b.ID[:]),
			)
		})
		pIdx := 0
		for _, p := range provisioners {
			if _, ok := replace[p.ID.String()]; !ok {
				replace[p.ID.String()] = fmt.Sprintf("00000000-0000-0000-aaaa-%012d", pIdx)
				pIdx++
			}
		}
		jobs, err := nicloudAPI.Database.GetProvisionerJobsCreatedAfter(systemCtx, time.Time{})
		require.NoError(t, err)
		slices.SortFunc(jobs, func(a, b database.ProvisionerJob) int {
			return cmp.Or(
				a.CreatedAt.Compare(b.CreatedAt),
				bytes.Compare(a.ID[:], b.ID[:]),
			)
		})
		jIdx := 0
		for _, j := range jobs {
			if _, ok := replace[j.ID.String()]; !ok {
				replace[j.ID.String()] = fmt.Sprintf("00000000-0000-0000-bbbb-%012d", jIdx)
				jIdx++
			}
		}
	}

	db, ps := dbtestutil.NewDB(t,
		dbtestutil.WithDumpOnFailure(),
		//nolint:gocritic // Use UTC for consistent timestamp length in golden files.
		dbtestutil.WithTimezone("UTC"),
	)
	client, _, nicloudAPI := nicloudtest.NewWithAPI(t, &nicloudtest.Options{
		IncludeProvisionerDaemon: false,
		Database:                 db,
		Pubsub:                   ps,
	})
	owner := nicloudtest.CreateFirstUser(t, client)
	templateAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.ScopedRoleOrgTemplateAdmin(owner.OrganizationID))
	_, member := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)

	// Create initial resources with a running provisioner.
	firstProvisioner := nicloudtest.NewTaggedProvisionerDaemon(t, nicloudAPI, "default-provisioner", map[string]string{"owner": "", "scope": "organization"})
	t.Cleanup(func() { _ = firstProvisioner.Close() })
	version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, completeWithAgent())
	version = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	require.Equal(t, nicloudsdk.ProvisionerJobSucceeded, version.Job.Status,
		"template version import should succeed, got error: %s", version.Job.Error)
	template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)

	workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
	wb := nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
	require.Equal(t, nicloudsdk.ProvisionerJobSucceeded, wb.Job.Status,
		"workspace build job should succeed, got error: %s", wb.Job.Error)

	// Stop the provisioner so it doesn't grab any more jobs.
	firstProvisioner.Close()

	// Sanitize the UUIDs for the initial resources.
	replace[version.ID.String()] = "00000000-0000-0000-cccc-000000000000"
	replace[workspace.LatestBuild.ID.String()] = "00000000-0000-0000-dddd-000000000000"

	// Base synthetic times off the latest real job's CreatedAt, not the
	// wall clock. Using dbtime.Now() here is racy because NTP clock
	// steps can make it return a time before the real jobs' CreatedAt.
	systemCtx := dbauthz.AsSystemRestricted(context.Background())
	existingJobs, err := nicloudAPI.Database.GetProvisionerJobsCreatedAfter(systemCtx, time.Time{})
	require.NoError(t, err)
	require.NotEmpty(t, existingJobs, "expected at least one provisioner job")
	latestJob := slices.MaxFunc(existingJobs, func(a, b database.ProvisionerJob) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	now := latestJob.CreatedAt.Add(time.Second)

	// Create a provisioner that's working on a job.
	pd1 := dbgen.ProvisionerDaemon(t, nicloudAPI.Database, database.ProvisionerDaemon{
		Name:       "provisioner-1",
		CreatedAt:  now.Add(time.Second),
		LastSeenAt: sql.NullTime{Time: nicloudAPI.Clock.Now().Add(time.Hour), Valid: true}, // Stale interval can't be adjusted, keep online.
		KeyID:      nicloudsdk.ProvisionerKeyUUIDBuiltIn,
		Tags:       database.StringMap{"owner": "", "scope": "organization", "foo": "bar"},
	})
	w1 := dbgen.Workspace(t, nicloudAPI.Database, database.WorkspaceTable{
		OwnerID:    member.ID,
		TemplateID: template.ID,
		CreatedAt:  now.Add(time.Second),
	})
	wb1ID := uuid.MustParse("00000000-0000-0000-dddd-000000000001")
	job1 := dbgen.ProvisionerJob(t, db, nicloudAPI.Pubsub, database.ProvisionerJob{
		WorkerID:  uuid.NullUUID{UUID: pd1.ID, Valid: true},
		Input:     json.RawMessage(`{"workspace_build_id":"` + wb1ID.String() + `"}`),
		CreatedAt: now.Add(time.Second),
		StartedAt: sql.NullTime{Time: nicloudAPI.Clock.Now(), Valid: true},
		Tags:      database.StringMap{"owner": "", "scope": "organization", "foo": "bar"},
	})
	dbgen.WorkspaceBuild(t, nicloudAPI.Database, database.WorkspaceBuild{
		ID:                wb1ID,
		JobID:             job1.ID,
		WorkspaceID:       w1.ID,
		TemplateVersionID: version.ID,
		CreatedAt:         now.Add(time.Second),
	})

	// Create a provisioner that completed a job previously and is offline.
	pd2 := dbgen.ProvisionerDaemon(t, nicloudAPI.Database, database.ProvisionerDaemon{
		Name:       "provisioner-2",
		CreatedAt:  now.Add(2 * time.Second),
		LastSeenAt: sql.NullTime{Time: nicloudAPI.Clock.Now().Add(-time.Hour), Valid: true},
		KeyID:      nicloudsdk.ProvisionerKeyUUIDBuiltIn,
		Tags:       database.StringMap{"owner": "", "scope": "organization"},
	})
	w2 := dbgen.Workspace(t, nicloudAPI.Database, database.WorkspaceTable{
		OwnerID:    member.ID,
		TemplateID: template.ID,
		CreatedAt:  now.Add(2 * time.Second),
	})
	wb2ID := uuid.MustParse("00000000-0000-0000-dddd-000000000002")
	job2 := dbgen.ProvisionerJob(t, db, nicloudAPI.Pubsub, database.ProvisionerJob{
		WorkerID:    uuid.NullUUID{UUID: pd2.ID, Valid: true},
		Input:       json.RawMessage(`{"workspace_build_id":"` + wb2ID.String() + `"}`),
		CreatedAt:   now.Add(2 * time.Second),
		StartedAt:   sql.NullTime{Time: nicloudAPI.Clock.Now().Add(-2 * time.Hour), Valid: true},
		CompletedAt: sql.NullTime{Time: nicloudAPI.Clock.Now().Add(-time.Hour), Valid: true},
		Tags:        database.StringMap{"owner": "", "scope": "organization"},
	})
	dbgen.WorkspaceBuild(t, nicloudAPI.Database, database.WorkspaceBuild{
		ID:                wb2ID,
		JobID:             job2.ID,
		WorkspaceID:       w2.ID,
		TemplateVersionID: version.ID,
		CreatedAt:         now.Add(2 * time.Second),
	})

	// Create a pending job.
	w3 := dbgen.Workspace(t, nicloudAPI.Database, database.WorkspaceTable{
		OwnerID:    member.ID,
		TemplateID: template.ID,
		CreatedAt:  now.Add(3 * time.Second),
	})
	wb3ID := uuid.MustParse("00000000-0000-0000-dddd-000000000003")
	job3 := dbgen.ProvisionerJob(t, db, nicloudAPI.Pubsub, database.ProvisionerJob{
		Input:     json.RawMessage(`{"workspace_build_id":"` + wb3ID.String() + `"}`),
		CreatedAt: now.Add(3 * time.Second),
		Tags:      database.StringMap{"owner": "", "scope": "organization"},
	})
	dbgen.WorkspaceBuild(t, nicloudAPI.Database, database.WorkspaceBuild{
		ID:                wb3ID,
		JobID:             job3.ID,
		WorkspaceID:       w3.ID,
		TemplateVersionID: version.ID,
		CreatedAt:         now.Add(3 * time.Second),
	})

	// Create a provisioner that is idle.
	_ = dbgen.ProvisionerDaemon(t, nicloudAPI.Database, database.ProvisionerDaemon{
		Name:       "provisioner-3",
		CreatedAt:  now.Add(4 * time.Second),
		LastSeenAt: sql.NullTime{Time: nicloudAPI.Clock.Now().Add(time.Hour), Valid: true}, // Stale interval can't be adjusted, keep online.
		KeyID:      nicloudsdk.ProvisionerKeyUUIDBuiltIn,
		Tags:       database.StringMap{"owner": "", "scope": "organization"},
	})

	updateReplaceUUIDs(nicloudAPI)

	for id, replaceID := range replace {
		t.Logf("replace[%q] = %q", id, replaceID)
	}

	// Test provisioners list with template admin as members are currently
	// unable to access provisioner jobs. In the future (with RBAC
	// changes), we may allow them to view _their_ jobs.
	t.Run("list", func(t *testing.T) {
		t.Parallel()

		var got bytes.Buffer
		inv, root := clitest.New(t,
			"provisioners",
			"list",
			"--column", "id,created at,last seen at,name,version,tags,key name,status,current job id,current job status,previous job id,previous job status,organization",
		)
		inv.Stdout = &got
		clitest.SetupConfig(t, templateAdminClient, root)
		err := inv.Run()
		require.NoError(t, err)

		clitest.TestGoldenFile(t, t.Name(), got.Bytes(), replace)
	})

	t.Run("list with offline provisioner daemons", func(t *testing.T) {
		t.Parallel()

		var got bytes.Buffer
		inv, root := clitest.New(t,
			"provisioners",
			"list",
			"--show-offline",
		)
		inv.Stdout = &got
		clitest.SetupConfig(t, templateAdminClient, root)
		err := inv.Run()
		require.NoError(t, err)

		clitest.TestGoldenFile(t, t.Name(), got.Bytes(), replace)
	})

	t.Run("list provisioner daemons by status", func(t *testing.T) {
		t.Parallel()

		var got bytes.Buffer
		inv, root := clitest.New(t,
			"provisioners",
			"list",
			"--status=idle,offline,busy",
		)
		inv.Stdout = &got
		clitest.SetupConfig(t, templateAdminClient, root)
		err := inv.Run()
		require.NoError(t, err)

		clitest.TestGoldenFile(t, t.Name(), got.Bytes(), replace)
	})

	t.Run("list provisioner daemons without offline", func(t *testing.T) {
		t.Parallel()

		var got bytes.Buffer
		inv, root := clitest.New(t,
			"provisioners",
			"list",
			"--status=idle,busy",
		)
		inv.Stdout = &got
		clitest.SetupConfig(t, templateAdminClient, root)
		err := inv.Run()
		require.NoError(t, err)

		clitest.TestGoldenFile(t, t.Name(), got.Bytes(), replace)
	})

	t.Run("list provisioner daemons by max age", func(t *testing.T) {
		t.Parallel()

		var got bytes.Buffer
		inv, root := clitest.New(t,
			"provisioners",
			"list",
			"--max-age=1h",
		)
		inv.Stdout = &got
		clitest.SetupConfig(t, templateAdminClient, root)
		err := inv.Run()
		require.NoError(t, err)

		clitest.TestGoldenFile(t, t.Name(), got.Bytes(), replace)
	})

	// Test jobs list with template admin as members are currently
	// unable to access provisioner jobs. In the future (with RBAC
	// changes), we may allow them to view _their_ jobs.
	t.Run("jobs list", func(t *testing.T) {
		t.Parallel()

		var got bytes.Buffer
		inv, root := clitest.New(t,
			"provisioners",
			"jobs",
			"list",
			"--column", "id,created at,status,worker id,tags,template version id,workspace build id,type,available workers,organization,queue",
		)
		inv.Stdout = &got
		clitest.SetupConfig(t, templateAdminClient, root)
		err := inv.Run()
		require.NoError(t, err)

		clitest.TestGoldenFile(t, t.Name(), got.Bytes(), replace)
	})
}
