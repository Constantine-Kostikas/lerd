---
title: XAMPP alternative for Linux
description: 'Move from XAMPP to Lerd on Linux or macOS: a .test domain and trusted HTTPS per project, a PHP version per project instead of one per install, and your htdocs projects and MariaDB databases carried over.'
head:
  - - meta
    - name: keywords
      content: xampp alternative linux, xampp alternative, xampp linux, xampp ubuntu alternative, lampp alternative, xampp to nginx, local php development linux
---

# XAMPP alternative for Linux

XAMPP bundles Apache, MariaDB and one PHP version into `/opt/lampp`, started with `sudo` and serving everything under `htdocs` at `localhost`. Lerd covers the same ground on Linux and macOS without root: each project gets its own `https://<name>.test` domain, its own PHP version, and shared MySQL, MariaDB or PostgreSQL, all running as rootless containers under your user. It is free and open source.

The one real difference is the web server. XAMPP serves through Apache, which reads `.htaccess` files, and Lerd serves through nginx, which does not. Most projects need no changes, and the [MAMP guide](/getting-started/mamp#do-you-need-to-translate-your-htaccess) shows how to carry over the rules of the ones that do.

## Every XAMPP feature, and the Lerd equivalent

| What you used in XAMPP | The same thing in Lerd |
|---|---|
| `localhost/myapp` | `https://myapp.test`, one domain per project with no hosts file edits |
| Projects inside `/opt/lampp/htdocs` | Projects anywhere in your home folder, `lerd park ~/code` serves a whole folder |
| `sudo /opt/lampp/lampp start` | `lerd start`, as your own user, or start at login with `lerd autostart enable` |
| One PHP version per XAMPP install | Every PHP version from 7.4 to 8.5 side by side, picked per project with `lerd isolate 8.4` or read from `composer.json` |
| MariaDB with `root` and no password | `lerd service start mysql`, or MariaDB after `lerd service preset mariadb`, wired into `.env` by `lerd env` |
| phpMyAdmin at `localhost/phpmyadmin` | `lerd service preset phpmyadmin`, then `lerd service start phpmyadmin`, or any database client |
| Self-signed SSL in `httpd-ssl.conf` | `lerd secure`, a certificate your browsers trust, no warning page |
| Editing `php.ini` in `/opt/lampp/etc` | `lerd php:ini`, per PHP version or shared, see [php.ini overrides](/usage/php-ini) |
| `.htaccess` rules | An [nginx override](/usage/nginx-overrides) per site |
| XAMPP Control Panel | The [web dashboard](/features/web-ui), a [terminal dashboard](/features/tui) and a [system tray](/features/system-tray) |

## Moving a project from XAMPP

**1. Export your databases.** In XAMPP's phpMyAdmin, open each database and use the **Export** tab, or run XAMPP's own dump tool:

```bash
/opt/lampp/bin/mysqldump -u root myapp > myapp.sql
```

**2. Stop XAMPP.** It holds ports 80, 443 and 3306, which Lerd needs.

```bash
sudo /opt/lampp/lampp stop
```

**3. Install Lerd and start it.**

```bash
curl -fsSL https://lerd.sh/install.sh | bash
lerd start
```

**4. Move the projects and serve them.** Copy them out of `htdocs` into a folder you own, then park it so every project becomes a site:

```bash
cp -r /opt/lampp/htdocs/myapp ~/code/
cd ~/code
lerd park
```

Each project is served at `https://<folder>.test`. Lerd detects the framework and its document root, and a plain PHP project with an `index.php` at the top works as it is. A project whose entry point lives elsewhere can set [`public_dir`](/configuration#per-project-config-lerd-yaml) in `.lerd.yaml`.

**5. Point the app at Lerd's database and import.** Start the database, let `lerd env` replace XAMPP's `root` with no password in `.env` (or `wp-config.php` on WordPress), and restore the dump:

```bash
lerd service start mysql
cd ~/code/myapp
lerd env
lerd db:import myapp.sql
```

**6. Update URLs.** Links that pointed at `http://localhost/myapp` now live at `https://myapp.test`. For Laravel and most frameworks that is `APP_URL` in `.env`. A WordPress site keeps its address in the database, so run a search and replace with WP-CLI:

```bash
wp search-replace 'http://localhost/myapp' 'https://myapp.test'
```

## What is genuinely different

- **No root after install.** XAMPP runs Apache and MariaDB as root from `/opt/lampp`. Lerd runs everything as rootless Podman containers and needs `sudo` only during installation, to point the system resolver at `.test`.
- **A domain per project, not a path.** Apps that hardcoded `/myapp/` as a base path need it removed, since each project is served from the root of its own domain.
- **Nginx, not Apache.** `.htaccess` is not read. Framework front controllers work without any rules, and custom rewrites move into an [nginx override](/usage/nginx-overrides).
- **Per-project PHP.** Two projects can run PHP 7.4 and 8.5 at the same time, which XAMPP needs two separate installs for.

## Removing XAMPP

Once your projects run on Lerd, XAMPP's own uninstaller removes it:

```bash
sudo /opt/lampp/uninstall
```

## Next steps

- [Quick start](/getting-started/quick-start), a project served in two commands
- [PHP versions per project](/usage/php-versions), switching and pinning versions
- [Database](/usage/database), imports, exports and switching engines
- [Laragon for Linux](/getting-started/laragon-linux) and [Moving from MAMP](/getting-started/mamp), the guides for the other all-in-one stacks
