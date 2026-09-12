import { useEffect, useState } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { getSession, type Session } from './lib/api'
import Shell from './components/Shell'
import Dashboard from './pages/Dashboard'
import DeviceTerminal from './pages/Terminal'
import Enroll from './pages/Enroll'
import Settings from './pages/Settings'
import Login from './pages/Login'

export default function App() {
  const [sess, setSess] = useState<Session | undefined>(undefined) // undefined = loading

  const refresh = () => getSession().then(setSess)
  useEffect(() => {
    refresh()
  }, [])

  if (sess === undefined) return null // brief load

  return (
    <Routes>
      <Route path="/login" element={<Login onDone={refresh} />} />
      {!sess ? (
        <Route path="*" element={<Navigate to="/login" replace />} />
      ) : (
        <Route element={<Shell session={sess} onLogout={() => setSess(null)} />}>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/devices/:id" element={<DeviceTerminal />} />
          <Route path="/enroll" element={<Enroll />} />
          <Route path="/settings" element={<Settings session={sess} />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Route>
      )}
    </Routes>
  )
}
