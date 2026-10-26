package claude

// cacheCreationSplitBound is the total a split is measured against. A missing
// total means the upstream reported no cache-write aggregate at all; the split
// itself is then the only evidence, so it must not be clamped away.
func cacheCreationSplitBound(totalTokens int, tokens5m int, tokens1h int) int {
	if totalTokens > 0 {
		return totalTokens
	}
	return tokens5m + tokens1h
}

// ClampCacheCreationSplit bounds a 5m/1h cache-write split by the reported
// cache-write total without attributing unlabelled tokens to either bucket.
// The long-lived share is preserved first, because it is the more specific and
// more expensive of the two.
func ClampCacheCreationSplit(totalTokens int, tokens5m int, tokens1h int) (int, int) {
	tokens5m = max(tokens5m, 0)
	tokens1h = max(tokens1h, 0)
	bound := cacheCreationSplitBound(totalTokens, tokens5m, tokens1h)
	tokens1h = min(tokens1h, bound)
	tokens5m = min(tokens5m, bound-tokens1h)
	return tokens5m, tokens1h
}

// NormalizeCacheCreationSplit bounds a split by the reported total and
// attributes the unlabelled remainder to 5m writes, so the returned buckets
// always sum to the total. Callers that must keep a split exactly as reported
// (for example the one from message_start) use ClampCacheCreationSplit instead.
func NormalizeCacheCreationSplit(totalTokens int, tokens5m int, tokens1h int) (int, int) {
	tokens5m, tokens1h = ClampCacheCreationSplit(totalTokens, tokens5m, tokens1h)
	remainder := max(cacheCreationSplitBound(totalTokens, tokens5m, tokens1h)-tokens5m-tokens1h, 0)
	return tokens5m + remainder, tokens1h
}
