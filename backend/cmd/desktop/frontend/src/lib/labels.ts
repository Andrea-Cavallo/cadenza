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
    case 'deepseek':
      return 'DeepSeek'
    case 'groq':
      return 'Groq'
    case 'mistral':
      return 'Mistral'
    default:
      return provider || 'Provider'
  }
}
