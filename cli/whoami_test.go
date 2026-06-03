package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
)

func TestWhoami(t *testing.T) {
	t.Parallel()

	t.Run("InitialUserNoTTY", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		root, _ := clitest.New(t, "login", client.URL.String())
		err := root.Run()
		require.Error(t, err)
	})

	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		_ = nicloudtest.CreateFirstUser(t, client)
		inv, root := clitest.New(t, "whoami")
		clitest.SetupConfig(t, client, root)
		buf := new(bytes.Buffer)
		inv.Stdout = buf
		err := inv.Run()
		require.NoError(t, err)
		whoami := buf.String()
		require.NotEmpty(t, whoami)
	})
}
