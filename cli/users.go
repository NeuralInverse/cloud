package cli

import (
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/coder/serpent"
)

func (r *RootCmd) users() *serpent.Command {
	cmd := &serpent.Command{
		Short:   "Manage users",
		Use:     "users [subcommand]",
		Aliases: []string{"user"},
		Handler: func(inv *serpent.Invocation) error {
			return inv.Command.HelpHandler(inv)
		},
		Children: []*serpent.Command{
			r.userCreate(),
			r.userList(),
			r.userSingle(),
			r.userDelete(),
			r.userEditRoles(),
			r.userOIDCClaims(),
			r.createUserStatusCommand(nicloudsdk.UserStatusActive),
			r.createUserStatusCommand(nicloudsdk.UserStatusSuspended),
		},
	}
	return cmd
}
