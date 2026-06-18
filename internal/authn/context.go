package authn

import (
	"context"

	"github.com/godatei/datei/internal/users"
)

// GetEmailIdentity retrieves the session Identity from the request context.
func GetEmailIdentity(ctx context.Context) (EmailIdentity, error) {
	if id, ok := ctx.Value(emailContextKey{}).(EmailIdentity); ok {
		return id, nil
	}
	return EmailIdentity{}, ErrNoAuthentication
}

// RequireEmailIdentity panics if no Identity is present (use after Middleware).
func RequireEmailIdentity(ctx context.Context) EmailIdentity {
	id, err := GetEmailIdentity(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// GetCurrentUser retrieves the authenticated user from ctx.
func GetCurrentUser(ctx context.Context) (users.UserAccount, error) {
	if user, ok := ctx.Value(userContextKey{}).(users.UserAccount); ok {
		return user, nil
	}
	return users.UserAccount{}, ErrNoAuthentication
}

// RequireCurrentUser panics if no user is present (use after Middleware).
func RequireCurrentUser(ctx context.Context) users.UserAccount {
	user, err := GetCurrentUser(ctx)
	if err != nil {
		panic(err)
	}
	return user
}

// PopulateContext injects the EmailIdentity and the authenticated user into ctx.
func PopulateContext(ctx context.Context, identity EmailIdentity, user users.UserAccount) context.Context {
	ctx = context.WithValue(ctx, emailContextKey{}, identity)
	ctx = context.WithValue(ctx, userContextKey{}, user)
	return ctx
}
