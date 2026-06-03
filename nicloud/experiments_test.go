package nicloud_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func Test_Experiments(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		cfg := nicloudtest.DeploymentValues(t)
		client := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues: cfg,
		})
		_ = nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		experiments, err := client.Experiments(ctx)
		require.NoError(t, err)
		require.NotNil(t, experiments)
		require.Empty(t, experiments)
		require.False(t, experiments.Enabled("foo"))
	})

	t.Run("multiple features", func(t *testing.T) {
		t.Parallel()
		cfg := nicloudtest.DeploymentValues(t)
		cfg.Experiments = []string{"foo", "BAR"}
		client := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues: cfg,
		})
		_ = nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		experiments, err := client.Experiments(ctx)
		require.NoError(t, err)
		require.NotNil(t, experiments)
		// Should be lower-cased.
		require.ElementsMatch(t, []nicloudsdk.Experiment{"foo", "bar"}, experiments)
		require.True(t, experiments.Enabled("foo"))
		require.True(t, experiments.Enabled("bar"))
		require.False(t, experiments.Enabled("baz"))
	})

	t.Run("wildcard", func(t *testing.T) {
		t.Parallel()
		cfg := nicloudtest.DeploymentValues(t)
		cfg.Experiments = []string{"*"}
		client := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues: cfg,
		})
		_ = nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		experiments, err := client.Experiments(ctx)
		require.NoError(t, err)
		require.NotNil(t, experiments)
		require.ElementsMatch(t, nicloudsdk.ExperimentsSafe, experiments)
		for _, ex := range nicloudsdk.ExperimentsSafe {
			require.True(t, experiments.Enabled(ex))
		}
		require.False(t, experiments.Enabled("danger"))
	})

	t.Run("alternate wildcard with manual opt-in", func(t *testing.T) {
		t.Parallel()
		cfg := nicloudtest.DeploymentValues(t)
		cfg.Experiments = []string{"*", "dAnGeR"}
		client := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues: cfg,
		})
		_ = nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		experiments, err := client.Experiments(ctx)
		require.NoError(t, err)
		require.NotNil(t, experiments)
		require.ElementsMatch(t, append(nicloudsdk.ExperimentsSafe, "danger"), experiments)
		for _, ex := range nicloudsdk.ExperimentsSafe {
			require.True(t, experiments.Enabled(ex))
		}
		require.True(t, experiments.Enabled("danger"))
		require.False(t, experiments.Enabled("herebedragons"))
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Parallel()
		cfg := nicloudtest.DeploymentValues(t)
		cfg.Experiments = []string{"*"}
		client := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues: cfg,
		})
		// Explicitly omit creating a user so we're unauthorized.
		// _ = nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.Experiments(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, httpmw.SignedOutErrorMessage)
	})

	t.Run("available experiments", func(t *testing.T) {
		t.Parallel()
		cfg := nicloudtest.DeploymentValues(t)
		client := nicloudtest.New(t, &nicloudtest.Options{
			DeploymentValues: cfg,
		})
		_ = nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		experiments, err := client.SafeExperiments(ctx)
		require.NoError(t, err)
		require.NotNil(t, experiments)
		require.ElementsMatch(t, nicloudsdk.ExperimentsSafe, experiments.Safe)
	})
}
