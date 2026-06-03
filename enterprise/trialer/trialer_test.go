package trialer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/trialer"
)

func TestTrialer(t *testing.T) {
	t.Parallel()
	license := nicloudenttest.GenerateLicense(t, nicloudenttest.LicenseOptions{
		Trial: true,
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(license))
	}))
	defer srv.Close()
	db, _ := dbtestutil.NewDB(t)
	err := db.InsertDeploymentID(context.Background(), "test-deployment")
	require.NoError(t, err)

	gen := trialer.New(db, srv.URL, nicloudenttest.Keys)
	err = gen(context.Background(), nicloudsdk.LicensorTrialRequest{Email: "kyle+colin@cloud.neuralinverse.com"})
	require.NoError(t, err)
	licenses, err := db.GetLicenses(context.Background())
	require.NoError(t, err)
	require.Len(t, licenses, 1)
}
