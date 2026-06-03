# PortSharing

## Get workspace agent port shares

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/workspaces/{workspace}/port-share \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/workspaces/{workspace}/port-share`

### Parameters

| Name        | In   | Type         | Required | Description  |
|-------------|------|--------------|----------|--------------|
| `workspace` | path | string(uuid) | true     | Workspace ID |

### Example responses

> 200 Response

```json
{
  "shares": [
    {
      "agent_name": "string",
      "port": 0,
      "protocol": "http",
      "share_level": "owner",
      "workspace_id": "0967198e-ec7b-4c6b-b4d3-f71244cadbe9"
    }
  ]
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                                           |
|--------|---------------------------------------------------------|-------------|----------------------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.WorkspaceAgentPortShares](schemas.md#nicloudsdkworkspaceagentportshares) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Upsert workspace agent port share

### Code samples

```shell
# Example request using curl
curl -X POST http://coder-server:8080/api/v2/workspaces/{workspace}/port-share \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`POST /api/v2/workspaces/{workspace}/port-share`

> Body parameter

```json
{
  "agent_name": "string",
  "port": 0,
  "protocol": "http",
  "share_level": "owner"
}
```

### Parameters

| Name        | In   | Type                                                                                                     | Required | Description                       |
|-------------|------|----------------------------------------------------------------------------------------------------------|----------|-----------------------------------|
| `workspace` | path | string(uuid)                                                                                             | true     | Workspace ID                      |
| `body`      | body | [nicloudsdk.UpsertWorkspaceAgentPortShareRequest](schemas.md#nicloudsdkupsertworkspaceagentportsharerequest) | true     | Upsert port sharing level request |

### Example responses

> 200 Response

```json
{
  "agent_name": "string",
  "port": 0,
  "protocol": "http",
  "share_level": "owner",
  "workspace_id": "0967198e-ec7b-4c6b-b4d3-f71244cadbe9"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                                         |
|--------|---------------------------------------------------------|-------------|--------------------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.WorkspaceAgentPortShare](schemas.md#nicloudsdkworkspaceagentportshare) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Delete workspace agent port share

### Code samples

```shell
# Example request using curl
curl -X DELETE http://coder-server:8080/api/v2/workspaces/{workspace}/port-share \
  -H 'Content-Type: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`DELETE /api/v2/workspaces/{workspace}/port-share`

> Body parameter

```json
{
  "agent_name": "string",
  "port": 0
}
```

### Parameters

| Name        | In   | Type                                                                                                     | Required | Description                       |
|-------------|------|----------------------------------------------------------------------------------------------------------|----------|-----------------------------------|
| `workspace` | path | string(uuid)                                                                                             | true     | Workspace ID                      |
| `body`      | body | [nicloudsdk.DeleteWorkspaceAgentPortShareRequest](schemas.md#nicloudsdkdeleteworkspaceagentportsharerequest) | true     | Delete port sharing level request |

### Responses

| Status | Meaning                                                 | Description | Schema |
|--------|---------------------------------------------------------|-------------|--------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          |        |

To perform this operation, you must be authenticated. [Learn more](authentication.md).
