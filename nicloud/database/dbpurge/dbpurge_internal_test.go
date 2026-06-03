package dbpurge

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/coder/quartz"
)

func TestDBPurgeAuthorization(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitShort)
	rawDB, _ := dbtestutil.NewDB(t)

	authz := rbac.NewAuthorizer(prometheus.NewRegistry())
	db := dbauthz.New(rawDB, authz, testutil.Logger(t), nicloudtest.AccessControlStorePointer())

	ctx = dbauthz.AsDBPurge(ctx)

	clk := quartz.NewMock(t)
	now := time.Date(2025, 1, 15, 7, 30, 0, 0, time.UTC)
	clk.Set(now)

	vals := &nicloudsdk.DeploymentValues{ /* same vals as before */ }

	inst := &instance{
		logger: testutil.Logger(t),
		vals:   vals,
		clk:    clk,
		// metrics can be nil in this test
	}

	err := inst.purgeTick(ctx, db, now)
	require.NoError(t, err)
}
