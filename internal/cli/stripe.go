package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/geodro/lerd/internal/config"
	"github.com/geodro/lerd/internal/feedback"
	gitpkg "github.com/geodro/lerd/internal/git"
	"github.com/geodro/lerd/internal/podman"
	"github.com/geodro/lerd/internal/services"
	"github.com/spf13/cobra"
)

// NewStripeCmds returns Stripe-related subcommands.
func NewStripeCmds() []*cobra.Command {
	return []*cobra.Command{
		newStripeListenCmd(),
		newStripeConfigCmd(),
		newStripeLoginCmd(),
	}
}

func newStripeLoginCmd() *cobra.Command {
	var projectName string
	cmd := &cobra.Command{
		Use:   "stripe:login",
		Short: "Log in to Stripe with lerd's Stripe CLI and save the login under a project name",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Without a terminal the Stripe CLI switches to a two-step device
			// flow and exits before the login is saved.
			if !containerTTY() {
				return fmt.Errorf("stripe:login needs an interactive terminal")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			site, _ := config.FindSiteByPath(cwd)
			if !cmd.Flags().Changed("project-name") {
				projectName = stripeLoginProject(site)
			}
			if projectName == "" {
				return fmt.Errorf("--project-name must not be empty")
			}
			if err := config.ValidateStripeProjectName(projectName); err != nil {
				return err
			}
			dir := filepath.Dir(config.StripeCLIConfigPath())
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}
			c := podman.Cmd(podman.StripeLoginArgs(dir, projectName)...)
			c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := c.Run(); err != nil {
				return fmt.Errorf("stripe login: %w", err)
			}
			if site == nil {
				feedback.Note("use it for a site: lerd stripe:config --project-name " + projectName)
				return nil
			}
			if err := SetStripeProjectName(site.Path, projectName); err != nil {
				return err
			}
			feedback.Note(site.Name + " now signs in with " + projectName)
			RestartStripeIfActive(site)
			return nil
		},
	}
	cmd.Flags().StringVar(&projectName, "project-name", "", "Name to save the login under (defaults to the site's project, else its name, else \"default\")")
	return cmd
}

// SetStripeProjectName saves the Stripe CLI project a site signs in with to
// its .lerd.local.yaml, excluding that file from git first.
func SetStripeProjectName(sitePath, name string) error {
	if err := gitpkg.ExcludeLocalOverride(sitePath); err != nil {
		return fmt.Errorf("excluding %s from git: %w", config.LocalOverrideFile, err)
	}
	return config.SetProjectStripeProjectName(sitePath, name)
}

// stripeLoginProject names a login run without --project-name: the site's
// saved project, else the site's name, else Stripe's own "default".
func stripeLoginProject(site *config.Site) string {
	if site == nil {
		return "default"
	}
	if name := config.StripeProjectName(site.Path); name != "" {
		return name
	}
	return site.Name
}

func newStripeConfigCmd() *cobra.Command {
	var webhookPath string
	var secretEnvKey string
	var projectName string

	cmd := &cobra.Command{
		Use:   "stripe:config",
		Short: "Show or set the Stripe webhook path, secret env key and CLI project for the current site (without starting the listener)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			site, err := ensureSiteForCwd()
			if err != nil {
				return err
			}

			// No flags: report the current config instead of writing anything.
			if !cmd.Flags().Changed("path") && !cmd.Flags().Changed("secret-env-key") && !cmd.Flags().Changed("project-name") {
				key, _ := config.ResolveStripeSecret(cwd)
				if key == "" {
					key = "(none found; looked for " + strings.Join(config.StripeSecretEnvCandidates, ", ") + ")"
				}
				fmt.Printf("Stripe config for %s\n", site.Name)
				fmt.Printf("  Webhook path:   %s\n", config.StripeWebhookPath(cwd))
				fmt.Printf("  Secret env key: %s\n", key)
				if name := config.StripeProjectName(cwd); name != "" {
					fmt.Printf("  CLI project:    %s (used instead of the .env key)\n", name)
				}
				return nil
			}
			if cmd.Flags().Changed("project-name") {
				if err := SetStripeProjectName(cwd, projectName); err != nil {
					return err
				}
			}

			// Only persist the path when --path was actually passed; its default
			// value must not overwrite a previously-saved custom route.
			pathArg := ""
			if cmd.Flags().Changed("path") {
				pathArg = webhookPath
			}
			if err := config.SetProjectStripe(cwd, pathArg, secretEnvKey); err != nil {
				return err
			}
			fmt.Printf("Updated Stripe config for %s\n", site.Name)
			fmt.Printf("  Webhook path:   %s\n", config.StripeWebhookPath(cwd))
			// Re-forward a running listener to the new route; no-op otherwise.
			RestartStripeIfActive(site)
			return nil
		},
	}
	cmd.Flags().StringVar(&webhookPath, "path", config.DefaultStripeWebhookPath, "Webhook route path on your app")
	cmd.Flags().StringVar(&secretEnvKey, "secret-env-key", "", "Which .env key holds the Stripe secret")
	cmd.Flags().StringVar(&projectName, "project-name", "", "Sign in with this Stripe CLI login (stripe login --project-name) instead of the .env key (\"\" clears it)")
	return cmd
}

func newStripeListenCmd() *cobra.Command {
	var apiKey string
	var webhookPath string
	var secretEnvKey string
	var projectName string

	cmd := &cobra.Command{
		Use:   "stripe:listen",
		Short: "Start a Stripe webhook listener for the current site as a systemd service",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			// --path and --secret-env-key persist to .lerd.yaml so the UI
			// toggle and install restore reuse them. Persist before resolving
			// so a same-invocation override takes effect immediately. Only pass
			// the path when --path was actually given, otherwise its default
			// value would clobber a previously-saved custom route.
			if cmd.Flags().Changed("path") || cmd.Flags().Changed("secret-env-key") {
				pathArg := ""
				if cmd.Flags().Changed("path") {
					pathArg = webhookPath
				}
				if err := config.SetProjectStripe(cwd, pathArg, secretEnvKey); err != nil {
					return err
				}
			}

			if cmd.Flags().Changed("project-name") {
				if err := SetStripeProjectName(cwd, projectName); err != nil {
					return err
				}
			}

			// An explicit key wins, then a pinned CLI project, then the shell's
			// STRIPE_SECRET, then the .env secret.
			if apiKey == "" && config.StripeProjectName(cwd) == "" {
				apiKey = os.Getenv("STRIPE_SECRET")
			}
			auth := config.StripeAuth{APIKey: apiKey}
			if apiKey == "" {
				if auth, err = config.ResolveStripeAuth(cwd); err != nil {
					return fmt.Errorf("%w: pass --api-key or --project-name", err)
				}
			}

			base := siteURL(cwd)
			if base == "" {
				if _, err := ensureSiteForCwd(); err != nil {
					return err
				}
				base = siteURL(cwd)
			}
			if base == "" {
				return errNotLinked()
			}

			siteName, err := queueSiteName(cwd)
			if err != nil {
				return err
			}

			if err := stripeStartExplicit(siteName, auth, base+config.StripeWebhookPath(cwd)); err != nil {
				return err
			}
			if site, err := config.FindSite(siteName); err == nil && !site.Paused {
				_ = config.SetProjectWorkers(site.Path, CollectDeclaredWorkerNames(site))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Stripe API key (defaults to the secret in .env)")
	cmd.Flags().StringVar(&webhookPath, "path", config.DefaultStripeWebhookPath, "Webhook route path on your app (persisted to .lerd.yaml)")
	cmd.Flags().StringVar(&secretEnvKey, "secret-env-key", "", "Which .env key holds the Stripe secret (persisted to .lerd.yaml)")
	cmd.Flags().StringVar(&projectName, "project-name", "", "Sign in with this Stripe CLI login (stripe login --project-name) instead of the .env key (persisted to .lerd.yaml)")
	// stop hangs off the starter rather than sitting beside it at the root:
	// two root commands whose Use began with stripe:listen made cobra resolve
	// the documented `lerd stripe:listen` to an arbitrary one of the two.
	cmd.AddCommand(newStripeListenStopCmd())
	return cmd
}

func newStripeListenStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the Stripe webhook listener for the current site",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			siteName, err := queueSiteName(cwd)
			if err != nil {
				return err
			}
			if err := StripeStopForSite(siteName); err != nil {
				return err
			}
			if site, err := config.FindSite(siteName); err == nil && !site.Paused {
				_ = config.SetProjectWorkers(site.Path, CollectDeclaredWorkerNames(site))
			}
			return nil
		},
	}
}

// writeStripeUnit writes (and enables on first write) the lerd-stripe-<site>
// service unit without starting it. Shared by the CLI path (starts right
// after) and the install restore path (defers start to the worker phase).
func writeStripeUnit(siteName string, auth config.StripeAuth, forwardTo string) error {
	changed, err := writeStripeUnitFile(siteName, auth, forwardTo)
	if err != nil || !changed {
		return err
	}
	if err := podman.DaemonReloadFn(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := syncWorkerBootArming("lerd-stripe-" + siteName); err != nil {
		feedback.Warn("enable: %v", err)
	}
	return nil
}

// writeStripeUnitFile writes the listener unit without reloading systemd, for
// a caller that reloads once for a batch. Reports whether the file changed.
func writeStripeUnitFile(siteName string, auth config.StripeAuth, forwardTo string) (bool, error) {
	unitName := "lerd-stripe-" + siteName
	unit := podman.StripeListenUnit(siteName, auth, forwardTo)

	changed, err := services.Mgr.WriteServiceUnitIfChanged(unitName, unit)
	if err != nil {
		return false, fmt.Errorf("writing service unit: %w", err)
	}
	return changed, nil
}

func stripeStartExplicit(siteName string, auth config.StripeAuth, forwardTo string) error {
	if err := writeStripeUnit(siteName, auth, forwardTo); err != nil {
		return err
	}
	return launchStripeUnit(siteName, forwardTo)
}

// launchStripeUnit starts a listener whose unit is already written and loaded.
func launchStripeUnit(siteName, forwardTo string) error {
	unitName := "lerd-stripe-" + siteName
	// podman.StartUnit (not services.Mgr.Start) so AfterUnitChange fires
	// and the dashboard reflects the new state without a manual refresh.
	if err := podman.StartUnit(unitName); err != nil {
		return fmt.Errorf("starting stripe listener: %w", err)
	}

	feedback.Start("starting stripe listener").OK("")
	feedback.Note("forwarding to: " + forwardTo)
	feedback.Note("logs: " + unitLogHint(unitName))
	return nil
}

// StripeStartForSite starts a Stripe listener for the given site, reading the
// sign-in and webhook path from the project's .env and .lerd.yaml.
func StripeStartForSite(siteName, sitePath, siteBaseURL string) error {
	changed, start, err := prepareStripeStart(siteName, sitePath, siteBaseURL)
	if err != nil {
		return err
	}
	if changed {
		if err := podman.DaemonReloadFn(); err != nil {
			return fmt.Errorf("daemon-reload: %w", err)
		}
	}
	return start()
}

// prepareStripeStart writes a site's listener unit without reloading systemd,
// so a batch of workers can share one reload. Reports whether the unit
// changed and returns the start to run once the reload is done.
func prepareStripeStart(siteName, sitePath, siteBaseURL string) (bool, func() error, error) {
	auth, err := config.ResolveStripeAuth(sitePath)
	if err != nil {
		return false, nil, err
	}
	forwardTo := siteBaseURL + config.StripeWebhookPath(sitePath)
	changed, err := writeStripeUnitFile(siteName, auth, forwardTo)
	if err != nil {
		return false, nil, err
	}
	start := func() error {
		if changed {
			if err := syncWorkerBootArming("lerd-stripe-" + siteName); err != nil {
				feedback.Warn("enable: %v", err)
			}
		}
		if err := launchStripeUnit(siteName, forwardTo); err != nil {
			return err
		}
		recordProjectWorker(sitePath, "stripe")
		ClearIdleSuspendOnStart(siteName, sitePath, "stripe")
		return nil
	}
	return changed, start, nil
}

// StripeRestoreUnit writes the stripe listener unit without starting it.
// Used by install's restore path so workers launch in phase order — FPM and
// nginx come up first, then `startRestoredServices` starts every worker.
func StripeRestoreUnit(siteName, sitePath, siteBaseURL string) error {
	auth, err := config.ResolveStripeAuth(sitePath)
	if err != nil {
		return err
	}
	return writeStripeUnit(siteName, auth, siteBaseURL+config.StripeWebhookPath(sitePath))
}

// restoreStripeWorker rewrites the listener unit even when it is already
// enabled, so an upgrade moves existing listeners onto the pinned image. The
// write is a no-op when the unit is unchanged.
func restoreStripeWorker(s config.Site) {
	if base := siteURL(s.Path); base != "" {
		StripeRestoreUnit(s.Name, s.Path, base) //nolint:errcheck
	}
}

// StripeStopForSite stops and removes the Stripe listener for the named site.
func StripeStopForSite(siteName string) error {
	return stopWorkerUnit("lerd-stripe-"+siteName, "stripe listener", "")
}

// StripeConfigured reports whether the site's listener can sign in, through a
// .env secret or a saved Stripe CLI project.
func StripeConfigured(sitePath string) bool {
	return config.StripeConfigured(sitePath)
}
