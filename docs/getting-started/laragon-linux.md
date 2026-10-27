---
title: Laragon for Linux
description: 'Laragon is Windows-only and has no Linux build. Lerd is the closest Laragon alternative for Linux: automatic .test domains, one-click HTTPS, per-project PHP versions, and bundled MySQL, PostgreSQL and Redis, free and open source.'
head:
  - - meta
    - name: keywords
      content: laragon for linux, laragon linux, laragon alternative linux, laragon ubuntu, laragon linux equivalent, laragon replacement, local php development linux
  - - script
    - type: application/ld+json
    - |
      {
        "@context": "https://schema.org",
        "@type": "FAQPage",
        "mainEntity": [
          {
            "@type": "Question",
            "name": "Is there a Laragon version for Linux?",
            "acceptedAnswer": {
              "@type": "Answer",
              "text": "No. Laragon is a Windows-only application and its maintainers have said there are no plans for a Linux build. To get the Laragon workflow on Linux you need a different tool. Lerd is the closest equivalent: automatic .test domains, one-click HTTPS, per-project PHP versions and bundled databases, on Linux and macOS."
            }
          },
          {
            "@type": "Question",
            "name": "What is the best Laragon alternative for Linux?",
            "acceptedAnswer": {
              "@type": "Answer",
              "text": "Lerd covers the same day-to-day surface as Laragon: pretty .test URLs with no hosts file edits, automatic virtual hosts, one-command TLS, scaffolding new Laravel, Symfony or WordPress projects, switching PHP versions per project, and shared MySQL, PostgreSQL, Redis, Meilisearch, S3 and Mailpit services. It is MIT licensed and free for commercial use. DDEV and Lando are alternatives if you prefer a per-project Docker stack."
            }
          },
          {
            "@type": "Question",
            "name": "Can I run Laragon on Ubuntu with Wine?",
            "acceptedAnswer": {
              "@type": "Answer",
              "text": "Laragon bundles Windows builds of Apache, Nginx, PHP, MySQL and its own tray application, so running it under Wine is not supported and not practical. Installing a native Linux tool is the better route."
            }
          },
          {
            "@type": "Question",
            "name": "Does Lerd need Docker or sudo?",
            "acceptedAnswer": {
              "@type": "Answer",
              "text": "No. Lerd runs Nginx, PHP-FPM and services as rootless Podman containers under your own user account. There is no Docker daemon and no sudo, except once during installation to point the system resolver at the .test domains."
            }
          },
          {
            "@type": "Question",
            "name": "How do I move my Laragon projects to Linux?",
            "acceptedAnswer": {
              "@type": "Answer",
              "text": "Copy the project folders out of C:\\laragon\\www to any directory on Linux, export your databases from Laragon with mysqldump, then run lerd link inside each project and lerd db:import to restore the dump. Lerd generates the virtual host, the .test hostname and the TLS certificate for you, so nothing in the project itself has to change."
            }
          }
        ]
      }
---

# Laragon for Linux

**Laragon does not run on Linux.** It is a Windows-only application, and its maintainers have repeatedly said a Linux port is not planned. There is no official `.deb`, no `.rpm`, no AppImage, and running the Windows build under Wine is not viable because Laragon ships Windows binaries of Apache, Nginx, PHP and MySQL along with its own tray application.

If you have moved to Linux, or you work on Linux and someone told you to "just use Laragon", **Lerd is the closest equivalent**. It is the same idea, a single tool that gives every project a pretty `.test` URL, HTTPS, a PHP version and the databases it needs, with nothing to configure per project. It is MIT licensed, free for commercial use, and runs on Linux and macOS.

```bash
curl -fsSL https://lerd.sh/install.sh | bash
cd ~/code/myapp
lerd link
```

Your project is live at `https://myapp.test`. That is the whole setup.

Prefer your package manager? Lerd is also available through an [apt PPA, a Fedora COPR or Homebrew](/getting-started/installation#install-via-apt-ubuntu-debian), so it updates with the rest of your system.

## Every Laragon feature, and the Lerd equivalent

| What you used in Laragon | The same thing in Lerd |
|---|---|
| Pretty URLs, `app.test` with no hosts file edits | Automatic `.test` domains, resolved by lerd's built-in DNS server wired into your system resolver, no `/etc/hosts` edits |
| Auto virtual hosts, a vhost file generated per project | An nginx vhost generated on `lerd link`, with [overrides](/usage/nginx-overrides) when you need them |
| One-click SSL, self-signed certificates | `lerd secure`, a real mkcert certificate trusted by your system and browsers, no warning page |
| Quick app, create Laravel, WordPress or Symfony in one click | `lerd new myapp`, scaffolds through the framework's own installer, then links and serves it |
| Multiple PHP versions, switched from the tray | PHP 7.4 and 8.0 to 8.5, picked per project by `lerd isolate 8.4` or auto-detected from `composer.json` |
| Bundled MySQL, PostgreSQL, Redis, Memcached | `lerd service start mysql`, plus PostgreSQL, Redis, Meilisearch, MongoDB, S3-compatible storage and Mailpit, shared across every site |
| The tray menu, start/stop, logs, terminal | A [system tray](/features/system-tray), a [web dashboard](/features/web-ui) at `127.0.0.1:7073` and a [terminal dashboard](/features/tui) |
| Cmder terminal with the right PHP on `PATH` | `lerd php`, `lerd composer` and `lerd shell`, always on the version that project is registered on |
| Portable, everything in `C:\laragon` | Everything under `~/.config/lerd` and `~/.local/share/lerd`, XDG-compliant, nothing written to system directories |
| Node, npm, Python and Go bundled alongside PHP | Per-project Node, plus [any other runtime](/getting-started/containers) through a `Containerfile.lerd` |

## Lerd vs Laragon

|  | Lerd | Laragon |
|---|---|---|
| Platforms | Linux (systemd), macOS, Windows via WSL2 | Windows only |
| License | Open source (MIT), free for commercial use | Proprietary, a licence is required from Laragon 7 onwards |
| Cost | Free | Free unlicenced tier with a reminder popup and no auto-updates; paid non-commercial and commercial licences |
| Stack | Nginx, PHP-FPM and services as rootless Podman containers | Apache or Nginx and services as native Windows binaries in a portable folder |
| `.test` domains | Automatic, through lerd's built-in DNS server | Automatic, through hosts file entries written by the tray app |
| HTTPS | `lerd secure`, mkcert certificate trusted system-wide | One-click self-signed certificate, trusted after you install it manually |
| PHP versions | 7.4, 8.0 to 8.5, per project | Multiple versions, downloaded into the Laragon folder |
| Per-project config | [`.lerd.yaml`](/configuration#per-project-config-lerd-yaml) committed to the repo, covering PHP, Node, services and workers | None, configuration lives in the Laragon install |
| Queue and scheduler workers | `lerd worker start queue` / `schedule`, as systemd user services | Not built in |
| Dashboard | Web UI, system tray, terminal dashboard, installable PWA | Native tray application |
| AI / MCP | Built-in [MCP server](/features/mcp) for Claude Code, Cursor, Junie and Windsurf | Not built in |
| Non-PHP projects | First-class through `Containerfile.lerd` (Node, Python, Go, Rails) | Node, Python and Go binaries bundled, not served as projects |

**Choose Laragon when:** you are on Windows, you want native binaries with no container layer, and the portable folder model suits how you work.

**Choose Lerd when:** you are on Linux or macOS, you want the same zero-config workflow without a Docker daemon or a per-project compose file, or you want the environment itself described in a file you can commit.

## Migrating from Laragon to Lerd

Nothing inside your projects has to change. Lerd reads them where they are, and the steps below map each part of a Laragon setup onto its Lerd equivalent.

### Project layout: `C:\laragon\www` becomes a parked folder

Laragon serves every folder in `C:\laragon\www` as `<folder>.test`. Lerd does the same with a parked directory: copy the projects anywhere on Linux, for example `~/code`, then park it once.

```bash
curl -fsSL https://lerd.sh/install.sh | bash
lerd start
cd ~/code
lerd park
```

Every PHP project inside `~/code` is now a site, and new folders you add later are picked up on their own, just like dropping a folder into `www`. To serve a single project from somewhere else, run `lerd link` inside it instead. Lerd detects the framework, picks the PHP version from `composer.json` and writes the nginx config. Use `lerd init` if you want a wizard to choose the PHP version, HTTPS and services and save the answers to `.lerd.yaml`.

### Virtual hosts: `app.test` stays `app.test`

Laragon's auto virtual hosts and Lerd's sites use the same `.test` names, so bookmarks and `APP_URL` values carry over. The differences:

- No hosts file. Lerd answers every `.test` lookup with its own DNS server, so a new site resolves the moment it is linked.
- HTTPS is one command, `lerd secure`, and the certificate is trusted by your browsers, with no warning page.
- Custom vhost lines, Laragon's per-site Apache or nginx files, go into an [nginx override](/usage/nginx-overrides). A project that relied on `.htaccess` needs its rules translated, as described on the [MAMP page](/getting-started/mamp#do-you-need-to-translate-your-htaccess).
- A different domain than the folder name is `lerd domain add`, covered in [Domains](/usage/domains).

### Databases: dump, then import

Dump each database from Laragon before you leave Windows, using its terminal or HeidiSQL's export:

```bash
mysqldump -u root myapp > myapp.sql
```

On Linux, start the matching service and let `lerd env` point the project at it. Laragon's MySQL runs as `root` with no password and Lerd's does not, so `lerd env` rewrites `DB_HOST`, `DB_USERNAME`, `DB_PASSWORD`, `REDIS_HOST` and `MAIL_HOST` to match the services Lerd runs instead of you editing `.env` by hand. Then import the dump into the site's database, which `lerd db:import` reads from that `.env`:

```bash
lerd service start mysql
cd ~/code/myapp
lerd env
lerd db:import myapp.sql
```

A desktop client such as DBeaver or TablePlus connects on `127.0.0.1` with the credentials `lerd env` wrote. PostgreSQL works the same way with `lerd service start postgres`.

Full detail lives in the [quick start](/getting-started/quick-start), [site management](/usage/sites) and [database](/usage/database) guides.

## What is genuinely different

Be aware of these before you switch, they are the places where the mental model changes:

- **Containers, not a portable folder.** PHP and the services run in rootless Podman containers, so there is no `C:\laragon` you can copy to another machine. What you commit instead is [`.lerd.yaml`](/configuration#per-project-config-lerd-yaml), which rebuilds the same environment anywhere.
- **Services are shared, not per project.** One MySQL, one Redis, one Mailpit across every site, which is why five running projects cost around 200 MB of RAM rather than five full stacks.
- **Nginx by default.** Lerd serves through Nginx and PHP-FPM. If a project depends on `.htaccess` rules, translate them into an [nginx override](/usage/nginx-overrides).
- **One `sudo` at install time.** Only to point the system resolver at the `.test` domains. Everything after that runs as your own user.

## Staying on Windows

If you are not leaving Windows, Lerd runs inside WSL2 and one installer sets it up. Your projects live on the Linux filesystem and are reachable from Windows browsers at their `.test` addresses. See [Lerd for Windows](/getting-started/windows).

## Frequently asked questions

**Is there a Laragon version for Linux?**
No, and there are no plans for one. Laragon is a Windows application built around Windows binaries and a Windows tray app.

**Can I run Laragon on Ubuntu with Wine?**
Not practically. The bundled Apache, Nginx, PHP and MySQL are Windows builds, and the parts of Laragon that make it worth using, the tray app, the hosts file automation and the vhost generation, are the parts least likely to work.

**What is the best Laragon alternative for Linux?**
Lerd, if you want the Laragon model of a shared zero-config stack that any project can be dropped into. [DDEV or Lando](/getting-started/comparison) if you would rather have a per-project Docker stack described in a committed config file.

**Does Lerd need Docker?**
No. It uses rootless Podman, which has no daemon, and it never asks for Docker Desktop.

**Is Lerd free for commercial work?**
Yes. MIT licensed, with no paid tier and no licence popup.

## Next steps

- [Requirements](/getting-started/requirements) and [installation](/getting-started/installation)
- [Quick start](/getting-started/quick-start), a project served in two commands
- [Full comparison](/getting-started/comparison) against Laravel Herd, Sail, DDEV and Lando
