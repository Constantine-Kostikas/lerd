package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/geodro/lerd/internal/envfile"
)

// DefaultStripeWebhookPath is the route a Stripe listener forwards events to
// when the project does not configure one. Matches Laravel Cashier's default.
const DefaultStripeWebhookPath = "/stripe/webhook"

// StripeSecretEnvCandidates are the env var names that commonly hold a Stripe
// secret key, in priority order: Laravel/Cashier, the Stripe Node/Nest
// convention, then the generic SDK name. Probed when a project pins no explicit
// secret_env_key, so the listener detects a secret regardless of framework.
var StripeSecretEnvCandidates = []string{"STRIPE_SECRET", "STRIPE_SECRET_KEY", "STRIPE_API_KEY"}

// StripeConfig is the optional per-project Stripe listener config in .lerd.yaml.
// Both fields are optional: empty values fall back to auto-detection and the
// default webhook path, so Laravel projects need no config at all.
type StripeConfig struct {
	// Path is the webhook route the listener forwards events to, e.g.
	// "/webhooks/stripe". Empty uses DefaultStripeWebhookPath.
	Path string `yaml:"path,omitempty"`
	// SecretEnvKey pins which .env key holds the Stripe secret. Empty probes
	// StripeSecretEnvCandidates in order.
	SecretEnvKey string `yaml:"secret_env_key,omitempty"`
	// ProjectName signs the listener in with a login the host's Stripe CLI
	// saved via `stripe login --project-name`, instead of a key from .env.
	// Lerd writes it to .lerd.local.yaml, since the login is one developer's.
	ProjectName string `yaml:"project_name,omitempty"`
}

// StripeAuth is how a listener signs in: an API key, or a project from the
// host's Stripe CLI config folder at ConfigDir.
type StripeAuth struct {
	APIKey      string
	ProjectName string
	ConfigDir   string
}

// stripeProjectNameRe keeps a project name safe to interpolate into the
// listener unit's ExecStart line.
var stripeProjectNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// ValidateStripeProjectName rejects anything but letters, digits, dot, dash and
// underscore. An empty name is valid and means "use the .env key".
func ValidateStripeProjectName(name string) error {
	if name == "" || stripeProjectNameRe.MatchString(name) {
		return nil
	}
	return fmt.Errorf("invalid Stripe project name %q: use letters, digits, '.', '-' or '_'", name)
}

// StripeProjectName returns the project's saved Stripe CLI project, or "" when
// none is set or the stored value is unsafe.
func StripeProjectName(sitePath string) string {
	proj, err := LoadProjectConfig(sitePath)
	if err != nil || proj.Stripe == nil || ValidateStripeProjectName(proj.Stripe.ProjectName) != nil {
		return ""
	}
	return proj.Stripe.ProjectName
}

// StripeCLIConfigPath is where the host's Stripe CLI keeps its logins, the
// same lookup the CLI itself does.
func StripeCLIConfigPath() string {
	return filepath.Join(xdgConfigHome(), "stripe", "config.toml")
}

// ResolveStripeAuth picks how a site's listener signs in. A pinned
// project_name wins over the .env secret, since choosing one is how a project
// says it keeps no key in .env.
func ResolveStripeAuth(sitePath string) (StripeAuth, error) {
	if name := StripeProjectName(sitePath); name != "" {
		cfg := StripeCLIConfigPath()
		if _, err := os.Stat(cfg); err != nil {
			return StripeAuth{}, fmt.Errorf("no Stripe CLI config at %s: run `stripe login --project-name %s` first", cfg, name)
		}
		// The path lands unquoted in the listener unit's ExecStart.
		if strings.ContainsAny(cfg, " \t\r\n") {
			return StripeAuth{}, fmt.Errorf("Stripe CLI config path %q contains whitespace", cfg)
		}
		return StripeAuth{ProjectName: name, ConfigDir: filepath.Dir(cfg)}, nil
	}
	if _, secret := ResolveStripeSecret(sitePath); secret != "" {
		return StripeAuth{APIKey: secret}, nil
	}
	return StripeAuth{}, fmt.Errorf("no Stripe secret found in %s/.env (looked for %s) and no Stripe CLI project set",
		sitePath, strings.Join(StripeSecretEnvCandidates, ", "))
}

// StripeConfigured reports whether a site's listener has a way to sign in.
func StripeConfigured(sitePath string) bool {
	return StripeProjectName(sitePath) != "" || StripeSecretSet(sitePath)
}

// ResolveStripeSecret returns the env key name and value of a project's Stripe
// secret. An explicit secret_env_key in .lerd.yaml wins; otherwise the common
// candidate keys are probed in .env. An empty value means none is set.
func ResolveStripeSecret(sitePath string) (envKey, value string) {
	envPath := filepath.Join(sitePath, ".env")
	if proj, err := LoadProjectConfig(sitePath); err == nil && proj.Stripe != nil && proj.Stripe.SecretEnvKey != "" {
		return proj.Stripe.SecretEnvKey, envfile.ReadKey(envPath, proj.Stripe.SecretEnvKey)
	}
	for _, k := range StripeSecretEnvCandidates {
		if v := envfile.ReadKey(envPath, k); v != "" {
			return k, v
		}
	}
	return "", ""
}

// StripeWebhookPath returns the configured webhook route for a project, or
// DefaultStripeWebhookPath when none is set. The stored value is normalised to
// a leading slash so a hand-edited ".lerd.yaml" can't yield a slash-less URL
// like "https://site.teststripe/webhook".
func StripeWebhookPath(sitePath string) string {
	if proj, err := LoadProjectConfig(sitePath); err == nil && proj.Stripe != nil && proj.Stripe.Path != "" {
		// Validate on read, not just on write: a hand-edited or hostile
		// .lerd.yaml can otherwise carry whitespace/newlines straight into the
		// listener unit's ExecStart line, so reject those and fall back to the
		// default rather than trusting the stored value.
		if validated, err := ValidateStripeWebhookPath(proj.Stripe.Path); err == nil {
			return validated
		}
	}
	return DefaultStripeWebhookPath
}

func ensureLeadingSlash(p string) string {
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

// ValidateStripeWebhookPath normalises a webhook route to a leading slash and
// rejects whitespace, which would otherwise break the systemd ExecStart line
// the path is interpolated into (a space adds an argument, a newline ends the
// directive). An empty path is returned unchanged for callers that leave it
// unset.
func ValidateStripeWebhookPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if strings.ContainsAny(path, " \t\r\n\x00") {
		return "", fmt.Errorf("invalid Stripe webhook path %q: must not contain whitespace or NUL", path)
	}
	return ensureLeadingSlash(path), nil
}

// StripeSecretSet reports whether a project has a Stripe secret in .env under
// its configured or any recognised env key.
func StripeSecretSet(sitePath string) bool {
	_, v := ResolveStripeSecret(sitePath)
	return v != ""
}

// RequireStripeSecretInEnv refuses a secret_env_key with no value in .env:
// pinning it would hide the UI's Stripe control, leaving no way back from the
// UI. An empty key (keep auto-detection) always passes.
func RequireStripeSecretInEnv(sitePath, key string) error {
	if key == "" || envfile.ReadKey(filepath.Join(sitePath, ".env"), key) != "" {
		return nil
	}
	return fmt.Errorf("%s has no value in .env", key)
}
