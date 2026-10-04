package tag

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/util"
)

type cacheTMDB struct {
	calls  atomic.Int32
	fail   atomic.Bool
	delay  time.Duration
	mu     sync.Mutex
	params []string
}

func (p *cacheTMDB) MovieDetails(options tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	p.calls.Add(1)
	p.mu.Lock()
	p.params = append(p.params, options.Params["append_to_response"])
	p.mu.Unlock()
	time.Sleep(p.delay)
	if p.fail.Load() {
		return tmdb.MovieDetails{}, errors.New("unavailable")
	}
	details := tmdb.MovieDetails{}
	addGenre(&details.ContentDetails, 16, "Animation")
	details.OriginalLanguage = "en"
	details.Keywords.Keywords = []tmdb.Keyword{{ID: 7, Name: "musical"}}
	return details, nil
}

func (p *cacheTMDB) ShowDetails(options tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	return tmdb.ShowDetails{}, errors.New("unexpected show lookup")
}

func TestMetadataCachePersistsFacetsAndUsesCurrentOwnerData(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "owner", Password: "password"}
	viewer := entity.User{Username: "viewer", Password: "password"}
	mustSave(t, db.Create(&owner).Error)
	mustSave(t, db.Create(&viewer).Error)
	tag := entity.Tag{UserID: owner.ID, Name: "Animation"}
	otherTag := entity.Tag{UserID: viewer.ID, Name: "Animation"}
	mustSave(t, db.Create(&tag).Error)
	mustSave(t, db.Create(&otherTag).Error)
	content := entity.Content{TmdbID: 10, Type: entity.MOVIE, Title: "Private owner title"}
	mustSave(t, db.Create(&content).Error)
	ownerItem := entity.Watched{UserID: owner.ID, ContentID: &content.ID, Status: entity.FINISHED}
	viewerItem := entity.Watched{UserID: viewer.ID, ContentID: &content.ID, Status: entity.PLANNED}
	mustSave(t, db.Create(&ownerItem).Error)
	mustSave(t, db.Create(&viewerItem).Error)

	provider := &cacheTMDB{}
	dir := t.TempDir()
	first := NewService(db, nil, provider, dir)
	options, err := first.getSuggestionOptions(owner.ID, tag.ID, suggestionKindGenre)
	if err != nil || len(options.Genres) != 1 || options.Genres[0].Count != 1 || provider.calls.Load() != 1 {
		t.Fatalf("first scan: options=%+v err=%v calls=%d", options, err, provider.calls.Load())
	}
	if provider.params[0] != "" {
		t.Fatal("genre scan must not request keywords or credits")
	}
	data, err := os.ReadFile(filepath.Join(dir, "movie-10-basic.json"))
	if err != nil || strings.Contains(strings.ToLower(string(data)), "watched") || strings.Contains(string(data), "Private owner title") {
		t.Fatalf("cache must contain only provider facets: %s, err=%v", data, err)
	}
	mustSave(t, db.Model(&content).Update("title", "Renamed").Error)
	second := NewService(db, nil, provider, dir)
	candidates, err := second.GetCandidates(viewer.ID, otherTag.ID, suggestionKindGenre, 16, "", "", "", util.PaginationParams{Page: 1, Limit: 30})
	if err != nil || len(candidates.Results) != 1 || candidates.Results[0].Media.Name != "Renamed" || candidates.Results[0].Media.Watched.ID != viewerItem.ID || provider.calls.Load() != 1 {
		t.Fatalf("restart must reuse facets with current owner data: %+v err=%v calls=%d", candidates, err, provider.calls.Load())
	}
	if _, err := second.getSuggestionOptions(viewer.ID, tag.ID, suggestionKindGenre); err == nil {
		t.Fatal("warm cache must not bypass tag ownership")
	}
	mustSave(t, db.Model(&otherTag).Association("Watched").Append(&viewerItem))
	options, err = second.getSuggestionOptions(viewer.ID, otherTag.ID, suggestionKindGenre)
	if err != nil || len(options.Genres) != 0 {
		t.Fatal("warm cache must exclude newly tagged items")
	}
	third := NewService(db, nil, provider, dir)
	third.now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	if _, err := third.getSuggestionOptions(owner.ID, tag.ID, suggestionKindGenre); err != nil || provider.calls.Load() != 2 {
		t.Fatal("expired disk cache must fetch fresh facets")
	}
}

func TestSuggestionCategoriesFetchOnlyRequiredMetadata(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "owner", Password: "password"}
	mustSave(t, db.Create(&owner).Error)
	tag := entity.Tag{UserID: owner.ID, Name: "Tag"}
	mustSave(t, db.Create(&tag).Error)
	content := entity.Content{TmdbID: 10, Type: entity.MOVIE, Title: "Film"}
	mustSave(t, db.Create(&content).Error)
	item := entity.Watched{UserID: owner.ID, ContentID: &content.ID, Status: entity.FINISHED}
	mustSave(t, db.Create(&item).Error)
	provider := &cacheTMDB{}
	service := NewService(db, nil, provider)
	for _, kind := range []suggestionKind{suggestionKindGameGenre, suggestionKindGameMode, suggestionKindGameCategory, suggestionKindGameFuture, suggestionKindFuture} {
		if _, err := service.getSuggestionOptions(owner.ID, tag.ID, kind); err != nil {
			t.Fatal(err)
		}
	}
	if provider.calls.Load() != 0 {
		t.Fatal("game/future options must not call TMDB")
	}
	for _, kind := range []suggestionKind{suggestionKindGenre, suggestionKindLanguage, suggestionKindCollection, suggestionKindKeyword, suggestionKindComposer} {
		if _, err := service.getSuggestionOptions(owner.ID, tag.ID, kind); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(provider.params, "|") != "|keywords|credits" {
		t.Fatalf("expected one shared basic lookup plus keyword/credit lookups: %v", provider.params)
	}
	if metadataParams("composers", entity.SHOW)["append_to_response"] != "aggregate_credits" {
		t.Fatal("show composers require aggregate credits")
	}
}

func TestMetadataCacheCoalescesRequestsAndRetriesFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		provider := &cacheTMDB{delay: time.Second}
		service := NewService(nil, nil, provider)
		content := entity.Content{TmdbID: 10, Type: entity.MOVIE}
		item := entity.Watched{Content: &content}
		var callers sync.WaitGroup
		for range 20 {
			callers.Go(func() {
				if _, err := service.cachedContentMetadata(item, "basic"); err != nil {
					t.Error(err)
				}
			})
		}
		callers.Wait()
		if provider.calls.Load() != 1 {
			t.Fatalf("concurrent scans fetched the same metadata %d times", provider.calls.Load())
		}
		provider.fail.Store(true)
		if _, err := service.cachedContentMetadata(item, "keywords"); err == nil {
			t.Fatal("expected failed keyword lookup")
		}
		provider.fail.Store(false)
		if _, err := service.cachedContentMetadata(item, "keywords"); err != nil || provider.calls.Load() != 3 {
			t.Fatal("failed lookups must be retried")
		}
	})
}
