import { useState, type FormEvent } from 'react'
import { loginWebAuthn, DEMO_AUTH } from '../lib/api'

export default function Login({ onDone }: { onDone: () => void }) {
  const [username, setUsername] = useState('alvaro')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setErr('')
    setBusy(true)
    try {
      if (DEMO_AUTH) {
        onDone()
        return
      }
      await loginWebAuthn(username)
      onDone()
    } catch (e: any) {
      setErr(e?.message || 'sign-in failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth">
      <div className="panel">
        <div className="brand" style={{ padding: 0, marginBottom: 20 }}>
          <div className="mark">m</div>
          <div className="name">marshmallows<small>TrustSentinel</small></div>
        </div>
        <h1>Welcome back</h1>
        <p className="lede">Passwordless sign-in with your security key.</p>
        <form onSubmit={submit}>
          <div className="field">
            <label>Username</label>
            <input value={username} onChange={(e) => setUsername(e.target.value)} autoFocus />
          </div>
          {err && <p style={{ color: 'var(--danger)', fontSize: 13, margin: '0 0 12px' }}>{err}</p>}
          <button className="btn primary" style={{ width: '100%', justifyContent: 'center' }} disabled={busy}>
            {busy ? 'Waiting for your key…' : 'Sign in with a security key'}
          </button>
        </form>
        <div className="hr" />
        <p className="muted" style={{ margin: 0 }}>No account yet? Ask an administrator for an invite link.</p>
      </div>
    </div>
  )
}
