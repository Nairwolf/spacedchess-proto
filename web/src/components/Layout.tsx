// App chrome for non-review screens: a thin, quiet header. The review
// session has its own even quieter header (DESIGN.md §4.1).
import { useEffect, useState } from 'react'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../auth'

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [dueCount, setDueCount] = useState<number | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .dueCards()
      .then((cards) => {
        if (!cancelled) setDueCount(cards.length)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [location])

  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="topbar__brand">
          SpacedChess
        </Link>
        <nav className="topbar__nav">
          <NavLink to="/" end>
            Library
          </NavLink>
          <NavLink to="/new">New card</NavLink>
          <NavLink to="/sets">Sets</NavLink>
          <NavLink to="/tags">Tags</NavLink>
        </nav>
        <div className="topbar__right">
          {dueCount !== null && dueCount > 0 ? (
            <Link to="/review" className="btn btn--primary btn--sm">
              Review {dueCount} due
            </Link>
          ) : (
            <span className="topbar__nodue">Nothing due</span>
          )}
          <span className="topbar__user" title={user?.username}>
            {user?.username}
          </span>
          <button
            type="button"
            className="btn btn--ghost btn--sm"
            onClick={async () => {
              await logout()
              navigate('/login')
            }}
          >
            Log out
          </button>
        </div>
      </header>
      <main className="app__main">
        <Outlet />
      </main>
    </div>
  )
}
