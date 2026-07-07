# SpacedChess

Most players analyze their games with an engine after the fact. The engine
finds the mistakes — a missed tactic, a blunder, a wrong plan — but it
doesn't help you remember them. Two months later, the same pattern shows
up in a different game, and you miss it again.

SpacedChess is a tool for storing and reviewing your own chess mistakes
over time, using spaced repetition.

After analyzing a game, you decide which positions are worth remembering —
not every inaccuracy, just the ones you judge to be genuinely instructive.
You save the position and write your own question and answer for it. There
are three kinds of cards:

- **Tactical opportunity** — your opponent made a mistake; the card asks
  you to find the winning continuation.
- **Blunder** — you made the mistake; the card shows the position before
  you played it, marks the move you're about to make, and asks you to find
  your opponent's refutation, then what you should have played instead.
- **Strategic mistake** — a positional or planning error, by either side,
  with a question and explanation you write yourself, since these usually
  don't reduce to a single correct move.

Cards are reviewed on a spaced schedule, like Anki: shown again just
before you'd likely forget them, less often as you keep getting them
right.

There's no automatic detection of what counts as a mistake, and no generic
puzzle database. The cards only contain what you chose to save, from your
own games. The tool assumes you're the best judge of what's worth
remembering — it just makes sure you actually do.
