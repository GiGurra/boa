package boa

type globalConfig struct {
	defaultOptional bool
}

var cfg globalConfig

// Option is a functional option for configuring global boa behavior.
type Option func(*globalConfig)

// Init configures global boa behavior. Call this before creating any commands.
// Without Init, the default behavior is that plain Go type fields are required.
func Init(opts ...Option) {
	for _, opt := range opts {
		opt(&cfg)
	}
}

// WithDefaultOptional makes plain Go type fields (string, int, etc.) optional by default
// instead of required. Explicit struct tags (required, req, optional, opt) and
// Required[T]/Optional[T] wrappers still override this setting.
func WithDefaultOptional() Option {
	return func(c *globalConfig) {
		c.defaultOptional = true
	}
}
