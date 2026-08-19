package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/accounts"
	"github.com/tuanp-github/unified-ai-proxy/internal/backup"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
	"github.com/tuanp-github/unified-ai-proxy/internal/provider"
	"github.com/tuanp-github/unified-ai-proxy/internal/proxy"
	"github.com/tuanp-github/unified-ai-proxy/internal/server"
	"github.com/tuanp-github/unified-ai-proxy/internal/tokenstore"
	"github.com/tuanp-github/unified-ai-proxy/internal/tui"
	"github.com/tuanp-github/unified-ai-proxy/internal/version"
)

// Run dispatches the CLI command and returns a process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		usage()
		return 1
	}
	cmd, rest := args[0], args[1:]

	var err error
	switch cmd {
	case "start":
		err = cmdStart(rest)
	case "tui":
		err = cmdTUI(rest)
	case "auth":
		err = cmdAuth(rest)
	case "accounts":
		err = cmdAccounts(rest)
	case "export":
		err = cmdExport(rest)
	case "import":
		err = cmdImport(rest)
	case "version", "--version", "-v":
		fmt.Println(version.Version)
		return 0
	case "help", "--help", "-h":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		return 1
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, `unified-ai-proxy - local AI provider proxy

Usage:
  unified-ai-proxy start [--config path.yaml]
  unified-ai-proxy tui [--config path.yaml]
  unified-ai-proxy auth <provider> --account <name> [--config path.yaml]
  unified-ai-proxy accounts [--config path.yaml]
  unified-ai-proxy export --output <file> --password <pass> [--config path.yaml]
  unified-ai-proxy import --input <file> --password <pass> [--config path.yaml]
  unified-ai-proxy version
  unified-ai-proxy help`)
}

// parseFlags extracts --key value (and --key=value) flags from args,
// leaving the positional arguments intact. Flags may appear anywhere.
func parseFlags(args []string) (flags map[string]string, positional []string) {
	flags = map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			positional = append(positional, a)
			continue
		}
		name := strings.TrimPrefix(a, "--")
		val := ""
		if idx := strings.Index(name, "="); idx >= 0 {
			val = name[idx+1:]
			name = name[:idx]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			val = args[i+1]
			i++
		}
		flags[name] = val
	}
	return flags, positional
}

func cmdStart(args []string) error {
	flags, _ := parseFlags(args)
	configPath := orDefault(flags["config"], config.DefaultPath())

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	mgr := accounts.New(cfg.Routing.Failover.UnhealthyCooldown.Duration())
	svc, err := proxy.New(cfg, mgr, provider.Build)
	if err != nil {
		return err
	}
	srv := server.New(cfg, svc, nil)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("unified-ai-proxy %s listening on http://%s\n", version.Version, srv.Addr())
	return srv.Run(ctx)
}

func cmdTUI(args []string) error {
	flags, _ := parseFlags(args)
	configPath := orDefault(flags["config"], config.DefaultPath())
	return tui.Run(configPath)
}

func cmdAuth(args []string) error {
	flags, positional := parseFlags(args)
	configPath := orDefault(flags["config"], config.DefaultPath())

	if len(positional) < 1 {
		return fmt.Errorf("auth requires a provider name")
	}
	providerName := normalizeProviderName(positional[0])
	accountName := flags["account"]
	if accountName == "" {
		return fmt.Errorf("--account is required")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	pc, ok := cfg.Providers[providerName]
	if !ok {
		return fmt.Errorf("provider %q not found in config", providerName)
	}
	if !pc.Enabled {
		return fmt.Errorf("provider %q is disabled", providerName)
	}

	var target config.AccountConfig
	found := false
	for _, a := range pc.Accounts {
		if a.Name == accountName {
			target = a
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("account %q not found for provider %q", accountName, providerName)
	}

	tokenPath := config.ExpandPath(target.TokenFile)
	var ts *model.TokenSet
	switch pc.Auth.Method {
	case "oauth":
		ts, err = provider.RunOAuthLogin(context.Background(), providerName, target.Name, tokenPath, pc)
	case "browser_key":
		ts, err = provider.RunBrowserKeyLogin(context.Background(), providerName, target.Name, tokenPath, pc)
	default:
		return fmt.Errorf("provider %q auth method %q does not support browser login", providerName, pc.Auth.Method)
	}
	if err != nil {
		return err
	}
	expiry := ts.ExpiresAt.Format(time.RFC3339)
	if ts.ExpiresAt.IsZero() {
		expiry = "never"
	}
	fmt.Printf("Stored token for %s/%s (expires %s)\n", providerName, target.Name, expiry)
	return nil
}

func cmdAccounts(args []string) error {
	flags, _ := parseFlags(args)
	configPath := orDefault(flags["config"], config.DefaultPath())

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	mgr := accounts.New(cfg.Routing.Failover.UnhealthyCooldown.Duration())
	for name, p := range cfg.EnabledProviders() {
		mgr.Register(name, p.Accounts)
	}

	fmt.Printf("%-14s %-14s %-34s %s\n", "provider", "account", "status", "expires")
	for _, s := range accounts.Summarize(mgr) {
		fmt.Printf("%-14s %-14s %-34s %s\n", s.Provider, s.Account, s.Status, s.Expiry)
	}
	return nil
}

func cmdExport(args []string) error {
	flags, _ := parseFlags(args)
	configPath := orDefault(flags["config"], config.DefaultPath())

	output := flags["output"]
	if output == "" {
		return fmt.Errorf("--output is required")
	}
	password := flags["password"]
	if password == "" {
		return fmt.Errorf("--password is required")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if err := backup.Export(cfg, configPath, password, output); err != nil {
		return err
	}
	fmt.Printf("Exported encrypted backup to %s\n", output)
	return nil
}

func cmdImport(args []string) error {
	flags, _ := parseFlags(args)
	configPath := orDefault(flags["config"], config.DefaultPath())

	input := flags["input"]
	if input == "" {
		return fmt.Errorf("--input is required")
	}
	password := flags["password"]
	if password == "" {
		return fmt.Errorf("--password is required")
	}

	payload, err := backup.Import(input, password)
	if err != nil {
		return err
	}
	if err := backup.Restore(payload, configPath); err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("imported config failed validation: %w", err)
	}

	var reauth []string
	for name, p := range cfg.EnabledProviders() {
		for _, a := range p.Accounts {
			if a.APIKey != "" {
				continue
			}
			path := config.ExpandPath(a.TokenFile)
			ts, err := tokenstore.Load(path)
			if err != nil {
				return fmt.Errorf("imported token file %s is not parseable: %w", path, err)
			}
			if ts == nil {
				reauth = append(reauth, name+"/"+a.Name)
				continue
			}
			if ts.NeedsRefresh(time.Now()) {
				acc := model.Account{Provider: name, Name: a.Name, TokenFile: path}
				prov, err := provider.Build(name, p, cfg.Routing.Failover.RequestTimeout.Duration())
				if err != nil {
					reauth = append(reauth, name+"/"+a.Name)
					continue
				}
				if _, err := prov.RefreshToken(context.Background(), acc); err != nil {
					reauth = append(reauth, name+"/"+a.Name)
				}
			}
		}
	}

	fmt.Printf("Imported config to %s\n", configPath)
	if len(reauth) > 0 {
		fmt.Printf("Accounts requiring reauthentication: %s\n", strings.Join(reauth, ", "))
	} else {
		fmt.Println("All accounts are ready.")
	}
	return nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func normalizeProviderName(name string) string {
	if name == "openai-codex" {
		return "openai_codex"
	}
	return name
}
