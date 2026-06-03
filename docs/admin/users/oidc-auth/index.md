# OpenID Connect

The following steps through how to integrate any OpenID Connect provider (Okta,
Active Directory, etc.) to Neural Inverse Cloud.

## Step 1: Set Redirect URI with your OIDC provider

Your OIDC provider will ask you for the following parameter:

- **Redirect URI**: Set to `https://coder.domain.com/api/v2/users/oidc/callback`

## Step 2: Configure Neural Inverse Cloud with the OpenID Connect credentials

Set the following environment variables on your Neural Inverse Cloud deployment and restart Neural Inverse Cloud:

```env
NEURALINVERSE_OIDC_ISSUER_URL="https://issuer.corp.com"
NEURALINVERSE_OIDC_EMAIL_DOMAIN="your-domain-1,your-domain-2"
NEURALINVERSE_OIDC_CLIENT_ID="533...des"
NEURALINVERSE_OIDC_CLIENT_SECRET="G0CSP...7qSM"
```

## OIDC Claims

When a user logs in for the first time via OIDC, Neural Inverse Cloud will merge both the
claims from the ID token and the claims obtained from hitting the upstream
provider's `userinfo` endpoint, and use the resulting data as a basis for
creating a new user or looking up an existing user.

To troubleshoot claims, set `NEURALINVERSE_LOG_FILTER=".*got oidc claims.*"` and follow the logs while
signing in via OIDC as a new user. Neural Inverse Cloud will log the claim fields returned by
the upstream identity provider in a message containing the string
`got oidc claims`, as well as the user info returned.

> [!NOTE]
> If you need to ensure that Neural Inverse Cloud only uses information from the ID
> token and does not hit the UserInfo endpoint, you can set the configuration
> option `NEURALINVERSE_OIDC_IGNORE_USERINFO=true`.

### Email Addresses

By default, Neural Inverse Cloud will look for the OIDC claim named `email` and use that value
for the newly created user's email address.

If your upstream identity provider users a different claim, you can set
`NEURALINVERSE_OIDC_EMAIL_FIELD` to the desired claim.

> [!NOTE]
> If this field is not present, Neural Inverse Cloud will attempt to use the claim
> field configured for `username` as an email address. If this field is not a
> valid email address, OIDC logins will fail.

### Email Address Verification

Neural Inverse Cloud requires all OIDC email addresses to be verified by default. If the
`email_verified` claim is present in the token response from the identity
provider, Neural Inverse Cloud will validate that its value is `true`. If needed, you can
disable this behavior with the following setting:

```env
NEURALINVERSE_OIDC_IGNORE_EMAIL_VERIFIED=true
```

> [!NOTE]
> This will cause Neural Inverse Cloud to implicitly treat all OIDC emails as
> "verified", regardless of what the upstream identity provider says.

### Usernames

When a new user logs in via OIDC, Neural Inverse Cloud will by default use the value of the
claim field named `preferred_username` as the the username.

If your upstream identity provider uses a different claim, you can set
`NEURALINVERSE_OIDC_USERNAME_FIELD` to the desired claim.

> [!NOTE]
> If this claim is empty, the email address will be stripped of the
> domain, and become the username (e.g. `example@cloud.neuralinverse.com` becomes `example`).
> To avoid conflicts, Neural Inverse Cloud may also append a random word to the resulting
> username.

## OIDC Login Customization

If you'd like to change the OpenID Connect button text and/or icon, you can
configure them like so:

```env
NEURALINVERSE_OIDC_SIGN_IN_TEXT="Sign in with Gitea"
NEURALINVERSE_OIDC_ICON_URL=https://gitea.io/images/gitea.png
```

To change the icon and text above the OpenID Connect button, see application
name and logo url in [appearance](../../setup/appearance.md) settings.

## Configure Refresh Tokens

By default, OIDC access tokens typically expire after a short period.
This is typically after one hour, but varies by provider.

Without refresh tokens, users will be automatically logged out when their access token expires.

Follow [Configure OIDC Refresh Tokens](./refresh-tokens.md) for provider-specific steps.

The general steps to configure persistent user sessions are:

1. Configure your Neural Inverse Cloud OIDC settings:

   For most providers, add the `offline_access` scope:

   ```env
   NEURALINVERSE_OIDC_SCOPES=openid,profile,email,offline_access
   ```

   For Google, add auth URL parameters (`NEURALINVERSE_OIDC_AUTH_URL_PARAMS`) too:

   ```env
   NEURALINVERSE_OIDC_SCOPES=openid,profile,email
   NEURALINVERSE_OIDC_AUTH_URL_PARAMS='{"access_type": "offline", "prompt": "consent"}'
   ```

1. Configure your identity provider to issue refresh tokens.

1. After configuration, have users log out and back in once to obtain refresh tokens

> [!IMPORTANT]
> Misconfigured refresh tokens can lead to frequent user authentication prompts.

## Disable Built-in Authentication

To remove email and password login, set the following environment variable on
your Neural Inverse Cloud deployment:

```env
NEURALINVERSE_DISABLE_PASSWORD_AUTH=true
```

## SCIM

> [!IMPORTANT]
> SCIM is a Premium feature
> ([learn more](https://cloud.neuralinverse.com/pricing#compare-plans)).
>
> Neural Inverse Cloud's SCIM 2.0 implementation is not a fully certified or guaranteed
> implementation of the [SCIM 2.0 specification](https://datatracker.ietf.org/doc/html/rfc7644).
> It is intended to cover common user provisioning and deprovisioning flows
> with the major identity providers (Okta, Microsoft Entra ID, etc.). Specific
> attributes, endpoints, or behaviors required by your IdP may not be
> supported, and compatibility may change between releases. If you depend on
> a specific SCIM behavior, [contact us](https://cloud.neuralinverse.com/contact) before
> rolling it out broadly. See
> [coder/coder#15830](https://github.com/NeuralInverse/cloud/issues/15830) for
> tracked gaps and ongoing work.

Neural Inverse Cloud supports user provisioning and deprovisioning via SCIM 2.0 with header
authentication. Upon deactivation, users are
[suspended](../index.md#suspend-a-user) and are not deleted.
[Configure](../../setup/index.md) your SCIM application with an auth key and supply
it the Neural Inverse Cloud server.

```env
NEURALINVERSE_SCIM_AUTH_HEADER="your-api-key"
```

## TLS

If your OpenID Connect provider requires client TLS certificates for
authentication, you can configure them like so:

```env
NEURALINVERSE_TLS_CLIENT_CERT_FILE=/path/to/cert.pem
NEURALINVERSE_TLS_CLIENT_KEY_FILE=/path/to/key.pem
```

## Next steps

- [Group Sync](../idp-sync.md)
- [Groups & Roles](../groups-roles.md)
- [Configure OIDC Refresh Tokens](./refresh-tokens.md)
