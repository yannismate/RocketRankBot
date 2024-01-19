package formatter

import (
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	tokenMatcher          = regexp.MustCompile("\\$\\(((\\w|\\.)+)\\)")
	tokenExtractor        = regexp.MustCompile("^([u123hrdst])\\.([rdm])\\.?([sml])?$")
	playlistAbbreviations = map[string]trackerggscraper.RankPlaylist{
		"u": trackerggscraper.RankPlaylist_UNRANKED,
		"1": trackerggscraper.RankPlaylist_RANKED_1V1,
		"2": trackerggscraper.RankPlaylist_RANKED_2V2,
		"3": trackerggscraper.RankPlaylist_RANKED_3V3,
		"h": trackerggscraper.RankPlaylist_HOOPS,
		"r": trackerggscraper.RankPlaylist_RUMBLE,
		"d": trackerggscraper.RankPlaylist_DROPSHOT,
		"s": trackerggscraper.RankPlaylist_SNOWDAY,
		"t": trackerggscraper.RankPlaylist_TOURNAMENTS,
	}
	ranksS = map[int]string{
		0: "UR", 1: "B1", 2: "B2", 3: "B3", 4: "S1", 5: "S2", 6: "S3", 7: "G1", 8: "G2", 9: "G3",
		10: "P1", 11: "P2", 12: "P3", 13: "D1", 14: "D2", 15: "D3",
		16: "C1", 17: "C2", 18: "C3", 19: "GC1", 20: "GC2",
		21: "GC3", 22: "SSL",
	}
	ranksM = map[int]string{
		0: "Unranked", 1: "Bronze I", 2: "Bronze II", 3: "Bronze III",
		4: "Silver I", 5: "Silver II", 6: "Silver III", 7: "Gold I", 8: "Gold II", 9: "Gold III",
		10: "Plat I", 11: "Plat II", 12: "Plat III", 13: "Dia I", 14: "Dia II", 15: "Dia III",
		16: "Champ I", 17: "Champ II", 18: "Champ III", 19: "Grand Champ I", 20: "Grand Champ II",
		21: "Grand Champ III", 22: "SSL",
	}
	ranksL = map[int]string{
		0: "Unranked", 1: "Bronze I", 2: "Bronze II", 3: "Bronze III",
		4: "Silver I", 5: "Silver II", 6: "Silver III", 7: "Gold I", 8: "Gold II", 9: "Gold III",
		10: "Platinum I", 11: "Platinum II", 12: "Platinum III", 13: "Diamond I", 14: "Diamond II", 15: "Diamond III",
		16: "Champion I", 17: "Champion II", 18: "Champion III", 19: "Grand Champion I", 20: "Grand Champion II",
		21: "Grand Champion III", 22: "Supersonic Legend",
	}
)

func FormatRankString(rankData *trackerggscraper.PlayerCurrentRanksRes, formatString string) string {
	var result strings.Builder

	matchesBytes := tokenMatcher.FindAllStringIndex(formatString, -1)

	var matches [][]int
	for _, x := range matchesBytes {
		matches = append(matches, []int{utf8.RuneCountInString(formatString[:x[0]]), utf8.RuneCountInString(formatString[:x[1]])})
	}

	if len(matches) == 0 {
		return formatString
	}

	formatChars := []rune(formatString)
	nextMatch := matches[0]
	matchIndex := 0

	for i, ch := range formatChars {
		if i == nextMatch[0] {
			// insert token for current match
			token := formatChars[nextMatch[0]+2 : nextMatch[1]-1]
			result.WriteString(evalToken(rankData, string(token)))
		} else if i+1 == nextMatch[1] {
			// use next match
			matchIndex++
			if matchIndex >= len(matches) {
				nextMatch = []int{-1, -1}
			} else {
				nextMatch = matches[matchIndex]
			}
		} else if i > nextMatch[0] && i < nextMatch[1] {
			// skip token chars
			continue
		} else {
			// non token char
			result.WriteRune(ch)
		}
	}

	return result.String()
}

func evalToken(rankData *trackerggscraper.PlayerCurrentRanksRes, token string) string {
	if token == "name" {
		return rankData.DisplayName
	}

	matches := tokenExtractor.FindAllStringSubmatch(token, -1)
	if len(matches) == 0 {
		return "$(" + string(token) + ")"
	}

	playlist := playlistAbbreviations[matches[0][1]]
	stat := matches[0][2]
	modifier := matches[0][3]

	if modifier == "" {
		modifier = "l"
	}

	for _, ranking := range rankData.Ranks {
		if playlist == ranking.Playlist {
			if stat == "r" {
				return rankToStr(int(ranking.Rank), modifier)
			} else if stat == "d" {
				if modifier == "l" || modifier == "m" {
					return toRoman(int(ranking.Division + 1))
				} else {
					return strconv.Itoa(int(ranking.Division + 1))
				}
			} else if stat == "m" {
				return strconv.Itoa(int(ranking.Mmr))
			}
		}
	}

	return "[no_data:" + token + "]"
}

func rankToStr(rank int, modifier string) string {
	if rank > 22 {
		return "?"
	}
	if modifier == "s" {
		return ranksS[rank]
	} else if modifier == "m" {
		return ranksM[rank]
	} else if modifier == "l" {
		return ranksL[rank]
	}
	return "??"
}

func toRoman(num int) string {
	switch num {
	case 1:
		return "I"
	case 2:
		return "II"
	case 3:
		return "III"
	case 4:
		return "IV"
	}
	return "?"
}
