# LocalAI Hub — Product Requirements Document

**Status:** v1 Draft
**Target audience:** Non-technical users with 4-8 GB RAM Windows PCs
**Distribution:** Single .exe, portable on USB

---

## 1. Executive Summary

LocalAI Hub is a single-binary desktop application that lets anyone run open-source LLMs locally with zero setup. No Docker, no Python, no Node.js, no Electron. Double-click → browser opens → chat. Designed for 4 GB RAM machines and portable USB operation.

---

## 2. Problem Statement

| Solution | Pain Point |
|----------|-----------|
| Open WebUI | Requires Docker (unusable on 4 GB) |
| LM Studio | Electron (500 MB+ overhead) |
| Ollama | CLI-only, no friendly UI |
| USB portable projects | Script-heavy, no unified web UI |

**Gap:** No tool offers a single lightweight binary with an embedded web UI, one-click model install, and 4 GB RAM targeting.

---

## 3. Target Users

| Persona | Needs | Constraints |
|---------|-------|-------------|
| Casual user wants private ChatGPT | Chat interface, no terminal | 4 GB RAM, Windows, no admin |
| Developer wants local coding assistant | OpenAI-compatible API | Wants to use with opencode / Cursor |
| Privacy-conscious user | 100% offline, no cloud | No data ever leaves machine |
| USB portable user | Runs from pendrive | Zero host footprint |
| Low-spec PC owner | Runs on old hardware | 4 GB RAM, HDD, CPU-only |

---

## 4. User Stories

### Must-Have (v1)

1. As a user, I download a single .exe and run it — no install wizard, no dependencies.
2. As a user, my browser opens automatically to `http://localhost:8080` showing a chat UI.
3. As a user, the app detects my RAM and recommends a suitable model.
4. As a user, I click "Install" and the model downloads from HuggingFace with a progress bar.
5. As a user, I type a message and see the AI response stream token-by-token.
6. As a user, my conversation history persists across sessions.
7. As a user, I can configure the system prompt, temperature, and context size.
8. As a developer, I point any OpenAI-compatible client at `http://localhost:8080/v1` and use the running model.
9. As a user, I can run the app from a USB drive — models and config live alongside the .exe.

### Nice-to-Have (v2)

10. As a user, I can upload documents and ask questions (RAG).
11. As a user, I can switch between multiple installed models.
12. As a user, I can see CPU/RAM usage in the UI.
13. As a user, the app runs as a system tray icon.

---

## 5. Functional Requirements

### FR-1: Application Startup

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-1.1 | Single .exe launches an HTTP server on a configurable port (default 8080) | P0 |
| FR-1.2 | Server auto-opens `http://localhost:{port}` in the default browser | P0 |
| FR-1.3 | All paths are relative to the .exe location (portable) | P0 |
| FR-1.4 | Config is stored in `config.json` alongside the .exe | P0 |
| FR-1.5 | On first launch, detect RAM and show setup wizard | P0 |

### FR-2: Hardware Detection

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-2.1 | Detect total physical RAM (GB) on Windows | P0 |
| FR-2.2 | Detect CPU core count | P1 |
| FR-2.3 | Detect free disk space in models directory | P1 |
| FR-2.4 | Return info via `GET /api/system/info` | P0 |

### FR-3: Model Management

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-3.1 | Ship curated list of 5 GGUF models fitting 2–4 GB RAM | P0 |
| FR-3.2 | Recommend best model based on free RAM | P0 |
| FR-3.3 | Download GGUF from HuggingFace Hub via HTTPS | P0 |
| FR-3.4 | Report download progress via SSE events | P0 |
| FR-3.5 | Support resumable downloads (HTTP Range headers) | P1 |
| FR-3.6 | Delete installed models | P1 |

#### Curated Models

| Model | Repo | File | Q4 Size | Min RAM |
|-------|------|------|---------|---------|
| Phi-4-mini 3.8B | `microsoft/Phi-4-mini-instruct-gguf` | `Phi-4-mini-instruct-q4_k_m.gguf` | ~2.5 GB | 4 GB |
| Qwen3 3B | `Qwen/Qwen3-3B-Instruct-GGUF` | `qwen3-3b-instruct-q4_k_m.gguf` | ~2.0 GB | 4 GB |
| Llama 3.2 3B | `unsloth/Llama-3.2-3B-Instruct-GGUF` | `Llama-3.2-3B-Instruct-Q4_K_M.gguf` | ~2.5 GB | 4 GB |
| Gemma 3 1B | `ggml-org/gemma-3-1b-it-GGUF` | `gemma-3-1b-it-Q4_K_M.gguf` | ~0.7 GB | 2 GB |
| Qwen3 1.5B | `Qwen/Qwen3-1.5B-Instruct-GGUF` | `qwen3-1.5b-instruct-q4_k_m.gguf` | ~1.0 GB | 2 GB |

### FR-4: Inference Engine

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-4.1 | Bundle or auto-download `llama-server` (llama.cpp) | P0 |
| FR-4.2 | Start llama-server as subprocess on a free port | P0 |
| FR-4.3 | Health-check the subprocess before declaring ready | P0 |
| FR-4.4 | Kill subprocess on app exit (Windows job object) | P0 |
| FR-4.5 | Pass sane defaults: CPU-only, threads = cores-1, context 4096 | P0 |

### FR-5: Chat API

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-5.1 | `POST /v1/chat/completions` — OpenAI-compatible | P0 |
| FR-5.2 | Support `stream: true` (SSE token-by-token) | P0 |
| FR-5.3 | Support `stream: false` (full JSON response) | P0 |
| FR-5.4 | Proxy requests to llama-server, forward streaming response | P0 |
| FR-5.5 | `GET /v1/models` — list installed models | P0 |

### FR-6: Web UI

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-6.1 | First-run setup wizard with model recommendation | P0 |
| FR-6.2 | Chat page with streaming messages, markdown rendering | P0 |
| FR-6.3 | Conversation history sidebar (create, select, delete) | P1 |
| FR-6.4 | Settings page: system prompt, temperature, context size, theme | P1 |
| FR-6.5 | Models page: install/delete models, see disk usage | P1 |
| FR-6.6 | Dark theme by default | P1 |

### FR-7: Conversation Persistence

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-7.1 | Save conversations as JSON files in `conversations/` | P1 |
| FR-7.2 | Load conversation on select | P1 |
| FR-7.3 | Delete conversation | P1 |
| FR-7.4 | Auto-save as messages arrive | P1 |

### FR-8: Packaging

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-8.1 | Single .exe for Windows amd64 | P0 |
| FR-8.2 | Cross-compile for Linux amd64, macOS amd64/arm64 | P1 |
| FR-8.3 | Binary size under 15 MB | P0 |
| FR-8.4 | Distributed as ZIP archive (portable) | P0 |
| FR-8.5 | Optional NSIS installer for Windows | P2 |

---

## 6. Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-1 | Binary size | ≤ 15 MB |
| NFR-2 | RAM overhead (idle, no model) | ≤ 50 MB |
| NFR-3 | RAM overhead (with model loaded) | Model size + 512 MB |
| NFR-4 | Time-to-chat (first run, download) | Model download time + 10 s |
| NFR-5 | Time-to-chat (subsequent runs) | ≤ 5 s to browser open |
| NFR-6 | Inference speed | ≥ 5 tok/s on 4 GB CPU |
| NFR-7 | Offline capable | Full functionality after model download |
| NFR-8 | Cross-platform | Windows primary, Linux/macOS secondary |

---

## 7. Architecture

```
┌──────────────────────────────────────────────────┐
│                  LocalAI.exe                      │
│                                                    │
│  ┌──────────────────┐    ┌─────────────────────┐  │
│  │  Svelte 5 SPA    │◄──►│  Go HTTP Server      │  │
│  │  (embedded via   │    │  (chi router)         │  │
│  │   //go:embed)    │    │                       │  │
│  └──────────────────┘    └───────┬─────────────┘  │
│                                  │                 │
│  ┌───────────────────────────────▼──────────────┐  │
│  │  llama-server (subprocess, spawned on demand) │  │
│  │  OpenAI-compatible API on localhost:{port2}   │  │
│  └───────────────────────────────────────────────┘  │
│                                                    │
│  ┌──────────────────────────────────────────────┐  │
│  │  Model Store: ./models/*.gguf                │  │
│  │  Config:      ./config.json                  │  │
│  │  Chats:       ./conversations/*.json         │  │
│  └──────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

### Data Flow (Chat)

```
Browser ──POST /v1/chat/completions──► Go ──HTTP──► llama-server
  ▲                                          │
  │           SSE: data: {"token":"..."}      │
  └──────────────────────────────────────────┘
```

---

## 8. API Specification

### System

```
GET /api/system/info
→ { "ram": { "total": 8, "free": 4 }, "cpu": 4, "disk": { "free": 50000 },
    "recommendedModel": "phi-4-mini", "installedModels": [...], "llamaServer": "running" }
```

### Models

```
GET  /api/models              → [{ id, name, size, installed, quality, ... }]
POST /api/models/pull          → SSE stream { type: "progress"|"done"|"error", ... }
       body: { "model": "phi-4-mini" }
DELETE /api/models/{id}        → 204
```

### Chat (OpenAI-compatible)

```
POST /v1/chat/completions
     body: { model, messages, stream, temperature, max_tokens }
     → SSE or JSON (same format as OpenAI)

GET /v1/models
     → { object: "list", data: [{ id, object, created, owned_by }] }
```

### Conversations

```
GET    /api/conversations        → [{ id, title, created, updated, messageCount }]
GET    /api/conversations/{id}   → { id, title, messages: [...] }
POST   /api/conversations        → { id } (create new)
DELETE /api/conversations/{id}   → 204
```

---

## 9. UI Wireframes (Text)

### Setup Wizard (first run only)

```
┌──────────────────────────────────────────────┐
│  Welcome to LocalAI Hub                       │
│                                               │
│  Your PC: 8 GB RAM, 4 cores, 120 GB free      │
│                                               │
│  Recommended for your PC:                      │
│                                               │
│  ┌────────────────────────────────────────┐   │
│  │  Phi-4-mini ★★★★                       │   │
│  │  2.5 GB — Best reasoning for 4GB PCs   │   │
│  │  [Install]                             │   │
│  └────────────────────────────────────────┘   │
│                                               │
│  ┌────────────────────────────────────────┐   │
│  │  Qwen3 3B  ★★★★                        │   │
│  │  2.0 GB — Best coding for 4GB PCs      │   │
│  │  [Install]                             │   │
│  └────────────────────────────────────────┘   │
│                                               │
│  [Skip — I'll choose later]                   │
└──────────────────────────────────────────────┘
```

### Chat Page

```
┌────────────────────────────────────────────────┐
│  ☰ New Chat  │  LocalAI Hub             ⚙️ 🌙  │
├──────────────┴─────────────────────────────────┤
│  ┌──────────────────────────────────────────┐  │
│  │  Conversations                            │  │
│  │  ────────────────────────                 │  │
│  │  📄 What is Rust borrow…  │  ← active    │  │
│  │  📄 Write a Python scra…  │              │  │
│  │  📄 Hello world           │              │  │
│  │                                          │  │
│  │  [Installed models]                      │  │
│  │  ● Phi-4-mini (active)                  │  │
│  │  ○ Qwen3 3B                             │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  Q: What is the capital of France?              │
│  ─────────────────────────────────────────────  │
│  The capital of France is **Paris**. It is      │
│  located in the Île-de-France region...         │
│   (streaming...)                                │
│                                                 │
│  ┌──────────────────────────────────────────┐   │
│  │  Type a message...                [Send] │   │
│  └──────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┘
```

---

## 10. Tech Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Language | Go 1.23+ | Single binary, cross-compile, small footprint |
| HTTP router | `chi` | Stdlib-compatible, lightweight, SSE-friendly |
| Frontend | Svelte 5 + SvelteKit (adapter-static) | ~3 KB bundle, SPA mode, runes for reactivity |
| Markdown | `marked` (client-side, ~5 KB) | Lightweight, no build-time rendering needed |
| CSS | Plain CSS | No framework dependency |
| Inference | `llama-server` (llama.cpp) subprocess | Best CPU inference, OpenAI API built-in |
| Model source | HuggingFace Hub (direct HTTPS) | No SDK needed, public models free |
| Persistence | JSON files | No database, portable, human-readable |
| Embedding | Go `//go:embed` | Bundles SPA into binary at compile time |

---

## 11. Development Phases

### Phase 1: Scaffold (Day 1)
- Go module + main.go
- SvelteKit SPA with adapter-static
- `//go:embed` + dev proxy
- Single binary build target

### Phase 2: Hardware Detection + Models (Day 2)
- RAM/CPU/disk detection
- Curated model list + recommender
- `GET /api/system/info`

### Phase 3: Model Downloader (Day 2-3)
- HuggingFace HTTPS download with progress
- Resumable downloads
- Model list/delete API

### Phase 4: Inference Engine (Day 3-4)
- llama-server subprocess manager
- Health checking
- Auto-download llama-server binary

### Phase 5: Chat API + Streaming (Day 4)
- OpenAI-compatible proxy
- SSE streaming
- Conversation persistence

### Phase 6: UI (Day 5)
- Setup wizard
- Chat page with streaming
- Settings page
- Models page

### Phase 7: Packaging (Day 6)
- Cross-compile targets
- ZIP distribution
- README

---

## 12. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| llama-server subprocess crashes | Medium | High | Restart automatically, log stderr |
| HuggingFace download fails mid-way | Medium | Medium | Resumable downloads, retry logic |
| 4 GB RAM insufficient for model + app | Low | High | Detect and warn before download |
| Windows Defender flags unknown .exe | Medium | Low | Open source, code signing (v2) |
| USB 2.0 too slow for model loading | Medium | Medium | Progress indicator, RAM caching |
| llama-server binary missing for arm64 | Low | Medium | Build from source fallback |

---

## 13. Out of Scope (v1)

- Multi-model parallel serving
- GPU acceleration configuration
- Mobile app / PWA
- Plugin system
- RAG / document Q&A
- System tray icon
- Auto-updater
- Authentication / multi-user
- Cloud sync
- Tool calling / function calling

---

## 14. Success Metrics

| Metric | Target |
|--------|--------|
| Binary size | ≤ 15 MB |
| First-run time-to-chat (1 GB model) | < 5 min (download) + 10 s (load) |
| Subsequent time-to-chat | < 5 s |
| RAM overhead (idle) | < 50 MB |
| Inference speed (4 GB CPU) | ≥ 5 tok/s |
| GitHub stars (3 months) | > 1000 |
| User-reported "worked first time" | > 90% |
