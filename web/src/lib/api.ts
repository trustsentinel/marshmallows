// API client for the marshmallows dashboard. Talks to mm-auth (identity) and
// the broker (device graph + terminal WebSockets). Endpoints are env-driven;
// when VITE_DEMO=1 (the default until the backends are wired) it serves local
// demo data so the UI is fully explorable offline.
import { create, get } from '@github/webauthn-json'

const AUTH = import.meta.env.VITE_AUTH_BASE ?? 'http://localhost:8090'
const BROKER = import.meta.env.VITE_BROKER_BASE ?? 'http://localhost:8081'
const BROKER_WS = import.meta.env.VITE_BROKER_WS ?? 'ws://localhost:8081'
export const CLOUD_ID = import.meta.env.VITE_CLOUD_ID ?? 'valc31'
export const DEMO = (import.meta.env.VITE_DEMO ?? '1') === '1'

export type DeviceStatus = 'online' | 'idle' | 'offline'
export type Device = {
  id: string // cloud_id.device_id
  name: string
  kind: 'gateway' | 'device'
  status: DeviceStatus
  addr: string
  os: string
  lastSeen: string
}
export type Session = { username: string; totp: boolean } | null

export const demoDevices: Device[] = [
  { id: `${CLOUD_ID}.gw01`, name: 'Gateway — Valencia', kind: 'gateway', status: 'online', addr: '10.8.0.1', os: 'Debian 12', lastSeen: 'now' },
  { id: `${CLOUD_ID}.5g0a`, name: 'Greenhouse sensor hub', kind: 'device', status: 'online', addr: '10.8.0.14', os: 'Raspbian (Pi 4)', lastSeen: '12s ago' },
  { id: `${CLOUD_ID}.7b2c`, name: 'Cold-store controller', kind: 'device', status: 'online', addr: '10.8.0.22', os: 'Raspbian (Pi 3)', lastSeen: '4s ago' },
  { id: `${CLOUD_ID}.9d41`, name: 'Perimeter camera relay', kind: 'device', status: 'idle', addr: '10.8.0.31', os: 'Alpine (Thingy)', lastSeen: '3m ago' },
  { id: `${CLOUD_ID}.a0f8`, name: 'Soil telemetry node', kind: 'device', status: 'offline', addr: '—', os: 'Raspbian (Pi Zero)', lastSeen: '2h ago' },
]

export async function getSession(): Promise<Session> {
  if (DEMO) return { username: 'alvaro', totp: true }
  try {
    const r = await fetch(`${AUTH}/api/session`, { credentials: 'include' })
    return r.ok ? await r.json() : null
  } catch {
    return null
  }
}

export async function logout(): Promise<void> {
  if (DEMO) return
  try {
    await fetch(`${AUTH}/api/logout`, { method: 'POST', credentials: 'include' })
  } catch {
    /* ignore */
  }
}

export async function listDevices(): Promise<Device[]> {
  if (DEMO) return demoDevices
  try {
    const r = await fetch(`${BROKER}/devices.json`)
    const g = await r.json()
    return (g.nodes ?? []).map((n: any): Device => ({
      id: n.id,
      name: n.id,
      kind: n.group === 1 ? 'gateway' : 'device',
      status: 'online',
      addr: n.addr ?? '—',
      os: n.os ?? 'unknown',
      lastSeen: 'now',
    }))
  } catch {
    return demoDevices
  }
}

export async function createAgentToken(): Promise<string> {
  if (DEMO) return 'mm_' + crypto.getRandomValues(new Uint32Array(4)).join('').slice(0, 32)
  const r = await fetch(`${AUTH}/api/agent-tokens`, { method: 'POST', credentials: 'include' })
  return (await r.json()).token
}

export async function loginWebAuthn(username: string): Promise<{ totp_required?: boolean }> {
  const beginResp = await fetch(`${AUTH}/api/login/begin`, {
    method: 'POST', credentials: 'include',
    headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username }),
  })
  if (!beginResp.ok) throw new Error('unknown user')
  const options = await beginResp.json()
  const credential = await get(options)
  const finishResp = await fetch(`${AUTH}/api/login/finish`, {
    method: 'POST', credentials: 'include',
    headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(credential),
  })
  if (!finishResp.ok) throw new Error('login failed')
  return finishResp.json()
}

export async function registerWebAuthn(username: string, invite: string): Promise<void> {
  const beginResp = await fetch(`${AUTH}/api/register/begin`, {
    method: 'POST', credentials: 'include',
    headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, invite_token: invite }),
  })
  if (!beginResp.ok) throw new Error('invalid invite')
  const options = await beginResp.json()
  const credential = await create(options)
  const finishResp = await fetch(`${AUTH}/api/register/finish`, {
    method: 'POST', credentials: 'include',
    headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(credential),
  })
  if (!finishResp.ok) throw new Error('registration failed')
}

export async function verifyTotp(code: string): Promise<boolean> {
  if (DEMO) return code.length === 6
  const r = await fetch(`${AUTH}/api/totp/verify`, {
    method: 'POST', credentials: 'include',
    headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ code }),
  })
  return r.ok
}

export function terminalControlURL(deviceId: string): string {
  return `${BROKER_WS}/open/${encodeURIComponent(deviceId)}`
}
