package domain

import "github.com/sbondCo/Watcharr/util"

type (
	TagAddRequest struct {
		Name    string `json:"name" binding:"required"`
		Color   string `json:"color"`
		BgColor string `json:"bgColor"`
	}
	TagOrderRequest struct {
		TagIDs []uint `json:"tagIds" binding:"required"`
	}
	TagBulkAddRequest struct {
		WatchedIDs []uint `json:"watchedIds" binding:"required"`
	}
	TagBulkAddResponse struct {
		Added int64 `json:"added"`
	}
	TagSuggestionOption struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	TagLanguageSuggestionOption struct {
		Code  string `json:"code"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	TagValueSuggestionOption struct {
		Value string `json:"value"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	TagSuggestionOptionsResponse struct {
		Genres                 []TagSuggestionOption         `json:"genres"`
		Keywords               []TagSuggestionOption         `json:"keywords"`
		Composers              []TagSuggestionOption         `json:"composers"`
		Languages              []TagLanguageSuggestionOption `json:"languages"`
		Collections            []TagSuggestionOption         `json:"collections"`
		GameGenres             []TagValueSuggestionOption    `json:"gameGenres"`
		GameModes              []TagValueSuggestionOption    `json:"gameModes"`
		GameCategories         []TagValueSuggestionOption    `json:"gameCategories"`
		FutureReleaseCount     int                           `json:"futureReleaseCount"`
		GameFutureReleaseCount int                           `json:"gameFutureReleaseCount"`
		Incomplete             bool                          `json:"incomplete"`
		SkippedCount           int                           `json:"skippedCount"`
	}
	TagCandidate struct {
		Media  Media  `json:"media"`
		Reason string `json:"reason,omitempty"`
	}
	TagCandidateMeta struct {
		Incomplete   bool `json:"incomplete"`
		SkippedCount int  `json:"skippedCount"`
	}
	TagCandidatesResponse = util.PaginationResponse[TagCandidate, TagCandidateMeta]
)
