# LocalAI Hub

Run LLMs locally on any Windows PC — no Docker, no Python, no cloud. One binary, double-click, chat.

**Single 12 MB .exe** → **auto-downloads llama-server + models** → **browser opens** → **chat.**

## How to use

1. Download `LocalAI.exe` (or the NSIS installer)
2. Put it in any folder (desktop, USB drive — paths are relative)
3. Double-click — browser opens to `http://localhost:8080`
4. Pick a model, click Install, wait for download
5. Type a message and press Enter

All data (`models/`, `conversations/`, `config.json`) lives alongside the .exe. Plug the folder onto a USB drive and it works on any PC.

For developers: OpenAI-compatible API at `http://localhost:8080/v1`.

## Build from source

```
git clone https://github.com/YOUR_USER/localai-hub
cd localai-hub/web && npm ci && npm run build
cd .. && go build -o dist/LocalAI.exe .
```

Requires Go 1.26+, Node 22+.
