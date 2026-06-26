# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.2] - 2026-06-26

### Fixed
- Fix install.sh download URL to include `v` prefix in version tag
- Fix systemd service to use `--foreground` flag, preventing daemonization restart loop
- Fix default admin port reference from 3000 to 9000 in install scripts

## [0.0.1] - 2026-06-26

### Added
- Route management with Host + PathPrefix matching
- Authentication middleware (API Key, Bearer Token)
- Web UI for configuration
- Hot reload configuration
- SQLite storage with auto-migration
- Access logs with visualization
- Certificate management with local CA
- Multi-language support (English, Chinese)
- Per-route custom request/response header manipulation
- Configurable listen ports, HTTPS toggle
- CLI setup flow and dev tooling
- Cross-platform installation scripts (Linux, macOS, Windows)
- Docker support
- GitHub Actions CI/CD pipeline

[Unreleased]: https://github.com/pallyoung/auth-gate/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/pallyoung/auth-gate/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/pallyoung/auth-gate/releases/tag/v0.0.1
