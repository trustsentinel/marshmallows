import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { listDevices, type Device, CLOUD_ID } from '../lib/api'

function CloudMap({ devices }: { devices: Device[] }) {
  const cx = 320, cy = 128, R = 92
  const nodes = devices.filter((d) => d.kind === 'device')
  return (
    <svg className="cloudmap" viewBox="0 0 640 256" role="img" aria-label="mini-cloud topology">
      <circle className="ring" cx={cx} cy={cy} r={R + 26} />
      {nodes.map((d, i) => {
        const a = (2 * Math.PI * i) / nodes.length - Math.PI / 2
        const x = cx + R * Math.cos(a)
        const y = cy + R * Math.sin(a)
        const off = d.status === 'offline'
        return (
          <g key={d.id}>
            <line className="edge" x1={cx} y1={cy} x2={x} y2={y} style={off ? { opacity: 0.4, strokeDasharray: '3 4' } : undefined} />
            <circle className="node" cx={x} cy={y} r={9} style={{ stroke: off ? 'var(--ink-faint)' : undefined, opacity: off ? 0.5 : 1 }} />
            <text x={x} y={y + 22} textAnchor="middle">{d.id.split('.')[1]}</text>
          </g>
        )
      })}
      <circle className="hub" cx={cx} cy={cy} r={16} />
      <text x={cx} y={cy + 36} textAnchor="middle" style={{ fill: 'var(--verdigris)', fontWeight: 600 }}>{CLOUD_ID}</text>
    </svg>
  )
}

function Stat({ label, value, delta }: { label: string; value: string; delta?: React.ReactNode }) {
  return (
    <div className="card stat">
      <div className="label">{label}</div>
      <div className="value">{value}</div>
      {delta && <div className="delta">{delta}</div>}
    </div>
  )
}

export default function Dashboard() {
  const [devices, setDevices] = useState<Device[]>([])
  const nav = useNavigate()
  useEffect(() => {
    listDevices().then(setDevices)
  }, [])

  const online = devices.filter((d) => d.status === 'online').length
  const gateways = devices.filter((d) => d.kind === 'gateway').length

  return (
    <div className="grid" style={{ gap: 18 }}>
      <div className="grid cols-4">
        <Stat label="Devices" value={String(devices.length)} delta={<><b>{online}</b> online now</>} />
        <Stat label="Gateways" value={String(gateways)} delta="1 mini-cloud" />
        <Stat label="Live sessions" value="1" delta="1 brokered shell" />
        <Stat label="Mini-cloud" value={CLOUD_ID} delta="federation-ready" />
      </div>

      <div className="grid cols-2">
        <div className="card">
          <div className="hd">
            <h3>Topology</h3>
            <span className="badge v">Noise · E2E</span>
          </div>
          <div className="bd">
            <CloudMap devices={devices} />
          </div>
        </div>

        <div className="card">
          <div className="hd">
            <h3>Connected devices</h3>
            <button className="btn sm primary" onClick={() => nav('/enroll')}>+ Enroll</button>
          </div>
          <div className="bd" style={{ paddingTop: 4, paddingBottom: 4 }}>
            <div className="devices">
              {devices.map((d) => (
                <div className="device" key={d.id}>
                  <div className="icn">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
                      {d.kind === 'gateway'
                        ? <path d="M12 2v6M4.9 7a8 8 0 0114.2 0M8 10.5a4 4 0 018 0M12 20h.01" strokeLinecap="round" />
                        : <><rect x="5" y="5" width="14" height="14" rx="2" /><path d="M9 3v2M15 3v2M9 19v2M15 19v2M3 9h2M3 15h2M19 9h2M19 15h2" strokeLinecap="round" /></>}
                    </svg>
                  </div>
                  <div>
                    <div className="id">{d.id}</div>
                    <div className="meta">{d.name} · {d.os}</div>
                  </div>
                  <div style={{ textAlign: 'right' }}>
                    <div className="addr">{d.addr}</div>
                    <div className="meta"><span className={`dot ${d.status}`} /> &nbsp;{d.status} · {d.lastSeen}</div>
                  </div>
                  <button className="btn sm" disabled={d.status === 'offline'} onClick={() => nav(`/devices/${encodeURIComponent(d.id)}`)}>
                    Connect
                  </button>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
