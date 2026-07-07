# DESIGN.md — Visual & UX Direction

This document defines the visual and interaction design direction. It
complements SPEC.md (what the product does) and ARCHITECTURE.md (how it's
built). Where this document is silent, the implementer should extrapolate
from the principles here rather than falling back to generic component-
library defaults.

## 1. Reference Point and Why

The explicit design reference is **Lichess** — not as a moodboard of
colors, but as a point of view: **the board is the product, everything
else is quiet infrastructure around it.** Lichess doesn't look like a
polished consumer SaaS app, and that's deliberate — it looks like a tool
built by people who play chess, for people who play chess. No marketing
gloss, no unnecessary chrome, no decoration that doesn't carry information.
Density and directness read as competence to this audience, not as a lack
of polish.

This matters because the audience here — club and tournament players
serious enough to review their own games with an engine — has the same
taste calibration as Lichess's userbase. A glossy, gradient-heavy,
consumer-app aesthetic would actively work against trust with this
audience; it would read as "built by people who don't really play."

## 2. Design Tokens

### Color

A dark-first palette, since serious chess players overwhelmingly play and
analyze in dark mode (Lichess and Chess.com both default long-session use
to dark boards). Light mode is a secondary, not a first pass.

| Token | Hex | Use |
|---|---|---|
| `bg-base` | `#161512` | App background (matches Lichess's own near-black-brown, not pure black — pure black creates harsh contrast against a bright board) |
| `bg-surface` | `#1e1c19` | Cards, panels, sidebars — one step up from base |
| `bg-raised` | `#26241f` | Modals, dropdowns, hover states |
| `text-primary` | `#e6e3dd` | Primary text — warm off-white, not stark white |
| `text-muted` | `#8b877e` | Secondary text, metadata, timestamps |
| `board-light` | `#f0d9b5` | Light squares (classic wood-tone, immediately legible to any chess player) |
| `board-dark` | `#b58863` | Dark squares |
| `accent-correct` | `#6a9955` | Correct answer / success states — muted green, not neon |
| `accent-incorrect` | `#c0564b` | Incorrect answer / error states — muted brick red, not alarming |
| `accent-focus` | `#7fa650` | Interactive highlight (matches Lichess's signature green — used sparingly, for the one or two things per screen that need it: primary buttons, active nav, move highlights) |

This is not a copy of Lichess's exact hex values, but the same *family*:
warm dark neutrals, a wood-toned board, and a single green accent used
sparingly rather than a rainbow of UI accent colors. Consistency of accent
color (one green, used for both "this is correct" and "this is the active
interactive thing") reduces visual noise rather than assigning a new color
to every UI state.

### Typography

| Role | Face | Notes |
|---|---|---|
| UI / body | **Inter** or **IBM Plex Sans** | Neutral, highly legible at small sizes, works well for dense data (review queues, stats, tag lists) |
| Monospace (moves, FEN, notation) | **IBM Plex Mono** or **JetBrains Mono** | Chess notation (`Qxe2`, `Rxf7+`) and FEN strings should be set in monospace — this is a small but meaningful detail that signals the product understands its own subject matter, the same way Lichess sets moves in a fixed-width, notation-friendly style in its move list |
| Display (rare — landing/marketing context only, not in-app) | A slightly heavier weight of the UI face, not a separate display font | This is a tool, not a magazine; there's no need for a distinct display face inside the app itself. Reserve any typographic personality for a future marketing/landing page, not the working product. |

No serif anywhere. No decorative face. Chess notation and clarity are the
personality — the type system's job is to disappear except where precision
matters (notation, FEN).

### Spacing & Shape

- Compact, information-dense spacing — closer to Lichess's tight layouts
  than to the generous whitespace of a typical consumer SaaS landing page.
  This is a working tool used in daily study sessions, not a marketing
  surface; density signals respect for the user's time.
- Minimal border-radius (2–4px, not the 12–16px "soft SaaS" default).
  Sharp-ish corners read as tool-like and precise, in keeping with the
  chess-notation monospace choice.
- Hairline borders (1px, low-contrast) to separate sections rather than
  heavy drop-shadows or card elevation effects.

## 3. Signature Element

**The board is always the largest, highest-contrast object on any screen
it appears on.** No screen should have UI chrome that competes with the
board for visual weight — no heavy header bars, no decorative sidebars, no
marketing copy alongside an active review session. When the board is on
screen, it is the unambiguous focal point; everything else (question text,
grading buttons, tags) is arranged around it at lower visual weight.

This is the one deliberate, consistent design rule that should be visible
across every screen with a board, and it's the throughline back to the
Lichess reference: the board is the product.

## 4. Layout Concepts

### 4.1 Review Session (the most important screen)

This is where users spend most of their time — it deserves the most
deliberate layout thinking.

```
┌─────────────────────────────────────────────────┐
│  [tag/set filter, subtle]        [3 / 12 due]    │  <- thin, quiet header
├───────────────────────────┬───────────────────────┤
│                           │  BLUNDER                │  <- card type as
│                           │  ─────────              │     a small label,
│                           │                          │     not decoration
│      [ CHESSBOARD ]       │  You're about to play:  │
│      (large, dominant)    │  Qxe2                   │
│                           │                          │
│                           │  Find Black's            │
│                           │  refutation.             │
│                           │                          │
│                           │  [Show refutation]       │
│                           │                          │
├───────────────────────────┴───────────────────────┤
│         Again        Hard        Good              │  <- grading, only
│                                                     │     shown after
└─────────────────────────────────────────────────┘     reveal
```

- Board on the left (or centered, full-width on mobile), taking up the
  majority of screen real estate.
- A narrow, quiet side panel holds: card type label, the annotated move
  (for Blunder cards), the question/prompt, and a reveal/next action.
- Grading controls only appear after an attempt or reveal — never
  competing for attention before the user has actually engaged with the
  position.
- Card-type label (Tactical Opportunity / Blunder / Strategic Mistake) is
  small, quiet, uppercase micro-label — informational, not a badge/pill
  competing visually with the board.
- Progress indicator ("3 / 12 due") is minimal — a number, not a progress
  bar with animation. This is a tool for tracking due cards, not a
  gamified streak mechanic.

### 4.2 Card Library / Browse

A dense, filterable table/list view — closer to a Lichess "my games" list
or a spreadsheet than a grid of visual cards. Each row: a small board
thumbnail (or FEN diagram), card type, tags, set membership, next-due date.
Sortable and filterable by tag, set, and type. No large imagery, no
decorative empty-state illustrations beyond a simple, direct message ("No
cards yet — create your first one" with a clear action, not a mascot or
graphic).

### 4.3 Card Creation

A focused, single-purpose form-like flow: position setup (FEN paste or
board editor) on one side, type-specific authoring fields on the other.
No multi-step wizard for what is fundamentally a short task — one screen,
clearly organized, matching the review session's board-dominant layout
convention.

## 5. Motion

Minimal and functional only:

- Board piece movement: smooth, brief transition (matching chessground's
  own default animation, not layered with additional custom motion).
- Reveal transitions (showing a solution, showing grading buttons): a
  brief fade/slide-in, not a flourish. Motion should communicate state
  change, not entertain.
- No page-load animation sequences, no scroll-triggered reveals, no
  decorative micro-interactions. This is a daily-use tool; motion that's
  delightful once becomes friction on the 200th review session.
- Respect `prefers-reduced-motion`.

## 6. Voice and Copy

- Plain, direct, second-person where addressing the user ("Find the best
  move," not "The user should identify the optimal continuation").
- No gamification language (no "streaks," "XP," "levels" — this is a
  serious training tool, not a habit app skinned in game mechanics). Due
  counts and review history are presented factually, not as a score to
  chase.
- Errors and empty states state what happened and what to do next,
  plainly — no apologetic tone, no jokes.
- Chess terminology is used correctly and without explanation — the
  audience is assumed to know what a "blunder," "refutation," or "FEN" is.
  This is not a beginner-facing product; over-explaining basic chess
  vocabulary would undercut credibility with the target audience the same
  way glossy visuals would.

## 7. Explicit Non-Goals

- No gradient-heavy, warm-cream-and-serif "AI generated landing page" look.
- No large hero illustrations, mascots, or decorative graphics anywhere in
  the working app (a future marketing/landing page is a separate surface
  and may reasonably take more visual liberty, but the product itself
  should not).
- No gamification visual language (badges, streak flames, confetti,
  progress bars with animation) — this cuts against the "serious tool"
  positioning and against the explicit non-goal in SPEC.md of not being a
  generic gamified trainer.
- No light-mode-first design; dark mode is the primary experience.
