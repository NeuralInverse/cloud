package nicloudtest_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, testutil.GoleakOptions...)
}

func TestNew(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, &nicloudtest.Options{
		IncludeProvisionerDaemon: true,
	})
	user := nicloudtest.CreateFirstUser(t, client)
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
	_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
	workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
	nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
	nicloudtest.AwaitWorkspaceAgents(t, client, workspace.ID)
	_, _ = nicloudtest.NewGoogleInstanceIdentity(t, "example", false)
	_, _ = nicloudtest.NewAWSInstanceIdentity(t, "an-instance")
}

func TestRandomName(t *testing.T) {
	t.Parallel()

	for range 10 {
		name := nicloudtest.RandomName(t)

		require.NotEmpty(t, name, "name should not be empty")
		require.NotContains(t, name, "_", "name should not contain underscores")

		// Should be title cased (e.g., "Happy Einstein").
		words := strings.Split(name, " ")
		require.Len(t, words, 2, "name should have exactly two words")
		for _, word := range words {
			firstRune := []rune(word)[0]
			require.True(t, unicode.IsUpper(firstRune), "word %q should start with uppercase letter", word)
		}
	}
}
