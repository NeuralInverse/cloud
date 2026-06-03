package nicloudsdk_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestDisconnectReason_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		reason nicloudsdk.DisconnectReason
		valid  bool
	}{
		{nicloudsdk.DisconnectReasonUnknown, true},
		{nicloudsdk.DisconnectReasonGraceful, true},
		{nicloudsdk.DisconnectReasonClientClosed, true},
		{nicloudsdk.DisconnectReasonServerShutdown, true},
		{nicloudsdk.DisconnectReasonNetworkError, true},
		{nicloudsdk.DisconnectReasonProtocolError, true},
		{nicloudsdk.DisconnectReasonWorkspaceStopped, true},
		{nicloudsdk.DisconnectReasonControlPlaneLost, true},
		{nicloudsdk.DisconnectReason("not_a_real_reason"), false},
	}

	for _, c := range cases {
		require.Equal(t, c.valid, c.reason.Valid(), "reason=%q", c.reason)
	}
}

func TestDisconnectReason_Expected(t *testing.T) {
	t.Parallel()

	expected := map[nicloudsdk.DisconnectReason]bool{
		nicloudsdk.DisconnectReasonGraceful:         true,
		nicloudsdk.DisconnectReasonClientClosed:     true,
		nicloudsdk.DisconnectReasonServerShutdown:   true,
		nicloudsdk.DisconnectReasonWorkspaceStopped: true,

		nicloudsdk.DisconnectReasonUnknown:          false,
		nicloudsdk.DisconnectReasonNetworkError:     false,
		nicloudsdk.DisconnectReasonProtocolError:    false,
		nicloudsdk.DisconnectReasonControlPlaneLost: false,
	}

	for reason, want := range expected {
		require.Equal(t, want, reason.Expected(), "reason=%q", reason)
	}

	// Unknown values default to not-expected so that uncategorized
	// emit sites surface in the "investigate" bucket.
	require.False(t, nicloudsdk.DisconnectReason("not_a_real_reason").Expected())
}

func TestDisconnectInitiator_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		initiator nicloudsdk.DisconnectInitiator
		valid     bool
	}{
		{nicloudsdk.DisconnectInitiatorUnknown, true},
		{nicloudsdk.DisconnectInitiatorClient, true},
		{nicloudsdk.DisconnectInitiatorAgent, true},
		{nicloudsdk.DisconnectInitiatorServer, true},
		{nicloudsdk.DisconnectInitiatorNetwork, true},
		{nicloudsdk.DisconnectInitiator("nobody"), false},
	}

	for _, c := range cases {
		require.Equal(t, c.valid, c.initiator.Valid(), "initiator=%q", c.initiator)
	}
}

func TestConnectionMethod_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		method nicloudsdk.ConnectionMethod
		valid  bool
	}{
		{nicloudsdk.ConnectionMethodUnknown, true},
		{nicloudsdk.ConnectionMethodDirect, true},
		{nicloudsdk.ConnectionMethodDERP, true},
		{nicloudsdk.ConnectionMethod("magic"), false},
	}

	for _, c := range cases {
		require.Equal(t, c.valid, c.method.Valid(), "method=%q", c.method)
	}
}
