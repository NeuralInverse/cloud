package portsharing

import (
	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type EnterprisePortSharer struct{}

func NewEnterprisePortSharer() *EnterprisePortSharer {
	return &EnterprisePortSharer{}
}

func (EnterprisePortSharer) AuthorizedLevel(template database.Template, level nicloudsdk.WorkspaceAgentPortShareLevel) error {
	maxLevel := nicloudsdk.WorkspaceAgentPortShareLevel(template.MaxPortSharingLevel)
	return level.IsCompatibleWithMaxLevel(maxLevel)
}

func (EnterprisePortSharer) ValidateTemplateMaxLevel(level nicloudsdk.WorkspaceAgentPortShareLevel) error {
	if !level.ValidMaxLevel() {
		return xerrors.New("invalid max port sharing level, value must be 'authenticated', 'organization', or 'public'.")
	}

	return nil
}

func (EnterprisePortSharer) ConvertMaxLevel(level database.AppSharingLevel) nicloudsdk.WorkspaceAgentPortShareLevel {
	return nicloudsdk.WorkspaceAgentPortShareLevel(level)
}
