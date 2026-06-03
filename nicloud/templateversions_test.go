package nicloud_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/db2sdk"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbgen"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/externalauth"
	"github.com/NeuralInverse/cloud/v2/nicloud/provisionerdserver"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/examples"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/provisionersdk"
	"github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestTemplateVersion(t *testing.T) {
	t.Parallel()
	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		client, _, api := nicloudtest.NewWithAPI(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		authz := nicloudtest.AssertRBAC(t, api, client).Reset()

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil, func(req *nicloudsdk.CreateTemplateVersionRequest) {
			req.Name = "bananas"
			req.Message = "first try"
		})
		authz.AssertChecked(t, policy.ActionCreate, rbac.ResourceTemplate.InOrg(user.OrganizationID))

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		authz.Reset()
		tv, err := client.TemplateVersion(ctx, version.ID)
		authz.AssertChecked(t, policy.ActionRead, tv)
		require.NoError(t, err)
		if assert.Equal(t, tv.Job.Status, nicloudsdk.ProvisionerJobPending) {
			assert.NotNil(t, tv.MatchedProvisioners)
			assert.Zero(t, tv.MatchedProvisioners.Available)
			assert.Zero(t, tv.MatchedProvisioners.Count)
			assert.False(t, tv.MatchedProvisioners.MostRecentlySeen.Valid)
		}

		assert.Equal(t, "bananas", tv.Name)
		assert.Equal(t, "first try", tv.Message)
	})

	t.Run("Message limit exceeded", func(t *testing.T) {
		t.Parallel()
		client, _, _ := nicloudtest.NewWithAPI(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		file, err := client.Upload(ctx, nicloudsdk.ContentTypeTar, bytes.NewReader([]byte{}))
		require.NoError(t, err)
		_, err = client.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			Name:          "bananas",
			Message:       strings.Repeat("a", 1048577),
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			FileID:        file.ID,
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.Error(t, err, "message too long, create should fail")
	})

	t.Run("MemberCanRead", func(t *testing.T) {
		t.Parallel()

		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		_ = nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		client1, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		tv, err := client1.TemplateVersion(ctx, version.ID)
		require.NoError(t, err)
		if assert.Equal(t, tv.Job.Status, nicloudsdk.ProvisionerJobPending) {
			assert.NotNil(t, tv.MatchedProvisioners)
			assert.Zero(t, tv.MatchedProvisioners.Available)
			assert.Zero(t, tv.MatchedProvisioners.Count)
			assert.False(t, tv.MatchedProvisioners.MostRecentlySeen.Valid)
		}
	})
}

func TestPostTemplateVersionsByOrganization(t *testing.T) {
	t.Parallel()
	t.Run("InvalidTemplate", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		templateID := uuid.New()
		_, err := client.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			TemplateID:    templateID,
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			FileID:        uuid.New(),
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("FileNotFound", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			FileID:        uuid.New(),
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("WithParameters", func(t *testing.T) {
		t.Parallel()
		auditor := audit.NewMock()
		client, db := nicloudtest.NewWithDatabase(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, Auditor: auditor})
		user := nicloudtest.CreateFirstUser(t, client)
		data, err := echo.Tar(&echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyComplete,
			ProvisionPlan:  echo.PlanComplete,
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		file, err := client.Upload(ctx, nicloudsdk.ContentTypeTar, bytes.NewReader(data))
		require.NoError(t, err)
		version, err := client.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			Name:          "bananas",
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			FileID:        file.ID,
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
		require.Equal(t, "bananas", version.Name)
		require.Equal(t, provisionersdk.ScopeOrganization, version.Job.Tags[provisionersdk.TagScope])
		if assert.Equal(t, version.Job.Status, nicloudsdk.ProvisionerJobPending) {
			assert.NotNil(t, version.MatchedProvisioners)
			assert.Equal(t, 1, version.MatchedProvisioners.Available)
			assert.Equal(t, 1, version.MatchedProvisioners.Count)
			assert.True(t, version.MatchedProvisioners.MostRecentlySeen.Valid)
		}

		require.Len(t, auditor.AuditLogs(), 2)
		assert.Equal(t, database.AuditActionCreate, auditor.AuditLogs()[1].Action)

		admin, err := client.User(ctx, user.UserID.String())
		require.NoError(t, err)
		tvDB, err := db.GetTemplateVersionByID(dbauthz.As(ctx, nicloudtest.AuthzUserSubject(admin)), version.ID)
		require.NoError(t, err)
		require.False(t, tvDB.SourceExampleID.Valid)
	})

	t.Run("Example", func(t *testing.T) {
		t.Parallel()
		client, db := nicloudtest.NewWithDatabase(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		ls, err := examples.List()
		require.NoError(t, err)

		// try a bad example ID
		_, err = client.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			Name:          "my-example",
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			ExampleID:     "not a real ID",
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "not found")

		// try file and example IDs
		_, err = client.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			Name:          "my-example",
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			ExampleID:     ls[0].ID,
			FileID:        uuid.New(),
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "example_id")
		require.ErrorContains(t, err, "file_id")

		// try a good example ID
		tv, err := client.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			Name:          "my-example",
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			ExampleID:     ls[0].ID,
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
		require.Equal(t, "my-example", tv.Name)

		admin, err := client.User(ctx, user.UserID.String())
		require.NoError(t, err)
		tvDB, err := db.GetTemplateVersionByID(dbauthz.As(ctx, nicloudtest.AuthzUserSubject(admin)), tv.ID)
		require.NoError(t, err)
		require.Equal(t, ls[0].ID, tvDB.SourceExampleID.String)

		// ensure the template tar was uploaded correctly
		fl, ct, err := client.Download(ctx, tv.Job.FileID)
		require.NoError(t, err)
		require.Equal(t, "application/x-tar", ct)
		tar, err := examples.Archive(ls[0].ID)
		require.NoError(t, err)
		require.EqualValues(t, tar, fl)

		// ensure we don't get file conflicts on multiple uses of the same example
		tv, err = client.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			Name:          "my-example",
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			ExampleID:     ls[0].ID,
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
	})

	t.Run("WorkspaceTags", func(t *testing.T) {
		t.Parallel()
		// This test ensures that when creating a template version from an archive continaining a ni_workspace_tags
		// data source, we automatically assign some "reasonable" provisioner tag values to the resulting template
		// import job.
		// TODO(Cian): I'd also like to assert that the correct raw tag values are stored in the database,
		//             but in order to do this, we need to actually run the job! This isn't straightforward right now.

		store, ps := dbtestutil.NewDB(t)
		client := nicloudtest.New(t, &nicloudtest.Options{
			Database: store,
			Pubsub:   ps,
		})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, templateAdminUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())

		for _, tt := range []struct {
			name        string
			files       map[string]string
			reqTags     map[string]string
			wantTags    map[string]string
			variables   []nicloudsdk.VariableValue
			expectError string
		}{
			{
				name:     "empty",
				wantTags: map[string]string{"owner": "", "scope": "organization"},
			},
			{
				name: "main.tf with no tags",
				files: map[string]string{
					`main.tf`: `
						variable "a" {
							type = string
							default = "1"
						}
						data "ni_parameter" "b" {
							name = "b"
							type = string
							default = "2"
						}
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {}`,
				},
				wantTags: map[string]string{"owner": "", "scope": "organization"},
			},
			{
				name: "main.tf with empty workspace tags",
				files: map[string]string{
					`main.tf`: `
						variable "a" {
							type = string
							default = "1"
						}
						data "ni_parameter" "b" {
							name = "b"
							type = string
							default = "2"
						}
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {}
						data "ni_workspace_tags" "tags" {
							tags = {}
						}`,
				},
				wantTags: map[string]string{"owner": "", "scope": "organization"},
			},
			{
				name: "main.tf with workspace tags",
				files: map[string]string{
					`main.tf`: `
						variable "a" {
							type = string
							default = "1"
						}
						data "ni_parameter" "b" {
							name = "b"
							type = string
							default = "2"
						}
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {}
						data "ni_workspace_tags" "tags" {
							tags = {
								"foo": "bar",
								"a": var.a,
								"b": data.ni_parameter.b.value,
							}
						}`,
				},
				wantTags: map[string]string{"owner": "", "scope": "organization", "foo": "bar", "a": "1", "b": "2"},
			},
			{
				name: "main.tf with request tags not clobbering workspace tags",
				files: map[string]string{
					`main.tf`: `
						// This file is, once again, the same as the above, except
						// for a slightly different comment.
						variable "a" {
							type = string
							default = "1"
						}
						data "ni_parameter" "b" {
							name = "b"
							type = string
							default = "2"
						}
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {}
						data "ni_workspace_tags" "tags" {
							tags = {
								"foo": "bar",
								"a": var.a,
								"b": data.ni_parameter.b.value,
							}
						}`,
				},
				reqTags:  map[string]string{"baz": "zap"},
				wantTags: map[string]string{"owner": "", "scope": "organization", "foo": "bar", "baz": "zap", "a": "1", "b": "2"},
			},
			{
				name: "main.tf with request tags clobbering workspace tags",
				files: map[string]string{
					`main.tf`: `
						// This file is the same as the above, except for this comment.
						variable "a" {
							type = string
							default = "1"
						}
						data "ni_parameter" "b" {
							name = "b"
							type = string
							default = "2"
						}
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {}
						data "ni_workspace_tags" "tags" {
							tags = {
								"foo": "bar",
								"a": var.a,
								"b": data.ni_parameter.b.value,
							}
						}`,
				},
				reqTags:  map[string]string{"baz": "zap", "foo": "clobbered"},
				wantTags: map[string]string{"owner": "", "scope": "organization", "foo": "clobbered", "baz": "zap", "a": "1", "b": "2"},
			},
			// FIXME(cian): we should skip evaluating tags for which values have already been provided.
			{
				name: "main.tf with variable missing default value but value is passed in request",
				files: map[string]string{
					`main.tf`: `
						variable "a" {
							type = string
						}
						data "ni_workspace_tags" "tags" {
							tags = {
								"a": var.a,
							}
						}`,
				},
				reqTags:   map[string]string{"a": "b"},
				wantTags:  map[string]string{"owner": "", "scope": "organization", "a": "b"},
				variables: []nicloudsdk.VariableValue{{Name: "a", Value: "b"}},
			},
			{
				name: "main.tf with resource reference",
				files: map[string]string{
					`main.tf`: `
						variable "a" {
							type = string
							default = "1"
						}
						data "ni_parameter" "b" {
							name = "b"
							type = string
							default = "2"
						}
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {
							name = "foo"
						}
						data "ni_workspace_tags" "tags" {
							tags = {
								"foo": "bar",
								"a": var.a,
								"b": data.ni_parameter.b.value,
								"test": null_resource.test.name,
							}
						}`,
				},
				reqTags:  map[string]string{"foo": "bar", "a": "1", "b": "2", "test": "foo"},
				wantTags: map[string]string{"owner": "", "scope": "organization", "foo": "bar", "a": "1", "b": "2", "test": "foo"},
			},
			// We will allow ni_workspace_tags to set the scope on a template version import job
			// BUT the user ID will be ultimately determined by the API key in the scope.
			// TODO(Cian): Is this what we want? Or should we just ignore these provisioner
			// tags entirely?
			{
				name: "main.tf with workspace tags that attempts to set user scope",
				files: map[string]string{
					`main.tf`: `
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {}
						data "ni_workspace_tags" "tags" {
							tags = {
								"scope": "user",
								"owner": "12345678-1234-1234-1234-1234567890ab",
							}
						}`,
				},
				wantTags: map[string]string{"owner": templateAdminUser.ID.String(), "scope": "user"},
			},
			{
				name: "main.tf with workspace tags that attempt to clobber org ID",
				files: map[string]string{
					`main.tf`: `
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {}
						data "ni_workspace_tags" "tags" {
							tags = {
								"scope": "organization",
								"owner": "12345678-1234-1234-1234-1234567890ab",
							}
						}`,
				},
				wantTags: map[string]string{"owner": "", "scope": "organization"},
			},
			{
				name: "main.tf with workspace tags that set scope=user",
				files: map[string]string{
					`main.tf`: `
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}
						resource "null_resource" "test" {}
						data "ni_workspace_tags" "tags" {
							tags = {
								"scope": "user",
							}
						}`,
				},
				wantTags: map[string]string{"owner": templateAdminUser.ID.String(), "scope": "user"},
			},
			// Ref: https://github.com/coder/coder/issues/16021
			{
				name: "main.tf with no workspace_tags and a function call in a parameter default",
				files: map[string]string{
					`main.tf`: `
						data "ni_parameter" "unrelated" {
							name    = "unrelated"
							type    = "list(string)"
							default = jsonencode(["a", "b"])
						}`,
				},
				wantTags: map[string]string{"owner": "", "scope": "organization"},
			},
			{
				name: "main.tf with tags from parameter with default value from variable no default",
				files: map[string]string{
					`main.tf`: `
						variable "provisioner" {
						  type        = string
						}
						variable "default_provisioner" {
						  type        = string
						  default     = "" # intentionally blank, set on template creation
						}
						data "ni_parameter" "provisioner" {
						  name         = "provisioner"
						  mutable      = false
						  default      = var.default_provisioner
						  dynamic "option" {
							for_each = toset(split(",", var.provisioner))
							content {
							  name  = option.value
							  value = option.value
							}
						  }
						}
						data "ni_workspace_tags" "tags" {
						  tags = {
							"provisioner" : data.ni_parameter.provisioner.value
						  }
						}`,
				},
				reqTags: map[string]string{
					"provisioner": "alpha",
				},
				wantTags: map[string]string{
					"provisioner": "alpha", "owner": "", "scope": "organization",
				},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ctx := testutil.Context(t, testutil.WaitShort)

				// Create an archive from the files provided in the test case.
				tarFile := testutil.CreateTar(t, tt.files)

				// Post the archive file
				fi, err := templateAdmin.Upload(ctx, "application/x-tar", bytes.NewReader(tarFile))
				require.NoError(t, err)

				// Create a template version from the archive
				tvName := testutil.GetRandomNameHyphenated(t)
				tv, err := templateAdmin.CreateTemplateVersion(ctx, owner.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
					Name:               tvName,
					StorageMethod:      nicloudsdk.ProvisionerStorageMethodFile,
					Provisioner:        nicloudsdk.ProvisionerTypeTerraform,
					FileID:             fi.ID,
					ProvisionerTags:    tt.reqTags,
					UserVariableValues: tt.variables,
				})

				if tt.expectError == "" {
					require.NoError(t, err)
					// Assert the expected provisioner job is created from the template version import
					pj, err := store.GetProvisionerJobByID(ctx, tv.Job.ID)
					require.NoError(t, err)
					require.EqualValues(t, tt.wantTags, pj.Tags)
					// Also assert that we get the expected information back from the API endpoint
					require.Zero(t, tv.MatchedProvisioners.Count)
					require.Zero(t, tv.MatchedProvisioners.Available)
					require.Zero(t, tv.MatchedProvisioners.MostRecentlySeen.Time)
				} else {
					require.ErrorContains(t, err, tt.expectError)
				}
			})
		}
	})

	t.Run("Presets", func(t *testing.T) {
		t.Parallel()
		store, ps := dbtestutil.NewDB(t)
		client := nicloudtest.New(t, &nicloudtest.Options{
			Database: store,
			Pubsub:   ps,
		})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())

		for _, tt := range []struct {
			name        string
			files       map[string]string
			expectError string
		}{
			{
				name: "valid preset",
				files: map[string]string{
					`main.tf`: `
						terraform {
							required_providers {
								coder = {
									source = "coder/coder"
									version = "2.8.0"
								}
							}
						}
						data "ni_parameter" "valid_parameter" {
							name = "valid_parameter_name"
							default = "valid_option_value"
							option {
								name = "valid_option_name"
								value = "valid_option_value"
							}
						}
						data "ni_workspace_preset" "valid_preset" {
							name = "valid_preset"
							parameters = {
								"valid_parameter_name" = "valid_option_value"
							}
						}
					`,
				},
			},
			{
				name: "invalid preset",
				files: map[string]string{
					`main.tf`: `
						terraform {
							required_providers {
								coder = {
									source = "coder/coder"
									version = "2.8.0"
								}
							}
						}
						data "ni_parameter" "valid_parameter" {
							name = "valid_parameter_name"
							default = "valid_option_value"
							option {
								name = "valid_option_name"
								value = "valid_option_value"
							}
						}
						data "ni_workspace_preset" "invalid_parameter_name" {
							name = "invalid_parameter_name"
							parameters = {
								"invalid_parameter_name" = "irrelevant_value"
							}
						}
					`,
				},
				expectError: "", // Presets are not validated unless they are for a prebuild
			},
			{
				name: "invalid prebuild",
				files: map[string]string{
					`main.tf`: `
						terraform {
							required_providers {
								coder = {
									source = "coder/coder"
									version = "2.8.0"
								}
							}
						}
						data "ni_parameter" "valid_parameter" {
							name = "valid_parameter_name"
							default = "valid_option_value"
							option {
								name = "valid_option_name"
								value = "valid_option_value"
							}
						}
						data "ni_workspace_preset" "invalid_parameter_name" {
							name = "invalid_parameter_name"
							parameters = {
								"invalid_parameter_name" = "irrelevant_value"
							}
							prebuilds {
								instances = 2
							}
						}
					`,
				},
				expectError: "Undefined Parameter",
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ctx := testutil.Context(t, testutil.WaitShort)

				// Create an archive from the files provided in the test case.
				tarFile := testutil.CreateTar(t, tt.files)

				// Post the archive file
				fi, err := templateAdmin.Upload(ctx, "application/x-tar", bytes.NewReader(tarFile))
				require.NoError(t, err)

				// Create a template version from the archive
				tvName := testutil.GetRandomNameHyphenated(t)
				tv, err := templateAdmin.CreateTemplateVersion(ctx, owner.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
					Name:          tvName,
					StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
					Provisioner:   nicloudsdk.ProvisionerTypeTerraform,
					FileID:        fi.ID,
				})

				if tt.expectError == "" {
					require.NoError(t, err)
					// Assert the expected provisioner job is created from the template version import
					pj, err := store.GetProvisionerJobByID(ctx, tv.Job.ID)
					require.NoError(t, err)
					require.NotNil(t, pj)
					// Also assert that we get the expected information back from the API endpoint
					require.Zero(t, tv.MatchedProvisioners.Count)
					require.Zero(t, tv.MatchedProvisioners.Available)
					require.Zero(t, tv.MatchedProvisioners.MostRecentlySeen.Time)
				} else {
					require.ErrorContains(t, err, tt.expectError)
					require.Equal(t, tv.Job.ID, uuid.Nil)
				}
			})
		}
	})
}

// TestTemplateVersionPresetValidation validates that presets with prebuilds
// are validated dynamically. A preset that enables a conditional parameter
// but doesn't provide the required value for the newly-visible parameter
// should fail validation during template version import.
//
// Scenario:
// - Parameter A (use_custom_image): defaults to false
// - Parameter B (custom_image_url): only exists when A is true, has no default
// - Preset with prebuilds enables A but doesn't provide B
//
// Static validation passes because B doesn't exist when evaluated with default
// values. ValidatePrebuilds catches this by evaluating with the preset's
// parameter values.
func TestTemplateVersionPresetValidation(t *testing.T) {
	t.Parallel()

	store, ps := dbtestutil.NewDB(t)
	client := nicloudtest.New(t, &nicloudtest.Options{
		Database: store,
		Pubsub:   ps,
	})
	owner := nicloudtest.CreateFirstUser(t, client)
	templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())

	ctx := testutil.Context(t, testutil.WaitShort)

	tf := func(valid bool, prebuildCount int) string {
		customImageURL := ""
		if valid {
			customImageURL = `custom_image_url = "ghcr.io/coder/example:latest"`
		}
		return fmt.Sprintf(`
		terraform {
		  required_providers {
			coder = {
			  source = "coder/coder"
			  version = "2.8.0"
			}
		  }
		}

		data "ni_parameter" "use_custom_image" {
		  name    = "use_custom_image"
		  type    = "bool"
		  default = "false"
		}

		data "ni_parameter" "custom_image_url" {
		  count   = data.ni_parameter.use_custom_image.value == "true" ? 1 : 0
		  name    = "custom_image_url"
		  type    = "string"
		  # No default - required when shown
		}

		data "ni_workspace_preset" "invalid" {
		  name = "Invalid Preset"
		  parameters = {
			"use_custom_image" = "true"
			%s
		  }
		  prebuilds {
			instances = %d
		  }
		}
		`, customImageURL, prebuildCount)
	}

	tarFile := testutil.CreateTar(t, map[string]string{
		`main.tf`: tf(false, 1),
	})

	fi, err := templateAdmin.Upload(ctx, "application/x-tar", bytes.NewReader(tarFile))
	require.NoError(t, err)

	_, err = templateAdmin.CreateTemplateVersion(ctx, owner.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
		Name:          testutil.GetRandomNameHyphenated(t),
		StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
		Provisioner:   nicloudsdk.ProvisionerTypeTerraform,
		FileID:        fi.ID,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "Parameter custom_image_url: Required parameter not provided; parameter value is null")

	// If the preset is not a prebuild, validation should pass. As presets can
	// be partially applied, we test with a prebuild count of 0.
	tarFile = testutil.CreateTar(t, map[string]string{
		`main.tf`: tf(false, 0),
	})

	fi, err = templateAdmin.Upload(ctx, "application/x-tar", bytes.NewReader(tarFile))
	require.NoError(t, err)

	_, err = templateAdmin.CreateTemplateVersion(ctx, owner.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
		Name:          testutil.GetRandomNameHyphenated(t),
		StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
		Provisioner:   nicloudsdk.ProvisionerTypeTerraform,
		FileID:        fi.ID,
	})
	require.NoError(t, err)

	// The valid preset should pass
	tarFile = testutil.CreateTar(t, map[string]string{
		`main.tf`: tf(true, 1),
	})

	fi, err = templateAdmin.Upload(ctx, "application/x-tar", bytes.NewReader(tarFile))
	require.NoError(t, err)

	_, err = templateAdmin.CreateTemplateVersion(ctx, owner.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
		Name:          testutil.GetRandomNameHyphenated(t),
		StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
		Provisioner:   nicloudsdk.ProvisionerTypeTerraform,
		FileID:        fi.ID,
	})
	require.NoError(t, err)
}

func TestPatchCancelTemplateVersion(t *testing.T) {
	t.Parallel()
	t.Run("AlreadyCompleted", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		err := client.CancelTemplateVersion(ctx, version.ID)
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
	})
	t.Run("AlreadyCanceled", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionPlan: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{},
				},
			}},
		})

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		nicloudtest.AwaitTemplateVersionJobRunning(t, client, version.ID)
		err := client.CancelTemplateVersion(ctx, version.ID)
		require.NoError(t, err)
		err = client.CancelTemplateVersion(ctx, version.ID)
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
		require.Eventually(t, func() bool {
			var err error
			version, err = client.TemplateVersion(ctx, version.ID)
			return assert.NoError(t, err) && version.Job.Status == nicloudsdk.ProvisionerJobFailed
		}, testutil.WaitShort, testutil.IntervalFast)
	})
	// TODO(Cian): until we are able to test cancellation properly, validating
	// Running -> Canceling is the best we can do for now.
	t.Run("Canceling", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionPlan: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{},
				},
			}},
		})

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		require.Eventually(t, func() bool {
			var err error
			version, err = client.TemplateVersion(ctx, version.ID)
			if !assert.NoError(t, err) {
				return false
			}
			t.Logf("Status: %s", version.Job.Status)
			return version.Job.Status == nicloudsdk.ProvisionerJobRunning
		}, testutil.WaitShort, testutil.IntervalFast)
		err := client.CancelTemplateVersion(ctx, version.ID)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			var err error
			version, err = client.TemplateVersion(ctx, version.ID)
			// job gets marked Failed when there is an Error; in practice we never get to Status = Canceled
			// because provisioners report an Error when canceled. We check the Error string to ensure we don't mask
			// other errors in this test.
			t.Logf("got version %s | %s", version.Job.Error, version.Job.Status)
			return assert.NoError(t, err) &&
				strings.HasSuffix(version.Job.Error, "canceled") &&
				version.Job.Status == nicloudsdk.ProvisionerJobFailed
		}, testutil.WaitShort, testutil.IntervalFast)
	})
}

func TestTemplateVersionsExternalAuth(t *testing.T) {
	t.Parallel()
	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.TemplateVersionExternalAuth(ctx, version.ID)
		require.NoError(t, err)
	})
	t.Run("Authenticated", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
			ExternalAuthConfigs: []*externalauth.Config{{
				InstrumentedOAuth2Config: &testutil.OAuth2Config{},
				ID:                       "github",
				Regex:                    regexp.MustCompile(`github\.com`),
				Type:                     nicloudsdk.EnhancedExternalAuthProviderGitHub.String(),
			}},
		})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionGraph: []*proto.Response{{
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						ExternalAuthProviders: []*proto.ExternalAuthProviderResource{{Id: "github", Optional: true}},
					},
				},
			}},
		})
		version = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		require.Empty(t, version.Job.Error)
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		// Not authenticated to start!
		providers, err := client.TemplateVersionExternalAuth(ctx, version.ID)
		require.NoError(t, err)
		require.Len(t, providers, 1)
		require.False(t, providers[0].Authenticated)

		// Perform the Git auth callback to authenticate the user...
		resp := nicloudtest.RequestExternalAuthCallback(t, "github", client)
		_ = resp.Body.Close()
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)

		// Ensure that the returned Git auth for the template is authenticated!
		providers, err = client.TemplateVersionExternalAuth(ctx, version.ID)
		require.NoError(t, err)
		require.Len(t, providers, 1)
		require.True(t, providers[0].Authenticated)
		require.True(t, providers[0].Optional)
	})
}

func TestTemplateVersionResources(t *testing.T) {
	t.Parallel()
	t.Run("ListRunning", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.TemplateVersionResources(ctx, version.ID)
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
	})
	t.Run("List", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionGraph: []*proto.Response{{
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						Resources: []*proto.Resource{{
							Name: "some",
							Type: "example",
							Agents: []*proto.Agent{{
								Id:   "something",
								Name: "dev",
								Auth: &proto.Agent_Token{},
							}},
						}, {
							Name: "another",
							Type: "example",
						}},
					},
				},
			}},
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		resources, err := client.TemplateVersionResources(ctx, version.ID)
		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Len(t, resources, 4)
		require.Equal(t, "some", resources[2].Name)
		require.Equal(t, "example", resources[2].Type)
		require.Len(t, resources[2].Agents, 1)
	})
}

func TestTemplateVersionLogs(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
	user := nicloudtest.CreateFirstUser(t, client)
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
		Parse:         echo.ParseComplete,
		ProvisionPlan: echo.PlanComplete,
		ProvisionGraph: []*proto.Response{{
			Type: &proto.Response_Log{
				Log: &proto.Log{
					Level:  proto.LogLevel_INFO,
					Output: "example",
				},
			},
		}, {
			Type: &proto.Response_Graph{
				Graph: &proto.GraphComplete{
					Resources: []*proto.Resource{{
						Name: "some",
						Type: "example",
						Agents: []*proto.Agent{{
							Id:   "something",
							Name: "dev",
							Auth: &proto.Agent_Token{
								Token: uuid.NewString(),
							},
						}},
					}, {
						Name: "another",
						Type: "example",
					}},
				},
			},
		}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	logs, closer, err := client.TemplateVersionLogsAfter(ctx, version.ID, 0)
	require.NoError(t, err)
	defer closer.Close()
	for {
		_, ok := <-logs
		if !ok {
			return
		}
	}
}

func TestTemplateVersionLogsFormat(t *testing.T) {
	t.Parallel()

	// Setup: Create template version with logs using dbfake.
	client, db := nicloudtest.NewWithDatabase(t, nil)
	user := nicloudtest.CreateFirstUser(t, client)

	tv := dbfake.TemplateVersion(t, db).
		Seed(database.TemplateVersion{
			OrganizationID: user.OrganizationID,
			CreatedBy:      user.UserID,
		}).
		Do()

	// Insert test log directly into database.
	jl := dbgen.ProvisionerJobLog(t, db, database.ProvisionerJobLog{
		JobID:  tv.TemplateVersion.JobID,
		Stage:  "Planning",
		Source: database.LogSourceProvisioner,
		Level:  database.LogLevelInfo,
		Output: "test log output",
	})

	tests := []struct {
		name                string
		queryParams         string
		expectedStatus      int
		expectedContentType string
		checkBody           func(t *testing.T, body string)
	}{
		{
			name:                "JSON",
			queryParams:         "",
			expectedStatus:      http.StatusOK,
			expectedContentType: "application/json",
			checkBody: func(t *testing.T, body string) {
				assert.NotEmpty(t, body) // This is checked more thoroughly in TestTemplateVersionLogs above.
			},
		},
		{
			name:                "Text",
			queryParams:         "?format=text",
			expectedStatus:      http.StatusOK,
			expectedContentType: "text/plain",
			checkBody: func(t *testing.T, body string) {
				expected := db2sdk.ProvisionerJobLog(jl).Text()
				assert.Contains(t, body, expected)
			},
		},
		{
			name:           "InvalidFormat",
			queryParams:    "?format=invalid",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				t.Log(body)
				var sdkErr nicloudsdk.Error
				assert.NoError(t, json.NewDecoder(strings.NewReader(body)).Decode(&sdkErr))
				assert.Equal(t, "Invalid format parameter.", sdkErr.Message)
			},
		},
		{
			name:           "TextWithFollowFails",
			queryParams:    "?format=text&follow",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				t.Log(body)
				var sdkErr nicloudsdk.Error
				assert.NoError(t, json.NewDecoder(strings.NewReader(body)).Decode(&sdkErr))
				assert.Equal(t, "Text format is not supported with follow mode.", sdkErr.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := testutil.Context(t, testutil.WaitLong)

			urlPath := fmt.Sprintf("/api/v2/templateversions/%s/logs%s", tv.TemplateVersion.ID, tt.queryParams)

			res, err := client.Request(ctx, http.MethodGet, urlPath, nil)
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, tt.expectedStatus, res.StatusCode)
			if tt.expectedContentType != "" {
				require.Contains(t, res.Header.Get("Content-Type"), tt.expectedContentType)
			}
			if assert.NotNil(t, tt.checkBody) {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				tt.checkBody(t, string(body))
			}
		})
	}
}

func TestTemplateVersionsByTemplate(t *testing.T) {
	t.Parallel()
	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		versions, err := client.TemplateVersionsByTemplate(ctx, nicloudsdk.TemplateVersionsByTemplateRequest{
			TemplateID: template.ID,
		})
		require.NoError(t, err)
		require.Len(t, versions, 1)
	})
}

func TestTemplateVersionByName(t *testing.T) {
	t.Parallel()

	// Single instance shared across all sub-tests. Each sub-test
	// creates its own template version and template with unique
	// IDs so parallel execution is safe.
	client := nicloudtest.New(t, nil)
	user := nicloudtest.CreateFirstUser(t, client)
	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.TemplateVersionByName(ctx, template.ID, "nothing")
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("Found", func(t *testing.T) {
		t.Parallel()
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		tv, err := client.TemplateVersionByName(ctx, template.ID, version.Name)
		require.NoError(t, err)

		if assert.Equal(t, tv.Job.Status, nicloudsdk.ProvisionerJobPending) {
			assert.NotNil(t, tv.MatchedProvisioners)
			assert.Zero(t, tv.MatchedProvisioners.Available)
			assert.Zero(t, tv.MatchedProvisioners.Count)
			assert.False(t, tv.MatchedProvisioners.MostRecentlySeen.Valid)
		}
	})
}

func TestPatchActiveTemplateVersion(t *testing.T) {
	t.Parallel()
	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		err := client.UpdateActiveTemplateVersion(ctx, template.ID, nicloudsdk.UpdateActiveTemplateVersion{
			ID: uuid.New(),
		})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("CanceledBuild", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		version = nicloudtest.UpdateTemplateVersion(t, client, user.OrganizationID, nil, template.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		err := client.CancelTemplateVersion(ctx, version.ID)
		require.NoError(t, err)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		err = client.UpdateActiveTemplateVersion(ctx, template.ID, nicloudsdk.UpdateActiveTemplateVersion{
			ID: version.ID,
		})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusForbidden, apiErr.StatusCode())
		require.Contains(t, apiErr.Detail, "canceled")
	})

	t.Run("PendingBuild", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		version = nicloudtest.UpdateTemplateVersion(t, client, user.OrganizationID, nil, template.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		err := client.UpdateActiveTemplateVersion(ctx, template.ID, nicloudsdk.UpdateActiveTemplateVersion{
			ID: version.ID,
		})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusForbidden, apiErr.StatusCode())
		require.Contains(t, apiErr.Detail, "pending")
	})

	t.Run("DoesNotBelong", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		version = nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		_ = nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		err := client.UpdateActiveTemplateVersion(ctx, template.ID, nicloudsdk.UpdateActiveTemplateVersion{
			ID: version.ID,
		})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
	})

	t.Run("Archived", func(t *testing.T) {
		t.Parallel()
		auditor := audit.NewMock()
		ownerClient := nicloudtest.New(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
			Auditor:                  auditor,
		})
		owner := nicloudtest.CreateFirstUser(t, ownerClient)
		client, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())

		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		version = nicloudtest.UpdateTemplateVersion(t, client, owner.OrganizationID, nil, template.ID)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		err := client.SetArchiveTemplateVersion(ctx, version.ID, true)
		require.NoError(t, err)

		err = client.UpdateActiveTemplateVersion(ctx, template.ID, nicloudsdk.UpdateActiveTemplateVersion{
			ID: version.ID,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "The provided template version is archived")
	})

	t.Run("SuccessfulBuild", func(t *testing.T) {
		t.Parallel()
		auditor := audit.NewMock()
		client := nicloudtest.New(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
			Auditor:                  auditor,
		})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		version = nicloudtest.UpdateTemplateVersion(t, client, user.OrganizationID, nil, template.ID)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		err := client.UpdateActiveTemplateVersion(ctx, template.ID, nicloudsdk.UpdateActiveTemplateVersion{
			ID: version.ID,
		})
		require.NoError(t, err)

		require.Len(t, auditor.AuditLogs(), 6)
		assert.Equal(t, database.AuditActionWrite, auditor.AuditLogs()[5].Action)
	})
}

func TestTemplateVersionDryRun(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		resource := &proto.Resource{
			Name: "cool-resource",
			Type: "cool_resource_type",
		}

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionGraph: []*proto.Response{
				{
					Type: &proto.Response_Log{
						Log: &proto.Log{},
					},
				},
				{
					Type: &proto.Response_Graph{
						Graph: &proto.GraphComplete{
							Resources: []*proto.Resource{resource},
						},
					},
				},
			},
		})
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		// Create template version dry-run
		job, err := client.CreateTemplateVersionDryRun(ctx, version.ID, nicloudsdk.CreateTemplateVersionDryRunRequest{})
		require.NoError(t, err)

		// Fetch template version dry-run
		newJob, err := client.TemplateVersionDryRun(ctx, version.ID, job.ID)
		require.NoError(t, err)
		require.Equal(t, job.ID, newJob.ID)

		// Check matched provisioners
		matched, err := client.TemplateVersionDryRunMatchedProvisioners(ctx, version.ID, job.ID)
		require.NoError(t, err)
		require.Equal(t, 1, matched.Count)
		require.Equal(t, 1, matched.Available)
		require.NotZero(t, matched.MostRecentlySeen.Time)

		// Stream logs
		logs, closer, err := client.TemplateVersionDryRunLogsAfter(ctx, version.ID, job.ID, 0)
		require.NoError(t, err)
		defer closer.Close()

		logsDone := make(chan struct{})
		go func() {
			defer close(logsDone)

			logCount := 0
			for range logs {
				logCount++
			}
			assert.GreaterOrEqual(t, logCount, 1, "unexpected log count")
		}()

		// Wait for the job to complete
		require.Eventually(t, func() bool {
			job, err := client.TemplateVersionDryRun(ctx, version.ID, job.ID)
			return assert.NoError(t, err) && job.Status == nicloudsdk.ProvisionerJobSucceeded
		}, testutil.WaitShort, testutil.IntervalFast)

		<-logsDone

		resources, err := client.TemplateVersionDryRunResources(ctx, version.ID, job.ID)
		require.NoError(t, err)
		require.Len(t, resources, 1)
		require.Equal(t, resource.Name, resources[0].Name)
		require.Equal(t, resource.Type, resources[0].Type)
	})

	t.Run("ImportNotFinished", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		// This import job will never finish
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionPlan: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{},
				},
			}},
		})

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.CreateTemplateVersionDryRun(ctx, version.ID, nicloudsdk.CreateTemplateVersionDryRunRequest{})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusTooEarly, apiErr.StatusCode())
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Parallel()

		t.Run("OK", func(t *testing.T) {
			t.Parallel()
			client, closer := nicloudtest.NewWithProvisionerCloser(t, nil)
			defer closer.Close()

			user := nicloudtest.CreateFirstUser(t, client)

			version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
				Parse: echo.ParseComplete,
				ProvisionApply: []*proto.Response{
					{
						Type: &proto.Response_Log{
							Log: &proto.Log{},
						},
					},
					{
						Type: &proto.Response_Apply{
							Apply: &proto.ApplyComplete{},
						},
					},
				},
			})

			version = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
			require.Equal(t, nicloudsdk.ProvisionerJobSucceeded, version.Job.Status)

			closer.Close()

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()

			// Create the dry-run
			job, err := client.CreateTemplateVersionDryRun(ctx, version.ID, nicloudsdk.CreateTemplateVersionDryRunRequest{})
			require.NoError(t, err)
			require.Equal(t, nicloudsdk.ProvisionerJobPending, job.Status)
			err = client.CancelTemplateVersionDryRun(ctx, version.ID, job.ID)
			require.NoError(t, err)
			job, err = client.TemplateVersionDryRun(ctx, version.ID, job.ID)
			require.NoError(t, err)
			require.Equal(t, nicloudsdk.ProvisionerJobCanceled, job.Status)
		})

		t.Run("AlreadyCompleted", func(t *testing.T) {
			t.Parallel()
			client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
			user := nicloudtest.CreateFirstUser(t, client)
			version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
			nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()

			// Create the dry-run
			job, err := client.CreateTemplateVersionDryRun(ctx, version.ID, nicloudsdk.CreateTemplateVersionDryRunRequest{})
			require.NoError(t, err)

			require.Eventually(t, func() bool {
				job, err := client.TemplateVersionDryRun(ctx, version.ID, job.ID)
				if !assert.NoError(t, err) {
					return false
				}

				t.Logf("Status: %s", job.Status)
				return job.Status == nicloudsdk.ProvisionerJobSucceeded
			}, testutil.WaitShort, testutil.IntervalFast)

			err = client.CancelTemplateVersionDryRun(ctx, version.ID, job.ID)
			var apiErr *nicloudsdk.Error
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
		})

		t.Run("AlreadyCanceled", func(t *testing.T) {
			t.Parallel()
			client, closer := nicloudtest.NewWithProvisionerCloser(t, nil)
			defer closer.Close()

			user := nicloudtest.CreateFirstUser(t, client)
			version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
				Parse: echo.ParseComplete,
				ProvisionApply: []*proto.Response{
					{
						Type: &proto.Response_Log{
							Log: &proto.Log{},
						},
					},
					{
						Type: &proto.Response_Apply{
							Apply: &proto.ApplyComplete{},
						},
					},
				},
			})

			version = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
			require.Equal(t, nicloudsdk.ProvisionerJobSucceeded, version.Job.Status)

			closer.Close()

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()

			// Create the dry-run
			job, err := client.CreateTemplateVersionDryRun(ctx, version.ID, nicloudsdk.CreateTemplateVersionDryRunRequest{})
			require.NoError(t, err)

			err = client.CancelTemplateVersionDryRun(ctx, version.ID, job.ID)
			require.NoError(t, err)

			err = client.CancelTemplateVersionDryRun(ctx, version.ID, job.ID)
			var apiErr *nicloudsdk.Error
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
		})
	})

	t.Run("Pending", func(t *testing.T) {
		t.Parallel()

		store, ps, db := dbtestutil.NewDBWithSQLDB(t)
		client, closer := nicloudtest.NewWithProvisionerCloser(t, &nicloudtest.Options{
			Database:                 store,
			Pubsub:                   ps,
			IncludeProvisionerDaemon: true,
		})
		defer closer.Close()

		owner := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyComplete,
		})
		version = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		require.Equal(t, nicloudsdk.ProvisionerJobSucceeded, version.Job.Status)

		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		ctx := testutil.Context(t, testutil.WaitShort)

		_, err := db.Exec("DELETE FROM provisioner_daemons")
		require.NoError(t, err)

		job, err := templateAdmin.CreateTemplateVersionDryRun(ctx, version.ID, nicloudsdk.CreateTemplateVersionDryRunRequest{
			WorkspaceName:       "test",
			RichParameterValues: []nicloudsdk.WorkspaceBuildParameter{},
			UserVariableValues:  []nicloudsdk.VariableValue{},
		})
		require.NoError(t, err)
		require.Equal(t, nicloudsdk.ProvisionerJobPending, job.Status)

		matched, err := templateAdmin.TemplateVersionDryRunMatchedProvisioners(ctx, version.ID, job.ID)
		require.NoError(t, err)
		require.Equal(t, 0, matched.Count)
		require.Equal(t, 0, matched.Available)
		require.Zero(t, matched.MostRecentlySeen.Time)
	})
}

func TestTemplateVersionDryRunLogsFormat(t *testing.T) {
	t.Parallel()

	// Setup: Create template version and dry-run job with logs using dbfake.
	client, db := nicloudtest.NewWithDatabase(t, nil)
	user := nicloudtest.CreateFirstUser(t, client)

	tv := dbfake.TemplateVersion(t, db).
		Seed(database.TemplateVersion{
			OrganizationID: user.OrganizationID,
			CreatedBy:      user.UserID,
		}).
		Do()

	// Create a dry-run provisioner job.
	dryRunInput, err := json.Marshal(provisionerdserver.TemplateVersionDryRunJob{
		TemplateVersionID: tv.TemplateVersion.ID,
	})
	require.NoError(t, err)

	dryRunJob := dbgen.ProvisionerJob(t, db, nil, database.ProvisionerJob{
		OrganizationID: user.OrganizationID,
		InitiatorID:    user.UserID,
		Type:           database.ProvisionerJobTypeTemplateVersionDryRun,
		Input:          dryRunInput,
	})

	// Insert test log directly into database.
	jl := dbgen.ProvisionerJobLog(t, db, database.ProvisionerJobLog{
		JobID:  dryRunJob.ID,
		Stage:  "Planning",
		Source: database.LogSourceProvisioner,
		Level:  database.LogLevelInfo,
		Output: "test dry-run log output",
	})

	tests := []struct {
		name                string
		queryParams         string
		expectedStatus      int
		expectedContentType string
		checkBody           func(t *testing.T, body string)
	}{
		{
			name:                "JSON",
			queryParams:         "",
			expectedStatus:      http.StatusOK,
			expectedContentType: "application/json",
			checkBody: func(t *testing.T, body string) {
				assert.NotEmpty(t, body)
			},
		},
		{
			name:                "Text",
			queryParams:         "?format=text",
			expectedStatus:      http.StatusOK,
			expectedContentType: "text/plain",
			checkBody: func(t *testing.T, body string) {
				expected := db2sdk.ProvisionerJobLog(jl).Text()
				assert.Contains(t, body, expected)
			},
		},
		{
			name:           "InvalidFormat",
			queryParams:    "?format=invalid",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid format")
			},
		},
		{
			name:           "TextWithFollowFails",
			queryParams:    "?format=text&follow",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "not supported with follow mode")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := testutil.Context(t, testutil.WaitLong)

			urlPath := fmt.Sprintf("/api/v2/templateversions/%s/dry-run/%s/logs%s", tv.TemplateVersion.ID, dryRunJob.ID, tt.queryParams)

			res, err := client.Request(ctx, http.MethodGet, urlPath, nil)
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, tt.expectedStatus, res.StatusCode)
			if tt.expectedContentType != "" {
				require.Contains(t, res.Header.Get("Content-Type"), tt.expectedContentType)
			}

			if assert.NotNil(t, tt.checkBody) {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				tt.checkBody(t, string(body))
			}
		})
	}
}

// TestPaginatedTemplateVersions creates a list of template versions and paginate.
func TestPaginatedTemplateVersions(t *testing.T) {
	t.Parallel()

	client := nicloudtest.New(t, nil)
	user := nicloudtest.CreateFirstUser(t, client)
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
	template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	// Populate database with template versions.
	total := 9
	eg, egCtx := errgroup.WithContext(ctx)
	templateVersionIDs := make([]uuid.UUID, total)
	data, err := echo.Tar(nil)
	require.NoError(t, err)
	file, err := client.Upload(egCtx, nicloudsdk.ContentTypeTar, bytes.NewReader(data))
	require.NoError(t, err)
	for i := 0; i < total; i++ {
		eg.Go(func() error {
			templateVersion, err := client.CreateTemplateVersion(egCtx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
				Name:          uuid.NewString(),
				TemplateID:    template.ID,
				FileID:        file.ID,
				StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
				Provisioner:   nicloudsdk.ProvisionerTypeEcho,
			})
			if err != nil {
				return err
			}
			templateVersionIDs[i] = templateVersion.ID
			return nil
		})
	}
	err = eg.Wait()
	require.NoError(t, err, "create templates failed")

	templateVersions, err := client.TemplateVersionsByTemplate(ctx,
		nicloudsdk.TemplateVersionsByTemplateRequest{
			TemplateID: template.ID,
		},
	)
	require.NoError(t, err)
	require.Len(t, templateVersions, 10, "wrong number of template versions created")

	type args struct {
		pagination nicloudsdk.Pagination
	}
	tests := []struct {
		name          string
		args          args
		want          []nicloudsdk.TemplateVersion
		expectedError string
	}{
		{
			name: "Single result",
			args: args{pagination: nicloudsdk.Pagination{Limit: 1}},
			want: templateVersions[:1],
		},
		{
			name: "Single result, second page",
			args: args{pagination: nicloudsdk.Pagination{Limit: 1, Offset: 1}},
			want: templateVersions[1:2],
		},
		{
			name: "Last two results",
			args: args{pagination: nicloudsdk.Pagination{Limit: 2, Offset: 8}},
			want: templateVersions[8:10],
		},
		{
			name: "AfterID returns next two results",
			args: args{pagination: nicloudsdk.Pagination{Limit: 2, AfterID: templateVersions[1].ID}},
			want: templateVersions[2:4],
		},
		{
			name: "No result after last AfterID",
			args: args{pagination: nicloudsdk.Pagination{Limit: 2, AfterID: templateVersions[9].ID}},
			want: []nicloudsdk.TemplateVersion{},
		},
		{
			name: "No result after last Offset",
			args: args{pagination: nicloudsdk.Pagination{Limit: 2, Offset: 10}},
			want: []nicloudsdk.TemplateVersion{},
		},
		{
			name:          "After_id does not exist",
			args:          args{pagination: nicloudsdk.Pagination{AfterID: uuid.New()}},
			expectedError: "does not exist",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
			defer cancel()

			got, err := client.TemplateVersionsByTemplate(ctx, nicloudsdk.TemplateVersionsByTemplateRequest{
				TemplateID: template.ID,
				Pagination: tt.args.pagination,
			})
			if tt.expectedError != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTemplateVersionByOrganizationTemplateAndName(t *testing.T) {
	t.Parallel()

	// Shared instance — see TestTemplateVersionByName for rationale.
	client := nicloudtest.New(t, nil)
	user := nicloudtest.CreateFirstUser(t, client)
	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.TemplateVersionByOrganizationAndName(ctx, user.OrganizationID, template.Name, "nothing")
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("Found", func(t *testing.T) {
		t.Parallel()
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.TemplateVersionByOrganizationAndName(ctx, user.OrganizationID, template.Name, version.Name)
		require.NoError(t, err)
	})
}

func TestPreviousTemplateVersion(t *testing.T) {
	t.Parallel()
	t.Run("Previous version not found", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)

		// Create two templates to be sure it is not returning a previous version
		// from another template
		templateAVersion1 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, templateAVersion1.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, templateAVersion1.ID)
		// Create two versions for template B so we can verify that requesting
		// the previous version of the first version returns nil.
		templateBVersion1 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		templateB := nicloudtest.CreateTemplate(t, client, user.OrganizationID, templateBVersion1.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, templateBVersion1.ID)
		templateBVersion2 := nicloudtest.UpdateTemplateVersion(t, client, user.OrganizationID, nil, templateB.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, templateBVersion2.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.PreviousTemplateVersion(ctx, user.OrganizationID, templateB.Name, templateBVersion1.Name)
		require.ErrorIs(t, err, nicloudsdk.ErrNoPreviousVersion)
	})

	t.Run("Previous version found", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)

		// Create two templates to be sure it is not returning a previous version
		// from another template
		templateAVersion1 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, templateAVersion1.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, templateAVersion1.ID)
		// Create two versions for the template B so we can try to get the previous
		// version of version 2
		templateBVersion1 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		templateB := nicloudtest.CreateTemplate(t, client, user.OrganizationID, templateBVersion1.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, templateBVersion1.ID)
		templateBVersion2 := nicloudtest.UpdateTemplateVersion(t, client, user.OrganizationID, nil, templateB.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, templateBVersion2.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		result, err := client.PreviousTemplateVersion(ctx, user.OrganizationID, templateB.Name, templateBVersion2.Name)
		require.NoError(t, err)
		require.Equal(t, templateBVersion1.ID, result.ID)
	})
}

func TestStarterTemplates(t *testing.T) {
	t.Parallel()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		_ = nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		ex, err := client.StarterTemplates(ctx)
		require.NoError(t, err)
		ls, err := examples.List()
		require.NoError(t, err)
		require.EqualValues(t, ls, ex)
	})
}

func TestTemplateVersionVariables(t *testing.T) {
	t.Parallel()

	createEchoResponses := func(templateVariables []*proto.TemplateVariable) *echo.Responses {
		return &echo.Responses{
			Parse: []*proto.Response{
				{
					Type: &proto.Response_Parse{
						Parse: &proto.ParseComplete{
							TemplateVariables: templateVariables,
						},
					},
				},
			},
			ProvisionPlan:  echo.PlanComplete,
			ProvisionApply: echo.ApplyComplete,
		}
	}

	t.Run("Pass value for required variable", func(t *testing.T) {
		t.Parallel()

		templateVariables := []*proto.TemplateVariable{
			{
				Name:        "first_variable",
				Description: "This is the first variable",
				Type:        "string",
				Required:    true,
			},
		}
		const firstVariableValue = "foobar"

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID,
			createEchoResponses(templateVariables),
			func(ctvr *nicloudsdk.CreateTemplateVersionRequest) {
				ctvr.UserVariableValues = []nicloudsdk.VariableValue{
					{
						Name:  templateVariables[0].Name,
						Value: firstVariableValue,
					},
				}
			},
		)
		templateVersion := nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		// As user passed the value for the first parameter, the job will succeed.
		require.Empty(t, templateVersion.Job.Error)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
		defer cancel()

		actualVariables, err := client.TemplateVersionVariables(ctx, templateVersion.ID)
		require.NoError(t, err)

		require.Len(t, actualVariables, 1)
		require.Equal(t, templateVariables[0].Name, actualVariables[0].Name)
		require.Equal(t, templateVariables[0].Description, actualVariables[0].Description)
		require.Equal(t, templateVariables[0].Type, actualVariables[0].Type)
		require.Equal(t, templateVariables[0].DefaultValue, actualVariables[0].DefaultValue)
		require.Equal(t, templateVariables[0].Required, actualVariables[0].Required)
		require.Equal(t, templateVariables[0].Sensitive, actualVariables[0].Sensitive)
		require.Equal(t, firstVariableValue, actualVariables[0].Value)
	})

	t.Run("Missing value for required variable", func(t *testing.T) {
		t.Parallel()

		templateVariables := []*proto.TemplateVariable{
			{
				Name:        "first_variable",
				Description: "This is the first variable",
				Type:        "string",
				Required:    true,
			},
			{
				Name:         "second_variable",
				Description:  "This is the second variable",
				DefaultValue: "123",
				Type:         "number",
			},
		}

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, createEchoResponses(templateVariables))
		templateVersion := nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		// As the first variable is marked as required and misses the default value,
		// the job will fail, but will populate the template_version_variables table with existing variables.
		require.Contains(t, templateVersion.Job.Error, "required template variables need values")

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
		defer cancel()

		actualVariables, err := client.TemplateVersionVariables(ctx, templateVersion.ID)
		require.NoError(t, err)

		require.Len(t, actualVariables, 2)
		for i := range templateVariables {
			require.Equal(t, templateVariables[i].Name, actualVariables[i].Name)
			require.Equal(t, templateVariables[i].Description, actualVariables[i].Description)
			require.Equal(t, templateVariables[i].Type, actualVariables[i].Type)
			require.Equal(t, templateVariables[i].DefaultValue, actualVariables[i].DefaultValue)
			require.Equal(t, templateVariables[i].Required, actualVariables[i].Required)
			require.Equal(t, templateVariables[i].Sensitive, actualVariables[i].Sensitive)
		}

		require.Equal(t, "", actualVariables[0].Value)
		require.Equal(t, templateVariables[1].DefaultValue, actualVariables[1].Value)
	})

	t.Run("Redact sensitive variables", func(t *testing.T) {
		t.Parallel()

		templateVariables := []*proto.TemplateVariable{
			{
				Name:        "first_variable",
				Description: "This is the first variable",
				Type:        "string",
				Required:    true,
				Sensitive:   true,
			},
		}
		const firstVariableValue = "foobar"

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID,
			createEchoResponses(templateVariables),
			func(ctvr *nicloudsdk.CreateTemplateVersionRequest) {
				ctvr.UserVariableValues = []nicloudsdk.VariableValue{
					{
						Name:  templateVariables[0].Name,
						Value: firstVariableValue,
					},
				}
			},
		)
		templateVersion := nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		// As user passed the value for the first parameter, the job will succeed.
		require.Empty(t, templateVersion.Job.Error)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
		defer cancel()

		actualVariables, err := client.TemplateVersionVariables(ctx, templateVersion.ID)
		require.NoError(t, err)

		require.Len(t, actualVariables, 1)
		require.Equal(t, templateVariables[0].Name, actualVariables[0].Name)
		require.Equal(t, templateVariables[0].Description, actualVariables[0].Description)
		require.Equal(t, templateVariables[0].Type, actualVariables[0].Type)
		require.Equal(t, templateVariables[0].Required, actualVariables[0].Required)
		require.Equal(t, templateVariables[0].Sensitive, actualVariables[0].Sensitive)
		require.Equal(t, "*redacted*", actualVariables[0].DefaultValue)
		require.Equal(t, "*redacted*", actualVariables[0].Value)
	})
}

func TestTemplateVersionPatch(t *testing.T) {
	t.Parallel()

	// Single instance shared across all 9 sub-tests. Each sub-test
	// creates its own template version(s) and template(s) with
	// unique IDs so parallel execution is safe.
	client := nicloudtest.New(t, nil)
	user := nicloudtest.CreateFirstUser(t, client)
	t.Run("Update the name", func(t *testing.T) {
		t.Parallel()
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		const newName = "new-name"
		updatedVersion, err := client.UpdateTemplateVersion(ctx, version.ID, nicloudsdk.PatchTemplateVersionRequest{
			Name: newName,
		})

		require.NoError(t, err)
		assert.Equal(t, newName, updatedVersion.Name)
		assert.NotEqual(t, updatedVersion.Name, version.Name)
	})

	t.Run("Update the message", func(t *testing.T) {
		t.Parallel()
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil, func(req *nicloudsdk.CreateTemplateVersionRequest) {
			req.Message = "Example message"
		})
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		wantMessage := "Updated message"
		updatedVersion, err := client.UpdateTemplateVersion(ctx, version.ID, nicloudsdk.PatchTemplateVersionRequest{
			Message: &wantMessage,
		})

		require.NoError(t, err)
		assert.Equal(t, wantMessage, updatedVersion.Message)
	})

	t.Run("Remove the message", func(t *testing.T) {
		t.Parallel()
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil, func(req *nicloudsdk.CreateTemplateVersionRequest) {
			req.Message = "Example message"
		})
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		wantMessage := ""
		updatedVersion, err := client.UpdateTemplateVersion(ctx, version.ID, nicloudsdk.PatchTemplateVersionRequest{
			Message: &wantMessage,
		})

		require.NoError(t, err)
		assert.Equal(t, wantMessage, updatedVersion.Message)
	})

	t.Run("Keep the message", func(t *testing.T) {
		t.Parallel()
		wantMessage := "Example message"
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil, func(req *nicloudsdk.CreateTemplateVersionRequest) {
			req.Message = wantMessage
		})
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		t.Log(version.Message)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		updatedVersion, err := client.UpdateTemplateVersion(ctx, version.ID, nicloudsdk.PatchTemplateVersionRequest{
			Message: nil,
		})

		require.NoError(t, err)
		assert.Equal(t, wantMessage, updatedVersion.Message)
	})

	t.Run("Use the same name if a new name is not passed", func(t *testing.T) {
		t.Parallel()
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		updatedVersion, err := client.UpdateTemplateVersion(ctx, version.ID, nicloudsdk.PatchTemplateVersionRequest{})
		require.NoError(t, err)
		assert.Equal(t, version.Name, updatedVersion.Name)
	})

	t.Run("Use the same name for two different templates", func(t *testing.T) {
		t.Parallel()
		version1 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, version1.ID)
		version2 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.CreateTemplate(t, client, user.OrganizationID, version2.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		const commonTemplateVersionName = "common-template-version-name"
		updatedVersion1, err := client.UpdateTemplateVersion(ctx, version1.ID, nicloudsdk.PatchTemplateVersionRequest{
			Name: commonTemplateVersionName,
		})
		require.NoError(t, err)

		updatedVersion2, err := client.UpdateTemplateVersion(ctx, version2.ID, nicloudsdk.PatchTemplateVersionRequest{
			Name: commonTemplateVersionName,
		})
		require.NoError(t, err)

		assert.NotEqual(t, updatedVersion1.ID, updatedVersion2.ID)
		assert.Equal(t, updatedVersion1.Name, updatedVersion2.Name)
	})

	t.Run("Use the same name for two versions for the same templates", func(t *testing.T) {
		t.Parallel()
		version1 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil, func(ctvr *nicloudsdk.CreateTemplateVersionRequest) {
			ctvr.Name = "v1"
		})
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version1.ID)

		version2 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil, func(ctvr *nicloudsdk.CreateTemplateVersionRequest) {
			ctvr.Name = "v2"
			ctvr.TemplateID = template.ID
		})

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()
		_, err := client.UpdateTemplateVersion(ctx, version2.ID, nicloudsdk.PatchTemplateVersionRequest{
			Name: version1.Name,
		})
		require.Error(t, err)
	})

	t.Run("Rename the unassigned template", func(t *testing.T) {
		t.Parallel()
		version1 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		const commonTemplateVersionName = "common-template-version-name"
		updatedVersion1, err := client.UpdateTemplateVersion(ctx, version1.ID, nicloudsdk.PatchTemplateVersionRequest{
			Name: commonTemplateVersionName,
		})
		require.NoError(t, err)
		assert.Equal(t, commonTemplateVersionName, updatedVersion1.Name)
	})

	t.Run("Use incorrect template version name", func(t *testing.T) {
		t.Parallel()
		version1 := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		const incorrectTemplateVersionName = "incorrect/name"
		_, err := client.UpdateTemplateVersion(ctx, version1.ID, nicloudsdk.PatchTemplateVersionRequest{
			Name: incorrectTemplateVersionName,
		})
		require.Error(t, err)
	})
}

func TestTemplateVersionParameters_Order(t *testing.T) {
	t.Parallel()

	const (
		firstParameterName  = "first_parameter"
		firstParameterType  = "string"
		firstParameterValue = "aaa"
		// no order

		secondParameterName  = "Second_parameter"
		secondParameterType  = "number"
		secondParameterValue = "2"
		secondParameterOrder = 3

		thirdParameterName  = "third_parameter"
		thirdParameterType  = "number"
		thirdParameterValue = "3"
		thirdParameterOrder = 3

		fourthParameterName  = "Fourth_parameter"
		fourthParameterType  = "number"
		fourthParameterValue = "3"
		fourthParameterOrder = 2

		fifthParameterName  = "Fifth_parameter"
		fifthParameterType  = "string"
		fifthParameterValue = "aaa"
		// no order
	)

	client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
	user := nicloudtest.CreateFirstUser(t, client)
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
		Parse: echo.ParseComplete,
		ProvisionGraph: []*proto.Response{
			{
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						Parameters: []*proto.RichParameter{
							{
								Name: firstParameterName,
								Type: firstParameterType,
								// No order
							},
							{
								Name:  secondParameterName,
								Type:  secondParameterType,
								Order: secondParameterOrder,
							},
							{
								Name:  thirdParameterName,
								Type:  thirdParameterType,
								Order: thirdParameterOrder,
							},
							{
								Name:  fourthParameterName,
								Type:  fourthParameterType,
								Order: fourthParameterOrder,
							},
							{
								Name: fifthParameterName,
								Type: fifthParameterType,
								// No order
							},
						},
					},
				},
			},
		},
		ProvisionApply: echo.ApplyComplete,
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	templateRichParameters, err := client.TemplateVersionRichParameters(ctx, version.ID)
	require.NoError(t, err)
	require.Len(t, templateRichParameters, 5)
	require.Equal(t, fifthParameterName, templateRichParameters[0].Name)
	require.Equal(t, firstParameterName, templateRichParameters[1].Name)
	require.Equal(t, fourthParameterName, templateRichParameters[2].Name)
	require.Equal(t, secondParameterName, templateRichParameters[3].Name)
	require.Equal(t, thirdParameterName, templateRichParameters[4].Name)
}

func TestTemplateArchiveVersions(t *testing.T) {
	t.Parallel()

	ownerClient := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
	owner := nicloudtest.CreateFirstUser(t, ownerClient)
	client, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())

	var totalVersions int
	// Create a template to archive
	initialVersion := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
	totalVersions++
	template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, initialVersion.ID)

	allFailed := make([]uuid.UUID, 0)
	expArchived := make([]uuid.UUID, 0)
	// create some failed versions
	for i := 0; i < 2; i++ {
		failed := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionPlan:  echo.PlanFailed,
			ProvisionApply: echo.ApplyFailed,
			ProvisionInit:  echo.InitComplete,
		}, func(req *nicloudsdk.CreateTemplateVersionRequest) {
			req.TemplateID = template.ID
		})
		allFailed = append(allFailed, failed.ID)
		totalVersions++
	}

	// Create some unused versions
	for i := 0; i < 2; i++ {
		unused := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil, func(req *nicloudsdk.CreateTemplateVersionRequest) {
			req.TemplateID = template.ID
		})
		expArchived = append(expArchived, unused.ID)
		totalVersions++
	}

	// Create some used template versions
	for i := 0; i < 2; i++ {
		used := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil, func(req *nicloudsdk.CreateTemplateVersionRequest) {
			req.TemplateID = template.ID
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, used.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, uuid.Nil, func(request *nicloudsdk.CreateWorkspaceRequest) {
			request.TemplateVersionID = used.ID
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		totalVersions++
	}

	ctx := testutil.Context(t, testutil.WaitMedium)
	versions, err := client.TemplateVersionsByTemplate(ctx, nicloudsdk.TemplateVersionsByTemplateRequest{
		TemplateID: template.ID,
		Pagination: nicloudsdk.Pagination{
			Limit: 100,
		},
	})
	require.NoError(t, err, "fetch all versions")
	require.Len(t, versions, totalVersions, "total versions")

	// Archive failed versions
	archiveFailed, err := client.ArchiveTemplateVersions(ctx, template.ID, false)
	require.NoError(t, err, "archive failed versions")
	require.ElementsMatch(t, archiveFailed.ArchivedIDs, allFailed, "all failed versions archived")

	remaining, err := client.TemplateVersionsByTemplate(ctx, nicloudsdk.TemplateVersionsByTemplateRequest{
		TemplateID: template.ID,
		Pagination: nicloudsdk.Pagination{
			Limit: 100,
		},
	})
	require.NoError(t, err, "fetch all non-failed versions")
	require.Len(t, remaining, totalVersions-len(allFailed), "remaining non-failed versions")

	// Try archiving "All" unused templates
	archived, err := client.ArchiveTemplateVersions(ctx, template.ID, true)
	require.NoError(t, err, "archive versions")
	require.ElementsMatch(t, archived.ArchivedIDs, expArchived, "all expected versions archived")

	remaining, err = client.TemplateVersionsByTemplate(ctx, nicloudsdk.TemplateVersionsByTemplateRequest{
		TemplateID: template.ID,
		Pagination: nicloudsdk.Pagination{
			Limit: 100,
		},
	})
	require.NoError(t, err, "fetch all versions")
	require.Len(t, remaining, totalVersions-len(expArchived)-len(allFailed), "remaining versions")

	// Unarchive a version
	err = client.SetArchiveTemplateVersion(ctx, expArchived[0], false)
	require.NoError(t, err, "unarchive a version")

	tv, err := client.TemplateVersion(ctx, expArchived[0])
	require.NoError(t, err, "fetch version")
	require.False(t, tv.Archived, "expect unarchived")

	// Check the remaining again
	remaining, err = client.TemplateVersionsByTemplate(ctx, nicloudsdk.TemplateVersionsByTemplateRequest{
		TemplateID: template.ID,
		Pagination: nicloudsdk.Pagination{
			Limit: 100,
		},
	})
	require.NoError(t, err, "fetch all versions")
	require.Len(t, remaining, totalVersions-len(expArchived)-len(allFailed)+1, "remaining versions")
}

func TestTemplateVersionHasExternalAgent(t *testing.T) {
	t.Parallel()

	client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
	user := nicloudtest.CreateFirstUser(t, client)

	ctx := testutil.Context(t, testutil.WaitMedium)
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
		Parse: echo.ParseComplete,
		ProvisionGraph: []*proto.Response{
			{
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						Resources: []*proto.Resource{
							{
								Name: "example",
								Type: "ni_external_agent",
							},
						},
						HasExternalAgents: true,
					},
				},
			},
		},
		ProvisionApply: echo.ApplyComplete,
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

	version, err := client.TemplateVersion(ctx, version.ID)
	require.NoError(t, err)
	require.True(t, version.HasExternalAgent)
}
