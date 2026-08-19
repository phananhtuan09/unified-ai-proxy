package config

// Config is the root configuration document.
type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Providers map[string]ProviderConfig `yaml:"providers"`
	Routing   RoutingConfig             `yaml:"routing"`
}

type ServerConfig struct {
	Host             string   `yaml:"host,omitempty"`
	Port             int      `yaml:"port,omitempty"`
	APIKeys          []string `yaml:"api_keys,omitempty"`
	DefaultMaxTokens int      `yaml:"default_max_tokens,omitempty"`
}
type ProviderConfig struct {
	Enabled  bool            `yaml:"enabled,omitempty"`
	Auth     AuthConfig      `yaml:"auth,omitempty"`
	API      APIConfig       `yaml:"api,omitempty"`
	Models   []ModelConfig   `yaml:"models,omitempty"`
	Accounts []AccountConfig `yaml:"accounts,omitempty"`
}
type AuthConfig struct {
	Method           string   `yaml:"method,omitempty"`
	ClientID         string   `yaml:"client_id,omitempty"`
	AuthorizationURL string   `yaml:"authorization_url,omitempty"`
	TokenURL         string   `yaml:"token_url,omitempty"`
	Scopes           []string `yaml:"scopes,omitempty"`
	RedirectHost     string   `yaml:"redirect_host,omitempty"`
	RedirectPort     int      `yaml:"redirect_port,omitempty"`
	RedirectPath     string   `yaml:"redirect_path,omitempty"`
	ExchangeFormat   string   `yaml:"exchange_format,omitempty"`
	PKCE             bool     `yaml:"pkce,omitempty"`
}
type APIConfig struct {
	BaseURL string `yaml:"base_url,omitempty"`
}
type ModelConfig struct {
	ID       string `yaml:"id,omitempty"`
	Upstream string `yaml:"upstream,omitempty"`
}
type AccountConfig struct {
	Name      string `yaml:"name,omitempty"`
	TokenFile string `yaml:"token_file,omitempty"`
	APIKey    string `yaml:"api_key,omitempty"`
}
type RoutingConfig struct {
	Strategy string         `yaml:"strategy,omitempty"`
	Failover FailoverConfig `yaml:"failover,omitempty"`
}
type FailoverConfig struct {
	Enabled           bool     `yaml:"enabled,omitempty"`
	MaxRetries        int      `yaml:"max_retries,omitempty"`
	UnhealthyCooldown Duration `yaml:"unhealthy_cooldown,omitempty"`
	RequestTimeout    Duration `yaml:"request_timeout,omitempty"`
}
