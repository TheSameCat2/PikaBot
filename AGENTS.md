# AGENTS.md

## Project Preferences (User)

- Repository owner and image namespace: `TheSameCat2` / `ghcr.io/thesamecat2/palbot`.
- Git identity for local + global use:
  - `user.name=TheSameCat2`
  - `user.email=thesamecat@proton.me`
- Workflow preference for this repo:
  - Commit and push changes without prompting.
- Deployment target:
  - Unraid, typically via Portainer stacks.

## CI/CD + Image Publishing

- CI/CD is implemented with GitHub Actions in `.github/workflows/ci-cd.yml`.
- Images are published to GHCR at:
  - `ghcr.io/thesamecat2/palbot`
- Expected tags include:
  - `main`, `latest` (default branch), `sha-*`, and version tags (`v*`).

## Unraid/Portainer Deployment Defaults

- Preferred server compose file: `docker-compose.server.yml`.
- Unraid-optimized defaults:
  - Data bind: `/mnt/user/appdata/palbot/data:/data`
  - Docker socket bind: `/var/run/docker.sock:/var/run/docker.sock`
  - `host.docker.internal` mapped with `host-gateway` for host-local access.
- Typical env var overrides in Portainer:
  - `PALBOT_IMAGE=ghcr.io/thesamecat2/palbot:main`
  - `PALBOT_DATA_PATH=/mnt/user/appdata/palbot/data`

## Tooling Learnings

- Docker daemon access requires the active shell user to be in group `docker`.
  - Symptom when missing: `permission denied while trying to connect to the docker API at unix:///var/run/docker.sock`.
  - Fix: add user to `docker` group and refresh session (`newgrp docker` or re-login).
- This environment originally lacked compose plugin/binary; `docker compose` is now available.
- Compose config validation requires an env file present when `env_file: .env` is used.
  - Fast local validation path: copy `.env.example` to `.env` first.

## Repository Hygiene

- Do not commit `.env` (contains deployment/runtime secrets).
- Keep server-safe defaults in `.env.example`, `docker-compose.server.yml`, and `README.md` synchronized.
- Keep CI workflow image path aligned with runtime deployment defaults.
