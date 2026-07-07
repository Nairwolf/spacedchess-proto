// Shared text helpers for presenting cards.
import type { BlunderDetails, Card, CardType, StrategicDetails, TacticalDetails } from './api'

export function typeLabel(t: CardType): string {
  switch (t) {
    case 'tactical_opportunity':
      return 'Tactical opportunity'
    case 'blunder':
      return 'Blunder'
    case 'strategic_mistake':
      return 'Strategic mistake'
  }
}

export function sideName(side: 'w' | 'b'): string {
  return side === 'w' ? 'White' : 'Black'
}

/** One-line summary for the library table. */
export function cardPromptSummary(card: Card): string {
  switch (card.card_type) {
    case 'tactical_opportunity': {
      const d = card.details as TacticalDetails
      return d.solution.join(' ')
    }
    case 'blunder': {
      const d = card.details as BlunderDetails
      return `${d.intended_move}? — ${d.refutation.join(' ')}`
    }
    case 'strategic_mistake': {
      const d = card.details as StrategicDetails
      return d.question.length > 80 ? d.question.slice(0, 77) + '…' : d.question
    }
  }
}
