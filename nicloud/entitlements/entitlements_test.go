package entitlements_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/entitlements"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestModify(t *testing.T) {
	t.Parallel()

	set := entitlements.New()
	require.False(t, set.Enabled(nicloudsdk.FeatureMultipleOrganizations))

	set.Modify(func(entitlements *nicloudsdk.Entitlements) {
		entitlements.Features[nicloudsdk.FeatureMultipleOrganizations] = nicloudsdk.Feature{
			Enabled:     true,
			Entitlement: nicloudsdk.EntitlementEntitled,
		}
	})
	require.True(t, set.Enabled(nicloudsdk.FeatureMultipleOrganizations))
}

func TestAllowRefresh(t *testing.T) {
	t.Parallel()

	now := time.Now()
	set := entitlements.New()
	set.Modify(func(entitlements *nicloudsdk.Entitlements) {
		entitlements.RefreshedAt = now
	})

	ok, wait := set.AllowRefresh(now)
	require.False(t, ok)
	require.InDelta(t, time.Minute.Seconds(), wait.Seconds(), 5)

	set.Modify(func(entitlements *nicloudsdk.Entitlements) {
		entitlements.RefreshedAt = now.Add(time.Minute * -2)
	})

	ok, wait = set.AllowRefresh(now)
	require.True(t, ok)
	require.Equal(t, time.Duration(0), wait)
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitShort)

	set := entitlements.New()
	require.False(t, set.Enabled(nicloudsdk.FeatureMultipleOrganizations))
	fetchStarted := make(chan struct{})
	firstDone := make(chan struct{})
	errCh := make(chan error, 2)
	go func() {
		err := set.Update(ctx, func(_ context.Context) (nicloudsdk.Entitlements, error) {
			close(fetchStarted)
			select {
			case <-firstDone:
				// OK!
			case <-ctx.Done():
				t.Error("timeout")
				return nicloudsdk.Entitlements{}, ctx.Err()
			}
			return nicloudsdk.Entitlements{
				Features: map[nicloudsdk.FeatureName]nicloudsdk.Feature{
					nicloudsdk.FeatureMultipleOrganizations: {
						Enabled: true,
					},
				},
			}, nil
		})
		errCh <- err
	}()
	testutil.TryReceive(ctx, t, fetchStarted)
	require.False(t, set.Enabled(nicloudsdk.FeatureMultipleOrganizations))
	// start a second update while the first one is in progress
	go func() {
		err := set.Update(ctx, func(_ context.Context) (nicloudsdk.Entitlements, error) {
			return nicloudsdk.Entitlements{
				Features: map[nicloudsdk.FeatureName]nicloudsdk.Feature{
					nicloudsdk.FeatureMultipleOrganizations: {
						Enabled: true,
					},
					nicloudsdk.FeatureAppearance: {
						Enabled: true,
					},
				},
			}, nil
		})
		errCh <- err
	}()
	close(firstDone)
	err := testutil.TryReceive(ctx, t, errCh)
	require.NoError(t, err)
	err = testutil.TryReceive(ctx, t, errCh)
	require.NoError(t, err)
	require.True(t, set.Enabled(nicloudsdk.FeatureMultipleOrganizations))
	require.True(t, set.Enabled(nicloudsdk.FeatureAppearance))
}

func TestUpdate_LicenseRequiresTelemetry(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitShort)
	set := entitlements.New()
	set.Modify(func(entitlements *nicloudsdk.Entitlements) {
		entitlements.Errors = []string{"some error"}
		entitlements.Features[nicloudsdk.FeatureAppearance] = nicloudsdk.Feature{
			Enabled: true,
		}
	})
	err := set.Update(ctx, func(_ context.Context) (nicloudsdk.Entitlements, error) {
		return nicloudsdk.Entitlements{}, entitlements.ErrLicenseRequiresTelemetry
	})
	require.NoError(t, err)
	require.True(t, set.Enabled(nicloudsdk.FeatureAppearance))
	require.Equal(t, []string{entitlements.ErrLicenseRequiresTelemetry.Error()}, set.Errors())
}
