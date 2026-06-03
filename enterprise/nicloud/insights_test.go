package nicloud_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestTemplateInsightsWithTemplateAdminACL(t *testing.T) {
	t.Parallel()

	y, m, d := time.Now().UTC().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	type test struct {
		interval nicloudsdk.InsightsReportInterval
	}

	tests := []test{
		{nicloudsdk.InsightsReportIntervalDay},
		{""},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("with interval=%q", tt.interval), func(t *testing.T) {
			t.Parallel()

			client, admin := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
				},
			}})
			templateAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID, rbac.RoleTemplateAdmin())

			version := nicloudtest.CreateTemplateVersion(t, client, admin.OrganizationID, nil)
			template := nicloudtest.CreateTemplate(t, client, admin.OrganizationID, version.ID)

			regular, regularUser := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID)

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
			defer cancel()

			err := templateAdminClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
				UserPerms: map[string]nicloudsdk.TemplateRole{
					regularUser.ID.String(): nicloudsdk.TemplateRoleAdmin,
				},
			})
			require.NoError(t, err)

			_, err = regular.TemplateInsights(ctx, nicloudsdk.TemplateInsightsRequest{
				StartTime:   today.AddDate(0, 0, -1),
				EndTime:     today,
				TemplateIDs: []uuid.UUID{template.ID},
			})
			require.NoError(t, err)
		})
	}
}

func TestTemplateInsightsWithRole(t *testing.T) {
	t.Parallel()

	y, m, d := time.Now().UTC().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	type test struct {
		interval nicloudsdk.InsightsReportInterval
		role     rbac.RoleIdentifier
		allowed  bool
	}

	tests := []test{
		{nicloudsdk.InsightsReportIntervalDay, rbac.RoleTemplateAdmin(), true},
		{"", rbac.RoleTemplateAdmin(), true},
		{nicloudsdk.InsightsReportIntervalDay, rbac.RoleAuditor(), true},
		{"", rbac.RoleAuditor(), true},
		{nicloudsdk.InsightsReportIntervalDay, rbac.RoleUserAdmin(), false},
		{"", rbac.RoleUserAdmin(), false},
		{nicloudsdk.InsightsReportIntervalDay, rbac.RoleMember(), false},
		{"", rbac.RoleMember(), false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("with interval=%q role=%q", tt.interval, tt.role), func(t *testing.T) {
			t.Parallel()

			client, admin := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
				},
			}})
			version := nicloudtest.CreateTemplateVersion(t, client, admin.OrganizationID, nil)
			template := nicloudtest.CreateTemplate(t, client, admin.OrganizationID, version.ID)

			aud, _ := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID, tt.role)

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
			defer cancel()

			_, err := aud.TemplateInsights(ctx, nicloudsdk.TemplateInsightsRequest{
				StartTime:   today.AddDate(0, 0, -1),
				EndTime:     today,
				TemplateIDs: []uuid.UUID{template.ID},
			})
			if tt.allowed {
				require.NoError(t, err)
			} else {
				var sdkErr *nicloudsdk.Error
				require.ErrorAs(t, err, &sdkErr)
				require.Equal(t, sdkErr.StatusCode(), http.StatusNotFound)
			}
		})
	}
}
