# HANDOVER.md

State of the SpacedChess implementation as of 2026-07-06. This is an
**AI-generated prototype** ("SpacedChess Proto"); the intended product is to
be rebuilt by hand from scratch. The app was
built in one pass from the four brief documents (`SPEC.md`,
`ARCHITECTURE.md`, `DESIGN.md`, `PITCH.md`) and is feature-complete for
v1 as specified. Nothing is committed to git yet.

## What exists

- **Backend (Go)**: REST JSON API per ARCHITECTURE.md §4 — session auth
  (register/login/logout/me), cards CRUD with type/tag/set/text filters,
  tags, sets, set membership, due queue, review submission. Embedded SQL
  migrations run at startup. Server-side chess validation rejects
  illegal FENs/solution lines at card write time.
- **SM-2** in `internal/srs`, isolated per the brief so FSRS can replace
  it later. `review_state` (one row per card) and append-only
  `review_log` tables.
- **Frontend (React/TS/Vite)**: login, library (dense filterable table
  with chessground mini-boards), card editor (FEN paste + solutions
  recorded by playing moves on the board), review session with distinct
  flows for all three card types, sets and tags management pages.
  Hand-written CSS from DESIGN.md tokens; no component library.
- **Deployment**: multi-stage `Dockerfile` (node build → Go build →
  alpine runtime serving the SPA), `docker-compose.yml` (api + postgres),
  `.env.example`. `cmd/devdb` runs an embedded Postgres for dev without
  Docker (an addition beyond the brief, kept as documented convenience).

## How it was verified

- `internal/srs` unit tests: interval sequence 1, 6, 17, 49 for
  consecutive correct answers; failure resets reps/interval; EF floor 1.3;
  recovery after failure.
- `internal/chessval` unit tests: FEN validation, legal/illegal lines.
- `go test -tags=integration ./internal/server/`: spins up an embedded
  Postgres and drives the real HTTP API through the whole core loop —
  register (incl. case-insensitive username conflict), create all three
  card types, reject illegal solutions, filter by type/tag/set, due
  queue, correct and incorrect reviews with exact SM-2 state assertions,
  card update, set membership endpoints, tag rename/delete, cross-user
  isolation (404s), logout, delete.
- **Browser-level** (headless Chromium + puppeteer-core, scripts lived in
  the session scratchpad, not committed): created a tactical card
  entirely through the editor UI (FEN paste, board-recorded `Qxf7#`,
  tag autocomplete), then reviewed it answering wrong (verdict, solution
  reveal, replay controls, SM-2 failure state confirmed via API); played
  a full blunder card through both steps by clicking moves, including a
  7-move refutation line with auto-played replies; strategic reveal +
  self-grade; session-complete tally. Screenshots were checked against
  DESIGN.md.
- **Docker Compose path** (verified 2026-07-07, Docker 29 / Compose 5):
  `docker compose up --build` with default env — image builds, Postgres
  comes up healthy, migrations apply, and the containerized API was
  driven end-to-end with curl: register/login/me, tactical card create
  (server-side validation active), due queue, correct review with the
  expected SM-2 transition (EF 2.5→2.6, interval 1, reps 1), JS assets
  served with correct MIME type, SPA fallback on client routes.

## Decisions made where the brief was ambiguous

Recorded so they aren't accidentally relitigated; also summarized in
CLAUDE.md where they affect day-to-day work.

1. **Grading buttons**: DESIGN.md §4.1's mockup shows "Again/Hard/Good";
   SPEC §5 and ARCHITECTURE §7 mandate binary grading. SPEC wins —
   tactical/blunder auto-grade correct/incorrect; strategic self-grades
   "Didn't remember / Remembered".
2. **SM-2 quality mapping**: correct → q=5, incorrect → q=2. Failure
   resets repetitions and schedules +1 day; EF updates on every review
   (so it can recover), floored at 1.3. New cards due immediately.
3. **Blunder = one SRS grade**: correct only if both steps correct. A
   failed step 1 reveals the refutation and still proceeds to step 2.
4. **jsonb `details` column** (ARCHITECTURE's suggested default) rather
   than per-type tables; server re-marshals from typed structs so only
   known fields persist.
5. **Line encoding**: SAN arrays including opponent replies; user plays
   even indices. Blunder refutation is relative to the position after
   the intended move.
6. **No intra-session re-queue** of failed cards (SPEC silent; classic
   SM-2 repeats same-day). Failed cards return the next day.
7. **Open registration** with `ALLOW_REGISTRATION=false` escape hatch.
8. **Dark mode only** — DESIGN.md calls light mode "secondary, not a
   first pass".
9. **Position setup is FEN paste** (ARCHITECTURE's required baseline) +
   board preview; solutions are recorded on the board. No piece-placement
   editor yet.
10. Editor enforces `correct_alternative[0] != intended_move`;
    usernames/tags/sets unique case-insensitively; `review_log.grade`
    is a 0/1 smallint.

## Known gaps / next steps (none are SPEC §11/§12 features)

- Promotion input: the Board component has a promotion picker
  (auto-queen deliberately avoided since solutions may underpromote),
  but no automated test exercises it.
- No unit tests for `validateCard` payload edge cases (malformed jsonb,
  wrong-type details) — covered only indirectly by the integration test.
- Review UX candidates from real usage: keyboard shortcuts (space =
  reveal/next), optional same-day repeat of failed cards, board editor
  for position setup.
- Hardening: no rate limiting on login; sessions expire after 30 days
  with no sliding renewal; `updated_at` is set in SQL rather than a
  trigger.
- Light mode, if ever, should be a second theme on the existing CSS
  custom properties in `web/src/styles.css`.

## One bug worth remembering

chessground's `turnColor` defaults to `'white'`; without setting it,
Black-to-move input is silently rejected (selection works, moves don't).
`Board.tsx` now always sets it from the FEN. If move input ever "stops
working" for one side, check this first.
