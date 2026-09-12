import { useState } from 'react'
import { createAgentToken } from '../lib/api'

export default function Enroll() {
  const [token, setToken] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const mint = async () => {
    setBusy(true)
    try {
      setToken(await createAgentToken())
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="grid cols-2" style={{ alignItems: 'start' }}>
      <div className="card">
        <div className="hd"><h3>Enroll a new device</h3></div>
        <div className="bd">
          <p className="muted">
            Mint a single-use enrollment token, then bake it into the device image. The agent
            presents it to the broker once; the broker verifies it with the identity service and
            the device joins the mini-cloud — no open inbound port on the device.
          </p>
          <button className="btn primary" onClick={mint} disabled={busy}>
            {busy ? 'Minting…' : 'Mint enrollment token'}
          </button>
          {token && (
            <>
              <div className="hr" />
              <div className="section-title">One-time token</div>
              <div className="field"><input readOnly value={token} onFocus={(e) => e.currentTarget.select()} /></div>
              <p className="muted">Expires in 60 minutes · single-use · consumed on first check.</p>
            </>
          )}
        </div>
      </div>
      <div className="card">
        <div className="hd"><h3>Install on the device</h3></div>
        <div className="bd">
          <ol className="muted" style={{ lineHeight: 2, paddingLeft: 18, margin: 0 }}>
            <li>Download the Raspberry Pi image for this mini-cloud.</li>
            <li>Flash it and boot the device on any network.</li>
            <li>Run <span className="kbd">mm-agent</span> and paste the token when prompted.</li>
            <li>The device registers over Noise and appears on the Overview map.</li>
          </ol>
        </div>
      </div>
    </div>
  )
}
