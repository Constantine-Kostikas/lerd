---
description: How to contribute to Lerd, building it from source, opening a pull request, and adding frameworks, services or plugin hooks.
---

# Contributing

Lerd is MIT licensed and developed in the open at [github.com/lerd-env/lerd](https://github.com/lerd-env/lerd). Most contributions do not need Go at all: frameworks and services are versioned YAML in their own stores, and a change there reaches every install within a day.

- [Building from source](/contributing/building) covers the toolchain, the make targets and the local checks CI runs.
- [Submitting a pull request](/contributing/pull-requests) explains formatting, commit style and what a pull request needs before it can merge.
- [Extending the plugin system](/contributing/plugins) is for plugins that need an event or a job lerd does not offer yet.

To add a framework or a new major version, open a pull request against [lerd-env/frameworks](https://github.com/lerd-env/frameworks); the schema is documented in [Framework definitions](/usage/framework-definitions). Services such as databases, caches and admin dashboards go to [lerd-env/services](https://github.com/lerd-env/services), described in [Service presets](/usage/service-presets).

Questions and ideas are welcome on [Discord](https://discord.gg/5JK54s7xCC) or in [GitHub Discussions](https://github.com/lerd-env/lerd/discussions).
