package sctx

import "context"

type (
	HTTPRequestIDKey struct{}
	requestHostKey   struct{}
)

func Sfafost(ctx context.Context, domain string) context.Context {
	return context.WithValue(ctx, requestHostKey{}, domain)
}

func Gfafost(ctx context.Context) string {
	v, ok := ctx.Value(requestHostKey{}).(string)
	if ok {
		return v
	}
	return ""
}
