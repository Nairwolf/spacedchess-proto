# ARCHITECTURE.md — Chess Mistakes SRS

This document defines the technical architecture. It should be read
alongside SPEC.md, which defines product behavior. Where this document is
silent on an implementation detail, the implementer (human or AI) should
make a reasonable decision consistent with the stack and conventions below,
rather than introducing new architectural patterns not described here.

## 1. Stack Overview

| Layer          | Choice                                              |
|----------------|------------------------------------------------------|
| Backend        | Go, REST JSON API                                   |
| Database       | PostgreSQL                                          |
| Frontend       | React SPA (JavaScript or TypeScript — TypeScript preferred if low additional overhead) |
| Chess board UI | chessground (Lichess's board library)               |
| Chess rules/move validation (frontend) | chess.js or chessops                |
| Chess rules/move validation (backend)  | A Go chess library (e.g. notnil/chess or corentings/chess) |
| Auth           | Session-based, username/password                    |
| Deployment     | Docker Compose (API container, Postgres container, frontend static build served either by the Go binary or a lightweight web server container) |

### Why these choices (brief rationale, for context — not to be re-litigated by the implementer)

- **React SPA over server-rendered templates:** the review session is a
  stateful, highly interactive UI (drag-and-drop board input, sequential
  reveals, per-card-type flows) that is a poor fit for server-rendered
  pages. A JSON API + SPA also cleanly separates concerns and leaves the
  door open for a future community layer without reworking the backend.
- **chessground:** built by Lichess specifically for this interaction
  pattern (drag input, arrows/annotations for the Blunder card type,
  highlighting), actively maintained.
- **PostgreSQL:** the data is inherently relational (cards, tags, sets,
  review history, users, all with many-to-many or one-to-many
  relationships). `jsonb` columns are used where a field is naturally
  variable-shaped (see Data Model) rather than reaching for a NoSQL store.
- **Session-based auth over JWT:** simpler and harder to get wrong for a
  low-user-count, self-hosted application. No need for refresh-token
  machinery.
- **SM-2 over FSRS:** see SPEC.md Section 10. Scheduling state is isolated
  in its own table specifically so this can be swapped later without
  touching card/content tables.

## 2. Project Structure

Left to the implementer's discretion, using idiomatic Go project
conventions (e.g. a `cmd/`, `internal/`, layout) and a standard React
project layout for the frontend (e.g. via Vite). No further constraints are
imposed here — use best judgment and prioritize consistency and
readability over any specific convention.

## 3. Data Model

This section describes entities and relationships, not exact SQL DDL. The
implementer should design the actual schema (types, indexes, constraints)
consistent with this model.

### 3.1 `users`

- id
- username
- password_hash
- created_at

Single-user or few-users in practice, but modeled as a proper users table
from the start (not hardcoded to one user), since session auth requires it
and it costs nothing extra.

### 3.2 `cards`

Core entity. One row per flashcard.

- id
- user_id (owner)
- card_type (enum: `tactical_opportunity`, `blunder`, `strategic_mistake`)
- fen (the position shown to the user; for `blunder` cards, this is the
  position **before** the mistake)
- side_to_move (derivable from FEN, but may be stored explicitly for
  convenience)
- created_at
- updated_at
- source_note (optional free-text: e.g. link to the source game, date
  played, opponent — purely for the user's own reference; never intended
  to be exposed publicly even in a hypothetical future sharing feature,
  per SPEC.md Section 12)

Type-specific fields are modeled via either separate related tables (one
per card type) or a `jsonb` "details" column, at the implementer's
discretion. Given the three types have meaningfully different shapes, a
recommended approach is:

- A shared `cards` table with the common fields above.
- A `card_details` `jsonb` column (or three separate related tables)
  holding type-specific data:
  - `tactical_opportunity`: `{ accepted_solution: <move or line> }`
  - `blunder`: `{ intended_move: <move>, refutation: <move or line>,
    correct_alternative: <move or line> }`
  - `strategic_mistake`: `{ question: <text>, answer: <text> }`

Either approach (jsonb vs. separate tables) is acceptable; jsonb is
suggested because the fields are simple, queried mostly by ID rather than
by content, and avoids three near-empty join tables. If the implementer
anticipates needing to query/filter on these fields directly, separate
typed tables are preferable — use judgment.

### 3.3 `tags`

- id
- user_id
- name

Tags are scoped per-user (not global) in v1, consistent with the tool being
private and per-user. Uniqueness constraint on (user_id, name).

### 3.4 `card_tags`

Many-to-many join table between `cards` and `tags`.

- card_id
- tag_id

### 3.5 `sets`

- id
- user_id
- name
- created_at

### 3.6 `set_cards`

Many-to-many join table between `sets` and `cards` (a card can belong to
multiple sets, per SPEC.md Section 7).

- set_id
- card_id

### 3.7 `review_state`

SRS scheduling state, **one row per card**, deliberately isolated from the
`cards` table so the algorithm (SM-2 now, potentially FSRS later) can be
swapped without touching card content.

- card_id (unique, one-to-one with cards)
- easiness_factor (SM-2 "EF", starts at 2.5)
- interval_days
- repetitions (consecutive correct reviews)
- due_at (timestamp — when this card is next due)
- last_reviewed_at

### 3.8 `review_log`

Append-only history of individual review attempts, for the user's own
tracking/stats and to support a future algorithm migration (e.g. FSRS
optimization from historical review data, per SPEC.md Section 12).

- id
- card_id
- reviewed_at
- grade (representation depends on card type: binary correct/incorrect for
  tactical_opportunity and blunder; self-assessed remembered/not for
  strategic_mistake — a single normalized field, e.g. an integer or enum,
  should be used so review_log is uniform across card types even though
  the grading UI differs)
- resulting_interval_days (the interval computed by SM-2 as a result of
  this review, for auditability)

## 4. API Surface

REST, JSON request/response bodies, session-cookie authenticated. Exact
route naming/versioning left to the implementer, but should cover at
minimum:

### Auth
- Register (if self-registration is desired for a self-hosted single/few
  user tool — otherwise a seeded/admin-created user is acceptable; use
  judgment)
- Login / logout
- Current user info

### Cards
- Create card (type + FEN + type-specific fields + tags + sets)
- Get card by ID
- List/search cards (filterable by tag, set, card_type)
- Update card
- Delete card

### Tags
- List user's tags (for autocomplete)
- Create/rename/delete tag

### Sets
- List sets
- Create/rename/delete set
- Add/remove card to/from set

### Review
- Get due cards (optionally scoped to a set or tag filter)
- Submit a review grade for a card (updates `review_state` via SM-2 and
  appends to `review_log`)

## 5. Frontend Structure Notes

- The three card types should share a common board component
  (chessground wrapper) but have distinct review-flow components, since
  their interaction patterns differ (see SPEC.md Section 3):
  - Tactical Opportunity: single move/line input, immediate binary result.
  - Blunder: annotated starting position (show intended move as an arrow
    or notation), sequential two-step reveal-then-attempt flow.
  - Strategic Mistake: board shown for reference, free-text question
    display, reveal button, self-grade buttons (no move input/validation).
- Card creation UI should let the user set up a position (via FEN paste or
  interactive board editing — implementer's choice which is built first,
  but FEN paste is the simpler baseline and should exist at minimum) and
  branch into type-specific authoring fields based on selected card type.
- No offline/PWA requirements. Standard authenticated SPA behavior
  (redirect to login when unauthenticated, etc.).

## 6. Deployment

- Docker Compose setup with at minimum:
  - `api` service (Go binary)
  - `db` service (PostgreSQL)
  - Frontend either built into a static bundle served by the Go binary
    directly, or served by a small dedicated static-file/nginx container —
    implementer's choice.
- Configuration via environment variables (DB connection string, session
  secret, etc.), not hardcoded.
- No specific cloud provider target — should run on any standard VPS via
  Docker Compose.

## 7. Explicit Constraints Carried Over From SPEC.md

These are product decisions with architectural implications; restated here
so they aren't accidentally violated during implementation:

- No public/shared data of any kind. All data is scoped to `user_id` and
  never exposed cross-user. Even though this is currently single/few-user,
  proper scoping should exist in the data model and every query, not be
  assumed away.
- No engine (Stockfish) integration in v1. Do not add an engine dependency.
- No external game-import integration (Lichess API, etc.) in v1.
- Grading is binary/self-assessed only — do not build multi-value grading
  UI or scoring logic in v1.
