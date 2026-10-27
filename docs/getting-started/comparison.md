---
title: Lerd compared with Herd, Laragon, DDEV, Lando and Sail
description: 'How Lerd compares with Laravel Herd, Laragon, Laravel Sail, DDEV, Lando and Laradock on OS support, licence, isolation and reproducibility, with a page per tool.'
---

# Lerd vs other tools

Lerd is a shared local stack: one nginx, PHP-FPM per PHP version and one instance of each service, run as rootless Podman containers, that every project drops into. Each tool below takes a different route, and each has its own page with the full comparison.

| Tool | OS support | Licence | How it runs | Project config in the repo |
|---|---|---|---|---|
| **Lerd** | Linux, macOS, Windows (WSL2) | MIT | Shared stack, rootless Podman | `.lerd.yaml`, optional |
| [Laravel Herd](#lerd-vs-laravel-herd) | macOS, Windows | Proprietary, paid Pro tier | Shared stack, native binaries | `herd.yml`, optional |
| [Laragon](#lerd-vs-laragon) | Windows | Proprietary, paid licence | Shared stack, native binaries | None |
| [Laravel Sail](#lerd-vs-laravel-sail) | Linux, macOS, Windows | MIT | Per-project Docker Compose | `docker-compose.yml`, required |
| [DDEV](#lerd-vs-ddev) | Linux, macOS, Windows | Apache 2.0 | Per-project containers | `.ddev/config.yaml`, required |
| [Lando](#lerd-vs-lando) | Linux, macOS, Windows | GPL-3.0 | Per-project containers | `.lando.yml`, required |
| [Laradock](#lerd-vs-laradock) | Linux, macOS, Windows | MIT | Per-project Docker Compose | `laradock/` folder, required |

## Lerd vs Laravel Herd

Herd is a shared stack of native binaries for macOS and Windows, free with a paid Pro tier for databases, mail and other services. Herd's own FAQ says Linux support is being worked on but is not ready. Lerd covers the same workflow on Linux and macOS, open source, with every service free. See [Laravel Herd for Linux](/getting-started/herd-linux#lerd-vs-laravel-herd), which also covers Herd 2.0.

## Lerd vs Laragon

Laragon is a portable Windows stack with `.test` URLs, auto virtual hosts and bundled databases, and needs a licence from version 7. It has no Linux build. Lerd gives the same workflow on Linux and macOS, and on Windows through WSL2. See [Laragon for Linux](/getting-started/laragon-linux#lerd-vs-laragon), with a migration guide.

## Lerd vs Laravel Sail

Sail is Laravel's official per-project Docker Compose setup, each project with its own containers and services. Lerd runs one shared stack instead, and its sail plugin moves a Sail project across, database and S3 files included, in one command. See [Laravel Sail alternative](/getting-started/sail).

## Lerd vs ddev

DDEV runs containers per project behind a shared router, configured by `.ddev/config.yaml` in the repo, which gives strong isolation between projects. Lerd shares one stack and needs nothing committed. See [Lerd vs DDEV](/getting-started/lerd-vs-ddev).

## Lerd vs Lando

Lando builds a per-project stack from a `.lando.yml` and platform recipes, with integrations for hosts such as Pantheon and Platform.sh. See [Lerd vs Lando](/getting-started/lerd-vs-lando).

## Lerd vs Laradock

Laradock is a set of Docker images behind a `docker-compose.yml` kept in a `laradock/` folder, with a workspace container for tooling. See [Laradock alternative](/getting-started/laradock) for the comparison and migration path.

Moving from an all-in-one stack instead? See [XAMPP alternative for Linux](/getting-started/xampp) and [Moving from MAMP](/getting-started/mamp).
