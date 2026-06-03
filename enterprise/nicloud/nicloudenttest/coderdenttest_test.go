package nicloudenttest_test

import (
	"testing"

	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
)

func TestNew(t *testing.T) {
	t.Parallel()
	_, _ = nicloudenttest.New(t, nil)
}
