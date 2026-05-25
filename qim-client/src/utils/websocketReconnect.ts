export interface ReconnectConfig {
  baseDelay: number
  maxDelay: number
  maxAttempts: number
  jitterRange: [number, number]
}

export const DEFAULT_RECONNECT_CONFIG: ReconnectConfig = {
  baseDelay: 1000,
  maxDelay: 30000,
  maxAttempts: 10,
  jitterRange: [0, 2000]
}

export function calculateReconnectDelay(
  attempt: number,
  config: ReconnectConfig = DEFAULT_RECONNECT_CONFIG
): number {
  const exponentialDelay = config.baseDelay * Math.pow(2, attempt)
  const cappedDelay = Math.min(exponentialDelay, config.maxDelay)
  const [minJitter, maxJitter] = config.jitterRange
  const jitter = minJitter + Math.random() * (maxJitter - minJitter)
  return cappedDelay + jitter
}

export function shouldReconnect(
  attempt: number,
  config: ReconnectConfig = DEFAULT_RECONNECT_CONFIG
): boolean {
  return attempt < config.maxAttempts
}
