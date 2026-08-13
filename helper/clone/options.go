package clone

type Option func(*Options)

type Options struct {
	IgnoreFields     map[string]struct{}
	IgnorePrimaryKey bool
	IgnoreCreatedAt  bool
}

func defaultOptions() *Options {
	return &Options{
		IgnoreFields:     map[string]struct{}{},
		IgnorePrimaryKey: true,
		IgnoreCreatedAt:  true,
	}
}

func WithIgnoreFields(fields ...string) Option {
	return func(o *Options) {
		for _, f := range fields {
			o.IgnoreFields[f] = struct{}{}
		}
	}
}

func WithPrimaryKey(ignore bool) Option {
	return func(o *Options) {
		o.IgnorePrimaryKey = ignore
	}
}

func WithCreatedAt(ignore bool) Option {
	return func(o *Options) {
		o.IgnoreCreatedAt = ignore
	}
}
