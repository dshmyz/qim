// Client version management types

export interface ClientVersion {
  id: number
  version: string
  platform: 'windows' | 'macos' | 'linux'
  releaseDate: string
  updateNotes: string
  forceUpdate: boolean
  rolloutPercentage: number
  downloadUrl: string
  status: 'active' | 'inactive'
  createdAt: string
  updatedAt: string
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
