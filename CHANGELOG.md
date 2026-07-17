# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-07-17

### Added

- Single 3.5 MB `.exe` — zero dependencies, no Electron, no Python runtime.
- Auto-setup wizard downloads `llama-server` (inference engine) and models from HuggingFace on first run.
- One-click model install from 5 curated GGUF models sized for 2–4 GB RAM.
- Streaming Chat UI with token-by-token responses, markdown rendering, and conversation history.
- Conversation management — create, select, delete, auto-save as JSON files.
- OpenAI-compatible API — `POST /v1/chat/completions` and `GET /v1/models`.
- Management API — system info, model pull/delete, config read/write.
- Hardware-aware model recommendation — detects RAM, CPU cores, free disk space.
- USB-portable operation — all data stored alongside the `.exe`.
- 100% offline after initial model download.
- Customizable system prompt, temperature, max tokens, context size.
- NSIS installer with Start Menu shortcut and uninstaller entry.
- Cross-compiled binaries for Windows, Linux, and macOS.
- Svelte 5 + SvelteKit SPA frontend embedded via Go `//go:embed`.
- Chi HTTP router with SSE support for streaming responses.
- Configurable API keys via Settings page.
