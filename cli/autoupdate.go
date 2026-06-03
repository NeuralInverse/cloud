package cli

import (
	"fmt"
	"strings"

	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/cli/cliui"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/coder/serpent"
)

func (r *RootCmd) autoupdate() *serpent.Command {
	cmd := &serpent.Command{
		Annotations: workspaceCommand,
		Use:         "autoupdate <workspace> <always|never>",
		Short:       "Toggle auto-update policy for a workspace",
		Middleware: serpent.Chain(
			serpent.RequireNArgs(2),
		),
		Handler: func(inv *serpent.Invocation) error {
			client, err := r.InitClient(inv)
			if err != nil {
				return err
			}

			policy := strings.ToLower(inv.Args[1])
			err = validateAutoUpdatePolicy(policy)
			if err != nil {
				return xerrors.Errorf("validate policy: %w", err)
			}

			workspace, err := client.ResolveWorkspace(inv.Context(), inv.Args[0])
			if err != nil {
				return xerrors.Errorf("get workspace: %w", err)
			}

			err = client.UpdateWorkspaceAutomaticUpdates(inv.Context(), workspace.ID, nicloudsdk.UpdateWorkspaceAutomaticUpdatesRequest{
				AutomaticUpdates: nicloudsdk.AutomaticUpdates(policy),
			})
			if err != nil {
				return xerrors.Errorf("update workspace automatic updates policy: %w", err)
			}
			_, _ = fmt.Fprintf(inv.Stdout, "Updated workspace %q auto-update policy to %q\n", workspace.Name, policy)
			return nil
		},
	}

	cmd.Options = append(cmd.Options, cliui.SkipPromptOption())
	return cmd
}

func validateAutoUpdatePolicy(arg string) error {
	switch nicloudsdk.AutomaticUpdates(arg) {
	case nicloudsdk.AutomaticUpdatesAlways, nicloudsdk.AutomaticUpdatesNever:
		return nil
	default:
		return xerrors.Errorf("invalid option %q must be either of %q or %q", arg, nicloudsdk.AutomaticUpdatesAlways, nicloudsdk.AutomaticUpdatesNever)
	}
}
