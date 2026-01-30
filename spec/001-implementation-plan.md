# BobBedrockBridge Implementation Plan

## Goal
Add AWS Bedrock support to Bob Translator by creating a bridge between Bob's plugin system and AWS Bedrock API.

## Analysis

### Option 1: Direct Bob Plugin (Not Recommended)
- Bob plugins run in JavaScriptCore (not Node.js/browser)
- **Problem**: SigV4 signing requires complex crypto operations (HMAC-SHA256, canonical request construction)
- JavaScriptCore environment lacks: AWS SDK, full crypto primitives, environment variable access
- Would need to implement SigV4 signing from scratch in JavaScript - error-prone and hard to maintain

### Option 2: Local Server Bridge (Recommended)
- Create a lightweight local HTTP server (Go)
- Bob plugin sends simple HTTP requests to local server
- Server handles AWS authentication via SDK and forwards to Bedrock
- Returns translation results to plugin (supports streaming)

**Advantages:**
- AWS SDK handles all SigV4 signing automatically
- Easy to test and debug
- Supports credential chain (env vars, AWS config files, IAM roles)
- User-configurable model selection
- Streaming support for real-time translation feedback

## Recommended Architecture

```
┌─────────────────┐   HTTP/SSE    ┌──────────────────┐    AWS SDK    ┌─────────────┐
│  Bob Translator │  ──────────>  │  Local Go Server │  ──────────>  │   Bedrock   │
│    (Plugin)     │  <─streaming─ │   (localhost)    │  <─streaming─ │   (Claude)  │
└─────────────────┘               └──────────────────┘               └─────────────┘
```

## Implementation Components

### 1. Bob Plugin (`bob-plugin-bedrock/`)
- `info.json` - Plugin metadata, user config options
- `main.js` - HTTP client with streaming support via `$http.streamRequest()`
- Uses `query.onStream()` for real-time translation feedback

### 2. Local Server (`server/`)
- Go HTTP server listening on localhost (default port: 18081)
- Endpoint: `POST /translate` (supports streaming via SSE)
- Request body: `{ "text": "...", "from": "en", "to": "zh", "stream": true }`
- Uses AWS SDK for Go v2 to call Bedrock ConverseStream API
- User-configurable model selection

## Detailed Steps

### Step 1: Create Go Server
**Files:**
- `server/main.go` - Entry point, HTTP server setup
- `server/bedrock/client.go` - Bedrock API wrapper with streaming
- `server/config/config.go` - Configuration handling
- `server/go.mod`, `server/go.sum` - Dependencies

**Key functionality:**
- Listen on `localhost:18081`
- Accept POST `/translate` with JSON body
- Support both streaming and non-streaming modes
- Call Bedrock ConverseStream API for streaming responses
- Return SSE format for streaming: `data: {"content": "..."}\n\n`

### Step 2: Create Bob Plugin
**Files:**
- `bob-plugin-bedrock/info.json` - Plugin metadata with config options
- `bob-plugin-bedrock/main.js` - Translation logic with streaming
- `bob-plugin-bedrock/icon.png` - Plugin icon

**Key functionality:**
- Use `$http.streamRequest()` for streaming responses
- Parse SSE chunks and call `query.onStream()` for incremental results
- Call `query.onCompletion()` when translation completes
- User-configurable options in Bob UI

### Step 3: Configuration Support
**Server configuration (via env vars):**
- `AWS_REGION` - Bedrock region (default: us-east-1)
- `SERVER_PORT` - Local server port (default: 18081)

**Plugin configuration (via Bob UI):**
- Server URL (default: http://localhost:18081)
- Model selection dropdown:
  - Claude 3 Haiku (fast, cost-effective)
  - Claude 3 Sonnet (balanced)
  - Claude 3.5 Sonnet (best quality)
  - Claude 3 Opus (most capable)

## Files to Create

```
BobBedrockBridge/
├── spec/
│   └── requirements.md (existing)
├── server/
│   ├── main.go           # HTTP server, routing, SSE streaming
│   ├── go.mod
│   ├── bedrock/
│   │   └── client.go     # Bedrock API wrapper with ConverseStream
│   └── config/
│       └── config.go     # Environment config handling
├── bob-plugin-bedrock/
│   ├── info.json         # Plugin metadata + user options
│   ├── main.js           # Streaming translation logic
│   └── icon.png          # Plugin icon
└── README.md             # Setup and usage instructions
```

## Verification

1. **Server test (non-streaming):**
   ```bash
   cd server && go run .
   curl -X POST http://localhost:18081/translate \
     -H "Content-Type: application/json" \
     -d '{"text":"Hello world","from":"en","to":"zh","model":"anthropic.claude-3-haiku-20240307-v1:0"}'
   ```

2. **Server test (streaming):**
   ```bash
   curl -X POST http://localhost:18081/translate \
     -H "Content-Type: application/json" \
     -d '{"text":"Hello world","from":"en","to":"zh","stream":true,"model":"anthropic.claude-3-haiku-20240307-v1:0"}'
   ```
   Should see SSE events: `data: {"content":"..."}\n\n`

3. **Plugin test:**
   - Package `bob-plugin-bedrock/` folder as `.bobplugin` (zip and rename)
   - Double-click to install in Bob Translator
   - Configure server URL and model in Bob plugin settings
   - Test translation with streaming feedback in Bob UI

## Dependencies

**Go Server:**
- `github.com/aws/aws-sdk-go-v2/config`
- `github.com/aws/aws-sdk-go-v2/service/bedrockruntime`

**Bob Plugin:**
- No external dependencies (uses Bob's built-in `$http`)
