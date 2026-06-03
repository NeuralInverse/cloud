# Prebuilds

## Get prebuilds settings

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/prebuilds/settings \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/prebuilds/settings`

### Example responses

> 200 Response

```json
{
  "reconciliation_paused": true
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                             |
|--------|---------------------------------------------------------|-------------|--------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.PrebuildsSettings](schemas.md#nicloudsdkprebuildssettings) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Update prebuilds settings

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/prebuilds/settings \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/prebuilds/settings`

> Body parameter

```json
{
  "reconciliation_paused": true
}
```

### Parameters

| Name   | In   | Type                                                               | Required | Description                |
|--------|------|--------------------------------------------------------------------|----------|----------------------------|
| `body` | body | [nicloudsdk.PrebuildsSettings](schemas.md#nicloudsdkprebuildssettings) | true     | Prebuilds settings request |

### Example responses

> 200 Response

```json
{
  "reconciliation_paused": true
}
```

### Responses

| Status | Meaning                                                         | Description  | Schema                                                             |
|--------|-----------------------------------------------------------------|--------------|--------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)         | OK           | [nicloudsdk.PrebuildsSettings](schemas.md#nicloudsdkprebuildssettings) |
| 304    | [Not Modified](https://tools.ietf.org/html/rfc7232#section-4.1) | Not Modified |                                                                    |

To perform this operation, you must be authenticated. [Learn more](authentication.md).
