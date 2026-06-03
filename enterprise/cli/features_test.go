package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
)

func TestFeaturesList(t *testing.T) {
	t.Parallel()
	t.Run("Table", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)
		client, admin := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID)
		inv, conf := newCLI(t, "features", "list")
		clitest.SetupConfig(t, anotherClient, conf)
		stdout := expecter.NewAttachedToInvocation(t, inv)
		clitest.Start(t, inv)
		stdout.ExpectMatchContext(ctx, "user_limit")
		stdout.ExpectMatchContext(ctx, "not_entitled")
	})
	t.Run("JSON", func(t *testing.T) {
		t.Parallel()

		client, admin := nicloudenttest.New(t, &nicloudenttest.Options{DontAddLicense: true})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID)
		inv, conf := newCLI(t, "features", "list", "-o", "json")
		clitest.SetupConfig(t, anotherClient, conf)
		doneChan := make(chan struct{})

		buf := bytes.NewBuffer(nil)
		inv.Stdout = buf
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()

		<-doneChan

		var entitlements nicloudsdk.Entitlements
		err := json.Unmarshal(buf.Bytes(), &entitlements)
		require.NoError(t, err, "unmarshal JSON output")
		assert.Empty(t, entitlements.Warnings)
		for _, featureName := range nicloudsdk.FeatureNames {
			assert.Equal(t, nicloudsdk.EntitlementNotEntitled, entitlements.Features[featureName].Entitlement)
		}
		assert.False(t, entitlements.HasLicense)
	})
}
