export interface Params {
  bpm: number
  key: string
  bars: number
  groove: string
  style: string
  provider: string
  model: string
  drums: boolean
}

export interface GenerateRequest {
  bpm: number
  key: string
  provider: string
  model: string
  noLlm: boolean
  bars: number
  groove: string
  offlineStyle: string
  seed: string
  reuseChords: boolean
  track: string
}

export interface GenerateResult {
  files: string[]
  elapsed: string
  seed: string
  preview: GenerationPreview
}

export interface ExportEditedRequest {
  bpm: number
  preview: GenerationPreview
}

export interface ExportEditedResult {
  files: string[]
}

export interface GenerationPreview {
  bars: number
  stepsPerBar: number
  chords: ChordPreview[]
  patterns: TrackPreview[]
}

export interface ChordPreview {
  name: string
  fromBar: number
  toBar: number
}

export interface TrackPreview {
  patternType: string
  label: string
  steps: StepPreview[]
}

export interface StepPreview {
  step: number
  note: string
  midi: number
  active: boolean
  accent: boolean
  ghost: boolean
  slide: boolean
  legato: boolean
  staccato: boolean
  durationSteps: number
  velocity: number
}

export interface GeneratedFile {
  path: string
  name?: string
  kind?: string
}

export interface ModelInfo {
  id: string
  name: string
}

export interface ProviderStatus {
  provider: string
  ready: boolean
  installed: boolean
  running: boolean
  authConfigured: boolean
  message: string
  setupHint: string
  defaultModel: string
  models: ModelInfo[]
  localModels: ModelInfo[]
}

