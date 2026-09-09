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
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	metadataWorkerCount      = 6
	originalMusicComposerJob = "Original Music Composer"
)

type suggestionKind string

const (
	suggestionKindAll        suggestionKind = "all"
	suggestionKindGenre      suggestionKind = "genre"
	suggestionKindKeyword    suggestionKind = "keyword"
	suggestionKindComposer   suggestionKind = "composer"
	suggestionKindLanguage   suggestionKind = "language"
	suggestionKindCollection suggestionKind = "collection"
	suggestionKindFuture     suggestionKind = "future"
)

type contentMetadata struct {
	media            domain.Media
	genres           []domain.TagSuggestionOption
	keywords         []domain.TagSuggestionOption
	composers        []domain.TagSuggestionOption
	originalLanguage string
	collection       *domain.TagSuggestionOption
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
	var keywords []domain.TagSuggestionOption
	var composers []domain.TagSuggestionOption
	var originalLanguage string
	var collection *domain.TagSuggestionOption
	switch item.Content.Type {
	case entity.MOVIE:
		details, err := s.tmdb.MovieDetails(tmdb.MovieDetailsOptions{
			ID:             id,
			Params:         map[string]string{"append_to_response": "keywords,credits"},
			DontRunDBCache: true,
		})
		if err != nil {
			return contentMetadata{}, err
		}
		genres = genreSuggestionMetadata(details.ContentDetails)
		keywords = keywordSuggestionMetadata(details.Keywords.Keywords)
		composers = movieComposerSuggestionMetadata(details.Credits)
		originalLanguage = details.OriginalLanguage
		if details.BelongsToCollection != nil {
			collection = &domain.TagSuggestionOption{
				ID:   details.BelongsToCollection.ID,
				Name: details.BelongsToCollection.Name,
			}
		}
	case entity.SHOW:
		details, err := s.tmdb.ShowDetails(tmdb.ShowDetailsOptions{
			ID:             id,
			Params:         map[string]string{"append_to_response": "keywords,aggregate_credits"},
			DontRunDBCache: true,
		})
		if err != nil {
			return contentMetadata{}, err
		}
		genres = genreSuggestionMetadata(details.ContentDetails)
		keywords = keywordSuggestionMetadata(details.Keywords.Results)
		composers = showComposerSuggestionMetadata(details.AggregateCredits)
		originalLanguage = details.OriginalLanguage
	default:
		return contentMetadata{}, errors.New("unsupported content type")
	}
	media := domain.NewMediaFromWatched(&item, ptr(domain.NewWatchedDtoForLists(&item)))
	return contentMetadata{
		media:            media,
		genres:           genres,
		keywords:         keywords,
		composers:        composers,
		originalLanguage: originalLanguage,
		collection:       collection,
	}, nil
}

func genreSuggestionMetadata(details tmdb.ContentDetails) []domain.TagSuggestionOption {
	genres := make([]domain.TagSuggestionOption, 0, len(details.Genres))
	for _, genre := range details.Genres {
		genres = append(genres, domain.TagSuggestionOption{ID: genre.ID, Name: genre.Name})
	}
	return genres
}

func keywordSuggestionMetadata(keywords []tmdb.Keyword) []domain.TagSuggestionOption {
	options := make([]domain.TagSuggestionOption, 0, len(keywords))
	for _, keyword := range keywords {
		options = append(options, domain.TagSuggestionOption{ID: keyword.ID, Name: keyword.Name})
	}
	return options
}

func movieComposerSuggestionMetadata(credits tmdb.ContentCredits) []domain.TagSuggestionOption {
	composers := make(map[int]domain.TagSuggestionOption)
	for _, crew := range credits.Crew {
		if crew.Job == originalMusicComposerJob {
			composers[crew.ID] = domain.TagSuggestionOption{ID: crew.ID, Name: crew.Name}
		}
	}
	return suggestionOptionValues(composers)
}

func showComposerSuggestionMetadata(credits tmdb.AggregateContentCredits) []domain.TagSuggestionOption {
	composers := make(map[int]domain.TagSuggestionOption)
	for _, crew := range credits.Crew {
		for _, job := range crew.Jobs {
			if job.Job == originalMusicComposerJob {
				composers[crew.ID] = domain.TagSuggestionOption{ID: crew.ID, Name: crew.Name}
				break
			}
		}
	}
	return suggestionOptionValues(composers)
}

func suggestionOptionValues(options map[int]domain.TagSuggestionOption) []domain.TagSuggestionOption {
	values := make([]domain.TagSuggestionOption, 0, len(options))
	for _, option := range options {
		values = append(values, option)
	}
	sortSuggestionOptions(values)
	return values
}

func (s *Service) GetSuggestionOptions(userID uint, tagID uint) (domain.TagSuggestionOptionsResponse, error) {
	watched, err := s.getUntaggedWatched(userID, tagID)
	if err != nil {
		return domain.TagSuggestionOptionsResponse{}, err
	}
	metadata, skipped, metadataErr := s.scanContentMetadata(watched)
	response := domain.TagSuggestionOptionsResponse{
		Genres:       []domain.TagSuggestionOption{},
		Keywords:     []domain.TagSuggestionOption{},
		Composers:    []domain.TagSuggestionOption{},
		Languages:    []domain.TagLanguageSuggestionOption{},
		Collections:  []domain.TagSuggestionOption{},
		Incomplete:   skipped > 0,
		SkippedCount: skipped,
	}
	response.FutureReleaseCount = len(s.futureCandidates(watched))
	if metadataErr != nil && response.FutureReleaseCount == 0 {
		return response, metadataErr
	}

	genreCounts := map[int]domain.TagSuggestionOption{}
	keywordCounts := map[int]domain.TagSuggestionOption{}
	composerCounts := map[int]domain.TagSuggestionOption{}
	languageCounts := map[string]domain.TagLanguageSuggestionOption{}
	collectionCounts := map[int]domain.TagSuggestionOption{}
	for _, item := range metadata {
		for _, genre := range item.genres {
			genre.Count = genreCounts[genre.ID].Count + 1
			genreCounts[genre.ID] = genre
		}
		for _, keyword := range item.keywords {
			keyword.Count = keywordCounts[keyword.ID].Count + 1
			keywordCounts[keyword.ID] = keyword
		}
		for _, composer := range item.composers {
			composer.Count = composerCounts[composer.ID].Count + 1
			composerCounts[composer.ID] = composer
		}
		if item.originalLanguage != "" {
			option := languageCounts[item.originalLanguage]
			option.Code = item.originalLanguage
			option.Name = languageName(item.originalLanguage)
			option.Count++
			languageCounts[item.originalLanguage] = option
		}
		if item.collection != nil {
			collection := *item.collection
			collection.Count = collectionCounts[collection.ID].Count + 1
			collectionCounts[collection.ID] = collection
		}
	}
	for _, genre := range genreCounts {
		response.Genres = append(response.Genres, genre)
	}
	for _, keyword := range keywordCounts {
		response.Keywords = append(response.Keywords, keyword)
	}
	for _, composer := range composerCounts {
		response.Composers = append(response.Composers, composer)
	}
	for _, originalLanguage := range languageCounts {
		response.Languages = append(response.Languages, originalLanguage)
	}
	for _, collection := range collectionCounts {
		response.Collections = append(response.Collections, collection)
	}
	sortSuggestionOptions(response.Genres)
	sortSuggestionOptions(response.Keywords)
	sortSuggestionOptions(response.Composers)
	sortSuggestionOptions(response.Collections)
	sort.Slice(response.Languages, func(i, j int) bool {
		return strings.ToLower(response.Languages[i].Name) < strings.ToLower(response.Languages[j].Name)
	})
	return response, nil
}

func languageName(code string) string {
	tag, err := language.Parse(code)
	if err != nil {
		return strings.ToUpper(code)
	}
	name := display.Languages(language.English).Name(tag)
	if name == "" {
		return strings.ToUpper(code)
	}
	return name
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
	originalLanguage string,
	query string,
	pp util.PaginationParams,
) (domain.TagCandidatesResponse, error) {
	if kind == "" {
		kind = suggestionKindAll
	}
	if (kind == suggestionKindGenre || kind == suggestionKindKeyword || kind == suggestionKindComposer || kind == suggestionKindCollection) && criterionID <= 0 {
		return domain.TagCandidatesResponse{}, errors.New("a suggestion criterion is required")
	}
	if kind == suggestionKindLanguage && strings.TrimSpace(originalLanguage) == "" {
		return domain.TagCandidatesResponse{}, errors.New("a language criterion is required")
	}
	if kind != suggestionKindAll && kind != suggestionKindGenre && kind != suggestionKindKeyword && kind != suggestionKindComposer && kind != suggestionKindLanguage && kind != suggestionKindCollection && kind != suggestionKindFuture {
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
	case suggestionKindGenre, suggestionKindKeyword, suggestionKindComposer, suggestionKindLanguage, suggestionKindCollection:
		metadata, skipped, err := s.scanContentMetadata(watched)
		if err != nil {
			return domain.TagCandidatesResponse{}, err
		}
		meta.Incomplete = skipped > 0
		meta.SkippedCount = skipped
		for _, item := range metadata {
			if query != "" && !strings.Contains(strings.ToLower(item.media.Name), query) {
				continue
			}
			if kind == suggestionKindLanguage {
				if item.originalLanguage == originalLanguage {
					candidates = append(candidates, domain.TagCandidate{
						Media:  item.media,
						Reason: "Original language: " + languageName(originalLanguage),
					})
				}
				continue
			}
			if kind == suggestionKindCollection {
				if item.collection != nil && item.collection.ID == criterionID {
					candidates = append(candidates, domain.TagCandidate{
						Media:  item.media,
						Reason: "Collection: " + item.collection.Name,
					})
				}
				continue
			}

			options := item.genres
			label := "Genre"
			if kind == suggestionKindKeyword {
				options = item.keywords
				label = "Keyword"
			} else if kind == suggestionKindComposer {
				options = item.composers
				label = "Composer"
			}
			for _, option := range options {
				if option.ID == criterionID {
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
