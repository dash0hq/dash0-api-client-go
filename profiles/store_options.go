package profiles

// StoreOption configures a [Store].
type StoreOption func(*storeConfig)

type storeConfig struct {
	configDir string
}

// WithConfigDir overrides the default configuration directory (~/.dash0/).
// This is useful for testing or for applications that store profiles in a
// non-standard location.
func WithConfigDir(dir string) StoreOption {
	return func(c *storeConfig) {
		c.configDir = dir
	}
}
