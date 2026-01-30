# Bob Translator Bedrock Bridge

A bridge that connects [Bob Translator](https://bobtranslate.com/) to AWS Bedrock Claude models for high-quality AI-powered translations.

Bob plugins run in a restricted JavaScriptCore environment without access to AWS SDK or proper crypto primitives for SigV4 signing. This project solves that limitation by providing a lightweight local Go server that handles AWS authentication and forwards translation requests to Bedrock.

## Architecture

```
+------------------+   HTTP/SSE    +-------------------+   AWS SDK    +--------------+
|  Bob Translator  | -----------> |  Local Go Server  | -----------> |   Bedrock    |
|    (Plugin)      | <-streaming- |   (localhost)     | <-streaming- |   (Claude)   |
+------------------+              +-------------------+              +--------------+
        |                                 |
        |  Plugin Options:                |  Environment:
        |  - Server URL                   |  - AWS_REGION
        |  - Model Selection              |  - SERVER_PORT
        |                                 |  - AWS credentials
```

## Prerequisites

- AWS credentials configured via one of:
  - AWS SSO (`aws sso login --profile your-profile`)
  - Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
  - AWS credentials file (`~/.aws/credentials`)
  - IAM role (when running on AWS infrastructure)
- [Bob Translator](https://bobtranslate.com/) installed (macOS)
- Access to AWS Bedrock Claude models in your AWS account

## Setup Instructions

### Option A: Homebrew (Recommended)

```bash
# Install
brew install --HEAD wd/bob-bedrock/bob-bedrock-bridge

# Configure AWS profile
vim ~/.config/bob-bedrock-bridge/config
# Set: AWS_PROFILE=your-profile

# If using AWS SSO
aws sso login --profile your-profile

# Start as background service
brew services start bob-bedrock-bridge

# Verify
curl http://localhost:18081/health
```

Service management:
```bash
brew services info bob-bedrock-bridge    # Check status
brew services restart bob-bedrock-bridge # Restart after config change
brew services stop bob-bedrock-bridge    # Stop service
```

Logs location: `/opt/homebrew/var/log/bob-bedrock-bridge/`

### Option B: Manual Installation

Requires Go 1.21 or later.

```bash
git clone https://github.com/wd/bob-plugin-bedrock.git
cd bob-plugin-bedrock/server
go build -ldflags="-s -w" -o bob-bedrock-bridge .
./bob-bedrock-bridge
```

The server will start on `localhost:18081` by default.

### Install the Bob Plugin

Option A: Build from source

```bash
cd bob-plugin-bedrock
zip -r bedrock-translator.bobplugin info.json main.js
# Double-click bedrock-translator.bobplugin to install
```

Option B: Download from releases (if available)

1. Download the `.bobplugin` file
2. Double-click to install in Bob Translator

### Configure Plugin Settings in Bob

1. Open Bob Translator preferences
2. Go to Plugins/Services
3. Find "Bedrock Translator" and click the settings icon
4. Configure:
   - **Server URL**: `http://localhost:18081` (default)
   - **Model**: Select your preferred Claude model

## Configuration

### Homebrew Config File

Location: `~/.config/bob-bedrock-bridge/config`

```bash
# AWS Profile (matches ~/.aws/config)
AWS_PROFILE=your-profile

# AWS Region for Bedrock
AWS_REGION=us-east-1

# Server port
SERVER_PORT=18081
```

After editing, restart the service:
```bash
brew services restart bob-bedrock-bridge
```

### Environment Variables (Manual Installation)

| Variable      | Default     | Description                      |
|---------------|-------------|----------------------------------|
| `AWS_REGION`  | `us-east-1` | AWS region for Bedrock API calls |
| `SERVER_PORT` | `18081`     | Port for the local HTTP server   |
| `AWS_PROFILE` | (none)      | AWS profile name from ~/.aws/config |

Example:

```bash
AWS_PROFILE=my-profile AWS_REGION=us-west-2 ./bob-bedrock-bridge
```

### Plugin Options (in Bob UI)

| Option     | Default                                    | Description                    |
|------------|--------------------------------------------|--------------------------------|
| Server URL | `http://localhost:18081`                   | URL of the local bridge server |
| Model      | `anthropic.claude-3-haiku-20240307-v1:0`   | Claude model for translation   |

## Usage Examples

### Testing the Server with curl

**Health check:**

```bash
curl http://localhost:18081/health
```

**Non-streaming translation:**

```bash
curl -X POST http://localhost:18081/translate \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello world",
    "from": "en",
    "to": "zh-Hans",
    "model": "anthropic.claude-3-haiku-20240307-v1:0"
  }'
```

**Streaming translation:**

```bash
curl -X POST http://localhost:18081/translate \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello world",
    "from": "en",
    "to": "zh-Hans",
    "stream": true,
    "model": "anthropic.claude-3-haiku-20240307-v1:0"
  }'
```

Streaming output format (Server-Sent Events):

```
data: {"content":"..."}

data: {"content":"..."}

data: [DONE]
```

### Using in Bob Translator

1. Ensure the bridge server is running
2. Select text in any application
3. Trigger Bob translation (default: double-tap Option key)
4. Select "Bedrock Translator" as the translation service
5. Translation results will stream in real-time

## Supported Models

| Model                    | Model ID                                    | Description                              |
|--------------------------|---------------------------------------------|------------------------------------------|
| Claude 3 Haiku           | `anthropic.claude-3-haiku-20240307-v1:0`    | Fastest, most cost-effective             |
| Claude 3 Sonnet          | `anthropic.claude-3-sonnet-20240229-v1:0`   | Balanced speed and quality               |
| Claude 3.5 Sonnet        | `anthropic.claude-3-5-sonnet-20241022-v2:0` | Best quality for most tasks              |
| Claude 3 Opus            | `anthropic.claude-3-opus-20240229-v1:0`     | Most capable, complex translations       |

Note: Model availability depends on your AWS account's Bedrock access. You may need to request access to specific models in the AWS Console.

## Supported Languages

The plugin supports translation between 30+ languages including:

- English, Chinese (Simplified/Traditional), Japanese, Korean
- French, German, Spanish, Italian, Portuguese, Russian
- Arabic, Thai, Vietnamese, Indonesian, Malay
- And more (see `main.js` for the full list)

## Troubleshooting

### Server won't start

**Error: "Failed to create Bedrock client"**

- Verify AWS credentials are configured:
  ```bash
  aws sts get-caller-identity
  ```
- Check that `AWS_REGION` is set to a region where Bedrock is available

**Error: "address already in use"**

- Another process is using port 18081
- Change the port: `SERVER_PORT=18082 go run .`
- Update the Server URL in Bob plugin settings accordingly

### Plugin shows "Network request failed"

- Ensure the bridge server is running
- Check that the Server URL in Bob settings matches the running server
- Verify no firewall is blocking localhost connections

### Translation returns error from Bedrock

**AccessDeniedException:**

- Your AWS account may not have access to the selected model
- Request model access in AWS Console: Bedrock > Model access
- Try a different model (Claude 3 Haiku is often enabled by default)

**ThrottlingException:**

- You've exceeded the API rate limits
- Wait a moment and try again
- Consider using a model with higher quotas

### Translations are slow

- Claude 3 Haiku is the fastest model; try selecting it
- Check your network connection to AWS
- The first request may be slower due to cold start

### Server logs show "stream error"

- Long translations may timeout
- Increase `WriteTimeout` in `server/main.go` if needed
- Check AWS Bedrock service health

## Development

### Project Structure

```
bob-plugin-bedrock/
├── server/
│   ├── main.go              # HTTP server entry point
│   ├── go.mod               # Go module dependencies
│   ├── bedrock/
│   │   └── client.go        # Bedrock API wrapper
│   └── config/
│       └── config.go        # Environment configuration
├── Formula/
│   └── bob-bedrock-bridge.rb  # Homebrew formula
├── info.json                # Bob plugin metadata
├── main.js                  # Bob plugin logic
├── spec/
│   └── *.md                 # Design docs
└── README.md
```

### Building for Production

```bash
cd server
go build -ldflags="-s -w" -o bob-bedrock-bridge .
```

## License

MIT
