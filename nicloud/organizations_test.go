package nicloud_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestOrganizationByUserAndName(t *testing.T) {
	t.Parallel()
	t.Run("NoExist", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		ctx := testutil.Context(t, testutil.WaitLong)

		_, err := client.OrganizationByUserAndName(ctx, nicloudsdk.Me, "nothing")
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		ctx := testutil.Context(t, testutil.WaitLong)

		org, err := client.Organization(ctx, user.OrganizationID)
		require.NoError(t, err)
		_, err = client.OrganizationByUserAndName(ctx, nicloudsdk.Me, org.Name)
		require.NoError(t, err)
	})
}
