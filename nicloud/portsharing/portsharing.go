package portsharing

import (
	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type PortSharer interface {
	AuthorizedLevel(template database.Template, level nicloudsdk.WorkspaceAgentPortShareLevel) error
	ValidateTemplateMaxLevel(level nicloudsdk.WorkspaceAgentPortShareLevel) error
	ConvertMaxLevel(level database.AppSharingLevel) nicloudsdk.WorkspaceAgentPortShareLevel
}

type AGPLPortSharer struct{}

func (AGPLPortSharer) AuthorizedLevel(_ database.Template, _ nicloudsdk.WorkspaceAgentPortShareLevel) error {
	return nil
}

func (AGPLPortSharer) ValidateTemplateMaxLevel(_ nicloudsdk.WorkspaceAgentPortShareLevel) error {
	return xerrors.New("Restricting port sharing level is an enterprise feature that is not enabled.")
}

func (AGPLPortSharer) ConvertMaxLevel(_ database.AppSharingLevel) nicloudsdk.WorkspaceAgentPortShareLevel {
	return nicloudsdk.WorkspaceAgentPortShareLevelPublic
}

var DefaultPortSharer PortSharer = AGPLPortSharer{}
