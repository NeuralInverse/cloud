# MCP Server

Power users can configure [claude.ai](https://claude.ai), Claude Desktop, Cursor, or other external agents to interact with Neural Inverse Cloud in order to:

- List workspaces
- Create/start/stop workspaces
- Run commands on workspaces
- Check in on agent activity

> [!NOTE]
> See our [toolsdk](https://pkg.go.dev/github.com/NeuralInverse/cloud/v2/nicloudsdk/toolsdk#pkg-variables) documentation for a full list of tools included in the MCP server

In this model, any custom agent could interact with a remote Neural Inverse Cloud workspace, or Neural Inverse Cloud can be used in a remote pipeline or a larger workflow.

## Local MCP server

The Neural Inverse Cloud CLI has options to automatically configure MCP servers for you. On your local machine, run the following command:

```sh
# First log in to Neural Inverse Cloud. 
coder login <https://coder.example.com>

# Configure your client with the Neural Inverse Cloud MCP
coder exp mcp configure claude-desktop # Configure Claude Desktop to interact with Neural Inverse Cloud
coder exp mcp configure cursor # Configure Cursor to interact with Neural Inverse Cloud
```

For other agents, run the MCP server with this command:

```sh
coder exp mcp server
```

> [!NOTE]
> The MCP server is authenticated with the same identity as your Neural Inverse Cloud CLI and can perform any action on the user's behalf. Fine-grained permissions are in development. [Contact us](https://cloud.neuralinverse.com/contact) if this use case is important to you.

## Remote MCP server

Neural Inverse Cloud can expose an MCP server via HTTP. This is useful for connecting web-based agents, like https://claude.ai/, to Neural Inverse Cloud. This is an experimental feature and is subject to change.

To enable this feature, activate the `oauth2` and `mcp-server-http` experiments using an environment variable or a CLI flag:

```sh
NEURALINVERSE_EXPERIMENTS="oauth2,mcp-server-http" coder server
# or
coder server --experiments=oauth2,mcp-server-http
```

The Neural Inverse Cloud server will expose the MCP server at:

```txt
https://coder.example.com/api/experimental/mcp/http
```

> [!NOTE]
> At this time, the remote MCP server is not compatible with web-based ChatGPT.

Users can authenticate applications to use the remote MCP server with [OAuth2](../admin/integrations/oauth2-provider.md). An authenticated application can perform any action on the user's behalf. Fine-grained permissions are in development.
