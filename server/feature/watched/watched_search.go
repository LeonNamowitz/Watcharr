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

// Search a user's movie/show titles and game names, returning only the watched
// row ids that fuzzy match query. GetWatchedPage uses these ids so its existing
// filtering, sorting, pinned ordering, and pagination remain authoritative.
func (s *Service) searchWatchedIDsByTitle(userID uint, query string) ([]uint, error) {
	var candidates []watchedTitleCandidate
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

	normalizedQuery := normalizeSearchText(query)
	ids := make([]uint, 0, len(candidates))
	for _, candidate := range candidates {
		if fuzzyNormalizedTitleMatch(
			normalizeSearchText(candidate.Name),
			normalizedQuery,
		) {
			ids = append(ids, candidate.ID)
		}
	}
	return ids, nil
}

// Match exact substrings first, then ignore separators and token order while
// allowing a small number of edits for longer search terms.
func fuzzyTitleMatch(title string, query string) bool {
	return fuzzyNormalizedTitleMatch(
		normalizeSearchText(title),
		normalizeSearchText(query),
	)
}

func fuzzyNormalizedTitleMatch(title string, query string) bool {
	if title == "" || query == "" {
		return false
	}
	if strings.Contains(title, query) {
		return true
	}

	compactTitle := strings.ReplaceAll(title, " ", "")
	compactQuery := strings.ReplaceAll(query, " ", "")
	if strings.Contains(compactTitle, compactQuery) {
		return true
	}

	titleTokens := strings.Fields(title)
	queryTokens := strings.Fields(query)
	for _, queryToken := range queryTokens {
		matched := false
		for _, titleToken := range titleTokens {
			if fuzzyTokenMatch(titleToken, queryToken) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// Lowercase text and collapse punctuation or other separators into spaces so
// formatting differences such as "Spider-Man" and "spider man" do not matter.
func normalizeSearchText(value string) string {
	var normalized strings.Builder
	lastWasSpace := true
	for _, char := range value {
		char = unicode.ToLower(char)
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
	if strings.Contains(titleToken, queryToken) {
		return true
	}
	titleFirst, _ := utf8.DecodeRuneInString(titleToken)
	queryFirst, _ := utf8.DecodeRuneInString(queryToken)
	// Avoid treating unrelated words of similar length as typos.
	if titleFirst != queryFirst {
		return false
	}
	queryLength := utf8.RuneCountInString(queryToken)
	maxDistance := 0
	// Keep short terms exact to avoid noisy matches.
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

// Calculate edit distance with a caller-provided cutoff. Returning
// maxDistance+1 signals that the strings cannot be a fuzzy match.
func levenshteinDistance(a string, b string, maxDistance int) int {
	aRunes := []rune(a)
	bRunes := []rune(b)
	difference := len(aRunes) - len(bRunes)
	if difference > maxDistance || difference < -maxDistance {
		return maxDistance + 1
	}

	previous := make([]int, len(bRunes)+1)
	current := make([]int, len(bRunes)+1)
	for i := range previous {
		previous[i] = i
	}
	for i, aRune := range aRunes {
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
		previous, current = current, previous
	}
	return previous[len(bRunes)]
}
