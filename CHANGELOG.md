# Changelog

> **Note**: As of v0.2.0, changelogs are automatically generated and published with each [GitHub Release](https://github.com/tareqmamari/cloud-logs-mcp/releases).
> This file contains historical changelog entries for reference only.

All notable changes to the IBM Cloud Logs MCP Server are documented in GitHub Releases.


## [0.4.1](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.4.0...v0.4.1) (2025-12-10)


### Bug Fixes

* **deps:** update codeql-action SHA to valid v4.31.7 commit ([e6779c4](https://github.com/tareqmamari/cloud-logs-mcp/commit/e6779c4ca3bee4bf486e4bc80ac040ee071a062a))
* **deps:** update codeql-action SHA to valid v4.31.7 commit ([f819fc7](https://github.com/tareqmamari/cloud-logs-mcp/commit/f819fc717114489e48dfbcb3a4770e6ba18fc143))

## [0.4.0](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.3.0...v0.4.0) (2025-12-09)


### Features

* add AI helper tools and expand configuration options ([810b8f6](https://github.com/tareqmamari/cloud-logs-mcp/commit/810b8f62a36c2b14d5c1b1f5f8c9ba4be5c311d6))
* automate CHANGELOG.md updates with GoReleaser ([d733f50](https://github.com/tareqmamari/cloud-logs-mcp/commit/d733f50bb259906ffb0f5e5488a56279d40f40e2))
* **client:** add SSE support and improve request handling ([bba39bd](https://github.com/tareqmamari/cloud-logs-mcp/commit/bba39bd758bed457321bc61f6561318d7ec2066b))
* **metrics:** integrate Prometheus client for metrics export ([ff00f39](https://github.com/tareqmamari/cloud-logs-mcp/commit/ff00f397413d4d35dbaa724bf1ef121949861d00))
* **observability:** add tracing, audit logging, and security utilities ([e1e8a7e](https://github.com/tareqmamari/cloud-logs-mcp/commit/e1e8a7e43d063eedaf7ac5c7a86c447f3340d0bb))
* **query:** add DataPrime validation and query builder ([f4e44f0](https://github.com/tareqmamari/cloud-logs-mcp/commit/f4e44f0d87cc7556c39e99e5633704e83ec968fb))
* **release:** migrate to Release Please for automated releases ([f5868d0](https://github.com/tareqmamari/cloud-logs-mcp/commit/f5868d02dba731ce46cf997fbd35f637fdead259))
* **server:** add metrics tracking for tool executions ([09a9456](https://github.com/tareqmamari/cloud-logs-mcp/commit/09a9456cea63998971b6673e448b169dbf59dfb3))
* **session:** add session context for tool chaining ([eeb6451](https://github.com/tareqmamari/cloud-logs-mcp/commit/eeb645181080e0b2cbd0fea572474abc2bc4f9e0))
* **tools:** add query templates library ([bad2928](https://github.com/tareqmamari/cloud-logs-mcp/commit/bad292823573733179d4d95f1e9685d18ecfbdb3))
* **tools:** add result analysis and smart suggestions ([f3e747a](https://github.com/tareqmamari/cloud-logs-mcp/commit/f3e747a8e2192dd9fb4d0bda9550c715b95f6e78))
* **tools:** add Tool interface, prompts registry, and capabilities ([504d1b3](https://github.com/tareqmamari/cloud-logs-mcp/commit/504d1b3f5c45ee05da96645c998639b1980e4368))
* **tools:** add workflow automation and query validation ([ac4643a](https://github.com/tareqmamari/cloud-logs-mcp/commit/ac4643a170e94377399cf6c892d9ad117f20f1df))
* **tools:** complete ToolCapabilities map for all resource types ([54fa3d9](https://github.com/tareqmamari/cloud-logs-mcp/commit/54fa3d955ca1f6dbffc7b13503e02aeff9e5dc92))
* **tools:** improve descriptions and add dry-run validation ([bd1a602](https://github.com/tareqmamari/cloud-logs-mcp/commit/bd1a602a9864d46b36183c1840f56177d8321ce1))


### Bug Fixes

* **config:** add missing scopes to commitlint config ([2e8b93e](https://github.com/tareqmamari/cloud-logs-mcp/commit/2e8b93e8c95d9e226411c3361dcba277afbb882f))
* Disable SBOM generation in GoReleaser ([71fb4d2](https://github.com/tareqmamari/cloud-logs-mcp/commit/71fb4d276ca4b60bb5336bd53f2eaf0f5a60df59))
* Only create tags after CI passes successfully ([0e49d52](https://github.com/tareqmamari/cloud-logs-mcp/commit/0e49d52c435d0aee7ae6299e8b9471dcbb0205a2))
* **release:** add SARIF fallback handling for SAST scanners ([cea366c](https://github.com/tareqmamari/cloud-logs-mcp/commit/cea366c6602fbf06b86ad090b204e1cea44255eb))
* **release:** disable SBOM generation in GoReleaser ([b463895](https://github.com/tareqmamari/cloud-logs-mcp/commit/b463895a103dba1d4c65d348bc3e08c61f7adc9b))
* Remove PR title validation from commitlint workflow ([e1fd29a](https://github.com/tareqmamari/cloud-logs-mcp/commit/e1fd29a3f7b4370749905883108a17a80febd409))
* Security issues and comprehensive enhancements ([#14](https://github.com/tareqmamari/cloud-logs-mcp/issues/14)) ([c25e2d1](https://github.com/tareqmamari/cloud-logs-mcp/commit/c25e2d1fbfb19994c811960593e0c903139d6807))
* **tools:** handle required parameter errors instead of ignoring ([508b79f](https://github.com/tareqmamari/cloud-logs-mcp/commit/508b79fb95379b7ddd1c163b01080a9b9c1b9193))
* **tools:** replace remaining ~~ operators with DataPrime methods ([ee08614](https://github.com/tareqmamari/cloud-logs-mcp/commit/ee086142853723ee55d2c275b168b31ab4d9517c))
* **tools:** resolve golangci-lint issues and improve CI ([aae38f2](https://github.com/tareqmamari/cloud-logs-mcp/commit/aae38f243c5bbd4308961487a87f86412f178d1c))
* **validator:** resolve linting issues ([677ef90](https://github.com/tareqmamari/cloud-logs-mcp/commit/677ef90fc53cb741d36b86ddebf54a153a05d833))

## [v0.3.0] - 2025-11-22
### Features
- automate CHANGELOG.md updates with GoReleaser
### Documentation
- clarify URL placeholders in installation instructions

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [v0.2.0] - 2025-11-15
### Documentation
- update description wording in README
- add Microsoft 365 Copilot configuration
- create concise professional README with Copilot support

### Features
- add explicit release targets for patch/minor/major versions

### Maintenance
- remove Formula directory
- **deps:** bump dependencies and setup changelog automation


## [v0.1.0] - 2025-11-15
### Bug Fixes
- Remove invalid folder property from Homebrew config

### Documentation
- Add Homebrew installation instructions

### Features
- Add automated semantic versioning with svu and Homebrew formula


## [Unreleased] - 2025-11-15
### Bug Fixes
- **ci:** Use Go version from go.mod for all test jobs
- **lint:** Fix linting issues and rename module to tareqmamari

### Documentation
- update documentation for dashboard tools

### Features
- **release:** Add GoReleaser support with Makefile targets
- **release:** Add production-ready tooling and workflows
- **tools:** Add comprehensive dashboard management tools

### Maintenance
- Replace observability-c with tareqmamari in all references


[v0.2.0]: https://github.com/tareqmamari/logs-mcp-server/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/tareqmamari/logs-mcp-server/compare/v0.0.0...v0.1.0
[v0.3.0]: https://github.com/tareqmamari/logs-mcp-server/compare/v0.3.0...v0.3.0
