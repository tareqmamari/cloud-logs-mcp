# Changelog

> **Note**: As of v0.2.0, changelogs are automatically generated and published with each [GitHub Release](https://github.com/tareqmamari/cloud-logs-mcp/releases).
> This file contains historical changelog entries for reference only.

All notable changes to the IBM Cloud Logs MCP Server are documented in GitHub Releases.


## [0.9.1](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.9.0...v0.9.1) (2026-03-09)


### Bug Fixes

* **ci:** add top-level read-all permissions to integration tests workflow ([fcacf71](https://github.com/tareqmamari/cloud-logs-mcp/commit/fcacf718aaa660bb5370ec73fedb7e63666771b5))
* **ci:** enable SAST scanning on pull requests ([d26af20](https://github.com/tareqmamari/cloud-logs-mcp/commit/d26af202e0fad9f9bf5abff13c58ae69397661f4))
* **ci:** pin commitlint dependency versions ([c394292](https://github.com/tareqmamari/cloud-logs-mcp/commit/c39429217d7ebb9e70be9cd92b0dcc3c1f71505e))
* **ci:** replace npm-based commitlint with GitHub Action ([e14ba55](https://github.com/tareqmamari/cloud-logs-mcp/commit/e14ba551a921cfd94e7d5ad88ff523a886c7d1b4))
* **ci:** run SAST on all commits to main, not just Go file changes ([cb92c49](https://github.com/tareqmamari/cloud-logs-mcp/commit/cb92c49d79b9da01b6f88e16d4c52fa3fc141d62))
* **ci:** scope SAST workflow permissions to job level ([19ce1b3](https://github.com/tareqmamari/cloud-logs-mcp/commit/19ce1b328e48244fa458ad6545915047bb9edfb3))
* **ci:** scope workflow token permissions to job level ([ed6f6c2](https://github.com/tareqmamari/cloud-logs-mcp/commit/ed6f6c24a1bf4c02b6f917df65ea32d720488c46))
* **client, tools:** resolve gosec G118 and G115 warnings ([fc0bfce](https://github.com/tareqmamari/cloud-logs-mcp/commit/fc0bfce5133d0041562c8f7821930e059f342558))
* **client:** isolate insecure TLS verification behind build tag ([d88e647](https://github.com/tareqmamari/cloud-logs-mcp/commit/d88e647fae3284c435eb71b85ae93fcd18da9362))
* **client:** refactor MockClient.Do to use defer-based mutex unlock ([af673fd](https://github.com/tareqmamari/cloud-logs-mcp/commit/af673fdca89ab49e493b842ce95324822758f499))
* **client:** remove TLSVerify config and enforce TLS verification ([e021830](https://github.com/tareqmamari/cloud-logs-mcp/commit/e021830b68471bc3ac39b7f96a745772c20451f6))
* **tools:** use concrete type for SSE event deserialization ([0f3e3e1](https://github.com/tareqmamari/cloud-logs-mcp/commit/0f3e3e1153db403e4e8b5cda2494929143b63ee0))

## [0.9.0](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.8.0...v0.9.0) (2026-02-26)


### Features

* **api:** add service abstraction layer with structured error handling ([52c1e4e](https://github.com/tareqmamari/cloud-logs-mcp/commit/52c1e4ecc1a2aabdf80a063944c746e898c87003))
* **client:** add Retry-After header support and instance info accessor ([6790dc3](https://github.com/tareqmamari/cloud-logs-mcp/commit/6790dc3267bfcbb55dd92cdf5e5e6301ff74f43e))
* **tools:** add instance info to query responses ([33c4518](https://github.com/tareqmamari/cloud-logs-mcp/commit/33c4518ce90d3dc2e64580805d528a6b8c5ab8b8))
* **tools:** add service adapter, log clustering, and context injection ([f2f397d](https://github.com/tareqmamari/cloud-logs-mcp/commit/f2f397daff08b44fbe46e59443c4c3dfc7c9c596))
* **validator:** add centralized DataPrime query validation pipeline ([f50de6a](https://github.com/tareqmamari/cloud-logs-mcp/commit/f50de6acf7e2b7ef24cc27d1d9d08423430b50d3))


### Bug Fixes

* **lint:** resolve all golangci-lint v2.10 warnings ([72220f8](https://github.com/tareqmamari/cloud-logs-mcp/commit/72220f875a3231251bbad027f677b9179edc4a36))
* **tools:** prevent context compaction failures and improve log formatting ([f2732ac](https://github.com/tareqmamari/cloud-logs-mcp/commit/f2732ac542c5e003e125f761137825ddb238144b))
* **tools:** update dashboard configuration as per the OpenAPI spec ([a3c941d](https://github.com/tareqmamari/cloud-logs-mcp/commit/a3c941de4ee83500d5a005fdd1f86012378b8c70))
* **tools:** use roundTime() for time bucketing in queries ([b4fcc67](https://github.com/tareqmamari/cloud-logs-mcp/commit/b4fcc672a173a85a939eea8f623a0c8c68796205))

## [0.8.0](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.7.0...v0.8.0) (2025-12-15)


### Features

* **server:** add about://service resource with service aliases ([2e072fc](https://github.com/tareqmamari/cloud-logs-mcp/commit/2e072fc2249877a2d6c2b3bbea1aaeff290e2982))
* **server:** load TCO policies at startup for tier selection ([a52deac](https://github.com/tareqmamari/cloud-logs-mcp/commit/a52deacc987ff2a37b21c7e4dbd78dbd8a1e2b98))
* **tools:** add budget-aware context management ([326701c](https://github.com/tareqmamari/cloud-logs-mcp/commit/326701c14a22001cb03e0b611da5bc089e76c267))
* **tools:** add get_dataprime_reference tool and shorten descriptions ([fb56957](https://github.com/tareqmamari/cloud-logs-mcp/commit/fb56957bfd3d357e5c6a0bd9526aaf1ec04eae5a))
* **tools:** add intent verification for destructive operations ([62d93a1](https://github.com/tareqmamari/cloud-logs-mcp/commit/62d93a1485e37d86066b67470861743ee4bda30b))
* **tools:** add mixed-type field auto-correction and response cleanup ([5fb42c9](https://github.com/tareqmamari/cloud-logs-mcp/commit/5fb42c95925556c214faf5fc4effc24f370040bb))
* **tools:** add scout_logs tool for pattern discovery and root cause analysis ([dcff66f](https://github.com/tareqmamari/cloud-logs-mcp/commit/dcff66fa0717589dd006c9e9713ac027fbdf0f02))
* **tools:** add SmartInvestigateTool for automated root cause analysis ([fc10d47](https://github.com/tareqmamari/cloud-logs-mcp/commit/fc10d47b0380a15ebbf6312c2db2ce4744033296))
* **tools:** add SRE-grade suggest_alert with burn rate alerting ([8485d43](https://github.com/tareqmamari/cloud-logs-mcp/commit/8485d4338049bc6e775caf54ee2e09521f3fc601))
* **tools:** add tool registry factory and per-tool-type timeouts ([8502f96](https://github.com/tareqmamari/cloud-logs-mcp/commit/8502f96ae8eb431804103b1afe4186ba063f2c8e))
* **tools:** auto-detect TCO policies for intelligent tier selection ([882af50](https://github.com/tareqmamari/cloud-logs-mcp/commit/882af50bff3577c7c508e553bbabc54df507aa70))


### Bug Fixes

* **client:** use crypto/rand for retry jitter ([71a59bb](https://github.com/tareqmamari/cloud-logs-mcp/commit/71a59bbfbcdb60bc82a3f475aef0645a0cacf6c2))
* **docs:** use correct CI workflow filename in badge URL ([562d9cc](https://github.com/tareqmamari/cloud-logs-mcp/commit/562d9ccf7dca1568931d3e90c618cce2ae7eaae1))
* **tools:** add tier parameter to scout_logs with archive default ([0246b2e](https://github.com/tareqmamari/cloud-logs-mcp/commit/0246b2e8ded725b9e179623d1f45fbdde443c19a))
* **tools:** correct TCO type_medium tier mapping ([730e3d0](https://github.com/tareqmamari/cloud-logs-mcp/commit/730e3d041208dd96429afc944be8a067b60ba069))
* **tools:** improve sort-to-sortby auto-correction pattern ([24043c4](https://github.com/tareqmamari/cloud-logs-mcp/commit/24043c45a1ac2dd5f677950a761bf34e91f0e4cf))
* **tools:** prevent concurrent map access in session summary ([7bf700f](https://github.com/tareqmamari/cloud-logs-mcp/commit/7bf700f0a5e5beeb4b0b67ce7952e3edfae19377))
* **tools:** use correct query API request format for scout_logs ([2ddf5bb](https://github.com/tareqmamari/cloud-logs-mcp/commit/2ddf5bb716f6671fc0aaa263df94eef54e0760fe))
* **validator:** auto-correct 'sort' to 'orderby' in DataPrime queries ([aa567e2](https://github.com/tareqmamari/cloud-logs-mcp/commit/aa567e2a318e171d1c34c1bc479c48298718df1f))

## [0.7.0](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.6.0...v0.7.0) (2025-12-13)


### Features

* add MCP best practices and infrastructure improvements ([af27af3](https://github.com/tareqmamari/cloud-logs-mcp/commit/af27af3f2fc03e326a7900b25a376bd57a2403da))

## [0.6.0](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.5.1...v0.6.0) (2025-12-10)


### Features

* **config:** extract region and instance ID from dev/stage service URLs ([a315caf](https://github.com/tareqmamari/cloud-logs-mcp/commit/a315cafb20877402700c6114436e40ec6bd454c2))
* **config:** support service URL construction from region and instance ID ([f1b1adf](https://github.com/tareqmamari/cloud-logs-mcp/commit/f1b1adf8970e93fd8c0788508e13780913751338))

## [0.5.1](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.5.0...v0.5.1) (2025-12-10)


### Bug Fixes

* **release:** update cosign signing for v4 bundle format ([a02bbd2](https://github.com/tareqmamari/cloud-logs-mcp/commit/a02bbd2c0ac03db0cdfbdb1d1e676ac7ae0679af))

## [0.5.0](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.4.3...v0.5.0) (2025-12-10)


### Features

* **release:** add cryptographic signing for release artifacts ([7e6601b](https://github.com/tareqmamari/cloud-logs-mcp/commit/7e6601b0117a0801b72dc3453a8624336948d4cc))

## [0.4.3](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.4.2...v0.4.3) (2025-12-10)


### Bug Fixes

* **ci:** use GitHub App token for Release Please to trigger GoReleaser ([fad3464](https://github.com/tareqmamari/cloud-logs-mcp/commit/fad34643ff29e301ccde4bef558621fcaa4861f7))

## [0.4.2](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.4.1...v0.4.2) (2025-12-10)


### Bug Fixes

* **ci:** trigger GoReleaser on release published instead of created ([df90508](https://github.com/tareqmamari/cloud-logs-mcp/commit/df9050822f23c01b21170c765b6ac1659351f369))

## [0.4.1](https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.4.0...v0.4.1) (2025-12-10)


### Bug Fixes

* **config:** use redhat versioning for Red Hat registry images ([31d4564](https://github.com/tareqmamari/cloud-logs-mcp/commit/31d45642304c93448251a08221fd27d68d6510a0))
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


[v0.2.0]: https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.0.0...v0.1.0
[v0.3.0]: https://github.com/tareqmamari/cloud-logs-mcp/compare/v0.3.0...v0.3.0
