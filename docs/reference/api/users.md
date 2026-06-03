# Users

## Get users

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users`

### Parameters

| Name       | In    | Type         | Required | Description  |
|------------|-------|--------------|----------|--------------|
| `q`        | query | string       | false    | Search query |
| `after_id` | query | string(uuid) | false    | After ID     |
| `limit`    | query | integer      | false    | Page limit   |
| `offset`   | query | integer      | false    | Page offset  |

### Example responses

> 200 Response

```json
{
  "count": 0,
  "users": [
    {
      "avatar_url": "http://example.com",
      "created_at": "2019-08-24T14:15:22Z",
      "email": "user@example.com",
      "has_ai_seat": true,
      "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
      "is_service_account": true,
      "last_seen_at": "2019-08-24T14:15:22Z",
      "login_type": "",
      "name": "string",
      "organization_ids": [
        "497f6eca-6276-4993-bfeb-53cbbbba6f08"
      ],
      "roles": [
        {
          "display_name": "string",
          "name": "string",
          "organization_id": "string"
        }
      ],
      "status": "active",
      "theme_preference": "string",
      "updated_at": "2019-08-24T14:15:22Z",
      "username": "string"
    }
  ]
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                           |
|--------|---------------------------------------------------------|-------------|------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.GetUsersResponse](schemas.md#nicloudsdkgetusersresponse) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Create new user

### Code samples

```shell
# Example request using curl
curl -X POST http://coder-server:8080/api/v2/users \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`POST /api/v2/users`

> Body parameter

```json
{
  "email": "user@example.com",
  "login_type": "",
  "name": "string",
  "organization_ids": [
    "497f6eca-6276-4993-bfeb-53cbbbba6f08"
  ],
  "password": "string",
  "roles": [
    "string"
  ],
  "service_account": true,
  "user_status": "active",
  "username": "string"
}
```

### Parameters

| Name   | In   | Type                                                                               | Required | Description         |
|--------|------|------------------------------------------------------------------------------------|----------|---------------------|
| `body` | body | [nicloudsdk.CreateUserRequestWithOrgs](schemas.md#nicloudsdkcreateuserrequestwithorgs) | true     | Create user request |

### Example responses

> 201 Response

```json
{
  "avatar_url": "http://example.com",
  "created_at": "2019-08-24T14:15:22Z",
  "email": "user@example.com",
  "has_ai_seat": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "is_service_account": true,
  "last_seen_at": "2019-08-24T14:15:22Z",
  "login_type": "",
  "name": "string",
  "organization_ids": [
    "497f6eca-6276-4993-bfeb-53cbbbba6f08"
  ],
  "roles": [
    {
      "display_name": "string",
      "name": "string",
      "organization_id": "string"
    }
  ],
  "status": "active",
  "theme_preference": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "username": "string"
}
```

### Responses

| Status | Meaning                                                      | Description | Schema                                   |
|--------|--------------------------------------------------------------|-------------|------------------------------------------|
| 201    | [Created](https://tools.ietf.org/html/rfc7231#section-6.3.2) | Created     | [nicloudsdk.User](schemas.md#nicloudsdkuser) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get authentication methods

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/authmethods \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/authmethods`

### Example responses

> 200 Response

```json
{
  "github": {
    "default_provider_configured": true,
    "enabled": true
  },
  "oidc": {
    "enabled": true,
    "iconUrl": "string",
    "signInText": "string"
  },
  "password": {
    "enabled": true
  },
  "terms_of_service_url": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                 |
|--------|---------------------------------------------------------|-------------|--------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.AuthMethods](schemas.md#nicloudsdkauthmethods) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Check initial user created

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/first \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/first`

### Example responses

> 200 Response

```json
{
  "detail": "string",
  "message": "string",
  "validations": [
    {
      "detail": "string",
      "field": "string"
    }
  ]
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                           |
|--------|---------------------------------------------------------|-------------|--------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.Response](schemas.md#nicloudsdkresponse) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Create initial user

### Code samples

```shell
# Example request using curl
curl -X POST http://coder-server:8080/api/v2/users/first \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`POST /api/v2/users/first`

> Body parameter

```json
{
  "email": "string",
  "name": "string",
  "onboarding_info": {
    "newsletter_marketing": true,
    "newsletter_releases": true
  },
  "password": "string",
  "trial": true,
  "trial_info": {
    "company_name": "string",
    "country": "string",
    "developers": "string",
    "first_name": "string",
    "job_title": "string",
    "last_name": "string",
    "phone_number": "string"
  },
  "username": "string"
}
```

### Parameters

| Name   | In   | Type                                                                         | Required | Description        |
|--------|------|------------------------------------------------------------------------------|----------|--------------------|
| `body` | body | [nicloudsdk.CreateFirstUserRequest](schemas.md#nicloudsdkcreatefirstuserrequest) | true     | First user request |

### Example responses

> 201 Response

```json
{
  "organization_id": "7c60d51f-b44e-4682-87d6-449835ea4de6",
  "user_id": "a169451c-8525-4352-b8ca-070dd449a1a5"
}
```

### Responses

| Status | Meaning                                                      | Description | Schema                                                                         |
|--------|--------------------------------------------------------------|-------------|--------------------------------------------------------------------------------|
| 201    | [Created](https://tools.ietf.org/html/rfc7231#section-6.3.2) | Created     | [nicloudsdk.CreateFirstUserResponse](schemas.md#nicloudsdkcreatefirstuserresponse) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Log out user

### Code samples

```shell
# Example request using curl
curl -X POST http://coder-server:8080/api/v2/users/logout \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`POST /api/v2/users/logout`

### Example responses

> 200 Response

```json
{
  "detail": "string",
  "message": "string",
  "validations": [
    {
      "detail": "string",
      "field": "string"
    }
  ]
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                           |
|--------|---------------------------------------------------------|-------------|--------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.Response](schemas.md#nicloudsdkresponse) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## OAuth 2.0 GitHub Callback

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/oauth2/github/callback \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/oauth2/github/callback`

### Responses

| Status | Meaning                                                                 | Description        | Schema |
|--------|-------------------------------------------------------------------------|--------------------|--------|
| 307    | [Temporary Redirect](https://tools.ietf.org/html/rfc7231#section-6.4.7) | Temporary Redirect |        |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get Github device auth

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/oauth2/github/device \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/oauth2/github/device`

### Example responses

> 200 Response

```json
{
  "device_code": "string",
  "expires_in": 0,
  "interval": 0,
  "user_code": "string",
  "verification_uri": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                               |
|--------|---------------------------------------------------------|-------------|----------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.ExternalAuthDevice](schemas.md#nicloudsdkexternalauthdevice) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get OIDC claims for the authenticated user

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/oidc-claims \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/oidc-claims`

### Example responses

> 200 Response

```json
{
  "claims": {}
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                               |
|--------|---------------------------------------------------------|-------------|----------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.OIDCClaimsResponse](schemas.md#nicloudsdkoidcclaimsresponse) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## OpenID Connect Callback

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/oidc/callback \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/oidc/callback`

### Responses

| Status | Meaning                                                                 | Description        | Schema |
|--------|-------------------------------------------------------------------------|--------------------|--------|
| 307    | [Temporary Redirect](https://tools.ietf.org/html/rfc7231#section-6.4.7) | Temporary Redirect |        |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get user by name

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user} \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}`

### Parameters

| Name   | In   | Type   | Required | Description              |
|--------|------|--------|----------|--------------------------|
| `user` | path | string | true     | User ID, username, or me |

### Example responses

> 200 Response

```json
{
  "avatar_url": "http://example.com",
  "created_at": "2019-08-24T14:15:22Z",
  "email": "user@example.com",
  "has_ai_seat": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "is_service_account": true,
  "last_seen_at": "2019-08-24T14:15:22Z",
  "login_type": "",
  "name": "string",
  "organization_ids": [
    "497f6eca-6276-4993-bfeb-53cbbbba6f08"
  ],
  "roles": [
    {
      "display_name": "string",
      "name": "string",
      "organization_id": "string"
    }
  ],
  "status": "active",
  "theme_preference": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "username": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                   |
|--------|---------------------------------------------------------|-------------|------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.User](schemas.md#nicloudsdkuser) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Delete user

### Code samples

```shell
# Example request using curl
curl -X DELETE http://coder-server:8080/api/v2/users/{user} \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`DELETE /api/v2/users/{user}`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Responses

| Status | Meaning                                                 | Description | Schema |
|--------|---------------------------------------------------------|-------------|--------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          |        |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get user appearance settings

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/appearance \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/appearance`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
{
  "terminal_font": "",
  "theme_dark": "string",
  "theme_light": "string",
  "theme_mode": "",
  "theme_preference": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                                       |
|--------|---------------------------------------------------------|-------------|------------------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.UserAppearanceSettings](schemas.md#nicloudsdkuserappearancesettings) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Update user appearance settings

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/appearance \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/appearance`

> Body parameter

```json
{
  "terminal_font": "",
  "theme_dark": "light",
  "theme_light": "light",
  "theme_mode": "sync",
  "theme_preference": "string"
}
```

### Parameters

| Name   | In   | Type                                                                                                   | Required | Description             |
|--------|------|--------------------------------------------------------------------------------------------------------|----------|-------------------------|
| `user` | path | string                                                                                                 | true     | User ID, name, or me    |
| `body` | body | [nicloudsdk.UpdateUserAppearanceSettingsRequest](schemas.md#nicloudsdkupdateuserappearancesettingsrequest) | true     | New appearance settings |

### Example responses

> 200 Response

```json
{
  "terminal_font": "",
  "theme_dark": "string",
  "theme_light": "string",
  "theme_mode": "",
  "theme_preference": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                                       |
|--------|---------------------------------------------------------|-------------|------------------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.UserAppearanceSettings](schemas.md#nicloudsdkuserappearancesettings) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get autofill build parameters for user

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/autofill-parameters?template_id=string \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/autofill-parameters`

### Parameters

| Name          | In    | Type   | Required | Description              |
|---------------|-------|--------|----------|--------------------------|
| `user`        | path  | string | true     | User ID, username, or me |
| `template_id` | query | string | true     | Template ID              |

### Example responses

> 200 Response

```json
[
  {
    "name": "string",
    "value": "string"
  }
]
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                              |
|--------|---------------------------------------------------------|-------------|---------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | array of [nicloudsdk.UserParameter](schemas.md#nicloudsdkuserparameter) |

<h3 id="get-autofill-build-parameters-for-user-responseschema">Response Schema</h3>

Status Code **200**

| Name           | Type   | Required | Restrictions | Description |
|----------------|--------|----------|--------------|-------------|
| `[array item]` | array  | false    |              |             |
| `» name`       | string | false    |              |             |
| `» value`      | string | false    |              |             |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get user Git SSH key

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/gitsshkey \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/gitsshkey`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
{
  "created_at": "2019-08-24T14:15:22Z",
  "public_key": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "user_id": "a169451c-8525-4352-b8ca-070dd449a1a5"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                             |
|--------|---------------------------------------------------------|-------------|----------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.GitSSHKey](schemas.md#nicloudsdkgitsshkey) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Regenerate user SSH key

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/gitsshkey \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/gitsshkey`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
{
  "created_at": "2019-08-24T14:15:22Z",
  "public_key": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "user_id": "a169451c-8525-4352-b8ca-070dd449a1a5"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                             |
|--------|---------------------------------------------------------|-------------|----------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.GitSSHKey](schemas.md#nicloudsdkgitsshkey) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Create new session key

### Code samples

```shell
# Example request using curl
curl -X POST http://coder-server:8080/api/v2/users/{user}/keys \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`POST /api/v2/users/{user}/keys`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 201 Response

```json
{
  "key": "string"
}
```

### Responses

| Status | Meaning                                                      | Description | Schema                                                                       |
|--------|--------------------------------------------------------------|-------------|------------------------------------------------------------------------------|
| 201    | [Created](https://tools.ietf.org/html/rfc7231#section-6.3.2) | Created     | [nicloudsdk.GenerateAPIKeyResponse](schemas.md#nicloudsdkgenerateapikeyresponse) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get user tokens

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/keys/tokens \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/keys/tokens`

### Parameters

| Name              | In    | Type    | Required | Description                        |
|-------------------|-------|---------|----------|------------------------------------|
| `user`            | path  | string  | true     | User ID, name, or me               |
| `include_expired` | query | boolean | false    | Include expired tokens in the list |

### Example responses

> 200 Response

```json
[
  {
    "allow_list": [
      {
        "id": "string",
        "type": "*"
      }
    ],
    "created_at": "2019-08-24T14:15:22Z",
    "expires_at": "2019-08-24T14:15:22Z",
    "id": "string",
    "last_used": "2019-08-24T14:15:22Z",
    "lifetime_seconds": 0,
    "login_type": "password",
    "scope": "all",
    "scopes": [
      "all"
    ],
    "token_name": "string",
    "updated_at": "2019-08-24T14:15:22Z",
    "user_id": "a169451c-8525-4352-b8ca-070dd449a1a5"
  }
]
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                |
|--------|---------------------------------------------------------|-------------|-------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | array of [nicloudsdk.APIKey](schemas.md#nicloudsdkapikey) |

<h3 id="get-user-tokens-responseschema">Response Schema</h3>

Status Code **200**

| Name                 | Type                                                     | Required | Restrictions | Description                     |
|----------------------|----------------------------------------------------------|----------|--------------|---------------------------------|
| `[array item]`       | array                                                    | false    |              |                                 |
| `» allow_list`       | array                                                    | false    |              |                                 |
| `»» id`              | string                                                   | false    |              |                                 |
| `»» type`            | [nicloudsdk.RBACResource](schemas.md#nicloudsdkrbacresource) | false    |              |                                 |
| `» created_at`       | string(date-time)                                        | true     |              |                                 |
| `» expires_at`       | string(date-time)                                        | true     |              |                                 |
| `» id`               | string                                                   | true     |              |                                 |
| `» last_used`        | string(date-time)                                        | true     |              |                                 |
| `» lifetime_seconds` | integer                                                  | true     |              |                                 |
| `» login_type`       | [nicloudsdk.LoginType](schemas.md#nicloudsdklogintype)       | true     |              |                                 |
| `» scope`            | [nicloudsdk.APIKeyScope](schemas.md#nicloudsdkapikeyscope)   | false    |              | Deprecated: use Scopes instead. |
| `» scopes`           | array                                                    | false    |              |                                 |
| `» token_name`       | string                                                   | true     |              |                                 |
| `» updated_at`       | string(date-time)                                        | true     |              |                                 |
| `» user_id`          | string(uuid)                                             | true     |              |                                 |

#### Enumerated Values

| Property     | Value(s)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
|--------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `type`       | `*`, `ai_gateway_key`, `ai_model_price`, `ai_provider`, `ai_seat`, `aibridge_interception`, `api_key`, `assign_org_role`, `assign_role`, `audit_log`, `boundary_log`, `boundary_usage`, `chat`, `connection_log`, `crypto_key`, `debug_info`, `deployment_config`, `deployment_stats`, `file`, `group`, `group_member`, `idpsync_settings`, `inbox_notification`, `license`, `notification_message`, `notification_preference`, `notification_template`, `oauth2_app`, `oauth2_app_code_token`, `oauth2_app_secret`, `organization`, `organization_member`, `prebuilt_workspace`, `provisioner_daemon`, `provisioner_jobs`, `replicas`, `system`, `tailnet_coordinator`, `task`, `template`, `usage_event`, `user`, `user_secret`, `user_skill`, `webpush_subscription`, `workspace`, `workspace_agent_devcontainers`, `workspace_agent_resource_monitor`, `workspace_dormant`, `workspace_proxy` |
| `login_type` | `github`, `oidc`, `password`, `token`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `scope`      | `all`, `application_connect`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Create token API key

### Code samples

```shell
# Example request using curl
curl -X POST http://coder-server:8080/api/v2/users/{user}/keys/tokens \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`POST /api/v2/users/{user}/keys/tokens`

> Body parameter

```json
{
  "allow_list": [
    {
      "id": "string",
      "type": "*"
    }
  ],
  "lifetime": 0,
  "scope": "all",
  "scopes": [
    "all"
  ],
  "token_name": "string"
}
```

### Parameters

| Name   | In   | Type                                                                 | Required | Description          |
|--------|------|----------------------------------------------------------------------|----------|----------------------|
| `user` | path | string                                                               | true     | User ID, name, or me |
| `body` | body | [nicloudsdk.CreateTokenRequest](schemas.md#nicloudsdkcreatetokenrequest) | true     | Create token request |

### Example responses

> 201 Response

```json
{
  "key": "string"
}
```

### Responses

| Status | Meaning                                                      | Description | Schema                                                                       |
|--------|--------------------------------------------------------------|-------------|------------------------------------------------------------------------------|
| 201    | [Created](https://tools.ietf.org/html/rfc7231#section-6.3.2) | Created     | [nicloudsdk.GenerateAPIKeyResponse](schemas.md#nicloudsdkgenerateapikeyresponse) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get API key by token name

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/keys/tokens/{keyname} \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/keys/tokens/{keyname}`

### Parameters

| Name      | In   | Type           | Required | Description          |
|-----------|------|----------------|----------|----------------------|
| `user`    | path | string         | true     | User ID, name, or me |
| `keyname` | path | string(string) | true     | Key Name             |

### Example responses

> 200 Response

```json
{
  "allow_list": [
    {
      "id": "string",
      "type": "*"
    }
  ],
  "created_at": "2019-08-24T14:15:22Z",
  "expires_at": "2019-08-24T14:15:22Z",
  "id": "string",
  "last_used": "2019-08-24T14:15:22Z",
  "lifetime_seconds": 0,
  "login_type": "password",
  "scope": "all",
  "scopes": [
    "all"
  ],
  "token_name": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "user_id": "a169451c-8525-4352-b8ca-070dd449a1a5"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                       |
|--------|---------------------------------------------------------|-------------|----------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.APIKey](schemas.md#nicloudsdkapikey) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get API key by ID

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/keys/{keyid} \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/keys/{keyid}`

### Parameters

| Name    | In   | Type           | Required | Description          |
|---------|------|----------------|----------|----------------------|
| `user`  | path | string         | true     | User ID, name, or me |
| `keyid` | path | string(string) | true     | Key ID               |

### Example responses

> 200 Response

```json
{
  "allow_list": [
    {
      "id": "string",
      "type": "*"
    }
  ],
  "created_at": "2019-08-24T14:15:22Z",
  "expires_at": "2019-08-24T14:15:22Z",
  "id": "string",
  "last_used": "2019-08-24T14:15:22Z",
  "lifetime_seconds": 0,
  "login_type": "password",
  "scope": "all",
  "scopes": [
    "all"
  ],
  "token_name": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "user_id": "a169451c-8525-4352-b8ca-070dd449a1a5"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                       |
|--------|---------------------------------------------------------|-------------|----------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.APIKey](schemas.md#nicloudsdkapikey) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Delete API key

### Code samples

```shell
# Example request using curl
curl -X DELETE http://coder-server:8080/api/v2/users/{user}/keys/{keyid} \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`DELETE /api/v2/users/{user}/keys/{keyid}`

### Parameters

| Name    | In   | Type           | Required | Description          |
|---------|------|----------------|----------|----------------------|
| `user`  | path | string         | true     | User ID, name, or me |
| `keyid` | path | string(string) | true     | Key ID               |

### Responses

| Status | Meaning                                                         | Description | Schema |
|--------|-----------------------------------------------------------------|-------------|--------|
| 204    | [No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5) | No Content  |        |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Expire API key

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/keys/{keyid}/expire \
  -H 'Accept: */*' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/keys/{keyid}/expire`

### Parameters

| Name    | In   | Type           | Required | Description          |
|---------|------|----------------|----------|----------------------|
| `user`  | path | string         | true     | User ID, name, or me |
| `keyid` | path | string(string) | true     | Key ID               |

### Example responses

> 404 Response

### Responses

| Status | Meaning                                                                    | Description           | Schema                                           |
|--------|----------------------------------------------------------------------------|-----------------------|--------------------------------------------------|
| 204    | [No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)            | No Content            |                                                  |
| 404    | [Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)             | Not Found             | [nicloudsdk.Response](schemas.md#nicloudsdkresponse) |
| 500    | [Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1) | Internal Server Error | [nicloudsdk.Response](schemas.md#nicloudsdkresponse) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get user login type

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/login-type \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/login-type`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
{
  "login_type": ""
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                     |
|--------|---------------------------------------------------------|-------------|------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.UserLoginType](schemas.md#nicloudsdkuserlogintype) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get organizations by user

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/organizations \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/organizations`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
[
  {
    "created_at": "2019-08-24T14:15:22Z",
    "description": "string",
    "display_name": "string",
    "icon": "string",
    "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
    "is_default": true,
    "name": "string",
    "updated_at": "2019-08-24T14:15:22Z"
  }
]
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                            |
|--------|---------------------------------------------------------|-------------|-------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | array of [nicloudsdk.Organization](schemas.md#nicloudsdkorganization) |

<h3 id="get-organizations-by-user-responseschema">Response Schema</h3>

Status Code **200**

| Name             | Type              | Required | Restrictions | Description |
|------------------|-------------------|----------|--------------|-------------|
| `[array item]`   | array             | false    |              |             |
| `» created_at`   | string(date-time) | true     |              |             |
| `» description`  | string            | false    |              |             |
| `» display_name` | string            | false    |              |             |
| `» icon`         | string            | false    |              |             |
| `» id`           | string(uuid)      | true     |              |             |
| `» is_default`   | boolean           | true     |              |             |
| `» name`         | string            | false    |              |             |
| `» updated_at`   | string(date-time) | true     |              |             |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get organization by user and organization name

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/organizations/{organizationname} \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/organizations/{organizationname}`

### Parameters

| Name               | In   | Type   | Required | Description          |
|--------------------|------|--------|----------|----------------------|
| `user`             | path | string | true     | User ID, name, or me |
| `organizationname` | path | string | true     | Organization name    |

### Example responses

> 200 Response

```json
{
  "created_at": "2019-08-24T14:15:22Z",
  "description": "string",
  "display_name": "string",
  "icon": "string",
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "is_default": true,
  "name": "string",
  "updated_at": "2019-08-24T14:15:22Z"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                   |
|--------|---------------------------------------------------------|-------------|----------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.Organization](schemas.md#nicloudsdkorganization) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Update user password

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/password \
  -H 'Content-Type: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/password`

> Body parameter

```json
{
  "old_password": "string",
  "password": "string"
}
```

### Parameters

| Name   | In   | Type                                                                               | Required | Description             |
|--------|------|------------------------------------------------------------------------------------|----------|-------------------------|
| `user` | path | string                                                                             | true     | User ID, name, or me    |
| `body` | body | [nicloudsdk.UpdateUserPasswordRequest](schemas.md#nicloudsdkupdateuserpasswordrequest) | true     | Update password request |

### Responses

| Status | Meaning                                                         | Description | Schema |
|--------|-----------------------------------------------------------------|-------------|--------|
| 204    | [No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5) | No Content  |        |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get user preference settings

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/preferences \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/preferences`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
{
  "agent_chat_send_shortcut": "enter",
  "code_diff_display_mode": "auto",
  "shell_tool_display_mode": "auto",
  "task_notification_alert_dismissed": true,
  "thinking_display_mode": "auto"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                                       |
|--------|---------------------------------------------------------|-------------|------------------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.UserPreferenceSettings](schemas.md#nicloudsdkuserpreferencesettings) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Update user preference settings

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/preferences \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/preferences`

> Body parameter

```json
{
  "agent_chat_send_shortcut": "enter",
  "code_diff_display_mode": "auto",
  "shell_tool_display_mode": "auto",
  "task_notification_alert_dismissed": true,
  "thinking_display_mode": "auto"
}
```

### Parameters

| Name   | In   | Type                                                                                                   | Required | Description             |
|--------|------|--------------------------------------------------------------------------------------------------------|----------|-------------------------|
| `user` | path | string                                                                                                 | true     | User ID, name, or me    |
| `body` | body | [nicloudsdk.UpdateUserPreferenceSettingsRequest](schemas.md#nicloudsdkupdateuserpreferencesettingsrequest) | true     | New preference settings |

### Example responses

> 200 Response

```json
{
  "agent_chat_send_shortcut": "enter",
  "code_diff_display_mode": "auto",
  "shell_tool_display_mode": "auto",
  "task_notification_alert_dismissed": true,
  "thinking_display_mode": "auto"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                                                       |
|--------|---------------------------------------------------------|-------------|------------------------------------------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.UserPreferenceSettings](schemas.md#nicloudsdkuserpreferencesettings) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Update user profile

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/profile \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/profile`

> Body parameter

```json
{
  "name": "string",
  "username": "string"
}
```

### Parameters

| Name   | In   | Type                                                                             | Required | Description          |
|--------|------|----------------------------------------------------------------------------------|----------|----------------------|
| `user` | path | string                                                                           | true     | User ID, name, or me |
| `body` | body | [nicloudsdk.UpdateUserProfileRequest](schemas.md#nicloudsdkupdateuserprofilerequest) | true     | Updated profile      |

### Example responses

> 200 Response

```json
{
  "avatar_url": "http://example.com",
  "created_at": "2019-08-24T14:15:22Z",
  "email": "user@example.com",
  "has_ai_seat": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "is_service_account": true,
  "last_seen_at": "2019-08-24T14:15:22Z",
  "login_type": "",
  "name": "string",
  "organization_ids": [
    "497f6eca-6276-4993-bfeb-53cbbbba6f08"
  ],
  "roles": [
    {
      "display_name": "string",
      "name": "string",
      "organization_id": "string"
    }
  ],
  "status": "active",
  "theme_preference": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "username": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                   |
|--------|---------------------------------------------------------|-------------|------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.User](schemas.md#nicloudsdkuser) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Get user roles

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/roles \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`GET /api/v2/users/{user}/roles`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
{
  "avatar_url": "http://example.com",
  "created_at": "2019-08-24T14:15:22Z",
  "email": "user@example.com",
  "has_ai_seat": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "is_service_account": true,
  "last_seen_at": "2019-08-24T14:15:22Z",
  "login_type": "",
  "name": "string",
  "organization_ids": [
    "497f6eca-6276-4993-bfeb-53cbbbba6f08"
  ],
  "roles": [
    {
      "display_name": "string",
      "name": "string",
      "organization_id": "string"
    }
  ],
  "status": "active",
  "theme_preference": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "username": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                   |
|--------|---------------------------------------------------------|-------------|------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.User](schemas.md#nicloudsdkuser) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Assign role to user

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/roles \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/roles`

> Body parameter

```json
{
  "roles": [
    "string"
  ]
}
```

### Parameters

| Name   | In   | Type                                                   | Required | Description          |
|--------|------|--------------------------------------------------------|----------|----------------------|
| `user` | path | string                                                 | true     | User ID, name, or me |
| `body` | body | [nicloudsdk.UpdateRoles](schemas.md#nicloudsdkupdateroles) | true     | Update roles request |

### Example responses

> 200 Response

```json
{
  "avatar_url": "http://example.com",
  "created_at": "2019-08-24T14:15:22Z",
  "email": "user@example.com",
  "has_ai_seat": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "is_service_account": true,
  "last_seen_at": "2019-08-24T14:15:22Z",
  "login_type": "",
  "name": "string",
  "organization_ids": [
    "497f6eca-6276-4993-bfeb-53cbbbba6f08"
  ],
  "roles": [
    {
      "display_name": "string",
      "name": "string",
      "organization_id": "string"
    }
  ],
  "status": "active",
  "theme_preference": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "username": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                   |
|--------|---------------------------------------------------------|-------------|------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.User](schemas.md#nicloudsdkuser) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Activate user account

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/status/activate \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/status/activate`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
{
  "avatar_url": "http://example.com",
  "created_at": "2019-08-24T14:15:22Z",
  "email": "user@example.com",
  "has_ai_seat": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "is_service_account": true,
  "last_seen_at": "2019-08-24T14:15:22Z",
  "login_type": "",
  "name": "string",
  "organization_ids": [
    "497f6eca-6276-4993-bfeb-53cbbbba6f08"
  ],
  "roles": [
    {
      "display_name": "string",
      "name": "string",
      "organization_id": "string"
    }
  ],
  "status": "active",
  "theme_preference": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "username": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                   |
|--------|---------------------------------------------------------|-------------|------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.User](schemas.md#nicloudsdkuser) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).

## Suspend user account

### Code samples

```shell
# Example request using curl
curl -X PUT http://coder-server:8080/api/v2/users/{user}/status/suspend \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

`PUT /api/v2/users/{user}/status/suspend`

### Parameters

| Name   | In   | Type   | Required | Description          |
|--------|------|--------|----------|----------------------|
| `user` | path | string | true     | User ID, name, or me |

### Example responses

> 200 Response

```json
{
  "avatar_url": "http://example.com",
  "created_at": "2019-08-24T14:15:22Z",
  "email": "user@example.com",
  "has_ai_seat": true,
  "id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
  "is_service_account": true,
  "last_seen_at": "2019-08-24T14:15:22Z",
  "login_type": "",
  "name": "string",
  "organization_ids": [
    "497f6eca-6276-4993-bfeb-53cbbbba6f08"
  ],
  "roles": [
    {
      "display_name": "string",
      "name": "string",
      "organization_id": "string"
    }
  ],
  "status": "active",
  "theme_preference": "string",
  "updated_at": "2019-08-24T14:15:22Z",
  "username": "string"
}
```

### Responses

| Status | Meaning                                                 | Description | Schema                                   |
|--------|---------------------------------------------------------|-------------|------------------------------------------|
| 200    | [OK](https://tools.ietf.org/html/rfc7231#section-6.3.1) | OK          | [nicloudsdk.User](schemas.md#nicloudsdkuser) |

To perform this operation, you must be authenticated. [Learn more](authentication.md).
