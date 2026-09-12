import { type Session } from '../lib/api'

export default function Settings({ session }: { session: Session }) {
  return (
    <div className="grid cols-2" style={{ alignItems: 'start' }}>
      <div className="card">
        <div className="hd"><h3>Identity</h3></div>
        <div className="bd">
          <p className="muted" style={{ margin: 0 }}>Signed in as</p>
          <p style={{ fontFamily: 'var(--serif)', fontSize: 22, margin: '4px 0 0' }}>{session?.username}</p>
          <div className="hr" />
          <div className="section-title">Security key · WebAuthn</div>
          <p style={{ margin: 0 }}>
            <span className="badge v">● Registered</span> &nbsp;
            <span className="muted">passwordless, phishing-resistant</span>
          </p>
        </div>
      </div>
      <div className="card">
        <div className="hd"><h3>Two-factor · TOTP</h3></div>
        <div className="bd">
          <p style={{ marginTop: 0 }}>
            {session?.totp ? <span className="badge v">● Enabled</span> : <span className="badge">Not set up</span>}
          </p>
          <p className="muted">
            A time-based one-time code is required after your security key at sign-in, as a second
            factor.
          </p>
        </div>
      </div>
    </div>
  )
}
