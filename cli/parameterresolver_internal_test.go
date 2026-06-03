package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestIsValidTemplateParameterOption(t *testing.T) {
	t.Parallel()

	options := []nicloudsdk.TemplateVersionParameterOption{
		{Name: "Vim", Value: "vim"},
		{Name: "Emacs", Value: "emacs"},
		{Name: "VS Code", Value: "vscode"},
	}

	t.Run("SingleSelectValid", func(t *testing.T) {
		t.Parallel()
		bp := nicloudsdk.WorkspaceBuildParameter{Name: "editor", Value: "vim"}
		tvp := nicloudsdk.TemplateVersionParameter{
			Name:    "editor",
			Type:    "string",
			Options: options,
		}
		assert.True(t, isValidTemplateParameterOption(bp, tvp))
	})

	t.Run("SingleSelectInvalid", func(t *testing.T) {
		t.Parallel()
		bp := nicloudsdk.WorkspaceBuildParameter{Name: "editor", Value: "notepad"}
		tvp := nicloudsdk.TemplateVersionParameter{
			Name:    "editor",
			Type:    "string",
			Options: options,
		}
		assert.False(t, isValidTemplateParameterOption(bp, tvp))
	})

	t.Run("MultiSelectAllValid", func(t *testing.T) {
		t.Parallel()
		bp := nicloudsdk.WorkspaceBuildParameter{Name: "editors", Value: `["vim","emacs"]`}
		tvp := nicloudsdk.TemplateVersionParameter{
			Name:    "editors",
			Type:    "list(string)",
			Options: options,
		}
		assert.True(t, isValidTemplateParameterOption(bp, tvp))
	})

	t.Run("MultiSelectOneInvalid", func(t *testing.T) {
		t.Parallel()
		bp := nicloudsdk.WorkspaceBuildParameter{Name: "editors", Value: `["vim","notepad"]`}
		tvp := nicloudsdk.TemplateVersionParameter{
			Name:    "editors",
			Type:    "list(string)",
			Options: options,
		}
		assert.False(t, isValidTemplateParameterOption(bp, tvp))
	})

	t.Run("MultiSelectEmptyArray", func(t *testing.T) {
		t.Parallel()
		bp := nicloudsdk.WorkspaceBuildParameter{Name: "editors", Value: `[]`}
		tvp := nicloudsdk.TemplateVersionParameter{
			Name:    "editors",
			Type:    "list(string)",
			Options: options,
		}
		assert.True(t, isValidTemplateParameterOption(bp, tvp))
	})

	t.Run("MultiSelectInvalidJSON", func(t *testing.T) {
		t.Parallel()
		bp := nicloudsdk.WorkspaceBuildParameter{Name: "editors", Value: `not-json`}
		tvp := nicloudsdk.TemplateVersionParameter{
			Name:    "editors",
			Type:    "list(string)",
			Options: options,
		}
		assert.False(t, isValidTemplateParameterOption(bp, tvp))
	})
}
