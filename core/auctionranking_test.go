package core

import (
	"testing"

	"github.com/peterldowns/testy/check"
)

func TestRankCoreBids_Integration(t *testing.T) {
	bids := []CoreBid{
		{ID: "bid_a_001", Bidder: "bidder_a", Price: 2.50},
		{ID: "bid_b_001", Bidder: "bidder_b", Price: 2.25},
		{ID: "bid_c_001", Bidder: "bidder_c", Price: 2.75},
	}

	rankingResult := RankCoreBids(bids)

	check.Equal(t, 3, len(rankingResult.SortedBidders))
	check.Equal(t, "bidder_c", rankingResult.SortedBidders[0]) // Highest (2.75)
	check.Equal(t, "bidder_a", rankingResult.SortedBidders[1]) // Middle (2.50)
	check.Equal(t, "bidder_b", rankingResult.SortedBidders[2]) // Lowest (2.25)

	check.Equal(t, 2.75, rankingResult.HighestBids["bidder_c"].Price)
	check.Equal(t, 2.50, rankingResult.HighestBids["bidder_a"].Price)
	check.Equal(t, 2.25, rankingResult.HighestBids["bidder_b"].Price)
}
