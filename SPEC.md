# SPEC.md — Chess Mistakes SRS

## 1. Purpose

A private, personal web application for chess improvement. The user manually
curates positions from their own games (or elsewhere) representing mistakes
or missed opportunities worth remembering, and reviews them over time using
spaced repetition (SRS), with the goal of internalizing patterns and avoiding
repeat errors.

This is **not** a generic tactics trainer. The value proposition is that the
cards are self-selected and self-authored by the user, based on their own
games and their own judgment of what's worth remembering — not algorithmically
generated or auto-detected.

## 2. Core Philosophy

- **The user is always in control of what gets saved.** No automated
  detection, triggering, or classification of "mistakes" from games. The user
  decides what is worth remembering and manually creates each card.
- **Manual authorship over automation.** Questions, answers, explanations,
  and tags are written by the user. This is deliberate: automated
  classification of tactical motifs or strategic themes is unreliable and
  would undermine trust in the tool.
- **v1 is single-user and 100% private.** No sharing, no community features,
  no public data. This may be revisited in a future version, but the initial
  product must work well as a private tool before any public/community layer
  is considered. Nothing in the architecture should make that future
  direction impossible, but nothing should be built for it prematurely
  either.
- **Simplicity over completeness.** Where a feature could be simple (binary
  grading, single accepted solution, manual FEN entry) or complex (partial
  credit, multi-solution acceptance via engine analysis, game import), v1
  takes the simple path. Complexity is added later once the core loop is
  validated through real usage.

## 3. Card Types

There are exactly three card types. Each has a distinct data shape and
review flow. New "categories" of mistake (opening deviations, endgame
technique, missed defenses, etc.) are **not** new card types — they are
expressed via tags on top of one of these three templates.

### 3.1 Tactical Opportunity

**Scenario:** The opponent made a mistake. There is a tactic available for
the user to find.

- **Position shown:** After the opponent's mistake, user to move.
- **Task:** Find the best move (and, if it's a forced sequence, the
  following forced moves — see "solution as a line" below).
- **Grading:** Binary. Correct or incorrect, checked against a single
  accepted solution (a move or short forced line) authored by the user at
  card-creation time.
- **Board interaction:** Fully interactive — user drags pieces to input
  their answer, illegal moves are rejected by the board itself.

### 3.2 Blunder

**Scenario:** The user played a mistake in one of their own games. The card
should train recognition of *why the intended move fails*, before it's
played — not just "what should I have played instead."

- **Position shown:** The position **before** the user's mistake, user to
  move. The move the user is "about to play" (the actual blunder) is
  annotated on the board (e.g. as an arrow) or shown as text/notation
  alongside the position.
- **Task, in two sequential steps:**
  1. Find the opponent's best refutation — the tactic that punishes the
     annotated move.
  2. After that refutation is revealed, find what the user should have
     played instead of the blunder.
- **Grading:** Binary per step. Both steps are checked against a single
  accepted solution authored at card-creation time. The two steps are
  revealed sequentially (step 2 is not attempted "blind" — the user sees the
  refutation from step 1 before attempting step 2), since the point is
  recognizing the danger, not guessing in the dark.
- **Board interaction:** Fully interactive for both steps.

### 3.3 Strategic Mistake

**Scenario:** A positional or strategic decision — by either player — that
doesn't reduce to a single correct move, but represents a conceptual
mistake worth reflecting on (bad plan, misjudged structure, wrong piece
placement, mishandled opening deviation, etc.).

- **Position shown:** Whatever position the user chooses (typically before
  the strategic decision in question).
- **Task:** A free-form question authored by the user (e.g. "Why is this
  plan wrong? What should be played instead, and why?").
- **Answer:** A free-form explanation authored by the user, shown on reveal.
- **Grading:** Self-graded. The user reveals their own answer and rates
  their own recall (e.g. "remembered" / "didn't remember"), similar to
  Anki's basic reveal-and-rate flow. There is no single objectively correct
  move to check against.
- **Board interaction:** Board is shown for context/reference. No move
  input or validation is required — this is not a move-finding exercise.
- **Future direction (not v1):** Optionally let the user play a move, run
  Stockfish locally, and check whether the evaluation has shifted
  significantly, as a soft check on self-grading. Deferred.

## 4. Solutions

- Each Tactical Opportunity and Blunder card has **exactly one authored
  accepted solution** in v1 (a move or short line). Multi-solution
  acceptance sets (e.g. accepting a second-best-but-still-winning move) and
  engine-assisted move validation are explicitly deferred.
- The user is responsible for deciding what the "correct" answer is when
  authoring the card. If a position genuinely has multiple reasonable
  solutions, the user picks the one they want to be tested on.

## 5. Grading

- **v1 uses binary grading only:** correct / incorrect for Tactical
  Opportunity and Blunder cards; self-assessed remembered / not-remembered
  for Strategic Mistake cards.
- Three-way grading (best / acceptable-but-not-best / wrong) is a
  deliberate future direction, deferred until multi-solution support is
  built. The scheduling/data model should not make this impossible to add
  later, but should not be built for it now.

## 6. Tags

- Free-form tags, authored/selected by the user, with autocomplete against
  previously-used tags to encourage a self-organizing personal vocabulary.
- No automated tag suggestion or classification in v1.
- Tags are used for categorization the user cares about beyond the three
  card types — examples: `opening`, `endgame`, `missed-defense`, `pin`,
  `back-rank`, `IQP`, or anything else the user finds useful. The taxonomy
  is entirely emergent from use, not predefined by the product.
- A card can have any number of tags.

## 7. Sets

- Sets are **freeform, saved collections** of cards — not a strict
  one-card-one-set model.
- A card can belong to any number of sets.
- A set is effectively a saved filter/collection the user can review as a
  unit (e.g. "Sicilian Najdorf mistakes," "Cards from June 2026," "Endgame
  technique").
- Sets are user-created and user-named. No smart/auto-generated sets in v1
  (though tag-based filtering that feels similar to a "smart set" may
  exist as a browsing feature — see Section 9).

## 8. Card Input / Creation

v1 supports **manual creation only**:

- User provides a FEN (or sets up a position on an interactive board) and
  the side to move.
- User selects the card type (Tactical Opportunity / Blunder / Strategic
  Mistake).
- User authors the question context as needed per card type (e.g. the
  intended blunder move, for Blunder cards).
- User authors the accepted solution (move/line) for Tactical Opportunity
  and Blunder cards, or the free-form question/answer for Strategic
  Mistake cards.
- User adds tags and optionally assigns the card to one or more sets.

**Deferred (not v1):** importing directly from a Lichess game URL, with
assisted position/ply selection. This is a natural v2 feature once the
manual flow is validated, but is out of scope now.

## 9. Review Flow

- The user starts a review session, either across all due cards or scoped
  to a specific set or tag filter.
- Cards due for review are presented one at a time, per the SRS scheduling
  algorithm (see Section 10).
- Each card type has its own review UI flow as described in Section 3.
- After grading (binary or self-assessed), the SRS algorithm updates the
  card's next-due date and scheduling state.

## 10. Spaced Repetition Algorithm

- **v1 uses SM-2** (the classic SuperMemo 2 algorithm), applied per-card
  regardless of card type, using binary/self-assessed grading as the input
  quality signal.
- SM-2 is chosen deliberately over more modern alternatives (e.g. FSRS) for
  v1 because it is simpler to implement correctly, and the priority is
  validating the product loop, not optimizing scheduling accuracy at low
  card volumes.
- **Future direction (not v1):** migrating to FSRS once card/review volume
  is large enough for it to provide real benefit over SM-2. The scheduling
  state should be isolated (see ARCHITECTURE.md) so this migration doesn't
  require reworking unrelated parts of the system.

## 11. Explicit Non-Goals for v1

- No community features: no public cards, no sharing, no voting, no
  discussion/commentary sections, no collaborative tagging.
- No automated mistake detection or classification from imported games.
- No Lichess/chess.com game import.
- No multi-solution move acceptance or engine-assisted validation.
- No three-way/partial-credit grading.
- No time-management or non-position-based journaling features.
- No mobile app (web-only, though the web app may be usable on mobile
  browsers as a secondary concern, not a design target).

## 12. Future Directions (explicitly out of scope, noted for context)

These are documented so future work doesn't have to be re-derived from
scratch, but none of them should influence v1 implementation decisions
beyond "don't make this impossible later":

- Community/public card sharing, with anonymization on publish (strip
  username, game link, and identifying metadata) and a fork-not-edit model
  for community corrections to annotations (never silently overwrite a
  user's authored explanation).
- Community tag governance via voting with thresholds.
- Discussion/commentary sections per position, especially valuable for
  Strategic Mistake cards.
- Lichess game URL import with assisted ply/position selection.
- Multi-solution acceptance sets, populated via local Stockfish analysis
  with user confirmation of which engine-suggested moves count as correct.
- Three-way grading (best / acceptable / wrong) feeding into SRS scheduling.
- FSRS as the scheduling algorithm.
- Endgame-technique and opening-deviation as potentially distinct card
  templates, if tag-based handling proves insufficient in practice.
