# SpacedChess Proto

> **An AI-generated prototype of SpacedChess.** The real SpacedChess will be
> built from scratch, by hand, with no code generation — this repo is
> throwaway scaffolding used to explore the idea.

A private, personal web app for storing and reviewing your own chess
mistakes with spaced repetition. See `PITCH.md` for the why, `SPEC.md` for
product behavior, `ARCHITECTURE.md` for technical decisions, and
`DESIGN.md` for the visual direction.

## Run it (Docker Compose)

```sh
docker compose up --build
```

Then open http://localhost:8080, register an account, and create your
first card. Data persists in the `db-data` volume.

Optional configuration via `.env` (see `.env.example`): host port,
Postgres password, `SECURE_COOKIES=true` when serving over HTTPS, and
`ALLOW_REGISTRATION=false` to close self-registration once your account
exists.

## Local development (no Docker)

Three processes:

```sh
go run ./cmd/devdb        # throwaway embedded PostgreSQL on :5433
DATABASE_URL='postgres://postgres:postgres@localhost:5433/spacedchess?sslmode=disable' \
  go run ./cmd/spacedchess   # API on :8080 (migrations run at startup)
cd web && npm install && npm run dev   # Vite dev server on :5173, proxies /api → :8080
```

Open http://localhost:5173. For a production-style single server, build
the frontend (`cd web && npm run build`) and start the API with
`STATIC_DIR=web/dist`.

## Tests

```sh
go test ./...                                  # unit tests (SM-2, chess validation)
go test -tags=integration ./internal/server/   # full API loop against embedded Postgres
```

The integration test downloads PostgreSQL binaries on first run and needs
network access.

## Layout

- `cmd/spacedchess` — server entry point; `cmd/devdb` — dev database helper.
- `internal/srs` — SM-2 scheduling (isolated so the algorithm can be swapped).
- `internal/chessval` — server-side FEN/SAN legality validation.
- `internal/store` — PostgreSQL data access, all queries scoped by user.
- `internal/server` — REST API, session auth, static SPA serving.
- `internal/migrate` — embedded SQL migrations, applied at startup.
- `web` — React SPA (Vite, TypeScript, chessground, chess.js).
