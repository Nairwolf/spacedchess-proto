// Card library: a dense, filterable table (DESIGN.md §4.2).
import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import type { Card, CardSet, CardType, Tag } from '../api'
import Board from '../components/Board'
import { cardPromptSummary, typeLabel } from '../cardText'

function dueText(dueAt: string): { text: string; dueNow: boolean } {
  const due = new Date(dueAt)
  const now = new Date()
  if (due <= now) return { text: 'due now', dueNow: true }
  const days = Math.ceil((due.getTime() - now.getTime()) / (24 * 3600 * 1000))
  return { text: days === 1 ? 'in 1 day' : `in ${days} days`, dueNow: false }
}

export default function Library() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const [cards, setCards] = useState<Card[] | null>(null)
  const [tags, setTags] = useState<Tag[]>([])
  const [sets, setSets] = useState<CardSet[]>([])
  const [search, setSearch] = useState(params.get('q') ?? '')
  const [error, setError] = useState('')

  const type = params.get('type') ?? ''
  const tag = params.get('tag') ?? ''
  const setID = params.get('set_id') ?? ''

  useEffect(() => {
    api.listTags().then(setTags).catch(() => {})
    api.listSets().then(setSets).catch(() => {})
  }, [])

  useEffect(() => {
    let cancelled = false
    api
      .listCards({
        type: type || undefined,
        tag: tag || undefined,
        set_id: setID ? Number(setID) : undefined,
        q: params.get('q') ?? undefined,
      })
      .then((c) => !cancelled && setCards(c))
      .catch((e) => !cancelled && setError(e.message))
    return () => {
      cancelled = true
    }
  }, [type, tag, setID, params])

  const setFilter = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  const setsById = useMemo(() => new Map(sets.map((s) => [s.id, s.name])), [sets])
  const filtered = Boolean(type || tag || setID || params.get('q'))

  const reviewHref = useMemo(() => {
    const qs = new URLSearchParams()
    if (setID) qs.set('set_id', setID)
    if (tag) qs.set('tag', tag)
    const s = qs.toString()
    return '/review' + (s ? `?${s}` : '')
  }, [setID, tag])

  return (
    <div className="library">
      <div className="library__toolbar">
        <select value={type} onChange={(e) => setFilter('type', e.target.value)}>
          <option value="">All types</option>
          <option value="tactical_opportunity">Tactical opportunity</option>
          <option value="blunder">Blunder</option>
          <option value="strategic_mistake">Strategic mistake</option>
        </select>
        <select value={tag} onChange={(e) => setFilter('tag', e.target.value)}>
          <option value="">All tags</option>
          {tags.map((t) => (
            <option key={t.id} value={t.name}>
              {t.name} ({t.card_count})
            </option>
          ))}
        </select>
        <select value={setID} onChange={(e) => setFilter('set_id', e.target.value)}>
          <option value="">All sets</option>
          {sets.map((s) => (
            <option key={s.id} value={String(s.id)}>
              {s.name} ({s.card_count})
            </option>
          ))}
        </select>
        <input
          type="search"
          placeholder="Search notes…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && setFilter('q', search)}
          onBlur={() => setFilter('q', search)}
        />
        {(tag || setID) && (
          <Link to={reviewHref} className="btn btn--sm">
            Review this selection
          </Link>
        )}
        <span className="library__count">
          {cards === null ? '' : `${cards.length} card${cards.length === 1 ? '' : 's'}`}
        </span>
      </div>

      {error && <p className="form-error">{error}</p>}

      {cards !== null && cards.length === 0 ? (
        <div className="empty-state">
          {filtered ? (
            <p>No cards match these filters.</p>
          ) : (
            <>
              <p>No cards yet — create your first one.</p>
              <Link to="/new" className="btn btn--primary">
                New card
              </Link>
            </>
          )}
        </div>
      ) : (
        cards !== null && (
          <table className="table">
            <thead>
              <tr>
                <th>Position</th>
                <th>Type</th>
                <th>Prompt</th>
                <th>Tags</th>
                <th>Sets</th>
                <th>Due</th>
                <th>Added</th>
              </tr>
            </thead>
            <tbody>
              {cards.map((card) => {
                const due = dueText(card.review.due_at)
                return (
                  <tr key={card.id} onClick={() => navigate(`/cards/${card.id}`)}>
                    <td>
                      <Board
                        fen={card.fen}
                        orientation={card.side_to_move === 'w' ? 'white' : 'black'}
                        mini
                      />
                    </td>
                    <td>
                      <span className="type-label">{typeLabel(card.card_type as CardType)}</span>
                    </td>
                    <td className={card.card_type === 'strategic_mistake' ? undefined : 'mono'}>
                      {cardPromptSummary(card)}
                    </td>
                    <td>
                      {card.tags.map((t) => (
                        <span key={t} className="tag-chip">
                          {t}
                        </span>
                      ))}
                    </td>
                    <td>
                      {card.set_ids
                        .map((id) => setsById.get(id))
                        .filter(Boolean)
                        .map((name) => (
                          <span key={name} className="tag-chip">
                            {name}
                          </span>
                        ))}
                    </td>
                    <td className={due.dueNow ? 'due-now' : undefined}>{due.text}</td>
                    <td>{new Date(card.created_at).toLocaleDateString()}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )
      )}
    </div>
  )
}
