# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | Yes       |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. Do NOT open a public issue
2. Email the maintainer directly or use GitHub's private vulnerability reporting
3. Include steps to reproduce and potential impact

## Scope

This project handles:
- API keys (Anthropic, Ollama) — stored only in environment variables, never logged
- File system writes (MIDI output) — restricted to the output directory
- HTTP calls to LLM providers — no user data beyond prompts

The project does NOT handle:
- User authentication
- Personal data
- Network-facing services (it's a CLI tool)
