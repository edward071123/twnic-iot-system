const ENGINEER_AUTH_SESSION_KEY = 'ward_engineer_authed'

export function isEngineerAuthed() {
  if (typeof window === 'undefined') return false
  return window.sessionStorage.getItem(ENGINEER_AUTH_SESSION_KEY) === '1'
}

export function markEngineerAuthed() {
  if (typeof window === 'undefined') return
  window.sessionStorage.setItem(ENGINEER_AUTH_SESSION_KEY, '1')
}

export function clearEngineerAuthed() {
  if (typeof window === 'undefined') return
  window.sessionStorage.removeItem(ENGINEER_AUTH_SESSION_KEY)
}
