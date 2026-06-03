package workspacesdk_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
	"tailscale.com/net/tsaddr"
	"tailscale.com/tailcfg"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/workspacesdk"
	"github.com/NeuralInverse/cloud/v2/tailnet"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/coder/websocket"
)

func TestWorkspaceRewriteDERPMap(t *testing.T) {
	t.Parallel()
	// This test ensures that RewriteDERPMap mutates built-in DERPs with the
	// client access URL.
	dm := &tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{
			1: {
				EmbeddedRelay: true,
				RegionID:      1,
				Nodes: []*tailcfg.DERPNode{{
					HostName: "bananas.org",
					DERPPort: 1,
				}},
			},
		},
	}
	parsed, err := url.Parse("https://coconuts.org:44558")
	require.NoError(t, err)
	client := workspacesdk.New(nicloudsdk.New(parsed))
	client.RewriteDERPMap(dm)
	region := dm.Regions[1]
	require.True(t, region.EmbeddedRelay)
	require.Len(t, region.Nodes, 1)
	node := region.Nodes[0]
	require.Equal(t, "coconuts.org", node.HostName)
	require.Equal(t, 44558, node.DERPPort)
}

func TestWorkspaceDialerFailure(t *testing.T) {
	t.Parallel()

	// Setup.
	ctx := testutil.Context(t, testutil.WaitShort)
	logger := testutil.Logger(t)

	// Given: a mock HTTP server which mimicks an unreachable database when calling the coordination endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpapi.Write(ctx, w, http.StatusInternalServerError, nicloudsdk.Response{
			Message: nicloudsdk.DatabaseNotReachable,
			Detail:  "oops",
		})
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	// When: calling the coordination endpoint.
	dialer := workspacesdk.NewWebsocketDialer(logger, u, &websocket.DialOptions{})
	_, err = dialer.Dial(ctx, nil)

	// Then: an error indicating a database issue is returned, to conditionalize the behavior of the caller.
	require.ErrorIs(t, err, nicloudsdk.ErrDatabaseNotReachable)
}

func TestClient_IsNIConnectRunning(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitShort)

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/workspaceagents/connection", r.URL.Path)
		httpapi.Write(ctx, rw, http.StatusOK, workspacesdk.AgentConnectionInfo{
			HostnameSuffix: "test",
		})
	}))
	defer srv.Close()

	apiURL, err := url.Parse(srv.URL)
	require.NoError(t, err)
	sdkClient := nicloudsdk.New(apiURL)
	client := workspacesdk.New(sdkClient)

	// Right name, right IP
	expectedName := fmt.Sprintf(tailnet.IsNIConnectEnabledFmtString, "test")
	ctxResolveExpected := workspacesdk.WithTestOnlyCoderContextResolver(ctx,
		&fakeResolver{t: t, hostMap: map[string][]net.IP{
			expectedName: {net.ParseIP(tsaddr.NIServiceIPv6().String())},
		}})

	result, err := client.IsNIConnectRunning(ctxResolveExpected, workspacesdk.NIConnectQueryOptions{})
	require.NoError(t, err)
	require.True(t, result)

	// Wrong name
	result, err = client.IsNIConnectRunning(ctxResolveExpected, workspacesdk.NIConnectQueryOptions{HostnameSuffix: "coder"})
	require.NoError(t, err)
	require.False(t, result)

	// Not found
	ctxResolveNotFound := workspacesdk.WithTestOnlyCoderContextResolver(ctx,
		&fakeResolver{t: t, err: &net.DNSError{IsNotFound: true}})
	result, err = client.IsNIConnectRunning(ctxResolveNotFound, workspacesdk.NIConnectQueryOptions{})
	require.NoError(t, err)
	require.False(t, result)

	// Some other error
	ctxResolverErr := workspacesdk.WithTestOnlyCoderContextResolver(ctx,
		&fakeResolver{t: t, err: xerrors.New("a bad thing happened")})
	_, err = client.IsNIConnectRunning(ctxResolverErr, workspacesdk.NIConnectQueryOptions{})
	require.Error(t, err)

	// Right name, wrong IP
	ctxResolverWrongIP := workspacesdk.WithTestOnlyCoderContextResolver(ctx,
		&fakeResolver{t: t, hostMap: map[string][]net.IP{
			expectedName: {net.ParseIP("2001::34")},
		}})
	result, err = client.IsNIConnectRunning(ctxResolverWrongIP, workspacesdk.NIConnectQueryOptions{})
	require.NoError(t, err)
	require.False(t, result)
}

type fakeResolver struct {
	t       testing.TB
	hostMap map[string][]net.IP
	err     error
}

func (f *fakeResolver) LookupIP(_ context.Context, network, host string) ([]net.IP, error) {
	assert.Equal(f.t, "ip6", network)
	return f.hostMap[host], f.err
}
