package stats

import "strings"

type ReviewLengths struct {
	Shortest *ReviewLength `json:"shortest,omitempty"`
	Longest  *ReviewLength `json:"longest,omitempty"`
}

type ReviewLength struct {
	Item      MediaCard `json:"item"`
	WordCount int       `json:"wordCount"`
}

// Callers supply the same scoped titles used by their other review statistics.
func buildReviewLengths(records []savedMediaRecord) *ReviewLengths {
	result := &ReviewLengths{}
	before := func(a, b MediaCard) bool {
		if a.Title != b.Title {
			return a.Title < b.Title
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.ID < b.ID
	}
	for _, record := range records {
		words := len(strings.Fields(record.watched.Thoughts))
		if words == 0 {
			continue
		}
		entry := &ReviewLength{Item: record.card, WordCount: words}
		if result.Shortest == nil || words < result.Shortest.WordCount || (words == result.Shortest.WordCount && before(entry.Item, result.Shortest.Item)) {
			result.Shortest = entry
		}
		if result.Longest == nil || words > result.Longest.WordCount || (words == result.Longest.WordCount && before(entry.Item, result.Longest.Item)) {
			result.Longest = entry
		}
	}
	return result
}
