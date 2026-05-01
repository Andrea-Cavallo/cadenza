package metrics

import "expvar"

var (
	GenerationsTotal = expvar.NewInt("cadenza_generations_total")
	GenerationErrors = expvar.NewInt("cadenza_generation_errors_total")
	CacheHits        = expvar.NewInt("cadenza_cache_hits_total")
	CacheMisses      = expvar.NewInt("cadenza_cache_misses_total")
	LLMCalls         = expvar.NewInt("cadenza_llm_calls_total")
	LLMErrors        = expvar.NewInt("cadenza_llm_errors_total")
	OfflinePatterns  = expvar.NewInt("cadenza_offline_patterns_total")
)
