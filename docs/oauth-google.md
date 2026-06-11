# Google OAuth Setup

GoMyAdmin v0.6 can accept Google OAuth logins through the built-in provider helper plus an application-specific actor resolver.

## Required environment variables

```bash
export GOMYADMIN_PUBLIC_URL="https://admin.example.com"
export GOMYADMIN_SESSION_SECRET="replace-me"
export GOOGLE_CLIENT_ID="..."
export GOOGLE_CLIENT_SECRET="..."
```

## Server wiring

```go
srv, err := server.New(ctx, server.Config{
    DatabaseURL: os.Getenv("DATABASE_URL"),
    PublicURL:   os.Getenv("GOMYADMIN_PUBLIC_URL"),
    OAuthProviders: map[string]auth.OAuthProvider{
        "google": auth.GoogleOAuthProvider(
            os.Getenv("GOOGLE_CLIENT_ID"),
            os.Getenv("GOOGLE_CLIENT_SECRET"),
        ),
    },
    ResolveOAuthActor: func(ctx context.Context, provider string, identity auth.OAuthIdentity) (admin.Actor, bool, error) {
        if provider != "google" || identity.Email == "" {
            return admin.Actor{}, false, nil
        }

        // Look up an existing admin user by email.
        actor, err := store.ActiveUserByEmail(ctx, identity.Email)
        if err != nil {
            return admin.Actor{}, false, err
        }
        return actor, true, nil
    },
})
```

## Routes

- `GET /admin/api/auth/providers`
- `GET /admin/api/auth/oauth/google/start`
- `GET /admin/api/auth/oauth/google/callback`

## Notes

- OAuth sign-in should usually allow only users that already exist in your admin user table.
- The callback URL registered in Google should be:

```text
https://admin.example.com/admin/api/auth/oauth/google/callback
```

- API keys remain available for automation and server-to-server access even when browser logins use OAuth.
