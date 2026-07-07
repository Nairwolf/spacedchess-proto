# CLAUDE.md

Guidance for working in this repository.

## What this is

SpacedChess: a private, single-user-ish web app for reviewing one's own
chess mistakes with spaced repetition. Four documents are the product
brief and are **binding constraints, not suggestions**:

- `SPEC.md` — product behavior (three card types, binary grading, SM-2).
  Section 11 lists explicit non-goals; Section 12 lists deferred features.
  Do not implement anything from either (no engine, no game import, no
  multi-solution grading, no community features).
- `ARCHITECTURE.md` — stack and data model. Don't introduce new
  architectural patterns not described there.
- `DESIGN.md` — visual direction. Signature rule: **the board is always
  the largest, highest-contrast element on any screen it appears on.**
  Dark-first, one green accent, monospace for chess notation/FEN only,
  no gamification visuals, no UI component libraries.
- `PITCH.md` — the product's purpose, source for any landing copy.

`HANDOVER.md` records implementation decisions already made where the
brief was ambiguous — follow them for consistency.

## Commands

```sh
go build ./... && go vet ./...                 # backend build
go test ./...                                  # unit tests (SM-2, chessval)
go test -tags=integration ./internal/server/   # full API test vs embedded Postgres (needs network on first run)
cd web && npm run build                        # frontend typecheck + build
cd web && npx oxlint src                       # frontend lint
```

Local dev (no Docker):

```sh
go run ./cmd/devdb        # embedded Postgres on :5433 (blocks; Ctrl+C stops)
DATABASE_URL='postgres://postgres:postgres@localhost:5433/spacedchess?sslmode=disable' go run ./cmd/spacedchess
cd web && npm run dev     # :5173, proxies /api → :8080
```

Deployment: `docker compose up --build` (single API container serves the
built SPA from `STATIC_DIR`; Postgres container; see `docker-compose.yml`).

## Layout

- `internal/srs` — SM-2 only. Deliberately isolated (future FSRS swap);
  keep scheduling logic out of handlers and store.
- `internal/chessval` — server-side FEN/SAN legality (notnil/chess). All
  card writes go through `validateCard` in `internal/server/cards.go`.
- `internal/store` — pgx data access. **Every query is scoped by
  user_id**; never add a query that isn't.
- `internal/server` — stdlib `http.ServeMux` routes (Go 1.22 method
  patterns), session-cookie auth, JSON helpers, SPA static fallback.
- `internal/migrate/migrations/*.sql` — embedded, applied in filename
  order at startup; add new files, never edit applied ones.
- `web/src` — React SPA. `components/Board.tsx` is the single chessground
  wrapper used everywhere (review, editor, library thumbnails via
  `mini`). `chessUtil.ts` wraps chess.js. Pages under `pages/`.
  All styling is hand-written in `styles.css` from DESIGN.md tokens.

## Conventions & gotchas

- Card type-specific fields live in a `details` jsonb column; shapes are
  the typed structs in `internal/server/cards.go` (Go) and `web/src/api.ts`
  (TS). Keep both in sync.
- Solution lines are SAN arrays *including* the opponent's forced
  replies; the reviewer plays even indices, odd indices auto-play.
  Blunder `refutation` is relative to the position *after*
  `intended_move`; `correct_alternative` is from the card's FEN.
- Grades are binary everywhere: `review_log.grade` is 0/1; the review
  endpoint takes `{"correct": bool}`. Do not add multi-value grading.
- chessground: always set `turnColor` alongside `fen` — it defaults to
  white and silently rejects Black-to-move input otherwise (this was a
  real bug once).
- SAN comparison uses `sameSan()` (strips `+`/`#` suffixes); don't
  compare raw strings.
- Session tokens are random, stored server-side in the `sessions` table;
  cookie name `spacedchess_session`. No JWT.
- Uniqueness for usernames/tags/sets is case-insensitive
  (`lower(...)` unique indexes); tag upsert relies on
  `ON CONFLICT (user_id, lower(name))`.
- Voice in UI copy: plain, direct, second person, correct chess
  terminology, no gamification language (see DESIGN.md §6).
