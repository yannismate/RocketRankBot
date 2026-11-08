package formatter

import (
	"RocketRankBot/services/commander/internal/tracker"
	"testing"
)

func TestFormatRankStringWithTwoCharacterPlaylistAbbreviation(t *testing.T) {
	rankData := &tracker.Ranks{
		Ranks: []*tracker.PlayerRank{
			{
				Playlist: tracker.Heatseeker,
				Mmr:      1350,
				Rank:     13,
				Division: 2,
			},
		},
	}

	got := FormatRankString(rankData, "$(hs.m) $(hs.r.s) $(hs.d)")
	want := "1350 D1 III"
	if got != want {
		t.Fatalf("FormatRankString() = %q, want %q", got, want)
	}
}

func TestFormatRankStringWithSingleCharacterPlaylistAbbreviation(t *testing.T) {
	rankData := &tracker.Ranks{
		Ranks: []*tracker.PlayerRank{
			{
				Playlist: tracker.Ranked2v2,
				Mmr:      1200,
			},
		},
	}

	got := FormatRankString(rankData, "$(2.m)")
	want := "1200"
	if got != want {
		t.Fatalf("FormatRankString() = %q, want %q", got, want)
	}
}

func TestFormatRankStringLeavesUnknownPlaylistAbbreviationUnchanged(t *testing.T) {
	got := FormatRankString(&tracker.Ranks{}, "$(z.m)")
	want := "$(z.m)"
	if got != want {
		t.Fatalf("FormatRankString() = %q, want %q", got, want)
	}
}

func TestFormatRankStringLeavesUnknownTwoChararacterPlaylistAbbreviationUnchanged(t *testing.T) {
	got := FormatRankString(&tracker.Ranks{}, "$(zz.m)")
	want := "$(zz.m)"
	if got != want {
		t.Fatalf("FormatRankString() = %q, want %q", got, want)
	}
}
