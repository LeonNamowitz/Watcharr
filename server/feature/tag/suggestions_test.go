package tag

import (
	"errors"
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/util"
)

type suggestionTMDB struct {
	movies    map[string]tmdb.MovieDetails
	shows     map[string]tmdb.ShowDetails
	failMovie map[string]bool
	failShow  map[string]bool
}

func (p *suggestionTMDB) MovieDetails(options tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	if options.Params["append_to_response"] != "keywords" {
		return tmdb.MovieDetails{}, errors.New("keywords were not requested")
	}
	if p.failMovie[options.ID] {
		return tmdb.MovieDetails{}, errors.New("movie lookup failed")
	}
	return p.movies[options.ID], nil
}

func (p *suggestionTMDB) ShowDetails(options tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	if options.Params["append_to_response"] != "keywords" {
		return tmdb.ShowDetails{}, errors.New("keywords were not requested")
	}
	if p.failShow[options.ID] {
		return tmdb.ShowDetails{}, errors.New("show lookup failed")
	}
	return p.shows[options.ID], nil
}

func addGenre(details *tmdb.ContentDetails, id int, name string) {
	details.Genres = append(details.Genres, struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{ID: id, Name: name})
}

func mustSave(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSuggestionsUseExactMetadataAndPreservePartialResults(t *testing.T) {
	db := testutil.SetupDB(t)
	user := entity.User{Username: "owner", Password: "password"}
	mustSave(t, db.Create(&user).Error)
	tag := entity.Tag{UserID: user.ID, Name: "Collection"}
	mustSave(t, db.Create(&tag).Error)

	past := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	documentary := entity.Content{TmdbID: 10, Title: "Animated Musical", Type: entity.MOVIE, ReleaseDate: &past}
	musicShow := entity.Content{TmdbID: 20, Title: "Music Show", Type: entity.SHOW, ReleaseDate: &future}
	collectionMovie := entity.Content{TmdbID: 25, Title: "Collection Film", Type: entity.MOVIE, ReleaseDate: &past}
	failing := entity.Content{TmdbID: 30, Title: "Unavailable", Type: entity.MOVIE, ReleaseDate: &past}
	alreadyTagged := entity.Content{TmdbID: 40, Title: "Already Tagged", Type: entity.MOVIE, ReleaseDate: &past}
	game := entity.Game{IgdbID: 50, Name: "A Game"}
	for _, value := range []any{&documentary, &musicShow, &collectionMovie, &failing, &alreadyTagged, &game} {
		mustSave(t, db.Create(value).Error)
	}

	watched := []entity.Watched{
		{UserID: user.ID, ContentID: &documentary.ID, Status: entity.FINISHED},
		{UserID: user.ID, ContentID: &musicShow.ID, Status: entity.FINISHED},
		{UserID: user.ID, ContentID: &collectionMovie.ID, Status: entity.FINISHED},
		{UserID: user.ID, ContentID: &failing.ID, Status: entity.PLANNED},
		{UserID: user.ID, ContentID: &alreadyTagged.ID, Status: entity.FINISHED},
		{UserID: user.ID, GameID: &game.ID, Status: entity.FINISHED},
	}
	for i := range watched {
		mustSave(t, db.Create(&watched[i]).Error)
	}
	mustSave(t, db.Model(&tag).Association("Watched").Append(&watched[4]))

	movie := tmdb.MovieDetails{Title: "Animated Musical"}
	movie.ID = 10
	movie.OriginalLanguage = "en"
	addGenre(&movie.ContentDetails, 99, "Documentary")
	addGenre(&movie.ContentDetails, 16, "Animation")
	movie.Keywords.Keywords = []tmdb.Keyword{{ID: 4344, Name: "musical"}}
	show := tmdb.ShowDetails{Name: "Music Show", FirstAirDate: "2026-12-01"}
	show.ID = 20
	show.OriginalLanguage = "ja"
	addGenre(&show.ContentDetails, 10402, "Music")
	show.Keywords.Results = []tmdb.Keyword{{ID: 4379, Name: "time travel"}}
	collection := tmdb.MovieDetails{Title: "Collection Film"}
	collection.ID = 25
	collection.OriginalLanguage = "en"
	collection.Keywords.Keywords = []tmdb.Keyword{{ID: 9672, Name: "based on true story"}}
	collection.BelongsToCollection = &tmdb.MovieCollection{ID: 119, Name: "Example Collection"}
	provider := &suggestionTMDB{
		movies:    map[string]tmdb.MovieDetails{"10": movie, "25": collection},
		shows:     map[string]tmdb.ShowDetails{"20": show},
		failMovie: map[string]bool{"30": true},
		failShow:  map[string]bool{},
	}
	service := NewService(db, nil, provider)
	service.now = func() time.Time {
		return time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	}

	options, err := service.GetSuggestionOptions(user.ID, tag.ID)
	if err != nil {
		t.Fatalf("GetSuggestionOptions failed: %v", err)
	}
	if !options.Incomplete || options.SkippedCount != 1 {
		t.Fatalf("partial metadata = (%v, %d), want (true, 1)", options.Incomplete, options.SkippedCount)
	}
	if len(options.Genres) != 3 || options.Genres[0].Name != "Animation" || options.Genres[1].Name != "Documentary" || options.Genres[2].Name != "Music" {
		t.Fatalf("genres = %#v", options.Genres)
	}
	if len(options.Keywords) != 3 || options.Keywords[0].Name != "based on true story" || options.Keywords[1].Name != "musical" || options.Keywords[2].Name != "time travel" {
		t.Fatalf("keywords = %#v", options.Keywords)
	}
	if len(options.Languages) != 2 || options.Languages[0].Name != "English" || options.Languages[0].Count != 2 || options.Languages[1].Name != "Japanese" {
		t.Fatalf("languages = %#v", options.Languages)
	}
	if len(options.Collections) != 1 || options.Collections[0].Name != "Example Collection" {
		t.Fatalf("collections = %#v", options.Collections)
	}
	if options.FutureReleaseCount != 1 {
		t.Fatalf("future count = %d, want 1", options.FutureReleaseCount)
	}

	genre, err := service.GetCandidates(user.ID, tag.ID, suggestionKindGenre, 16, "", "", util.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("genre candidates failed: %v", err)
	}
	if len(genre.Results) != 1 || genre.Results[0].Media.Name != "Animated Musical" || genre.Results[0].Reason != "Genre: Animation" {
		t.Fatalf("genre candidates = %#v", genre.Results)
	}
	if !genre.Meta.Incomplete || genre.Meta.SkippedCount != 1 {
		t.Fatalf("genre partial metadata = %#v", genre.Meta)
	}

	keyword, err := service.GetCandidates(user.ID, tag.ID, suggestionKindKeyword, 4344, "", "", util.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("keyword candidates failed: %v", err)
	}
	if len(keyword.Results) != 1 || keyword.Results[0].Media.Name != "Animated Musical" || keyword.Results[0].Reason != "Keyword: musical" {
		t.Fatalf("keyword candidates = %#v", keyword.Results)
	}

	languageCandidates, err := service.GetCandidates(user.ID, tag.ID, suggestionKindLanguage, 0, "en", "", util.PaginationParams{Page: 1, Limit: 10})
	if err != nil || len(languageCandidates.Results) != 2 || languageCandidates.Results[0].Reason != "Original language: English" {
		t.Fatalf("language candidates = %#v, err = %v", languageCandidates.Results, err)
	}

	collectionCandidates, err := service.GetCandidates(user.ID, tag.ID, suggestionKindCollection, 119, "", "collection", util.PaginationParams{Page: 1, Limit: 10})
	if err != nil || len(collectionCandidates.Results) != 1 || collectionCandidates.Results[0].Reason != "Collection: Example Collection" {
		t.Fatalf("collection candidates = %#v, err = %v", collectionCandidates.Results, err)
	}

	upcoming, err := service.GetCandidates(user.ID, tag.ID, suggestionKindFuture, 0, "", "", util.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("future candidates failed: %v", err)
	}
	if len(upcoming.Results) != 1 || upcoming.Results[0].Media.Name != "Music Show" {
		t.Fatalf("future candidates = %#v", upcoming.Results)
	}

	manual, err := service.GetCandidates(user.ID, tag.ID, suggestionKindAll, 0, "", "game", util.PaginationParams{Page: 1, Limit: 1})
	if err != nil {
		t.Fatalf("manual candidates failed: %v", err)
	}
	if manual.TotalResults != 1 || len(manual.Results) != 1 || manual.Results[0].Media.Name != "A Game" {
		t.Fatalf("manual candidates = %#v", manual)
	}
	manual, err = service.GetCandidates(user.ID, tag.ID, suggestionKindAll, 0, "", "", util.PaginationParams{Page: 2, Limit: 3})
	if err != nil || manual.TotalResults != 5 || manual.TotalPages != 2 || len(manual.Results) != 2 {
		t.Fatalf("paginated manual candidates = %#v, err = %v", manual, err)
	}
	empty, err := service.GetCandidates(user.ID, tag.ID, suggestionKindAll, 0, "", "not present", util.PaginationParams{Page: 1, Limit: 10})
	if err != nil || empty.TotalResults != 0 || len(empty.Results) != 0 {
		t.Fatalf("empty manual candidates = %#v, err = %v", empty, err)
	}
	if _, err := service.GetCandidates(user.ID, tag.ID, suggestionKindKeyword, 0, "", "", util.PaginationParams{}); err == nil {
		t.Fatal("keyword candidates should require an exact criterion id")
	}
	if _, err := service.GetCandidates(user.ID, tag.ID, suggestionKindLanguage, 0, "", "", util.PaginationParams{}); err == nil {
		t.Fatal("language candidates should require an exact language code")
	}
}

func TestSuggestionOptionsAllowLibrariesWithoutTMDBContent(t *testing.T) {
	db := testutil.SetupDB(t)
	user := entity.User{Username: "owner", Password: "password"}
	mustSave(t, db.Create(&user).Error)
	tag := entity.Tag{UserID: user.ID, Name: "Games"}
	mustSave(t, db.Create(&tag).Error)
	game := entity.Game{IgdbID: 60, Name: "Only Game"}
	mustSave(t, db.Create(&game).Error)
	mustSave(t, db.Create(&entity.Watched{UserID: user.ID, GameID: &game.ID, Status: entity.FINISHED}).Error)

	options, err := NewService(db, nil, nil).GetSuggestionOptions(user.ID, tag.ID)
	if err != nil {
		t.Fatalf("game-only suggestions failed: %v", err)
	}
	if len(options.Genres) != 0 || len(options.Keywords) != 0 || len(options.Languages) != 0 || len(options.Collections) != 0 || options.FutureReleaseCount != 0 {
		t.Fatalf("game-only suggestions = %#v", options)
	}
}

func TestBulkAddWatchedIsOwnerScopedAtomicAndIdempotent(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "owner", Password: "password"}
	other := entity.User{Username: "other", Password: "password"}
	mustSave(t, db.Create(&owner).Error)
	mustSave(t, db.Create(&other).Error)
	tag := entity.Tag{UserID: owner.ID, Name: "Tag"}
	mustSave(t, db.Create(&tag).Error)
	foreignTag := entity.Tag{UserID: other.ID, Name: "Foreign tag"}
	mustSave(t, db.Create(&foreignTag).Error)
	items := []entity.Watched{
		{UserID: owner.ID, Status: entity.FINISHED},
		{UserID: owner.ID, Status: entity.PLANNED},
		{UserID: other.ID, Status: entity.FINISHED},
	}
	for i := range items {
		mustSave(t, db.Create(&items[i]).Error)
	}

	if _, err := NewService(db, nil, nil).BulkAddWatched(owner.ID, tag.ID, []uint{items[0].ID, items[0].ID}); err == nil {
		t.Fatal("duplicate watched ids should fail")
	}
	if _, err := NewService(db, nil, nil).BulkAddWatched(owner.ID, tag.ID, []uint{items[0].ID, items[2].ID}); err == nil {
		t.Fatal("foreign watched id should fail")
	}
	if _, err := NewService(db, nil, nil).BulkAddWatched(owner.ID, foreignTag.ID, []uint{items[0].ID}); err == nil {
		t.Fatal("foreign tag should fail")
	}
	var count int64
	db.Table("watched_tags").Where("tag_id = ?", tag.ID).Count(&count)
	if count != 0 {
		t.Fatalf("failed bulk request added %d rows", count)
	}

	service := NewService(db, nil, nil)
	added, err := service.BulkAddWatched(owner.ID, tag.ID, []uint{items[0].ID, items[1].ID})
	if err != nil || added != 2 {
		t.Fatalf("bulk add = (%d, %v), want (2, nil)", added, err)
	}
	added, err = service.BulkAddWatched(owner.ID, tag.ID, []uint{items[0].ID, items[1].ID})
	if err != nil || added != 0 {
		t.Fatalf("repeated bulk add = (%d, %v), want (0, nil)", added, err)
	}
}
