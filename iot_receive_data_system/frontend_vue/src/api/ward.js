const API_PREFIX = (import.meta.env.VITE_API_BASE_URL || '/api').replace(/\/$/, '')

async function requestJson(path) {
  const response = await fetch(`${API_PREFIX}${path}`, { cache: 'no-store' })
  if (!response.ok) throw new Error(`API ${response.status}: ${path}`)

  const payload = await response.json()
  if (!payload.success || !payload.data) throw new Error(`Invalid API response: ${path}`)
  return payload.data
}

export function fetchWardFloorOverview(floor) {
  return requestJson(`/ward/floors/${encodeURIComponent(floor)}/overview`)
}

export function fetchWardFloors() {
  return requestJson('/ward/floors')
}

export function fetchSensorHistory(sensorNumber, query = '') {
  const suffix = query ? `?${query}` : ''
  return requestJson(`/ward/sensors/${encodeURIComponent(sensorNumber)}/history${suffix}`)
}

export function fetchSensorThermalTimeline(sensorNumber, query = '') {
  const suffix = query ? `?${query}` : ''
  return requestJson(`/ward/sensors/${encodeURIComponent(sensorNumber)}/thermal/timeline${suffix}`)
}

export function fetchSensorThermalLatest(sensorNumber) {
  return requestJson(`/ward/sensors/${encodeURIComponent(sensorNumber)}/thermal/latest`)
}

export function fetchSensorThermalFrame(sensorNumber, dataId) {
  return requestJson(`/ward/sensors/${encodeURIComponent(sensorNumber)}/thermal/${encodeURIComponent(dataId)}`)
}

export function openWardFloorStream(floor) {
  return new EventSource(`${API_PREFIX}/ward/floors/${encodeURIComponent(floor)}/stream`)
}
