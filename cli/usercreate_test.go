package cli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
)

func TestUserCreate(t *testing.T) {
	t.Parallel()
	t.Run("Prompts", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		ctx := testutil.Context(t, testutil.WaitLong)
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		inv, root := clitest.New(t, "users", "create")
		clitest.SetupConfig(t, client, root)
		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()
		matches := []string{
			"Username", "dean",
			"Email", "dean@cloud.neuralinverse.com",
			"Full name (optional):", "Mr. Dean Deanington",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)
			stdin.WriteLine(value)
		}
		_ = testutil.TryReceive(ctx, t, doneChan)
		created, err := client.User(ctx, matches[1])
		require.NoError(t, err)
		assert.Equal(t, matches[1], created.Username)
		assert.Equal(t, matches[3], created.Email)
		assert.Equal(t, matches[5], created.Name)
	})

	t.Run("PromptsNoName", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		ctx := testutil.Context(t, testutil.WaitLong)
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		inv, root := clitest.New(t, "users", "create")
		clitest.SetupConfig(t, client, root)
		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()
		matches := []string{
			"Username", "noname",
			"Email", "noname@cloud.neuralinverse.com",
			"Full name (optional):", "",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)
			stdin.WriteLine(value)
		}
		_ = testutil.TryReceive(ctx, t, doneChan)
		created, err := client.User(ctx, matches[1])
		require.NoError(t, err)
		assert.Equal(t, matches[1], created.Username)
		assert.Equal(t, matches[3], created.Email)
		assert.Empty(t, created.Name)
	})

	t.Run("Args", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		args := []string{
			"users", "create",
			"-e", "dean@cloud.neuralinverse.com",
			"-u", "dean",
			"-n", "Mr. Dean Deanington",
			"-p", "1n5ecureP4ssw0rd!",
		}
		inv, root := clitest.New(t, args...)
		clitest.SetupConfig(t, client, root)
		err := inv.Run()
		require.NoError(t, err)
		ctx := testutil.Context(t, testutil.WaitShort)
		created, err := client.User(ctx, "dean")
		require.NoError(t, err)
		assert.Equal(t, args[3], created.Email)
		assert.Equal(t, args[5], created.Username)
		assert.Equal(t, args[7], created.Name)
	})

	t.Run("ArgsNoName", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		args := []string{
			"users", "create",
			"-e", "dean@cloud.neuralinverse.com",
			"-u", "dean",
			"-p", "1n5ecureP4ssw0rd!",
		}
		inv, root := clitest.New(t, args...)
		clitest.SetupConfig(t, client, root)
		err := inv.Run()
		require.NoError(t, err)
		ctx := testutil.Context(t, testutil.WaitShort)
		created, err := client.User(ctx, args[5])
		require.NoError(t, err)
		assert.Equal(t, args[3], created.Email)
		assert.Equal(t, args[5], created.Username)
		assert.Empty(t, created.Name)
	})

	tests := []struct {
		name string
		args []string
		err  string
	}{
		{
			name: "ServiceAccount",
			args: []string{"--service-account", "-u", "dean"},
			err:  "Premium feature",
		},
		{
			name: "ServiceAccountLoginType",
			args: []string{"--service-account", "-u", "dean", "--login-type", "none"},
			err:  "You cannot use --login-type with --service-account",
		},
		{
			name: "ServiceAccountDisableLogin",
			args: []string{"--service-account", "-u", "dean", "--disable-login"},
			err:  "You cannot use --disable-login with --service-account",
		},
		{
			name: "ServiceAccountEmail",
			args: []string{"--service-account", "-u", "dean", "--email", "dean@cloud.neuralinverse.com"},
			err:  "You cannot use --email with --service-account",
		},
		{
			name: "ServiceAccountPassword",
			args: []string{"--service-account", "-u", "dean", "--password", "1n5ecureP4ssw0rd!"},
			err:  "You cannot use --password with --service-account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := nicloudtest.New(t, nil)
			nicloudtest.CreateFirstUser(t, client)
			inv, root := clitest.New(t, append([]string{"users", "create"}, tt.args...)...)
			clitest.SetupConfig(t, client, root)
			err := inv.Run()
			if tt.err == "" {
				require.NoError(t, err)
				ctx := testutil.Context(t, testutil.WaitShort)
				created, err := client.User(ctx, "dean")
				require.NoError(t, err)
				assert.Equal(t, nicloudsdk.LoginTypeNone, created.LoginType)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.err)
			}
		})
	}
}
