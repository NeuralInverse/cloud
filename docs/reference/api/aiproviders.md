# AI Providers

## List AI providers

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/ai/providers \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/ai/providers`

### Example responses

> 200 Response

```json
[
  {
    "api_keys": [
      {
        "created_at": "2019-08-24T14:15:22Z",
        "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
        "masked": "string"
      }
    ],
    "base_url": "string",
    "created_at": "2019-08-24T14:15:22Z",
    "display_name": "string",
    "enabled": true,
    "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
    "name": "string",
    "settings": {},
    "type": "openai",
    "updated_at": "2019-08-24T14:15:22Z"
  }
]
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                        |
|--------|---------------------------------------------------------|-------------|---------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | array of [nicloudsdk.AIProvider](schemas.md#nicloudsdkaiprovider) |

<h3 id="list-ai-providers-responseschema">Response Schema</h3>

Status Code **200**

| Name             | Type                                                                 | Required | Restrictions | Description |
|------------------|----------------------------------------------------------------------|----------|--------------|-------------|
| `[array item]`   | array                                                                | false    |              |             |
| `» api_keys`     | array                                                                | false    |              |             |
| `»» created_at`  | string(date-time)                                                    | false    |              |             |
| `»» id`          | string(uuid)                                                         | false    |              |             |
| `»» masked`      | string                                                               | false    |              |             |
| `» base_url`     | string                                                               | false    |              |             |
| `» created_at`   | string(date-time)                                                    | false    |              |             |
| `» display_name` | string                                                               | false    |              |             |
| `» enabled`      | boolean                                                              | false    |              |             |
| `» id`           | string(uuid)                                                         | false    |              |             |
| `» name`         | string                                                               | false    |              |             |
| `» settings`     | [nicloudsdk.AIProviderSettings](schemas.md#nicloudsdkaiprovidersettings) | false    |              |             |
| `» type`         | [nicloudsdk.AIProviderType](schemas.md#nicloudsdkaiprovidertype)         | false    |              |             |
| `» updated_at`   | string(date-time)                                                    | false    |              |             |

#### Enumerated Values

| Property | Value(s)                                                                                                |
|----------|---------------------------------------------------------------------------------------------------------|
| `type`   | `anthropic`, `azure`, `bedrock`, `copilot`, `google`, `openai`, `openai-compat`, `openrouter`, `vercel` |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Create an AI provider

### Code samples

```shell
# Example request using curl
curl -X POST http://coder-server:8080/api/v2/ai/providers \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`POST /api/v2/ai/providers`

> Body parameter

```json
{
  "api_keys": [
    "string"
  ],
  "base_url": "string",
  "display_name": "string",
  "enabled": true,
  "name": "string",
  "settings": {},
  "type": "openai"
}
```

### Parameters

| Name   | In   | Type                                                                           | Required | Description                |
|--------|------|--------------------------------------------------------------------------------|----------|----------------------------|
| `body` | body | [nicloudsdk.CreateAIProviderRequest](schemas.md#nicloudsdkcreateaiproviderrequest) | true     | Create AI provider request |

### Example responses

> 201 Response

```json
{
  "api_keys": [
    {
      "created_at": "2019-08-24T14:15:22Z",
      "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
      "masked": "string"
    }
  ],
  "base_url": "string",
  "created_at": "2019-08-24T14:15:22Z",
  "display_name": "string",
  "enabled": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "name": "string",
  "settings": {},
  "type": "openai",
  "updated_at": "2019-08-24T14:15:22Z"
}
```

### Responses

| Status | Meaning                                                      | Description | Schema                                               |
|--------|--------------------------------------------------------------|-------------|------------------------------------------------------|
| 201    | [Created](https://tools.ietf.org/html/rfc7231#section-6.3.2) | Created     | [nicloudsdk.AIProvider](schemas.md#nicloudsdkaiprovider) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get an AI provider

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/ai/providers/{idOrName} \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/ai/providers/{idOrName}`

### Parameters

| Name       | In   | Type   | Required | Description         |
|------------|------|--------|----------|---------------------|
| `idOrName` | path | string | true     | Provider ID or name |

### Example responses

> 200 Response

```json
{
  "api_keys": [
    {
      "created_at": "2019-08-24T14:15:22Z",
      "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
      "masked": "string"
    }
  ],
  "base_url": "string",
  "created_at": "2019-08-24T14:15:22Z",
  "display_name": "string",
  "enabled": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "name": "string",
  "settings": {},
  "type": "openai",
  "updated_at": "2019-08-24T14:15:22Z"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                               |
|--------|---------------------------------------------------------|-------------|------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.AIProvider](schemas.md#nicloudsdkaiprovider) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Delete an AI provider

### Code samples

```shell
# Example request using curl
curl -X DELETE http://coder-server:8080/api/v2/ai/providers/{idOrName} \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`DELETE /api/v2/ai/providers/{idOrName}`

### Parameters

| Name       | In   | Type   | Required | Description         |
|------------|------|--------|----------|---------------------|
| `idOrName` | path | string | true     | Provider ID or name |

### Responses

| Status | Meaning                                                         | Description | Schema |
|--------|-----------------------------------------------------------------|-------------|--------|
| 204    | [No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5) | No Content  |        |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Update an AI provider

### Code samples

```shell
# Example request using curl
curl -X PATCH http://coder-server:8080/api/v2/ai/providers/{idOrName} \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PATCH /api/v2/ai/providers/{idOrName}`

> Body parameter

```json
{
  "api_keys": [
    {
      "api_key": "string",
      "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08"
    }
  ],
  "base_url": "string",
  "display_name": "string",
  "enabled": true,
  "settings": {}
}
```

### Parameters

| Name       | In   | Type                                                                           | Required | Description                |
|------------|------|--------------------------------------------------------------------------------|----------|----------------------------|
| `idOrName` | path | string                                                                         | true     | Provider ID or name        |
| `body`     | body | [nicloudsdk.UpdateAIProviderRequest](schemas.md#nicloudsdkupdateaiproviderrequest) | true     | Update AI provider request |

### Example responses

> 200 Response

```json
{
  "api_keys": [
    {
      "created_at": "2019-08-24T14:15:22Z",
      "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
      "masked": "string"
    }
  ],
  "base_url": "string",
  "created_at": "2019-08-24T14:15:22Z",
  "display_name": "string",
  "enabled": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "name": "string",
  "settings": {},
  "type": "openai",
  "updated_at": "2019-08-24T14:15:22Z"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                               |
|--------|---------------------------------------------------------|-------------|------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.AIProvider](schemas.md#nicloudsdkaiprovider) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).
