package claudemessages

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
)

const startNoSplit = `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5","content":[],"usage":{"input_tokens":12,"cache_creation_input_tokens":17383,"cache_read_input_tokens":0,"output_tokens":1}}}`

func feed(t *testing.T, info *ClaudeResponseInfo, events ...string) *dto.ClaudeResponse {
	t.Helper()
	var ev *dto.ClaudeResponse
	for _, raw := range events {
		ev = &dto.ClaudeResponse{}
		if err := kitutil.UnmarshalJsonStr(raw, ev); err != nil {
			t.Fatal(err)
		}
		FormatClaudeResponseInfo(ev, nil, info)
	}
	return ev
}

// message_start 无分档时分档未知，不应合成为全部 5m。
func TestDeltaPatchSynthesizes5mWhenStartHasNoSplit(t *testing.T) {
	info := &ClaudeResponseInfo{Usage: &dto.Usage{}}
	delta := feed(t, info, startNoSplit,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":12,"cache_creation_input_tokens":17383,"cache_read_input_tokens":0,"output_tokens":5}}`)
	patched := BuildMessageDeltaPatchUsage(delta, info)
	if patched.CacheCreation != nil {
		got, _ := kitutil.Marshal(patched.CacheCreation)
		t.Errorf("want cache_creation omitted (split unknown), got %s", got)
	}
}

// 非官方的 delta 分档不应只覆盖 5m 而保留 start 的 1h。
func TestDeltaCacheSplitNotDoubleCounted(t *testing.T) {
	info := &ClaudeResponseInfo{Usage: &dto.Usage{}}
	feed(t, info,
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5","content":[],"usage":{"input_tokens":12,"cache_creation_input_tokens":17383,"cache_read_input_tokens":0,"output_tokens":1,"cache_creation":{"ephemeral_5m_input_tokens":8691,"ephemeral_1h_input_tokens":8692}}}}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":12,"cache_creation_input_tokens":17383,"cache_read_input_tokens":0,"output_tokens":5,"cache_creation":{"ephemeral_5m_input_tokens":17383}}}`)
	u := info.Usage
	m5, h1, total := u.ClaudeCacheCreation5mTokens, u.ClaudeCacheCreation1hTokens, u.PromptTokensDetails.CachedCreationTokens
	bill := "BillingUsage canonical: none"
	if c, ok := u.BillingUsage.CanonicalUsage(); ok {
		bill = fmt.Sprintf("BillingUsage canonical: 5m=%d 1h=%d total=%d", c.ClaudeCacheCreation5mTokens, c.ClaudeCacheCreation1hTokens, c.PromptTokensDetails.CachedCreationTokens)
		if c.ClaudeCacheCreation5mTokens+c.ClaudeCacheCreation1hTokens > c.PromptTokensDetails.CachedCreationTokens {
			t.Errorf("billing split exceeds its total; %s", bill)
		}
	}
	if m5+h1 > total || m5 != 8691 || h1 != 8692 {
		t.Errorf("want 5m=8691 1h=8692 (sum<=%d), got 5m=%d 1h=%d sum=%d; %s", total, m5, h1, m5+h1, bill)
	}
}
