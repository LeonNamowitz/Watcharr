package tag

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const metadataWorkerCount = 6

type suggestionKind string

const (
	suggestionKindAll     suggestionKind = "all"
	suggestionKindGenre   suggestionKind = "genre"
	suggestionKindCompany suggestionKind = "company"
	suggestionKindFuture  suggestionKind = "future"
)

type contentMetadata struct {
	media     domain.Media
	genres    []domain.TagSuggestionOption
	companies []domain.TagSuggestionOption
}

func (s *Service) getUntaggedWatched(userID uint, tagID uint) ([]entity.Watched, error) {
	var tag entity.Tag
	if err := s.db.Where("id = ? AND user_id = ?", tagID, userID).Take(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag does not exist")
		}
		return nil, errors.New("failed getting tag")
	}

	taggedIDs := s.db.Table("watched_tags").
		Select("watched_id").
		Where("tag_id = ?", tagID)
	var watched []entity.Watched
	res := s.db.Model(&entity.Watched{}).
		Where("user_id = ?", userID).
		Where("id NOT IN (?)", taggedIDs).
		Preload("Content").
		Preload("Game").
		Preload("Game.Poster").
		Find(&watched)
	if res.Error != nil {
		return nil, errors.New("failed getting tag candidates")
	}
	sort.SliceStable(watched, func(i, j int) bool {
		return strings.ToLower(watchedName(&watched[i])) < strings.ToLower(watchedName(&watched[j]))
	})
	return watched, nil
}

func watchedName(w *entity.Watched) string {
	if w.Content != nil {
		return w.Content.Title
	}
	if w.Game != nil {
		return w.Game.Name
	}
	return ""
}

func (s *Service) scanContentMetadata(watched []entity.Watched) ([]contentMetadata, int, error) {
	eligible := make([]entity.Watched, 0, len(watched))
	for _, item := range watched {
		if item.Content != nil {
			eligible = append(eligible, item)
		}
	}
	if len(eligible) == 0 {
		return []contentMetadata{}, 0, nil
	}
	if s.tmdb == nil {
		return nil, len(eligible), errors.New("metadata provider is unavailable")
	}

	type scanResult struct {
		metadata contentMetadata
		err      error
	}
	jobs := make(chan entity.Watched)
	results := make(chan scanResult, len(eligible))
	workerCount := min(metadataWorkerCount, len(eligible))
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for item := range jobs {
				metadata, err := s.fetchContentMetadata(item)
				results <- scanResult{metadata: metadata, err: err}
			}
		}()
	}
	go func() {
		for _, item := range eligible {
			jobs <- item
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()

	metadata := make([]contentMetadata, 0, len(eligible))
	skipped := 0
	for result := range results {
		if result.err != nil {
			skipped++
			continue
		}
		metadata = append(metadata, result.metadata)
	}
	if len(metadata) == 0 {
		return nil, skipped, errors.New("failed getting metadata for tag suggestions")
	}
	sort.SliceStable(metadata, func(i, j int) bool {
		return strings.ToLower(metadata[i].media.Name) < strings.ToLower(metadata[j].media.Name)
	})
	return metadata, skipped, nil
}

func (s *Service) fetchContentMetadata(item entity.Watched) (contentMetadata, error) {
	id := strconv.Itoa(item.Content.TmdbID)
	var genres []domain.TagSuggestionOption
	var companies []domain.TagSuggestionOption
	switch item.Content.Type {
	case entity.MOVIE:
		details, err := s.tmdb.MovieDetails(tmdb.MovieDetailsOptions{
			ID:             id,
			DontRunDBCache: true,
		})
		if err != nil {
			return contentMetadata{}, err
		}
		genres, companies = suggestionMetadata(details.ContentDetails)
	case entity.SHOW:
		details, err := s.tmdb.ShowDetails(tmdb.ShowDetailsOptions{
			ID:             id,
			DontRunDBCache: true,
		})
		if err != nil {
			return contentMetadata{}, err
		}
		genres, companies = suggestionMetadata(details.ContentDetails)
	default:
		return contentMetadata{}, errors.New("unsupported content type")
	}
	media := domain.NewMediaFromWatched(&item, ptr(domain.NewWatchedDtoForLists(&item)))
	return contentMetadata{
		media:     media,
		genres:    genres,
		companies: companies,
	}, nil
}

func suggestionMetadata(details tmdb.ContentDetails) ([]domain.TagSuggestionOption, []domain.TagSuggestionOption) {
	genres := make([]domain.TagSuggestionOption, 0, len(details.Genres))
	for _, genre := range details.Genres {
		genres = append(genres, domain.TagSuggestionOption{ID: genre.ID, Name: genre.Name})
	}
	companies := make([]domain.TagSuggestionOption, 0, len(details.ProductionCompanies))
	for _, company := range details.ProductionCompanies {
		companies = append(companies, domain.TagSuggestionOption{ID: company.ID, Name: company.Name})
	}
	return genres, companies
}

func (s *Service) GetSuggestionOptions(userID uint, tagID uint) (domain.TagSuggestionOptionsResponse, error) {
	watched, err := s.getUntaggedWatched(userID, tagID)
	if err != nil {
		return domain.TagSuggestionOptionsResponse{}, err
	}
	metadata, skipped, metadataErr := s.scanContentMetadata(watched)
	response := domain.TagSuggestionOptionsResponse{
		Genres:       []domain.TagSuggestionOption{},
		Companies:    []domain.TagSuggestionOption{},
		Incomplete:   skipped > 0,
		SkippedCount: skipped,
	}
	response.FutureReleaseCount = len(s.futureCandidates(watched))
	if metadataErr != nil && response.FutureReleaseCount == 0 {
		return response, metadataErr
	}

	genreCounts := map[int]domain.TagSuggestionOption{}
	companyCounts := map[int]domain.TagSuggestionOption{}
	for _, item := range metadata {
		for _, genre := range item.genres {
			genre.Count = genreCounts[genre.ID].Count + 1
			genreCounts[genre.ID] = genre
		}
		for _, company := range item.companies {
			company.Count = companyCounts[company.ID].Count + 1
			companyCounts[company.ID] = company
		}
	}
	for _, genre := range genreCounts {
		response.Genres = append(response.Genres, genre)
	}
	for _, company := range companyCounts {
		response.Companies = append(response.Companies, company)
	}
	sortSuggestionOptions(response.Genres)
	sortSuggestionOptions(response.Companies)
	return response, nil
}

func sortSuggestionOptions(options []domain.TagSuggestionOption) {
	sort.Slice(options, func(i, j int) bool {
		return strings.ToLower(options[i].Name) < strings.ToLower(options[j].Name)
	})
}

func (s *Service) GetCandidates(
	userID uint,
	tagID uint,
	kind suggestionKind,
	criterionID int,
	query string,
	pp util.PaginationParams,
) (domain.TagCandidatesResponse, error) {
	if kind == "" {
		kind = suggestionKindAll
	}
	if (kind == suggestionKindGenre || kind == suggestionKindCompany) && criterionID <= 0 {
		return domain.TagCandidatesResponse{}, errors.New("a suggestion criterion is required")
	}
	if kind != suggestionKindAll && kind != suggestionKindGenre && kind != suggestionKindCompany && kind != suggestionKindFuture {
		return domain.TagCandidatesResponse{}, errors.New("unsupported suggestion kind")
	}

	watched, err := s.getUntaggedWatched(userID, tagID)
	if err != nil {
		return domain.TagCandidatesResponse{}, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	candidates := []domain.TagCandidate{}
	meta := domain.TagCandidateMeta{}

	switch kind {
	case suggestionKindAll:
		for i := range watched {
			if query != "" && !strings.Contains(strings.ToLower(watchedName(&watched[i])), query) {
				continue
			}
			media := domain.NewMediaFromWatched(&watched[i], ptr(domain.NewWatchedDtoForLists(&watched[i])))
			candidates = append(candidates, domain.TagCandidate{Media: media})
		}
	case suggestionKindFuture:
		for _, candidate := range s.futureCandidates(watched) {
			if query == "" || strings.Contains(strings.ToLower(candidate.Media.Name), query) {
				candidates = append(candidates, candidate)
			}
		}
	case suggestionKindGenre, suggestionKindCompany:
		metadata, skipped, err := s.scanContentMetadata(watched)
		if err != nil {
			return domain.TagCandidatesResponse{}, err
		}
		meta.Incomplete = skipped > 0
		meta.SkippedCount = skipped
		for _, item := range metadata {
			options := item.genres
			label := "Genre"
			if kind == suggestionKindCompany {
				options = item.companies
				label = "Studio"
			}
			for _, option := range options {
				if option.ID == criterionID && (query == "" || strings.Contains(strings.ToLower(item.media.Name), query)) {
					candidates = append(candidates, domain.TagCandidate{
						Media:  item.media,
						Reason: fmt.Sprintf("%s: %s", label, option.Name),
					})
					break
				}
			}
		}
	}

	return paginateCandidates(candidates, meta, pp), nil
}

func ptr[T any](value T) *T {
	return &value
}

func (s *Service) futureCandidates(watched []entity.Watched) []domain.TagCandidate {
	today := s.now().UTC().Truncate(24 * time.Hour)
	candidates := []domain.TagCandidate{}
	for i := range watched {
		item := &watched[i]
		if item.Content == nil || item.Content.ReleaseDate == nil || !item.Content.ReleaseDate.After(today) {
			continue
		}
		media := domain.NewMediaFromWatched(item, ptr(domain.NewWatchedDtoForLists(item)))
		candidates = append(candidates, domain.TagCandidate{
			Media:  media,
			Reason: "Releases " + item.Content.ReleaseDate.UTC().Format("2 Jan 2006"),
		})
	}
	return candidates
}

func paginateCandidates(
	candidates []domain.TagCandidate,
	meta domain.TagCandidateMeta,
	pp util.PaginationParams,
) domain.TagCandidatesResponse {
	if pp.Page < 1 {
		pp.Page = 1
	}
	if pp.Limit < 1 {
		pp.Limit = 40
	}
	response := domain.TagCandidatesResponse{
		PaginationParams: pp,
		TotalResults:     int64(len(candidates)),
		Results:          []domain.TagCandidate{},
		Meta:             meta,
	}
	start := (pp.Page - 1) * pp.Limit
	if start < len(candidates) {
		end := min(start+pp.Limit, len(candidates))
		response.Results = candidates[start:end]
	}
	response.Finished(pp)
	return response
}

func (s *Service) BulkAddWatched(userID uint, tagID uint, watchedIDs []uint) (int64, error) {
	if len(watchedIDs) == 0 {
		return 0, errors.New("no watched ids provided")
	}
	seen := make(map[uint]struct{}, len(watchedIDs))
	for _, watchedID := range watchedIDs {
		if watchedID == 0 {
			return 0, errors.New("invalid watched id")
		}
		if _, exists := seen[watchedID]; exists {
			return 0, errors.New("duplicate watched id")
		}
		seen[watchedID] = struct{}{}
	}

	var added int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var tagCount int64
		if err := tx.Model(&entity.Tag{}).
			Where("id = ? AND user_id = ?", tagID, userID).
			Count(&tagCount).Error; err != nil || tagCount != 1 {
			return errors.New("tag does not exist")
		}
		var watchedCount int64
		if err := tx.Model(&entity.Watched{}).
			Where("user_id = ? AND id IN ?", userID, watchedIDs).
			Count(&watchedCount).Error; err != nil {
			return errors.New("failed validating watched entries")
		}
		if watchedCount != int64(len(watchedIDs)) {
			return errors.New("one or more watched entries do not belong to user")
		}

		rows := make([]map[string]any, 0, len(watchedIDs))
		for _, watchedID := range watchedIDs {
			rows = append(rows, map[string]any{"tag_id": tagID, "watched_id": watchedID})
		}
		res := tx.Table("watched_tags").Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
		if res.Error != nil {
			return errors.New("failed adding watched entries to tag")
		}
		added = res.RowsAffected
		return nil
	})
	return added, err
}
