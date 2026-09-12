import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { logout, type Session, CLOUD_ID } from '../lib/api'

const TITLES: Record<string, { h: string; sub: string }> = {
  dashboard: { h: 'Overview', sub: `Mini-cloud ${CLOUD_ID} · devices and live sessions` },
  enroll: { h: 'Enroll a device', sub: 'Mint a one-time token and flash an image' },
  settings: { h: 'Settings', sub: 'Identity and two-factor' },
  devices: { h: 'Terminal', sub: 'Brokered, end-to-end encrypted shell' },
}

function Icon({ d }: { d: string }) {
  return (
    <svg className="ico" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d={d} />
    </svg>
  )
}
const ICONS = {
  grid: 'M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z',
  chip: 'M9 3v3M15 3v3M9 18v3M15 18v3M3 9h3M3 15h3M18 9h3M18 15h3M6 6h12v12H6z',
  plus: 'M12 5v14M5 12h14',
  gear: 'M12 15a3 3 0 100-6 3 3 0 000 6zM19.4 15a1.6 1.6 0 00.3 1.8l.1.1a2 2 0 11-2.8 2.8l-.1-.1a1.6 1.6 0 00-2.7 1.1V21a2 2 0 11-4 0v-.1A1.6 1.6 0 005 19.4l-.1.1a2 2 0 11-2.8-2.8l.1-.1a1.6 1.6 0 00-1.1-2.7H1a2 2 0 110-4h.1A1.6 1.6 0 002.6 5l-.1-.1a2 2 0 112.8-2.8l.1.1a1.6 1.6 0 001.8.3H9a1.6 1.6 0 001-1.5V1a2 2 0 114 0v.1a1.6 1.6 0 001 1.5 1.6 1.6 0 001.8-.3l.1-.1a2 2 0 112.8 2.8l-.1.1a1.6 1.6 0 00-.3 1.8V9a1.6 1.6 0 001.5 1H23a2 2 0 110 4h-.1a1.6 1.6 0 00-1.5 1z',
}

export default function Shell({ session, onLogout }: { session: Session; onLogout: () => void }) {
  const loc = useLocation()
  const nav = useNavigate()
  const seg = loc.pathname.split('/')[1] || 'dashboard'
  const t = TITLES[seg] ?? TITLES.dashboard
  const initial = (session?.username ?? '?').charAt(0).toUpperCase()

  const doLogout = async () => {
    await logout()
    onLogout()
    nav('/login')
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <div className="mark">m</div>
          <div className="name">
            marshmallows
            <small>TrustSentinel</small>
          </div>
        </div>
        <NavLink to="/dashboard" className={({ isActive }) => 'nav-item' + (isActive ? ' active' : '')}>
          <Icon d={ICONS.grid} /> Overview
        </NavLink>
        <NavLink to="/enroll" className={({ isActive }) => 'nav-item' + (isActive ? ' active' : '')}>
          <Icon d={ICONS.plus} /> Enroll device
        </NavLink>
        <NavLink to="/settings" className={({ isActive }) => 'nav-item' + (isActive ? ' active' : '')}>
          <Icon d={ICONS.gear} /> Settings
        </NavLink>
        <div className="spacer" />
        <div className="foot">
          Noise-encrypted overlay · the hub relays ciphertext only.
        </div>
      </aside>

      <div className="main">
        <header className="topbar">
          <div>
            <h1>{t.h}</h1>
            <div className="sub">{t.sub}</div>
          </div>
          <div className="user">
            <span className="muted mono">{session?.username}</span>
            <div className="avatar">{initial}</div>
            <button className="btn sm" onClick={doLogout}>Sign out</button>
          </div>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
