package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func modelMarketPrice(value float64) *float64 {
	return &value
}

func TestBuildModelMarketEntriesUsesActiveChannelsAndPreservesZeroPrice(t *testing.T) {
	channels := []service.AvailableChannel{
		{
			ID: 2, Name: "second", Status: "active",
			SupportedModels: []service.SupportedModel{
				{
					Name: "gpt-5", Platform: "openai", Adapted: true,
					Pricing: &service.ChannelModelPricing{InputPrice: modelMarketPrice(2e-6)},
				},
			},
		},
		{
			ID: 1, Name: "first", Status: "active",
			SupportedModels: []service.SupportedModel{
				{
					Name: "GPT-5", Platform: "OpenAI",
					Pricing: &service.ChannelModelPricing{InputPrice: modelMarketPrice(0)},
				},
				{Name: "o3", Platform: "openai"},
			},
		},
		{
			ID: 3, Name: "disabled", Status: "disabled",
			SupportedModels: []service.SupportedModel{
				{Name: "hidden", Platform: "openai"},
			},
		},
	}

	entries, activeChannels := buildModelMarketEntries(channels, "", "all")
	require.Equal(t, 2, activeChannels)
	require.Len(t, entries, 2)

	gpt := entries[0]
	require.Equal(t, "openai:gpt-5", gpt.ID)
	require.Equal(t, []int64{1, 2}, gpt.ChannelIDs)
	require.Equal(t, 2, gpt.ChannelCount)
	require.True(t, gpt.Recommended)
	require.True(t, gpt.PlatformAdapted)
	require.Equal(t, "ADAPTED", gpt.Type)
	require.Equal(t, "recommended", gpt.Category)
	require.NotNil(t, gpt.InputPrice)
	require.Zero(t, *gpt.InputPrice)
	require.Nil(t, gpt.OutputPrice)

	official := entries[1]
	require.Equal(t, "OFFICIAL", official.Type)
	require.False(t, official.Recommended)
	require.Nil(t, official.InputPrice)

	recommended, _ := buildModelMarketEntries(channels, "", "recommended")
	require.Len(t, recommended, 1)
	require.Equal(t, gpt.ID, recommended[0].ID)

	adapted, _ := buildModelMarketEntries(channels, "GPT", "platform")
	require.Len(t, adapted, 1)
	require.Equal(t, gpt.ID, adapted[0].ID)
}
