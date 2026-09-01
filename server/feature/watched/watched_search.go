package watched

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sbondCo/Watcharr/database/entity"
)

type watchedTitleCandidate struct {
	ID   uint
	Name string
}

func (s *Service) searchWatchedIDsByTitle(userID uint, query string) ([]uint, error) {
	candidates := []watchedTitleCandidate{}
	res := s.db.
		Model(&entity.Watched{}).
		Select("watcheds.id AS id, COALESCE(`Content`.`title`, `Game`.`name`, '') AS name").
		Joins("Content").
		Joins("Game").
		Where("watcheds.user_id = ?", userID).
		Scan(&candidates)
	if res.Error != nil {
		return nil, res.Error
	}

	ids := make([]uint, 0, len(candidates))
	for _, candidate := range candidates {
		if fuzzyTitleMatch(candidate.Name, query) {
			ids = append(ids, candidate.ID)
		}
	}
	return ids, nil
}

func fuzzyTitleMatch(title string, query string) bool {
	title = normalizeSearchText(title)
	query = normalizeSearchText(query)
	if title == "" || query == "" {
		return false
	}
	if strings.Contains(title, query) {
		return true
	}

	titleTokens := strings.Fields(title)
	queryTokens := strings.Fields(query)
	titleIndex := 0
	for _, queryToken := range queryTokens {
		matched := false
		for titleIndex < len(titleTokens) {
			if fuzzyTokenMatch(titleTokens[titleIndex], queryToken) {
				matched = true
				titleIndex++
				break
			}
			titleIndex++
		}
		if !matched {
			return false
		}
	}
	return true
}

func normalizeSearchText(value string) string {
	var normalized strings.Builder
	lastWasSpace := true
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			normalized.WriteRune(char)
			lastWasSpace = false
		} else if !lastWasSpace {
			normalized.WriteByte(' ')
			lastWasSpace = true
		}
	}
	return strings.TrimSpace(normalized.String())
}

func fuzzyTokenMatch(titleToken string, queryToken string) bool {
	if titleToken == queryToken || strings.Contains(titleToken, queryToken) {
		return true
	}
	queryLength := utf8.RuneCountInString(queryToken)
	maxDistance := 0
	switch {
	case queryLength >= 12:
		maxDistance = 3
	case queryLength >= 7:
		maxDistance = 2
	case queryLength >= 4:
		maxDistance = 1
	}
	return maxDistance > 0 &&
		levenshteinDistance(titleToken, queryToken, maxDistance) <= maxDistance
}

func levenshteinDistance(a string, b string, maxDistance int) int {
	aRunes := []rune(a)
	bRunes := []rune(b)
	difference := len(aRunes) - len(bRunes)
	if difference > maxDistance || difference < -maxDistance {
		return maxDistance + 1
	}

	previous := make([]int, len(bRunes)+1)
	for i := range previous {
		previous[i] = i
	}
	for i, aRune := range aRunes {
		current := make([]int, len(bRunes)+1)
		current[0] = i + 1
		rowMin := current[0]
		for j, bRune := range bRunes {
			cost := 1
			if aRune == bRune {
				cost = 0
			}
			current[j+1] = min(
				current[j]+1,
				previous[j+1]+1,
				previous[j]+cost,
			)
			rowMin = min(rowMin, current[j+1])
		}
		if rowMin > maxDistance {
			return maxDistance + 1
		}
		previous = current
	}
	return previous[len(bRunes)]
}
