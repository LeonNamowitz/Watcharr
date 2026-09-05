package content

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/watched/addedtocontent"
	"github.com/sbondCo/Watcharr/media/tmdb"
	appRouter "github.com/sbondCo/Watcharr/router"
	"github.com/sbondCo/Watcharr/util"
)

type publicPersonTMDB struct {
	person  tmdb.PersonDetails
	credits tmdb.PersonCombinedCredits
	movie   tmdb.MovieDetails
	show    tmdb.ShowDetails
}

func (p *publicPersonTMDB) MovieDetails(tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	return p.movie, nil
}

func (p *publicPersonTMDB) MovieCredits(string) (tmdb.ContentCredits, error) {
	return tmdb.ContentCredits{}, nil
}

func (p *publicPersonTMDB) ShowDetails(tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	return p.show, nil
}

func (p *publicPersonTMDB) ShowCredits(string) (tmdb.ContentCredits, error) {
	return tmdb.ContentCredits{}, nil
}

func (p *publicPersonTMDB) SeasonDetails(string, string) (tmdb.SeasonDetails, error) {
	return tmdb.SeasonDetails{}, nil
}

func (p *publicPersonTMDB) PersonDetails(string) (tmdb.PersonDetails, error) {
	return p.person, nil
}

func (p *publicPersonTMDB) PersonCredits(string) (tmdb.PersonCombinedCredits, error) {
	return p.credits, nil
}

func (p *publicPersonTMDB) Regions() (tmdb.Regions, error) {
	return tmdb.Regions{}, nil
}

type publicPersonWatched struct {
	validateErr error
	items       []entity.Watched
	optional    *entity.Watched
}

func (p *publicPersonWatched) UpdateWatchedLastViewedSeason(uint, uint, int) error {
	return nil
}

func (p *publicPersonWatched) GetWatchedItemBySupportedMediaId(uint, uint, util.SupportedMedia) (entity.Watched, error) {
	return entity.Watched{}, errors.New("not implemented")
}

func (p *publicPersonWatched) GetWatchedItemsBySupportedMediaIds(uint, []addedtocontent.IdToTypePair) ([]entity.Watched, error) {
	return p.items, nil
}

func (p *publicPersonWatched) GetPublicWatchedItem(uint, string, uint, util.SupportedMedia) (entity.Watched, bool, error) {
	if p.optional == nil {
		return entity.Watched{}, false, errors.New("not on list")
	}
	return *p.optional, true, nil
}

func (p *publicPersonWatched) GetOptionalPublicWatchedItem(uint, string, uint, util.SupportedMedia) (*entity.Watched, bool, error) {
	if p.validateErr != nil {
		return nil, false, p.validateErr
	}
	return p.optional, true, nil
}

func (p *publicPersonWatched) ValidatePublicWatchedList(uint, string) error {
	return p.validateErr
}

func newPublicPersonEngine(
	t *testing.T,
	provider *publicPersonTMDB,
	watched *publicPersonWatched,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")
	br := appRouter.NewBaseRouter(nil, api, &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, nil, watched, provider).AddRoutes()
	return engine
}

func publicPersonRequest(t *testing.T, engine http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestPublicPersonRoutesReturnOwnerScopedWatchedData(t *testing.T) {
	movieContent := entity.Content{TmdbID: 101, Type: entity.MOVIE}
	provider := &publicPersonTMDB{
		person: tmdb.PersonDetails{ID: 7, Name: "Public Actor", ProfilePath: "/actor.jpg"},
		credits: tmdb.PersonCombinedCredits{
			Cast: []tmdb.PersonCombinedCreditsCastResult{{
				ID: 101, Title: "Listed Movie", MediaType: "movie",
			}},
			Crew: []tmdb.PersonCombinedCreditsCrewResult{{
				ID: 202, Title: "Directed Movie", MediaType: "movie",
				Department: "Directing", Job: "Director",
			}},
		},
	}
	watchedItem := entity.Watched{
		Content: &movieContent, Status: entity.FINISHED, Rating: 8,
		Thoughts: "must stay private",
		Activity: []entity.Activity{{Type: entity.RATING_CHANGED, Data: "private"}},
	}
	watchedItem.ID = 44
	watched := &publicPersonWatched{items: []entity.Watched{watchedItem}}
	engine := newPublicPersonEngine(t, provider, watched)

	personRecorder := publicPersonRequest(t, engine, "/api/public/users/1/leon/content/person/7")
	if personRecorder.Code != http.StatusOK {
		t.Fatalf("public person status = %d, want 200; body=%s", personRecorder.Code, personRecorder.Body.String())
	}

	creditsRecorder := publicPersonRequest(t, engine, "/api/public/users/1/leon/content/person/7/credits?creditsType=acting")
	if creditsRecorder.Code != http.StatusOK {
		t.Fatalf("public credits status = %d, want 200; body=%s", creditsRecorder.Code, creditsRecorder.Body.String())
	}
	var response domain.PersonCreditsResponse
	if err := json.Unmarshal(creditsRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode public credits: %v", err)
	}
	if !response.HasActing || !response.HasDirecting || len(response.Credits) != 1 {
		t.Fatalf("unexpected credits response: %#v", response)
	}
	credit := response.Credits[0]
	if credit.Watched.ID != 44 || credit.Watched.Rating != 8 || credit.Watched.Status != entity.FINISHED {
		t.Fatalf("owner watched data = %#v", credit.Watched)
	}
	if credit.Watched.Thoughts != "" || len(credit.Watched.Activity) != 0 {
		t.Fatalf("public credits leaked private data: %#v", credit.Watched)
	}

	directingRecorder := publicPersonRequest(t, engine, "/api/public/users/1/leon/content/person/7/credits?creditsType=directing")
	if directingRecorder.Code != http.StatusOK {
		t.Fatalf("directing credits status = %d, want 200", directingRecorder.Code)
	}
	response = domain.PersonCreditsResponse{}
	if err := json.Unmarshal(directingRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode directing credits: %v", err)
	}
	if len(response.Credits) != 1 || response.Credits[0].Name != "Directed Movie" {
		t.Fatalf("directing credits = %#v", response.Credits)
	}
}

func TestPublicPersonRoutesRejectInvalidOwner(t *testing.T) {
	provider := &publicPersonTMDB{person: tmdb.PersonDetails{Name: "Hidden Actor"}}
	watched := &publicPersonWatched{validateErr: errors.New("private or mismatched owner")}
	engine := newPublicPersonEngine(t, provider, watched)

	for _, path := range []string{
		"/api/public/users/1/wrong/content/person/7",
		"/api/public/users/1/wrong/content/person/7/credits",
	} {
		recorder := publicPersonRequest(t, engine, path)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s status = %d, want 403", path, recorder.Code)
		}
	}
}

func TestOffListPublicMediaDetailsHaveNoOwnerData(t *testing.T) {
	provider := &publicPersonTMDB{movie: tmdb.MovieDetails{
		ContentDetails: tmdb.ContentDetails{ID: 303, Overview: "Public overview"},
		Title:          "Off-list Movie",
		ReleaseDate:    "2026-01-02",
	}, show: tmdb.ShowDetails{
		ContentDetails: tmdb.ContentDetails{ID: 404, Overview: "Public show overview"},
		Name:           "Off-list Show",
		FirstAirDate:   "2026-01-02",
		LastAirDate:    "2026-02-03",
	}}
	engine := newPublicPersonEngine(t, provider, &publicPersonWatched{})

	recorder := publicPersonRequest(t, engine, "/api/public/users/1/leon/content/movie/303")
	if recorder.Code != http.StatusOK {
		t.Fatalf("off-list movie status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var response domain.PublicMediaDetailsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode off-list movie: %v", err)
	}
	if response.Media.Name != "Off-list Movie" || response.Media.Watched.ID != 0 {
		t.Fatalf("off-list response = %#v", response)
	}
	if response.ThoughtsPublic {
		t.Fatal("off-list response unexpectedly marks thoughts as public")
	}

	recorder = publicPersonRequest(t, engine, "/api/public/users/1/leon/content/tv/404")
	if recorder.Code != http.StatusOK {
		t.Fatalf("off-list show status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	response = domain.PublicMediaDetailsResponse{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode off-list show: %v", err)
	}
	if response.Media.Name != "Off-list Show" || response.Media.Watched.ID != 0 {
		t.Fatalf("off-list show response = %#v", response)
	}
	if response.ThoughtsPublic {
		t.Fatal("off-list show unexpectedly marks thoughts as public")
	}
}
