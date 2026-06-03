package cli_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/cli/cliui"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
	"github.com/coder/pretty"
)

func TestLogin(t *testing.T) {
	t.Parallel()
	t.Run("InitialUserNoTTY", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		root, _ := clitest.New(t, "login", client.URL.String())
		err := root.Run()
		require.Error(t, err)
	})

	t.Run("InitialUserBadLoginURL", func(t *testing.T) {
		t.Parallel()
		badLoginURL := "https://fcca2077f06e68aaf9"
		root, _ := clitest.New(t, "login", badLoginURL)
		err := root.Run()
		errMsg := fmt.Sprintf("Failed to check server %q for first user, is the URL correct and is neuralinverse accessible from your browser?", badLoginURL)
		require.ErrorContains(t, err, errMsg)
	})

	t.Run("InitialUserNonNIURLFail", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}))
		defer ts.Close()

		badLoginURL := ts.URL
		root, _ := clitest.New(t, "login", badLoginURL)
		err := root.Run()
		errMsg := fmt.Sprintf("Failed to check server %q for first user, is the URL correct and is neuralinverse accessible from your browser?", badLoginURL)
		require.ErrorContains(t, err, errMsg)
	})

	t.Run("InitialUserNonNIURLSuccess", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(nicloudsdk.BuildVersionHeader, "something")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}))
		defer ts.Close()

		badLoginURL := ts.URL
		root, _ := clitest.New(t, "login", badLoginURL)
		err := root.Run()
		// this means we passed the check for a valid neuralinverse server
		require.ErrorContains(t, err, "the initial user cannot be created in non-interactive mode")
	})

	t.Run("InitialUserTTY", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		client := nicloudtest.New(t, nil)
		// The --force-tty flag is required on Windows, because the `isatty` library does not
		// accurately detect Windows ptys when they are not attached to a process:
		// https://github.com/mattn/go-isatty/issues/59
		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", "--force-tty", client.URL.String())
		stdout := expecter.NewAttachedToInvocation(t, root)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), root)
		ctx := testutil.Context(t, testutil.WaitMedium)
		go func() {
			defer close(doneChan)
			err := root.Run()
			assert.NoError(t, err)
		}()

		matches := []string{
			"first user?", "yes",
			"username", nicloudtest.FirstUserParams.Username,
			"name", nicloudtest.FirstUserParams.Name,
			"email", nicloudtest.FirstUserParams.Email,
			"password", nicloudtest.FirstUserParams.Password,
			"password", nicloudtest.FirstUserParams.Password, // confirm
			"trial", "yes",
			"firstName", nicloudtest.TrialUserParams.FirstName,
			"lastName", nicloudtest.TrialUserParams.LastName,
			"phoneNumber", nicloudtest.TrialUserParams.PhoneNumber,
			"jobTitle", nicloudtest.TrialUserParams.JobTitle,
			"companyName", nicloudtest.TrialUserParams.CompanyName,
			// `developers` and `country` `cliui.Select` automatically selects the first option during tests.
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)
			stdin.WriteLine(value)
		}
		stdout.ExpectMatchContext(ctx, "Welcome to Coder")
		<-doneChan
		resp, err := client.LoginWithPassword(ctx, nicloudsdk.LoginWithPasswordRequest{
			Email:    nicloudtest.FirstUserParams.Email,
			Password: nicloudtest.FirstUserParams.Password,
		})
		require.NoError(t, err)
		client.SetSessionToken(resp.SessionToken)
		me, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err)
		assert.Equal(t, nicloudtest.FirstUserParams.Username, me.Username)
		assert.Equal(t, nicloudtest.FirstUserParams.Name, me.Name)
		assert.Equal(t, nicloudtest.FirstUserParams.Email, me.Email)
	})

	t.Run("InitialUserTTYWithNoTrial", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		client := nicloudtest.New(t, nil)
		// The --force-tty flag is required on Windows, because the `isatty` library does not
		// accurately detect Windows ptys when they are not attached to a process:
		// https://github.com/mattn/go-isatty/issues/59
		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", "--force-tty", client.URL.String())
		stdout := expecter.NewAttachedToInvocation(t, root)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), root)
		ctx := testutil.Context(t, testutil.WaitMedium)
		go func() {
			defer close(doneChan)
			err := root.Run()
			assert.NoError(t, err)
		}()

		matches := []string{
			"first user?", "yes",
			"username", nicloudtest.FirstUserParams.Username,
			"name", nicloudtest.FirstUserParams.Name,
			"email", nicloudtest.FirstUserParams.Email,
			"password", nicloudtest.FirstUserParams.Password,
			"password", nicloudtest.FirstUserParams.Password, // confirm
			"trial", "no",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)
			stdin.WriteLine(value)
		}
		stdout.ExpectMatchContext(ctx, "Welcome to Coder")
		<-doneChan
		resp, err := client.LoginWithPassword(ctx, nicloudsdk.LoginWithPasswordRequest{
			Email:    nicloudtest.FirstUserParams.Email,
			Password: nicloudtest.FirstUserParams.Password,
		})
		require.NoError(t, err)
		client.SetSessionToken(resp.SessionToken)
		me, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err)
		assert.Equal(t, nicloudtest.FirstUserParams.Username, me.Username)
		assert.Equal(t, nicloudtest.FirstUserParams.Name, me.Name)
		assert.Equal(t, nicloudtest.FirstUserParams.Email, me.Email)
	})

	t.Run("InitialUserTTYNameOptional", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		client := nicloudtest.New(t, nil)
		// The --force-tty flag is required on Windows, because the `isatty` library does not
		// accurately detect Windows ptys when they are not attached to a process:
		// https://github.com/mattn/go-isatty/issues/59
		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", "--force-tty", client.URL.String())
		stdout := expecter.NewAttachedToInvocation(t, root)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), root)
		ctx := testutil.Context(t, testutil.WaitMedium)
		go func() {
			defer close(doneChan)
			err := root.Run()
			assert.NoError(t, err)
		}()

		matches := []string{
			"first user?", "yes",
			"username", nicloudtest.FirstUserParams.Username,
			"name", "",
			"email", nicloudtest.FirstUserParams.Email,
			"password", nicloudtest.FirstUserParams.Password,
			"password", nicloudtest.FirstUserParams.Password, // confirm
			"trial", "yes",
			"firstName", nicloudtest.TrialUserParams.FirstName,
			"lastName", nicloudtest.TrialUserParams.LastName,
			"phoneNumber", nicloudtest.TrialUserParams.PhoneNumber,
			"jobTitle", nicloudtest.TrialUserParams.JobTitle,
			"companyName", nicloudtest.TrialUserParams.CompanyName,
			// `developers` and `country` `cliui.Select` automatically selects the first option during tests.
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)
			stdin.WriteLine(value)
		}
		stdout.ExpectMatchContext(ctx, "Welcome to Coder")
		<-doneChan
		resp, err := client.LoginWithPassword(ctx, nicloudsdk.LoginWithPasswordRequest{
			Email:    nicloudtest.FirstUserParams.Email,
			Password: nicloudtest.FirstUserParams.Password,
		})
		require.NoError(t, err)
		client.SetSessionToken(resp.SessionToken)
		me, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err)
		assert.Equal(t, nicloudtest.FirstUserParams.Username, me.Username)
		assert.Equal(t, nicloudtest.FirstUserParams.Email, me.Email)
		assert.Empty(t, me.Name)
	})

	t.Run("InitialUserTTYFlag", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		client := nicloudtest.New(t, nil)
		// The --force-tty flag is required on Windows, because the `isatty` library does not
		// accurately detect Windows ptys when they are not attached to a process:
		// https://github.com/mattn/go-isatty/issues/59
		inv, _ := clitest.New(t, "--url", client.URL.String(), "login", "--force-tty")
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		ctx := testutil.Context(t, testutil.WaitMedium)

		clitest.Start(t, inv)

		stdout.ExpectMatchContext(ctx, fmt.Sprintf("Attempting to authenticate with flag URL: '%s'", client.URL.String()))
		matches := []string{
			"first user?", "yes",
			"username", nicloudtest.FirstUserParams.Username,
			"name", nicloudtest.FirstUserParams.Name,
			"email", nicloudtest.FirstUserParams.Email,
			"password", nicloudtest.FirstUserParams.Password,
			"password", nicloudtest.FirstUserParams.Password, // confirm
			"trial", "yes",
			"firstName", nicloudtest.TrialUserParams.FirstName,
			"lastName", nicloudtest.TrialUserParams.LastName,
			"phoneNumber", nicloudtest.TrialUserParams.PhoneNumber,
			"jobTitle", nicloudtest.TrialUserParams.JobTitle,
			"companyName", nicloudtest.TrialUserParams.CompanyName,
			// `developers` and `country` `cliui.Select` automatically selects the first option during tests.
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)
			stdin.WriteLine(value)
		}
		stdout.ExpectMatchContext(ctx, "Welcome to Coder")
		resp, err := client.LoginWithPassword(ctx, nicloudsdk.LoginWithPasswordRequest{
			Email:    nicloudtest.FirstUserParams.Email,
			Password: nicloudtest.FirstUserParams.Password,
		})
		require.NoError(t, err)
		client.SetSessionToken(resp.SessionToken)
		me, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err)
		assert.Equal(t, nicloudtest.FirstUserParams.Username, me.Username)
		assert.Equal(t, nicloudtest.FirstUserParams.Name, me.Name)
		assert.Equal(t, nicloudtest.FirstUserParams.Email, me.Email)
	})

	t.Run("InitialUserFlags", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		client := nicloudtest.New(t, nil)
		inv, _ := clitest.New(
			t, "login", client.URL.String(),
			"--first-user-username", nicloudtest.FirstUserParams.Username,
			"--first-user-full-name", nicloudtest.FirstUserParams.Name,
			"--first-user-email", nicloudtest.FirstUserParams.Email,
			"--first-user-password", nicloudtest.FirstUserParams.Password,
			"--first-user-trial",
		)
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		ctx := testutil.Context(t, testutil.WaitMedium)
		w := clitest.StartWithWaiter(t, inv)
		stdout.ExpectMatchContext(ctx, "firstName")
		stdin.WriteLine(nicloudtest.TrialUserParams.FirstName)
		stdout.ExpectMatchContext(ctx, "lastName")
		stdin.WriteLine(nicloudtest.TrialUserParams.LastName)
		stdout.ExpectMatchContext(ctx, "phoneNumber")
		stdin.WriteLine(nicloudtest.TrialUserParams.PhoneNumber)
		stdout.ExpectMatchContext(ctx, "jobTitle")
		stdin.WriteLine(nicloudtest.TrialUserParams.JobTitle)
		stdout.ExpectMatchContext(ctx, "companyName")
		stdin.WriteLine(nicloudtest.TrialUserParams.CompanyName)
		// `developers` and `country` `cliui.Select` automatically selects the first option during tests.
		stdout.ExpectMatchContext(ctx, "Welcome to Coder")
		w.RequireSuccess()
		resp, err := client.LoginWithPassword(ctx, nicloudsdk.LoginWithPasswordRequest{
			Email:    nicloudtest.FirstUserParams.Email,
			Password: nicloudtest.FirstUserParams.Password,
		})
		require.NoError(t, err)
		client.SetSessionToken(resp.SessionToken)
		me, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err)
		assert.Equal(t, nicloudtest.FirstUserParams.Username, me.Username)
		assert.Equal(t, nicloudtest.FirstUserParams.Name, me.Name)
		assert.Equal(t, nicloudtest.FirstUserParams.Email, me.Email)
	})

	t.Run("InitialUserFlagsNameOptional", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		client := nicloudtest.New(t, nil)
		inv, _ := clitest.New(
			t, "login", client.URL.String(),
			"--first-user-username", nicloudtest.FirstUserParams.Username,
			"--first-user-email", nicloudtest.FirstUserParams.Email,
			"--first-user-password", nicloudtest.FirstUserParams.Password,
			"--first-user-trial",
		)
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		ctx := testutil.Context(t, testutil.WaitMedium)
		w := clitest.StartWithWaiter(t, inv)
		stdout.ExpectMatchContext(ctx, "firstName")
		stdin.WriteLine(nicloudtest.TrialUserParams.FirstName)
		stdout.ExpectMatchContext(ctx, "lastName")
		stdin.WriteLine(nicloudtest.TrialUserParams.LastName)
		stdout.ExpectMatchContext(ctx, "phoneNumber")
		stdin.WriteLine(nicloudtest.TrialUserParams.PhoneNumber)
		stdout.ExpectMatchContext(ctx, "jobTitle")
		stdin.WriteLine(nicloudtest.TrialUserParams.JobTitle)
		stdout.ExpectMatchContext(ctx, "companyName")
		stdin.WriteLine(nicloudtest.TrialUserParams.CompanyName)
		// `developers` and `country` `cliui.Select` automatically selects the first option during tests.
		stdout.ExpectMatchContext(ctx, "Welcome to Coder")
		w.RequireSuccess()
		resp, err := client.LoginWithPassword(ctx, nicloudsdk.LoginWithPasswordRequest{
			Email:    nicloudtest.FirstUserParams.Email,
			Password: nicloudtest.FirstUserParams.Password,
		})
		require.NoError(t, err)
		client.SetSessionToken(resp.SessionToken)
		me, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err)
		assert.Equal(t, nicloudtest.FirstUserParams.Username, me.Username)
		assert.Equal(t, nicloudtest.FirstUserParams.Email, me.Email)
		assert.Empty(t, me.Name)
	})

	t.Run("InitialUserTTYConfirmPasswordFailAndReprompt", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		client := nicloudtest.New(t, nil)
		// The --force-tty flag is required on Windows, because the `isatty` library does not
		// accurately detect Windows ptys when they are not attached to a process:
		// https://github.com/mattn/go-isatty/issues/59
		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", "--force-tty", client.URL.String())
		stdout := expecter.NewAttachedToInvocation(t, root)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), root)
		go func() {
			defer close(doneChan)
			err := root.WithContext(ctx).Run()
			assert.NoError(t, err)
		}()

		matches := []string{
			"first user?", "yes",
			"username", nicloudtest.FirstUserParams.Username,
			"name", nicloudtest.FirstUserParams.Name,
			"email", nicloudtest.FirstUserParams.Email,
			"password", nicloudtest.FirstUserParams.Password,
			"password", "something completely different",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)
			stdin.WriteLine(value)
		}

		// Validate that we reprompt for matching passwords.
		stdout.ExpectMatchContext(ctx, "Passwords do not match")
		stdout.ExpectMatchContext(ctx, "Enter a "+pretty.Sprint(cliui.DefaultStyles.Field, "password"))
		stdin.WriteLine(nicloudtest.FirstUserParams.Password)
		stdout.ExpectMatchContext(ctx, "Confirm")
		stdin.WriteLine(nicloudtest.FirstUserParams.Password)
		stdout.ExpectMatchContext(ctx, "trial")
		stdin.WriteLine("yes")
		stdout.ExpectMatchContext(ctx, "firstName")
		stdin.WriteLine(nicloudtest.TrialUserParams.FirstName)
		stdout.ExpectMatchContext(ctx, "lastName")
		stdin.WriteLine(nicloudtest.TrialUserParams.LastName)
		stdout.ExpectMatchContext(ctx, "phoneNumber")
		stdin.WriteLine(nicloudtest.TrialUserParams.PhoneNumber)
		stdout.ExpectMatchContext(ctx, "jobTitle")
		stdin.WriteLine(nicloudtest.TrialUserParams.JobTitle)
		stdout.ExpectMatchContext(ctx, "companyName")
		stdin.WriteLine(nicloudtest.TrialUserParams.CompanyName)
		stdout.ExpectMatchContext(ctx, "Welcome to Coder")
		<-doneChan
	})

	t.Run("ExistingUserValidTokenTTY", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		ctx := testutil.Context(t, testutil.WaitMedium)

		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", "--force-tty", client.URL.String(), "--no-open")
		stdout := expecter.NewAttachedToInvocation(t, root)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), root)
		go func() {
			defer close(doneChan)
			err := root.Run()
			assert.NoError(t, err)
		}()

		stdout.ExpectMatchContext(ctx, fmt.Sprintf("Attempting to authenticate with argument URL: '%s'", client.URL.String()))
		stdout.ExpectMatchContext(ctx, "Paste your token here:")
		stdin.WriteLine(client.SessionToken())
		stdout.ExpectMatchContext(ctx, "Welcome to Coder")
		<-doneChan
	})

	t.Run("ExistingUserURLSavedInConfig", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		ctx := testutil.Context(t, testutil.WaitMedium)
		client := nicloudtest.New(t, nil)
		url := client.URL.String()
		nicloudtest.CreateFirstUser(t, client)

		inv, root := clitest.New(t, "login", "--no-open")
		clitest.SetupConfig(t, client, root)

		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()

		stdout.ExpectMatchContext(ctx, fmt.Sprintf("Attempting to authenticate with config URL: '%s'", url))
		stdout.ExpectMatchContext(ctx, "Paste your token here:")
		stdin.WriteLine(client.SessionToken())
		<-doneChan
	})

	t.Run("ExistingUserURLSavedInEnv", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		ctx := testutil.Context(t, testutil.WaitMedium)
		client := nicloudtest.New(t, nil)
		url := client.URL.String()
		nicloudtest.CreateFirstUser(t, client)

		inv, _ := clitest.New(t, "login", "--no-open")
		inv.Environ.Set("NEURALINVERSE_URL", url)

		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()

		stdout.ExpectMatchContext(ctx, fmt.Sprintf("Attempting to authenticate with environment URL: '%s'", url))
		stdout.ExpectMatchContext(ctx, "Paste your token here:")
		stdin.WriteLine(client.SessionToken())
		<-doneChan
	})

	t.Run("ExistingUserInvalidTokenTTY", func(t *testing.T) {
		t.Parallel()
		logger := testutil.Logger(t)
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)

		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()
		doneChan := make(chan struct{})
		root, _ := clitest.New(t, "login", client.URL.String(), "--no-open")
		stdout := expecter.NewAttachedToInvocation(t, root)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), root)
		go func() {
			defer close(doneChan)
			err := root.WithContext(ctx).Run()
			// An error is expected in this case, since the login wasn't successful:
			assert.Error(t, err)
		}()

		stdout.ExpectMatchContext(ctx, "Paste your token here:")
		stdin.WriteLine("an-invalid-token")
		stdout.ExpectMatchContext(ctx, "That's not a valid token!")
		cancelFunc()
		<-doneChan
	})

	// TokenFlag should generate a new session token and store it in the session file.
	t.Run("TokenFlag", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		root, cfg := clitest.New(t, "login", client.URL.String(), "--token", client.SessionToken())
		err := root.Run()
		require.NoError(t, err)
		sessionFile, err := cfg.Session().Read()
		require.NoError(t, err)
		// This **should not be equal** to the token we passed in.
		require.NotEqual(t, client.SessionToken(), sessionFile)
	})

	t.Run("SessionTokenEnvVar", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		root, _ := clitest.New(t, "login", client.URL.String())
		root.Environ.Set("NEURALINVERSE_SESSION_TOKEN", "invalid-token")
		err := root.Run()
		require.Error(t, err)
		require.Contains(t, err.Error(), "NEURALINVERSE_SESSION_TOKEN is set")
		require.Contains(t, err.Error(), "unset NEURALINVERSE_SESSION_TOKEN")
	})

	t.Run("SessionTokenEnvVarWithUseTokenAsSession", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		root, _ := clitest.New(t, "login", client.URL.String(), "--use-token-as-session")
		root.Environ.Set("NEURALINVERSE_SESSION_TOKEN", client.SessionToken())
		err := root.Run()
		require.NoError(t, err)
	})

	t.Run("SessionTokenEnvVarWithTokenFlag", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)
		// Using --token with NEURALINVERSE_SESSION_TOKEN set should succeed.
		// This is the standard pattern used by coder/setup-action.
		root, _ := clitest.New(t, "login", client.URL.String(), "--token", client.SessionToken())
		root.Environ.Set("NEURALINVERSE_SESSION_TOKEN", client.SessionToken())
		err := root.Run()
		require.NoError(t, err)
	})

	t.Run("KeepOrganizationContext", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		first := nicloudtest.CreateFirstUser(t, client)
		root, cfg := clitest.New(t, "login", client.URL.String(), "--token", client.SessionToken())

		err := cfg.Organization().Write(first.OrganizationID.String())
		require.NoError(t, err, "write bad org to config")

		err = root.Run()
		require.NoError(t, err)
		sessionFile, err := cfg.Session().Read()
		require.NoError(t, err)
		require.NotEqual(t, client.SessionToken(), sessionFile)

		// Organization config should be deleted since the org does not exist
		selected, err := cfg.Organization().Read()
		require.NoError(t, err)
		require.Equal(t, selected, first.OrganizationID.String())
	})
}

func TestLoginToken(t *testing.T) {
	t.Parallel()

	t.Run("PrintsToken", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)

		inv, root := clitest.New(t, "login", "token", "--url", client.URL.String())
		clitest.SetupConfig(t, client, root)
		stdout := expecter.NewAttachedToInvocation(t, inv)
		ctx := testutil.Context(t, testutil.WaitShort)
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		stdout.ExpectMatchContext(ctx, client.SessionToken())
	})

	t.Run("NoTokenStored", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		inv, _ := clitest.New(t, "login", "token", "--url", client.URL.String())
		ctx := testutil.Context(t, testutil.WaitShort)
		err := inv.WithContext(ctx).Run()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no session token found")
	})

	t.Run("NoURLProvided", func(t *testing.T) {
		t.Parallel()
		inv, _ := clitest.New(t, "login", "token")
		ctx := testutil.Context(t, testutil.WaitShort)
		err := inv.WithContext(ctx).Run()
		require.Error(t, err)
		require.Contains(t, err.Error(), "You are not logged in")
	})

	t.Run("URLMismatchFileBackend", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		nicloudtest.CreateFirstUser(t, client)

		inv, root := clitest.New(t, "login", "token", "--url", "https://other.example.com")
		clitest.SetupConfig(t, client, root)
		ctx := testutil.Context(t, testutil.WaitShort)
		err := inv.WithContext(ctx).Run()
		require.Error(t, err)
		require.Contains(t, err.Error(), "file session token storage only supports one server")
	})
}
