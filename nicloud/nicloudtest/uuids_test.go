package nicloudtest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
)

func TestDeterministicUUIDGenerator(t *testing.T) {
	t.Parallel()

	ids := nicloudtest.NewDeterministicUUIDGenerator()
	require.Equal(t, ids.ID("g1"), ids.ID("g1"))
	require.NotEqual(t, ids.ID("g1"), ids.ID("g2"))
}
