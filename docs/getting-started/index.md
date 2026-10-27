---
description: Install lerd and get a PHP site running on a local .test domain with HTTPS, on Linux, macOS or Windows via WSL2.
---

# Getting Started

Lerd is a local PHP development environment for Linux and macOS, with Windows supported through WSL2. It runs Nginx, PHP-FPM and your services as rootless Podman containers, so there is no Docker daemon, no sudo for day to day work, and nothing installed system wide.

## Start here

Three pages, in order, and you have a site running.

- [Requirements](/getting-started/requirements) covers the supported distributions and the handful of packages lerd expects to find.
- [Installation](/getting-started/installation) is the main path, a single install script that sets up directories, the container network, DNS and certificates.
- [Quick Start](/getting-started/quick-start) is the short version once lerd is installed: link a directory, get a `.test` domain with HTTPS.

Something not working? [Troubleshooting](/troubleshooting) lists the common errors with their fixes, and `lerd doctor` checks your setup. Already running an older lerd? [Updating from before 1.26](/getting-started/updating-from-pre-1.26) covers the one release that needs a manual step.

## Platform guides

The install script covers mainstream Linux and macOS on its own. These pages are for the systems that need something extra, or that have their own integration worth knowing about.

- [Omarchy](/getting-started/omarchy) is the Arch and Hyprland desktop, with the `crun` and `nss` prerequisites, the systemd-resolved detail and the lerd bar widget.
- [NixOS](/getting-started/nixos) documents the flake based route for immutable and declarative systems.
- [Lerd for Windows](/getting-started/windows) installs everything with one installer, and [Windows (WSL2)](/getting-started/wsl2) explains the systemd and mirrored networking setup Windows needs, most of which `lerd wsl:setup` does for you.

## Framework walkthroughs

Lerd detects your framework from the project itself and configures workers, environment wiring and health checks from a versioned store definition rather than hardcoded rules.

- [Laravel](/getting-started/laravel)
- [Symfony](/getting-started/symfony)
- [WordPress](/getting-started/wordpress)
- [Containers (Node, Python, Go, …)](/getting-started/containers) for stacks that are not PHP at all.

There are walkthroughs for [Drupal](/getting-started/drupal), [TYPO3](/getting-started/typo3), [CakePHP](/getting-started/cakephp), [CodeIgniter](/getting-started/codeigniter), [Statamic](/getting-started/statamic), [Winter CMS](/getting-started/winter), [Yii](/getting-started/yii), [Tempest](/getting-started/tempest) and [Magento](/getting-started/magento) too. Each framework's one-shot actions, cache clears and migrations among them, are listed in [Framework commands](/usage/framework-commands).

## Coming from another tool

Each of these is a migration guide, not just a feature table: what the equivalent command is, and how to move your projects and databases across.

- [Comparison](/getting-started/comparison) sets lerd against Herd, Laragon, Laradock, DDEV, Lando and Sail in one place.
- [Laravel Herd for Linux](/getting-started/herd-linux) is aimed at people who used Herd on a Mac and moved to Linux.
- [Laragon for Linux](/getting-started/laragon-linux) is aimed at people moving over from Windows.
- [XAMPP alternative for Linux](/getting-started/xampp) moves `htdocs` projects and their MariaDB databases across.
- [Laradock alternative](/getting-started/laradock) is for people leaving a per-project Docker Compose stack behind.
- [Laravel Sail alternative](/getting-started/sail) is the one with an automated importer, the sail plugin's `lerd sail:import`, database and S3 files included.

## Add-ons

- [Services](/getting-started/services) adds MongoDB, phpMyAdmin, Redis and the rest of the service presets.
- [NativePHP](/getting-started/nativephp) builds desktop and mobile apps from a Laravel project lerd is serving.
- [PhpStorm plugin](/features/phpstorm-plugin) shows the site's logs, workers, services and queries inside the IDE.

Once you are set up, [Usage](/usage/sites) covers day to day site management and [Features](/features/web-ui) covers the web UI, TUI, MCP server and the rest.
