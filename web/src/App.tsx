import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth } from './auth'
import Layout from './components/Layout'
import Login from './pages/Login'
import Library from './pages/Library'
import CardEditor from './pages/CardEditor'
import Review from './pages/Review'
import Sets from './pages/Sets'
import Tags from './pages/Tags'

function RequireAuth({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) return null
  if (!user) return <Navigate to="/login" state={{ from: location }} replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <Routes>
              <Route element={<Layout />}>
                <Route path="/" element={<Library />} />
                <Route path="/new" element={<CardEditor />} />
                <Route path="/cards/:id" element={<CardEditor />} />
                <Route path="/sets" element={<Sets />} />
                <Route path="/tags" element={<Tags />} />
              </Route>
              <Route path="/review" element={<Review />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </RequireAuth>
        }
      />
    </Routes>
  )
}
