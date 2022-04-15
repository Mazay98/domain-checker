package environment

import "context"

type key string

const (
	keyEnv       key = "environment"
	keyVersion   key = "version"
	keyBuildTime key = "build_time"
)

// CtxWithEnv puts passed env into the context.
func CtxWithEnv(ctx context.Context, env Env) context.Context {
	return context.WithValue(ctx, keyEnv, env)
}

// EnvFromCtx returns environment, if any, previously
// put in the context with CtxWithEnv.
func EnvFromCtx(ctx context.Context) Env {
	v := ctx.Value(keyEnv)
	if v == nil {
		return ""
	}

	env, ok := v.(Env)
	if !ok {
		return ""
	}

	return env
}

// CtxWithVersion puts passed version into the context.
func CtxWithVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, keyVersion, version)
}

// VersionFromCtx returns version, if any, previously
// put in the context with CtxWithVersion.
func VersionFromCtx(ctx context.Context) string {
	v := ctx.Value(keyVersion)
	if v == nil {
		return ""
	}

	env, ok := v.(string)
	if !ok {
		return ""
	}

	return env
}

// CtxWithBuildTime puts passed buildTime string into the context.
func CtxWithBuildTime(ctx context.Context, ts string) context.Context {
	return context.WithValue(ctx, keyBuildTime, ts)
}

// BuildTimeFromCtx returns build time name, if any, previously
// put in the context with CtxWithBuildTime.
func BuildTimeFromCtx(ctx context.Context) string {
	v := ctx.Value(keyBuildTime)
	if v == nil {
		return ""
	}

	ts, ok := v.(string)
	if !ok {
		return ""
	}

	return ts
}
