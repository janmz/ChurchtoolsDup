# Security Policy

## Supported versions

| Version | Supported |
| --- | --- |
| 0.1.x | yes |

## Reporting a vulnerability

Please report security issues privately to the repository maintainer.
Do not open public issues for credential leaks or exploit details.

## Handling secrets

- Never commit `config.json`, `.env`, or login tokens.
- Use `config.example.json` and environment variables in CI.
- Restrict file permissions on local config files (0600).

## API credentials

ChurchTools login tokens grant full access as the associated user.
Rotate tokens if they may have been exposed.
