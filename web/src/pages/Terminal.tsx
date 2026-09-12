import { useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { DEMO_DATA, terminalControlURL } from '../lib/api'

export default function DeviceTerminal() {
  const { id } = useParams()
  const nav = useNavigate()
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!ref.current) return
    const dev = id ?? 'device'
    const term = new Terminal({
      cursorBlink: true,
      fontFamily: 'IBM Plex Mono, ui-monospace, monospace',
      fontSize: 13,
      theme: { background: '#f4f1fe', foreground: '#2a2440', cursor: '#7c5cff', selectionBackground: '#e2d9ff' },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(ref.current)
    fit.fit()

    term.writeln('\x1b[35mmarshmallows\x1b[0m — brokered shell (Noise, mutually authenticated)')
    term.writeln(`connecting to \x1b[1m${dev}\x1b[0m through the hub …`)

    let cleanup = () => {}

    if (DEMO_DATA) {
      // Offline demo: a local fake shell.
      term.writeln('\x1b[35m✓ session established\x1b[0m — the hub relayed ciphertext only\r\n')
      const prompt = `\x1b[35mpi@${dev.split('.').slice(1).join('.')}\x1b[0m:~$ `
      term.write(prompt)
      let line = ''
      const disp = term.onData((d) => {
        for (const ch of d) {
          if (ch === '\r') {
            term.write('\r\n')
            if (line.trim() === 'uptime') term.write(' 14:22:07 up 9 days,  load 0.07\r\n')
            else if (line.trim()) term.write(`${line.trim()}: command not found\r\n`)
            line = ''
            term.write(prompt)
          } else if (ch === '\x7f') {
            if (line.length) { line = line.slice(0, -1); term.write('\b \b') }
          } else {
            line += ch
            term.write(ch)
          }
        }
      })
      cleanup = () => disp.dispose()
    } else {
      // Live: pipe the terminal to the broker, which bridges to the agent shell.
      const ws = new WebSocket(terminalControlURL(dev))
      ws.binaryType = 'arraybuffer'
      ws.onopen = () => {
        term.writeln('\x1b[35m✓ session established\x1b[0m — the hub relays ciphertext only\r\n')
        term.focus()
      }
      ws.onmessage = (ev) => {
        if (typeof ev.data === 'string') term.write(ev.data)
        else term.write(new Uint8Array(ev.data))
      }
      ws.onclose = () => term.writeln('\r\n\x1b[31m[session closed]\x1b[0m')
      ws.onerror = () => term.writeln('\r\n\x1b[31m[connection error]\x1b[0m')
      const disp = term.onData((d) => {
        if (ws.readyState === WebSocket.OPEN) ws.send(d)
      })
      cleanup = () => { disp.dispose(); ws.close() }
    }

    const onResize = () => fit.fit()
    window.addEventListener('resize', onResize)
    return () => {
      window.removeEventListener('resize', onResize)
      cleanup()
      term.dispose()
    }
  }, [id])

  return (
    <div className="grid" style={{ gap: 16 }}>
      <div className="card">
        <div className="hd">
          <h3 className="mono" style={{ fontFamily: 'var(--mono)', fontSize: 14 }}>{id}</h3>
          <button className="btn sm" onClick={() => nav('/dashboard')}>← Back to overview</button>
        </div>
        <div className="bd">
          <div className="term-wrap">
            <div className="term-bar">
              <span className="tl" style={{ background: '#e05b49' }} />
              <span className="tl" style={{ background: '#e0a23a' }} />
              <span className="tl" style={{ background: '#35b07f' }} />
              <span className="term-title">{id} — /bin/sh</span>
            </div>
            <div ref={ref} style={{ height: 380 }} />
          </div>
        </div>
      </div>
    </div>
  )
}
