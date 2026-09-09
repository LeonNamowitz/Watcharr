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
	TagSuggestionOptionsResponse struct {
		Genres             []TagSuggestionOption `json:"genres"`
		Companies          []TagSuggestionOption `json:"companies"`
		FutureReleaseCount int                   `json:"futureReleaseCount"`
		Incomplete         bool                  `json:"incomplete"`
		SkippedCount       int                   `json:"skippedCount"`
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
