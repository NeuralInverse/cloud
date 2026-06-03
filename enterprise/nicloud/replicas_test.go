package nicloud_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/workspacesdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestReplicas(t *testing.T) {
	t.Parallel()

	t.Run("ErrorWithoutLicense", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitLong)
		// This will error because replicas are expected to instantly report
		// errors when the license is not present.
		db, pubsub := dbtestutil.NewDB(t)
		firstClient, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				Database:                 db,
				Pubsub:                   pubsub,
			},
			DontAddLicense:          true,
			ReplicaErrorGracePeriod: time.Nanosecond,
		})
		secondClient, _, secondAPI, _ := nicloudenttest.NewWithAPI(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				Database: db,
				Pubsub:   pubsub,
			},
			DontAddFirstUser:        true,
			DontAddLicense:          true,
			ReplicaErrorGracePeriod: time.Nanosecond,
		})
		secondClient.SetSessionToken(firstClient.SessionToken())

		testutil.Eventually(ctx, t, func(ctx context.Context) (done bool) {
			ents, err := secondClient.Entitlements(ctx)
			return assert.NoError(t, err, "unexpected error from secondClient.Entitlements") &&
				len(ents.Errors) == 1
		}, testutil.IntervalFast)
		_ = secondAPI.Close()

		testutil.Eventually(ctx, t, func(ctx context.Context) (done bool) {
			ents, err := firstClient.Entitlements(ctx)
			return assert.NoError(t, err, "unexpected error from firstClient.Entitlements") &&
				len(ents.Warnings) == 0
		}, testutil.IntervalFast)
	})
	t.Run("DoesNotErrorBeforeGrace", func(t *testing.T) {
		t.Parallel()
		db, pubsub := dbtestutil.NewDB(t)
		firstClient, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				Database:                 db,
				Pubsub:                   pubsub,
			},
			DontAddLicense: true,
		})
		secondClient, _, secondAPI, _ := nicloudenttest.NewWithAPI(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				Database: db,
				Pubsub:   pubsub,
			},
			DontAddFirstUser: true,
			DontAddLicense:   true,
		})
		secondClient.SetSessionToken(firstClient.SessionToken())
		ents, err := secondClient.Entitlements(context.Background())
		require.NoError(t, err)
		require.Len(t, ents.Errors, 0)
		_ = secondAPI.Close()

		ents, err = firstClient.Entitlements(context.Background())
		require.NoError(t, err)
		require.Len(t, ents.Errors, 0)
	})
	t.Run("ConnectAcrossMultiple", func(t *testing.T) {
		t.Parallel()
		db, pubsub := dbtestutil.NewDB(t)
		firstClient, firstUser := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				Database:                 db,
				Pubsub:                   pubsub,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureHighAvailability: 1,
				},
			},
		})

		secondClient, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				Database: db,
				Pubsub:   pubsub,
			},
			DontAddLicense:   true,
			DontAddFirstUser: true,
		})
		secondClient.SetSessionToken(firstClient.SessionToken())
		replicas, err := secondClient.Replicas(context.Background())
		require.NoError(t, err)
		require.Len(t, replicas, 2)

		r := setupWorkspaceAgent(t, firstClient, firstUser, 0)
		conn, err := workspacesdk.New(secondClient).
			DialAgent(context.Background(), r.sdkAgent.ID, &workspacesdk.DialAgentOptions{
				BlockEndpoints: true,
				Logger:         testutil.Logger(t),
			})
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			ctx, cancelFunc := context.WithTimeout(context.Background(), testutil.WaitShort)
			defer cancelFunc()
			_, _, _, err = conn.Ping(ctx)
			return err == nil
		}, testutil.WaitLong, testutil.IntervalFast)
		_ = conn.Close()
	})
	t.Run("ConnectAcrossMultipleTLS", func(t *testing.T) {
		t.Parallel()
		db, pubsub := dbtestutil.NewDB(t)
		certificates := []tls.Certificate{testutil.GenerateTLSCertificate(t, "localhost")}
		firstClient, firstUser := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				Database:                 db,
				Pubsub:                   pubsub,
				TLSCertificates:          certificates,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureHighAvailability: 1,
				},
			},
		})

		secondClient, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				Database:        db,
				Pubsub:          pubsub,
				TLSCertificates: certificates,
			},
			DontAddFirstUser: true,
			DontAddLicense:   true,
		})
		secondClient.SetSessionToken(firstClient.SessionToken())
		replicas, err := secondClient.Replicas(context.Background())
		require.NoError(t, err)
		require.Len(t, replicas, 2)

		r := setupWorkspaceAgent(t, firstClient, firstUser, 0)
		conn, err := workspacesdk.New(secondClient).
			DialAgent(context.Background(), r.sdkAgent.ID, &workspacesdk.DialAgentOptions{
				BlockEndpoints: true,
				Logger:         testutil.Logger(t).Named("client"),
			})
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			ctx, cancelFunc := context.WithTimeout(context.Background(), testutil.IntervalSlow)
			defer cancelFunc()
			_, _, _, err = conn.Ping(ctx)
			return err == nil
		}, testutil.WaitLong, testutil.IntervalFast)
		_ = conn.Close()
		replicas, err = secondClient.Replicas(context.Background())
		require.NoError(t, err)
		require.Len(t, replicas, 2)
		for _, replica := range replicas {
			require.Empty(t, replica.Error)
		}
	})
}
