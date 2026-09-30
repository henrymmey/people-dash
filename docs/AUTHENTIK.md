# Authentik OIDC setup

People Dash uses Authentik as its only identity provider.

Existing Authentik:

```text
https://auth.meyerwolke.de
192.168.176.102
```

No LDAP integration is required.

## 1. Create the groups

In Authentik, create:

```text
people-users
people-admins
```

Recommended meaning:

```text
people-users
    Can use People Dash.

people-admins
    Can use the People Dash administration area.
```

An admin should also belong to `people-users`.

Use group bindings in Authentik to restrict access to the application in addition to the dashboard's own group check.

## 2. Create the OIDC application

In the Authentik Admin interface:

```text
Applications -> Applications -> Create with provider
```

Choose:

```text
Provider type:
OAuth2/OIDC
```

Use these values:

```text
Application name:
People Dash

Slug:
people

Client type:
Confidential

Redirect URI:
https://p.meyerbrief.de/auth/callback
```

The exact form labels can change slightly between Authentik releases. The redirect URI itself must be exact.

## 3. Scopes

People Dash requests:

```text
openid
profile
email
```

The `profile` scope supplied by current Authentik releases includes basic profile information and group membership by default. The application still verifies the `people-users` group itself.

The dashboard obtains claims through the OIDC UserInfo endpoint.

## 4. Verify the provider issuer

For a provider slug of `people`, the issuer normally looks like:

```text
https://auth.meyerwolke.de/application/o/people/
```

Do not blindly copy this value if you selected another slug.

Open your provider metadata in Authentik and copy the actual issuer into:

```dotenv
OIDC_ISSUER=https://auth.meyerwolke.de/application/o/people/
```

## 5. Required claims

After login, People Dash expects at least:

```text
sub
preferred_username
```

and normally:

```text
email
name
groups
```

The `sub` is the stable identity key stored by People Dash.

The username becomes the immutable People/Linux username.

## 6. If the groups claim is missing

Some customized Authentik configurations replace the default profile mapping.

Check your provider's configured scope/property mappings first.

If you need an explicit profile scope mapping, create a Scope Mapping under:

```text
Customization -> Property Mappings
```

For example:

```text
Name:
People Profile

Scope name:
profile
```

A minimal explicit expression can include:

```python
return {
    "name": request.user.name,
    "preferred_username": request.user.username,
    "nickname": request.user.username,
    "groups": [group.name for group in request.user.all_groups()],
}
```

Keep the normal email scope mapping enabled.

After adding or changing mappings, verify that the user's UserInfo response contains:

```json
{
  "sub": "...",
  "preferred_username": "henry",
  "email": "henry@example.com",
  "name": "Henry Meyer",
  "groups": [
    "people-users"
  ]
}
```

Do not copy a real access token into GitHub issues or documentation.

## 7. Application binding

Bind the People Dash application to the:

```text
people-users
```

group.

This provides an Authentik-side gate before People Dash even receives the callback.

Keep the explicit group check in Laravel as a second, application-side authorization boundary.

## 8. Admin authorization

People Dash sets `is_admin=true` when the OIDC groups claim contains:

```text
people-admins
```

Removing the user from the group takes effect on the next OIDC login.

The People database does not need an independent admin password.

## 9. Client secret

Store the client secret only in the dashboard `.env`:

```dotenv
OIDC_CLIENT_ID=...
OIDC_CLIENT_SECRET=...
```

Never commit it.

## 10. Useful Authentik references

- OAuth2/OIDC provider: https://docs.goauthentik.io/add-secure-apps/providers/oauth2/
- Create an OAuth2 provider: https://docs.goauthentik.io/add-secure-apps/providers/oauth2/create-oauth2-provider
- Groups: https://docs.goauthentik.io/users-sources/groups/
- User and group expressions: https://docs.goauthentik.io/users-sources/user/user_ref
