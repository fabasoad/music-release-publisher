# Music Release Publisher

[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)
![unit-tests](https://github.com/fabasoad/music-release-publisher/actions/workflows/unit-tests.yml/badge.svg)
![security](https://github.com/fabasoad/music-release-publisher/actions/workflows/security.yml/badge.svg)
![linting](https://github.com/fabasoad/music-release-publisher/actions/workflows/linting.yml/badge.svg)

A Go CLI bot that runs daily via GitHub Actions to discover new music releases
using Gemini AI and publish them to multiple social platforms.

## How it works

1. Queries `gemini-2.5-flash` with a structured JSON schema to discover notable
   releases from the past 24 hours across a set of configured genres.
2. Parses the response into typed `MusicRelease` structs.
3. Publishes each release to every configured platform concurrently.

## Project structure

```text
music-release-publisher/
├── .github/workflows/daily-publish.yml  # scheduled cron job
├── cmd/bot/main.go                       # entrypoint
├── internal/
│   ├── ai/curator.go                    # Gemini structured output
│   └── publisher/
│       ├── publisher.go                 # Publisher interface + MusicRelease struct
│       ├── format.go                    # shared message formatter
│       ├── discord.go                   # Discord webhook
│       ├── telegram.go                  # Telegram Bot API
│       └── threads.go                   # Threads (Meta) API
├── go.mod
└── go.sum
```

## Adding a new platform

Implement the `Publisher` interface in a new file under `internal/publisher/`:

```go
type Publisher interface {
    Name() string
    Publish(ctx context.Context, release MusicRelease) error
}
```

Then wire it up with an env-var guard in `buildPublishers()` in `cmd/bot/main.go`.

## Configuration

All configuration is via environment variables. The bot skips any platform whose
vars are absent, so you only need to set the ones you want.

| Variable | Required | Description |
|---|---|---|
| `GEMINI_API_KEY` | Yes | Google AI Studio API key |
| `DISCORD_WEBHOOK_URL` | No | Discord channel webhook URL |
| `TELEGRAM_BOT_TOKEN` | No | Telegram bot token (`123:ABC...`) |
| `TELEGRAM_CHAT_ID` | No | Target Telegram chat/channel ID |
| `THREADS_CLIENT_ID` | No | Meta app client ID |
| `THREADS_CLIENT_SECRET` | No | Meta app client secret |
| `THREADS_REDIRECT_URI` | No | OAuth redirect URI registered in Meta console |
| `THREADS_ACCESS_TOKEN` | No | Pre-obtained long-lived access token |

## Running locally

```sh
export GEMINI_API_KEY=...
export DISCORD_WEBHOOK_URL=...  # optional

go run ./cmd/bot/...
```

## Deployment

The bot runs automatically at **09:00 UTC** daily via the GitHub Actions workflow.
You can also trigger it manually from the Actions tab.

Add the env vars above as repository secrets under **Settings → Secrets and
variables → Actions**.

## Genres

Genres are configured in `internal/ai/curator.go`:

```go
var genres = []string{
    "electronic", "ambient", "post-rock", "indie", "jazz",
}
```

## Tech stack

- **Go** 1.22+
- **AI** — `google.golang.org/genai` (`gemini-2.5-flash`, structured JSON output)
- **Threads** — `github.com/tirthpatell/threads-go`
- **Discord / Telegram** — standard `net/http`
