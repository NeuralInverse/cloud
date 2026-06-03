package clitest_test

import (
	"testing"

	"go.uber.org/goleak"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, testutil.GoleakOptions...)
}

func TestCli(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitMedium)
	clitest.CreateTemplateVersionSource(t, nil)
	client := nicloudtest.New(t, nil)
	i, config := clitest.New(t)
	clitest.SetupConfig(t, client, config)
	stdout := expecter.NewAttachedToInvocation(t, i)
	clitest.Start(t, i)
	stdout.ExpectMatchContext(ctx, "coder")
}
