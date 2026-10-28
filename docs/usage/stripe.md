# Stripe

The webhook listener works for any project type, not just Laravel. It runs the Stripe CLI in a container and forwards events to your app, so a NestJS or plain Node site reached through the host proxy is supported the same way a Laravel Cashier site is. The secret is auto-detected from common env keys and the forward route is configurable per project.

## stripe-mock

A local Stripe API mock for feature tests that exercise Cashier without hitting the real API and without needing a Stripe account.

```yaml
# ~/.config/lerd/services/stripe-mock.yaml
name: stripe-mock
image: docker.io/stripemock/stripe-mock:latest
description: "Local Stripe API mock for Cashier testing"
ports:
  - 12111:12111
```

```bash
lerd service add ~/.config/lerd/services/stripe-mock.yaml
lerd service start stripe-mock
```

Point the Stripe PHP SDK at the mock in your `AppServiceProvider` or test bootstrap:

```php
\Stripe\Stripe::$apiBase = 'http://lerd-stripe-mock:12111';
```

---

## stripe:listen

Forwards live or test webhook events from Stripe to your local app. Lerd runs the Stripe CLI in a container as a background **systemd user service**, so it persists across terminal sessions and restarts automatically on failure.

### Starting the listener

```bash
cd ~/Lerd/myapp
lerd stripe:listen                         # forwards to https://myapp.test/stripe/webhook
lerd stripe:listen --path /webhooks/stripe # custom webhook path (persisted to .lerd.yaml)
lerd stripe:listen --api-key sk_test_...   # override key
lerd stripe:listen --secret-env-key STRIPE_SECRET_KEY # pin the .env key (persisted)
lerd stripe:listen --project-name acme     # sign in with a saved Stripe CLI login (persisted)
```

The target URL is auto-detected from the registered site in the current directory. Run `lerd link` first if the project is not yet registered.

The listener forwards every snapshot event, the classic webhook events Stripe sends to a v1 endpoint (`--all-snapshot`), and runs a pinned Stripe CLI release, since the CLI refuses to listen without an event selection from 1.51 on.

### Secret detection

Lerd resolves the Stripe secret from the project's `.env` automatically, probing these keys in order until one is set:

1. `STRIPE_SECRET`: Laravel / Cashier
2. `STRIPE_SECRET_KEY`: the common Stripe Node / NestJS convention
3. `STRIPE_API_KEY`: the generic SDK name

No flags are required if any of these is present. To force a specific key (for example a project that stores it under a non-standard name), pin it with `--secret-env-key` or the `stripe.secret_env_key` field in `.lerd.yaml`.

### Signing in with a saved Stripe CLI login

If you would rather not keep the secret in `.env`, log in from the project folder:

```bash
cd ~/Lerd/myapp
lerd stripe:login
```

`lerd stripe:login` runs lerd's own Stripe CLI image, so nothing needs installing on the host: it prints a link and a pairing code, and once you confirm in the browser the login is saved under that name. A Stripe CLI installed on the host works the same way, `stripe login --project-name acme` writes to the same file. Run inside a linked site, the login is saved under the site's name (or the project it already uses) and the site switches to it, restarting a running listener. Pass `--project-name` to pick the name yourself; outside a site the login is saved as `default` and `lerd stripe:config --project-name` points a site at it later.

The listener then mounts your Stripe CLI config folder (`~/.config/stripe`, or under `$XDG_CONFIG_HOME`) into its container and runs with `--project-name acme` instead of `--api-key`. The folder is writable because the Stripe CLI refreshes the login's token in it. A project name takes precedence over any `.env` secret; clear it with `--project-name ""` to go back to the `.env` key. Starting fails with a hint to run `stripe login` if the config file is missing.

The project name is your own login, not the team's, so lerd writes it to the untracked `.lerd.local.yaml` (excluded from git) rather than `.lerd.yaml`. The `stripe:` section merges field by field: `project_name` comes from your local file while `path` and `secret_env_key` keep coming from the committed one, and a field set in `.lerd.local.yaml` wins over the same field in `.lerd.yaml`.

### Configuring the route without starting

`lerd stripe:config` shows or sets the webhook path, secret env key and CLI project without touching the listener:

```bash
lerd stripe:config                              # show current route and resolved secret key
lerd stripe:config --path /webhooks/stripe      # set the forward route
lerd stripe:config --secret-env-key STRIPE_API_KEY
lerd stripe:config --project-name acme          # sign in with a saved Stripe CLI login
lerd stripe:config --project-name ""            # back to the .env secret
```

Both `stripe:config` and `stripe:listen` write the same optional `stripe:` block to the project's `.lerd.yaml` (`project_name` goes to `.lerd.local.yaml`, see above), so the route survives reinstalls and is shared by the CLI, the MCP server, and the web UI:

```yaml
# .lerd.yaml
stripe:
  path: /webhooks/stripe      # optional, defaults to /stripe/webhook
  secret_env_key: STRIPE_API_KEY  # optional, defaults to auto-detection
  project_name: acme          # optional, kept in .lerd.local.yaml: a `stripe login` login used instead of .env
```

All fields are optional. Project names may contain only letters, digits, `.`, `-` and `_`. Routes are normalised to a leading slash and rejected if they contain whitespace, so the generated systemd unit stays well formed. Changing the route while the listener is running re-forwards it to the new path automatically.

### Stopping the listener

```bash
lerd stripe:listen stop
```

Starting and stopping the listener updates the `workers` list in `.lerd.yaml` (when the file exists), so the stripe listener is restored automatically after a reinstall when you run `lerd start`.

### HTTPS

If you run `lerd secure` or `lerd unsecure` while the listener is active, Lerd automatically restarts it so `--forward-to` stays in sync with the site's current scheme. No manual restart needed.

### Logs

```bash
journalctl --user -u lerd-stripe-myapp -f
```

Logs are also available live in the **web UI**, see [Web UI](#web-ui) below.

### Options

These flags apply to `stripe:listen`; `stripe:config` accepts `--path`, `--secret-env-key` and `--project-name` only.

| Flag | Default | Description |
|---|---|---|
| `--api-key` | resolved from `.env` | Stripe secret key (`sk_test_…` or `sk_live_…`) |
| `--path` | `/stripe/webhook` | Webhook route path on your app (persisted to `.lerd.yaml`) |
| `--secret-env-key` | auto-detected | Which `.env` key holds the Stripe secret (persisted to `.lerd.yaml`) |
| `--project-name` | none | Sign in with this `stripe login --project-name` login instead of the `.env` key (persisted to `.lerd.yaml`) |

### Web UI

When a Stripe secret is detected in a site's `.env` (under any of the recognised keys), or a Stripe CLI project is set, a **Stripe** toggle appears in the site detail panel alongside HTTPS and Queue. Toggling it starts or stops the listener. A gear next to the toggle opens a small modal for setting the webhook route, the `.env` key that holds the secret and the Stripe CLI project, mirroring the Horizon control. The key field is prefilled with the detected key; saving a key that has no value in `.env` is refused unless a CLI project is set, since that would hide the control. While running:

- A violet dot appears next to the site in the sidebar.
- A **Stripe** log tab opens automatically beside PHP-FPM and Queue.
- The listener also appears in the **Services** tab with a `stripe` badge.
