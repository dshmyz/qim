// Client version management types

export interface ClientVersion {
  id: number
  version: string
  platform: 'windows' | 'macos' | 'linux'
  releaseDate: string
  updateNotes: string
  forceUpdate: boolean
  rolloutPercentage?: number
  downloadUrl: string
  status: 'active' | 'inactive'
  createdAt: string
  updatedAt?: string
}

// CLI version management types
export interface CLIVersion {
  id: number
  version: string
  platform: string   // "darwin-arm64" 等
  os: string          // darwin/linux/windows
  arch: string        // amd64/arm64
  downloadUrl: string
  sha256: string
  fileSize: number
  updateNotes: string
  forceUpdate: boolean
  rolloutPercentage?: number
  minVersion: string
  status: 'active' | 'inactive'
  releaseDate: string
  createdAt: string
  updatedAt?: string
}

export interface VersionDistribution {
  version: string
  count: number
}

export interface CreateVersionParams {
  version: string
  platform: 'windows' | 'macos' | 'linux'
  releaseDate: string
  updateNotes: string
  forceUpdate: boolean
  rolloutPercentage: number
  downloadUrl: string
}

export interface UpdateVersionParams {
  updateNotes?: string
  forceUpdate?: boolean
  rolloutPercentage?: number
  status?: 'active' | 'inactive'
}

export interface CrashLog {
  id: number
  platform: string
  appVersion: string
  crashType: string
  crashMessage: string
  stackTrace: string
  deviceInfo: string
  createdAt: string
}

export interface UserFeedback {
  id: number
  userId?: number
  username?: string
  nickname?: string
  handlerName?: string
  type: string
  content: string
  status: string
  priority?: string
  screenshot?: string
  reply?: string
  handlerId?: number
  createdAt?: string
  updatedAt?: string
}
