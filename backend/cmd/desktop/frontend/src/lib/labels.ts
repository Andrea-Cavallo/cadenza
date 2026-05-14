export function providerLabel(provider: string): string {
  switch (provider) {
    case 'claude':
      return 'Claude'
    case 'openai':
      return 'OpenAI'
    case 'gemini':
      return 'Gemini'
    case 'ollama':
      return 'Ollama'
    case 'offline':
      return 'Offline'
    default:
      return provider || 'Provider'
  }
}
