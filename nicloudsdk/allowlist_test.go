package nicloudsdk_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestAPIAllowListTarget_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	all := nicloudsdk.AllowAllTarget()
	b, err := json.Marshal(all)
	require.NoError(t, err)
	require.JSONEq(t, `"*:*"`, string(b))
	var rt nicloudsdk.APIAllowListTarget
	require.NoError(t, json.Unmarshal(b, &rt))
	require.Equal(t, nicloudsdk.ResourceWildcard, rt.Type)
	require.Equal(t, policy.WildcardSymbol, rt.ID)

	ty := nicloudsdk.AllowTypeTarget(nicloudsdk.ResourceWorkspace)
	b, err = json.Marshal(ty)
	require.NoError(t, err)
	require.JSONEq(t, `"workspace:*"`, string(b))
	require.NoError(t, json.Unmarshal(b, &rt))
	require.Equal(t, nicloudsdk.ResourceWorkspace, rt.Type)
	require.Equal(t, policy.WildcardSymbol, rt.ID)

	id := uuid.New()
	res := nicloudsdk.AllowResourceTarget(nicloudsdk.ResourceTemplate, id)
	b, err = json.Marshal(res)
	require.NoError(t, err)
	exp := `"template:` + id.String() + `"`
	require.JSONEq(t, exp, string(b))
}
