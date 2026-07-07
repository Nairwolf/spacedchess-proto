// Typed fetch wrapper for the SpacedChess REST API.

export interface User {
  id: number
  username: string
  created_at: string
}

export type CardType = 'tactical_opportunity' | 'blunder' | 'strategic_mistake'

export interface TacticalDetails {
  solution: string[]
}

export interface BlunderDetails {
  intended_move: string
  refutation: string[]
  correct_alternative: string[]
}

export interface StrategicDetails {
  question: string
  answer: string
}

export type CardDetails = TacticalDetails | BlunderDetails | StrategicDetails

export interface ReviewState {
  easiness_factor: number
  interval_days: number
  repetitions: number
  due_at: string
  last_reviewed_at: string | null
}

export interface Card {
  id: number
  card_type: CardType
  fen: string
  side_to_move: 'w' | 'b'
  details: CardDetails
  source_note: string
  created_at: string
  updated_at: string
  tags: string[]
  set_ids: number[]
  review: ReviewState
}

export interface CardPayload {
  card_type: CardType
  fen: string
  details: CardDetails
  source_note: string
  tags: string[]
  set_ids: number[]
}

export interface Tag {
  id: number
  name: string
  card_count: number
}

export interface CardSet {
  id: number
  name: string
  created_at: string
  card_count: number
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const resp = await fetch(path, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  const data = await resp.json().catch(() => ({}))
  if (!resp.ok) {
    throw new ApiError(resp.status, (data as { error?: string }).error ?? `HTTP ${resp.status}`)
  }
  return data as T
}

export const api = {
  // auth
  me: () => request<User>('GET', '/api/auth/me'),
  login: (username: string, password: string) =>
    request<User>('POST', '/api/auth/login', { username, password }),
  register: (username: string, password: string) =>
    request<User>('POST', '/api/auth/register', { username, password }),
  logout: () => request<{ ok: boolean }>('POST', '/api/auth/logout'),

  // cards
  listCards: (params?: { type?: string; tag?: string; set_id?: number; q?: string }) => {
    const qs = new URLSearchParams()
    if (params?.type) qs.set('type', params.type)
    if (params?.tag) qs.set('tag', params.tag)
    if (params?.set_id) qs.set('set_id', String(params.set_id))
    if (params?.q) qs.set('q', params.q)
    const s = qs.toString()
    return request<Card[]>('GET', '/api/cards' + (s ? `?${s}` : ''))
  },
  getCard: (id: number) => request<Card>('GET', `/api/cards/${id}`),
  createCard: (p: CardPayload) => request<Card>('POST', '/api/cards', p),
  updateCard: (id: number, p: CardPayload) => request<Card>('PUT', `/api/cards/${id}`, p),
  deleteCard: (id: number) => request<{ ok: boolean }>('DELETE', `/api/cards/${id}`),

  // tags
  listTags: () => request<Tag[]>('GET', '/api/tags'),
  renameTag: (id: number, name: string) => request<{ ok: boolean }>('PATCH', `/api/tags/${id}`, { name }),
  deleteTag: (id: number) => request<{ ok: boolean }>('DELETE', `/api/tags/${id}`),

  // sets
  listSets: () => request<CardSet[]>('GET', '/api/sets'),
  createSet: (name: string) => request<CardSet>('POST', '/api/sets', { name }),
  renameSet: (id: number, name: string) => request<{ ok: boolean }>('PATCH', `/api/sets/${id}`, { name }),
  deleteSet: (id: number) => request<{ ok: boolean }>('DELETE', `/api/sets/${id}`),
  addCardToSet: (setId: number, cardId: number) =>
    request<{ ok: boolean }>('PUT', `/api/sets/${setId}/cards/${cardId}`),
  removeCardFromSet: (setId: number, cardId: number) =>
    request<{ ok: boolean }>('DELETE', `/api/sets/${setId}/cards/${cardId}`),

  // review
  dueCards: (params?: { tag?: string; set_id?: number }) => {
    const qs = new URLSearchParams()
    if (params?.tag) qs.set('tag', params.tag)
    if (params?.set_id) qs.set('set_id', String(params.set_id))
    const s = qs.toString()
    return request<Card[]>('GET', '/api/review/due' + (s ? `?${s}` : ''))
  },
  submitReview: (cardId: number, correct: boolean) =>
    request<ReviewState>('POST', `/api/cards/${cardId}/review`, { correct }),
}
