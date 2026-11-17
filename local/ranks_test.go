package main

import "testing"

func TestApplyRankHitUsesCurrentThenPeak(t *testing.T) {
	icons := map[int]string{19: "diamond.png", 21: "ascendant.png"}
	current := player{Rank: "Silver 1", Tier: 9}
	applyRankHit(&current, rankHit{tier: 19, rr: 45, peak: 21, known: true}, icons)
	if current.Rank != "Diamond 2" || current.RR != 45 || current.RankPeak || current.RankIcon != "/pack/rank/19.png" || current.Peak != "Ascendant 1" || current.PeakIcon != "/pack/rank/21.png" {
		t.Fatalf("current %+v", current)
	}
	peaked := player{Rank: "Gold 1", Tier: 12}
	applyRankHit(&peaked, rankHit{tier: 0, peak: 21, known: true}, icons)
	if peaked.Rank != "Ascendant 1" || !peaked.RankPeak || peaked.Tier != 0 || peaked.RR != 0 || peaked.RankIcon != "/pack/rank/21.png" {
		t.Fatalf("peak %+v", peaked)
	}
	none := player{}
	applyRankHit(&none, rankHit{tier: 0, peak: 0, known: true}, icons)
	if none.Rank != "Unranked" || none.RankPeak || none.RankIcon != "" {
		t.Fatalf("unranked %+v", none)
	}
	kept := player{Rank: "Gold 1", Tier: 12}
	applyRankHit(&kept, rankHit{tier: 0, peak: 0, known: true}, icons)
	if kept.Rank != "Gold 1" || kept.Tier != 12 {
		t.Fatalf("kept %+v", kept)
	}
}

func TestPeakReadsHighestTier(t *testing.T) {
	body := map[string]any{
		"LatestCompetitiveUpdate": map[string]any{"TierAfterUpdate": float64(15)},
		"QueueSkills": map[string]any{
			"competitive": map[string]any{
				"SeasonalInfoBySeasonID": map[string]any{
					"act": map[string]any{"CompetitiveTier": float64(18), "HighestTierAchieved": float64(21), "RankedRating": float64(40)},
				},
			},
		},
	}
	if peakTier(body) != 21 {
		t.Fatal(peakTier(body))
	}
	tier, rr := tierFromMMR(body)
	if tier != 18 || rr != 40 {
		t.Fatalf("tier %d rr %d", tier, rr)
	}
}

func TestTierFromMMRReadsSeasonRank(t *testing.T) {
	body := map[string]any{
		"QueueSkills": map[string]any{
			"competitive": map[string]any{
				"SeasonalInfoBySeasonID": map[string]any{
					"act": map[string]any{"Rank": float64(21), "RankedRating": float64(12)},
				},
			},
		},
	}
	tier, rr := tierFromMMR(body)
	if tier != 21 || rr != 12 {
		t.Fatalf("tier %d rr %d", tier, rr)
	}
	if peakTier(body) != 21 {
		t.Fatalf("peak %d", peakTier(body))
	}
}
