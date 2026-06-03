//go:build !linux && !(windows && amd64)

package agent

import (
	"time"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type osListeningPortsGetter struct {
	cacheDuration time.Duration
}

func (*osListeningPortsGetter) GetListeningPorts() ([]nicloudsdk.WorkspaceAgentListeningPort, error) {
	// Can't scan for ports on non-linux or non-windows_amd64 systems at the
	// moment. The UI will not show any "no ports found" message to the user, so
	// the user won't suspect a thing.
	return []nicloudsdk.WorkspaceAgentListeningPort{}, nil
}
