package chatcost_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd/chatcost"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestCalculateTotalCostMicros(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		usage nicloudsdk.ChatMessageUsage
		cost  *nicloudsdk.ModelCostConfig
		want  *int64
	}{
		{
			name:  "nil cost returns nil",
			usage: nicloudsdk.ChatMessageUsage{InputTokens: ptr.Ref[int64](1000)},
			cost:  nil,
			want:  nil,
		},
		{
			name: "all priced usage fields nil returns nil",
			usage: nicloudsdk.ChatMessageUsage{
				TotalTokens:  ptr.Ref[int64](1234),
				ContextLimit: ptr.Ref[int64](8192),
			},
			cost: &nicloudsdk.ModelCostConfig{
				InputPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("3")),
			},
			want: nil,
		},
		{
			name:  "sub-micro total rounds up to 1",
			usage: nicloudsdk.ChatMessageUsage{InputTokens: ptr.Ref[int64](1)},
			cost: &nicloudsdk.ModelCostConfig{
				InputPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("0.01")),
			},
			want: ptr.Ref[int64](1),
		},
		{
			name:  "simple input only",
			usage: nicloudsdk.ChatMessageUsage{InputTokens: ptr.Ref[int64](1000)},
			cost: &nicloudsdk.ModelCostConfig{
				InputPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("3")),
			},
			want: ptr.Ref[int64](3000),
		},
		{
			name:  "simple output only",
			usage: nicloudsdk.ChatMessageUsage{OutputTokens: ptr.Ref[int64](500)},
			cost: &nicloudsdk.ModelCostConfig{
				OutputPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("15")),
			},
			want: ptr.Ref[int64](7500),
		},
		{
			name: "reasoning tokens included in output total",
			usage: nicloudsdk.ChatMessageUsage{
				OutputTokens:    ptr.Ref[int64](500),
				ReasoningTokens: ptr.Ref[int64](200),
			},
			cost: &nicloudsdk.ModelCostConfig{
				OutputPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("15")),
			},
			want: ptr.Ref[int64](7500),
		},
		{
			name:  "cache read tokens",
			usage: nicloudsdk.ChatMessageUsage{CacheReadTokens: ptr.Ref[int64](10000)},
			cost: &nicloudsdk.ModelCostConfig{
				CacheReadPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("0.3")),
			},
			want: ptr.Ref[int64](3000),
		},
		{
			name:  "cache creation tokens",
			usage: nicloudsdk.ChatMessageUsage{CacheCreationTokens: ptr.Ref[int64](5000)},
			cost: &nicloudsdk.ModelCostConfig{
				CacheWritePricePerMillionTokens: ptr.Ref(decimal.RequireFromString("3.75")),
			},
			want: ptr.Ref[int64](18750),
		},
		{
			name: "full mixed usage totals all components exactly",
			usage: nicloudsdk.ChatMessageUsage{
				InputTokens:         ptr.Ref[int64](101),
				OutputTokens:        ptr.Ref[int64](201),
				ReasoningTokens:     ptr.Ref[int64](52),
				CacheReadTokens:     ptr.Ref[int64](1005),
				CacheCreationTokens: ptr.Ref[int64](33),
				TotalTokens:         ptr.Ref[int64](1391),
				ContextLimit:        ptr.Ref[int64](4096),
			},
			cost: &nicloudsdk.ModelCostConfig{
				InputPricePerMillionTokens:      ptr.Ref(decimal.RequireFromString("1.23")),
				OutputPricePerMillionTokens:     ptr.Ref(decimal.RequireFromString("4.56")),
				CacheReadPricePerMillionTokens:  ptr.Ref(decimal.RequireFromString("0.7")),
				CacheWritePricePerMillionTokens: ptr.Ref(decimal.RequireFromString("7.89")),
			},
			want: ptr.Ref[int64](2005),
		},
		{
			name: "partial pricing only input contributes",
			usage: nicloudsdk.ChatMessageUsage{
				InputTokens:         ptr.Ref[int64](1234),
				OutputTokens:        ptr.Ref[int64](999),
				ReasoningTokens:     ptr.Ref[int64](111),
				CacheReadTokens:     ptr.Ref[int64](500),
				CacheCreationTokens: ptr.Ref[int64](250),
			},
			cost: &nicloudsdk.ModelCostConfig{
				InputPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("2.5")),
			},
			want: ptr.Ref[int64](3085),
		},
		{
			name:  "zero tokens with pricing returns zero pointer",
			usage: nicloudsdk.ChatMessageUsage{InputTokens: ptr.Ref[int64](0)},
			cost: &nicloudsdk.ModelCostConfig{
				InputPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("3")),
			},
			want: ptr.Ref[int64](0),
		},
		{
			name:  "usage only in unpriced categories returns nil",
			usage: nicloudsdk.ChatMessageUsage{InputTokens: ptr.Ref[int64](1000)},
			cost: &nicloudsdk.ModelCostConfig{
				OutputPricePerMillionTokens: ptr.Ref(decimal.RequireFromString("15")),
			},
			want: nil,
		},
		{
			name:  "non nil usage with empty cost config returns nil",
			usage: nicloudsdk.ChatMessageUsage{InputTokens: ptr.Ref[int64](42)},
			cost:  &nicloudsdk.ModelCostConfig{},
			want:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := chatcost.CalculateTotalCostMicros(tt.usage, tt.cost)

			if tt.want == nil {
				require.Nil(t, got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, *tt.want, *got)
			}
		})
	}
}
