# PalBot (Matrix + Go)

PalBot is a Matrix room bot that controls a Palworld Docker container on the same Unraid host.

It supports:
- `!startpal`
- `!stoppal` with fail-safe RCON player check

The bot is locked to one Matrix room ID and an allowlist of sender MXIDs.

## Features

- Matrix client built with `mautrix-go`
- Docker control via official Docker Go SDK
- Minimal Minecraft-style RCON implementation in Go for `ShowPlayers`
- Fail-safe stop behavior:
  - already stopped/running checks
  - stop blocked when players are online
  - stop blocked when RCON check fails
- Busy lock for command serialization (`busy, try again`)
- Sync token persisted to disk to avoid replaying old messages on restart
- Graceful shutdown on `SIGINT`/`SIGTERM`

## Configuration

All config is environment-based:

- `MATRIX_HOMESERVER` (required)
- `MATRIX_ACCESS_TOKEN` (preferred)
- `MATRIX_USER` + `MATRIX_PASSWORD` (fallback login if token not provided)
- `MATRIX_USER_ID` (recommended, e.g. `@palbot:matrix.pikipika.com`)
- `MATRIX_ROOM_ID` (required, exact room ID, e.g. `!abcdef:matrix.pikipika.com`)
- `ALLOWED_MXIDS` (required, comma-separated MXIDs)
- `DOCKER_CONTAINER_NAME` (default: `Palworld`)
- `RCON_HOST` (default: `127.0.0.1`)
- `RCON_PORT` (default: `25575`)
- `RCON_PASS` (required)
- `COMMAND_PREFIX` (default: `!`)
- `DATA_DIR` (default: `./data`, use `/data` in Docker)
- `LOG_LEVEL` (`debug`, `info`, `warn`, `error`; default `info`)

See `.env.example`.

## Matrix Setup

1. Create a dedicated bot user in your Matrix homeserver.
2. Invite the bot to the target room.
3. Ensure the target room is unencrypted.
4. Collect:
   - homeserver URL (`MATRIX_HOMESERVER`)
   - room ID (`MATRIX_ROOM_ID`) from room settings (Advanced/Internal room ID)
   - bot access token (`MATRIX_ACCESS_TOKEN`, preferred)

### Getting an access token (preferred)

Example login request:

```bash
curl -sS -X POST "$MATRIX_HOMESERVER/_matrix/client/v3/login" \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "m.login.password",
    "identifier": {"type": "m.id.user", "user": "@palbot:matrix.pikipika.com"},
    "password": "YOUR_PASSWORD"
  }'
```

Use the returned `access_token` as `MATRIX_ACCESS_TOKEN`.

If you omit `MATRIX_ACCESS_TOKEN`, the bot logs in using `MATRIX_USER` + `MATRIX_PASSWORD` and stores the token at `/data/matrix_access.token` (or `DATA_DIR/matrix_access.token`).

## Run Locally

```bash
cp .env.example .env
# edit .env
make test
make run
```

## Build and Run in Docker

```bash
docker build -t palbot .
docker run --rm -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/data:/data \
  --env-file .env \
  palbot
```

## docker-compose (Unraid style)

`docker-compose.yml` is included:

```bash
docker compose up -d --build
```

It mounts:
- `/var/run/docker.sock:/var/run/docker.sock`
- `./data:/data`

## Command Behavior

- `!startpal`
  - If running: replies `server is already running`
  - Else: starts container and replies `starting Palworld server...`

- `!stoppal`
  1. If container not running: replies `server is already stopped`
  2. Runs RCON `ShowPlayers` (5s timeout)
     - if players found: aborts and lists names
     - if RCON fails: aborts (`refused to stop: could not confirm zero players via RCON`)
  3. Stops container only when zero players are confirmed

## Security Notes

- Bot only processes events in `MATRIX_ROOM_ID`.
- Bot only accepts commands from `ALLOWED_MXIDS`.
- Bot ignores its own messages.
- Sync token is persisted (`/data/sync.token`) to avoid replaying old events on restart.
- Access token is never logged and stored as a local secret file when login fallback is used.
- Container only needs:
  - Docker socket mount
  - network reachability to `RCON_HOST:RCON_PORT`

## Development

```bash
make test
make build
```

Project layout:
- `cmd/palbot/main.go`
- `internal/config`
- `internal/commands`
- `internal/dockerctl`
- `internal/rcon`
- `internal/matrix`
