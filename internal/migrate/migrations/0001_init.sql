-- Initial schema for SpacedChess (ARCHITECTURE.md §3).

CREATE TABLE users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_username_key ON users (lower(username));

-- Server-side session store for cookie-based auth.
CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE cards (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    card_type    TEXT NOT NULL CHECK (card_type IN ('tactical_opportunity', 'blunder', 'strategic_mistake')),
    fen          TEXT NOT NULL,
    side_to_move CHAR(1) NOT NULL CHECK (side_to_move IN ('w', 'b')),
    -- Type-specific fields (jsonb per ARCHITECTURE.md §3.2 recommendation):
    --   tactical_opportunity: { "solution": ["Nf3", ...] }          (SAN line from fen)
    --   blunder:              { "intended_move": "Qxe2",
    --                           "refutation": ["Rxe2", ...],         (SAN line from fen+intended_move)
    --                           "correct_alternative": ["Nf3", ...] }(SAN line from fen)
    --   strategic_mistake:    { "question": "...", "answer": "..." }
    details      JSONB NOT NULL,
    source_note  TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cards_user_id_idx ON cards (user_id);

CREATE TABLE tags (
    id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name    TEXT NOT NULL CHECK (name <> '')
);

CREATE UNIQUE INDEX tags_user_name_key ON tags (user_id, lower(name));

CREATE TABLE card_tags (
    card_id BIGINT NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    tag_id  BIGINT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, tag_id)
);

CREATE INDEX card_tags_tag_id_idx ON card_tags (tag_id);

CREATE TABLE sets (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL CHECK (name <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX sets_user_name_key ON sets (user_id, lower(name));

CREATE TABLE set_cards (
    set_id  BIGINT NOT NULL REFERENCES sets (id) ON DELETE CASCADE,
    card_id BIGINT NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    PRIMARY KEY (set_id, card_id)
);

CREATE INDEX set_cards_card_id_idx ON set_cards (card_id);

-- SM-2 scheduling state, isolated from card content (ARCHITECTURE.md §3.7).
CREATE TABLE review_state (
    card_id          BIGINT PRIMARY KEY REFERENCES cards (id) ON DELETE CASCADE,
    easiness_factor  DOUBLE PRECISION NOT NULL DEFAULT 2.5,
    interval_days    INTEGER NOT NULL DEFAULT 0,
    repetitions      INTEGER NOT NULL DEFAULT 0,
    due_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_reviewed_at TIMESTAMPTZ
);

CREATE INDEX review_state_due_at_idx ON review_state (due_at);

-- Append-only review history (ARCHITECTURE.md §3.8).
CREATE TABLE review_log (
    id                      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    card_id                 BIGINT NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    reviewed_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Normalized binary grade across all card types:
    -- 1 = correct / remembered, 0 = incorrect / didn't remember.
    grade                   SMALLINT NOT NULL CHECK (grade IN (0, 1)),
    resulting_interval_days INTEGER NOT NULL
);

CREATE INDEX review_log_card_id_idx ON review_log (card_id);
