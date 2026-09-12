import { useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { DEMO } from '../lib/api'

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
    term.writeln('\x1b[35m✓ session established\x1b[0m — the hub relayed ciphertext only\r\n')
    const prompt = `\x1b[35mpi@${dev.split('.').slice(1).join('.')}\x1b[0m:~$ `
    if (DEMO) {
      term.writeln('Linux ' + (dev.split('.').pop() ?? 'node') + ' 6.1.0-rpi8 #1 SMP aarch64 GNU/Linux')
      term.write(prompt)
    }

    let line = ''
    const disp = term.onKey(({ key, domEvent: e }) => {
      if (e.key === 'Enter') {
        term.write('\r\n')
        if (line.trim() === 'uptime') term.write(' 14:22:07 up 9 days,  1 user,  load average: 0.07, 0.04, 0.01\r\n')
        else if (line.trim() === 'whoami') term.write('pi\r\n')
        else if (line.trim()) term.write(`${line.trim()}: command not found\r\n`)
        line = ''
        term.write(prompt)
      } else if (e.key === 'Backspace') {
        if (line.length) { line = line.slice(0, -1); term.write('\b \b') }
      } else if (key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
        line += key
        term.write(key)
      }
    })
    const onResize = () => fit.fit()
    window.addEventListener('resize', onResize)
    return () => { disp.dispose(); window.removeEventListener('resize', onResize); term.dispose() }
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
              <span className="term-title">{id} — /bin/bash</span>
            </div>
            <div ref={ref} style={{ height: 380 }} />
          </div>
        </div>
      </div>
    </div>
  )
}
