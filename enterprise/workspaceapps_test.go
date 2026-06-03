package enterprise_test

import (
	"net"
	"testing"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloud/workspaceapps/apptest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/coder/serpent"
)

func TestWorkspaceApps(t *testing.T) {
	t.Parallel()

	apptest.Run(t, true, func(t *testing.T, opts *apptest.DeploymentOptions) *apptest.Deployment {
		deploymentValues := nicloudtest.DeploymentValues(t)
		deploymentValues.DisablePathApps = serpent.Bool(opts.DisablePathApps)
		deploymentValues.Dangerous.AllowPathAppSharing = serpent.Bool(opts.DangerousAllowPathAppSharing)
		deploymentValues.Dangerous.AllowPathAppSiteOwnerAccess = serpent.Bool(opts.DangerousAllowPathAppSiteOwnerAccess)
		deploymentValues.Experiments = []string{
			"*",
		}

		if opts.DisableSubdomainApps {
			opts.AppHost = ""
		}

		flushStatsCollectorCh := make(chan chan<- struct{}, 1)
		opts.StatsCollectorOptions.Flush = flushStatsCollectorCh
		flushStats := func() {
			flushStatsCollectorDone := make(chan struct{}, 1)
			flushStatsCollectorCh <- flushStatsCollectorDone
			<-flushStatsCollectorDone
		}

		db, pubsub := dbtestutil.NewDB(t)

		client, _, _, user := nicloudenttest.NewWithAPI(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues:         deploymentValues,
				AppHostname:              opts.AppHost,
				IncludeProvisionerDaemon: true,
				RealIPConfig: &httpmw.RealIPConfig{
					TrustedOrigins: []*net.IPNet{{
						IP:   net.ParseIP("127.0.0.1"),
						Mask: net.CIDRMask(8, 32),
					}},
					TrustedHeaders: []string{
						"CF-Connecting-IP",
					},
				},
				WorkspaceAppsStatsCollectorOptions: opts.StatsCollectorOptions,
				Database:                           db,
				Pubsub:                             pubsub,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		return &apptest.Deployment{
			Options:        opts,
			SDKClient:      client,
			FirstUser:      user,
			PathAppBaseURL: client.URL,
			FlushStats:     flushStats,
		}
	})
}
