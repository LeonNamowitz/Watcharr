package stats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	watchedFeature "github.com/sbondCo/Watcharr/feature/watched"
	"github.com/sbondCo/Watcharr/internal/testutil"
	appRouter "github.com/sbondCo/Watcharr/router"
)

func TestReviewLengthsWordCountsAndTies(t *testing.T) {
	record := func(id int, title, media, thoughts string) savedMediaRecord {
		return savedMediaRecord{watched: entity.Watched{Thoughts: thoughts}, card: MediaCard{ID: id, Title: title, Type: media}}
	}
	for _, test := range []struct {
		name     string
		records  []savedMediaRecord
		shortest *ReviewLength
		longest  *ReviewLength
	}{
		{name: "no titles"},
		{name: "blank reviews", records: []savedMediaRecord{record(1, "Blank", "movie", ""), record(2, "Whitespace", "movie", " \n\t\u00a0\u2003 ")}},
		{name: "single Unicode review", records: []savedMediaRecord{record(1, "Solo", "tv", " \tTrès\u00a0bon\u2003film!\n ")}, shortest: &ReviewLength{Item: MediaCard{ID: 1, Title: "Solo", Type: "tv"}, WordCount: 3}, longest: &ReviewLength{Item: MediaCard{ID: 1, Title: "Solo", Type: "tv"}, WordCount: 3}},
		{name: "different lengths", records: []savedMediaRecord{record(1, "Long", "game", "One\n two\tthree, four!"), record(2, "Short", "game", "Great!"), record(3, "Empty", "game", " ")}, shortest: &ReviewLength{Item: MediaCard{ID: 2, Title: "Short", Type: "game"}, WordCount: 1}, longest: &ReviewLength{Item: MediaCard{ID: 1, Title: "Long", Type: "game"}, WordCount: 4}},
		{name: "ties by title type and ID", records: []savedMediaRecord{record(1, "Zulu", "game", "Nice"), record(1, "Alpha", "tv", "Nice"), record(2, "Alpha", "movie", "Nice"), record(9, "Alpha", "game", "Nice"), record(3, "Alpha", "game", "Nice")}, shortest: &ReviewLength{Item: MediaCard{ID: 3, Title: "Alpha", Type: "game"}, WordCount: 1}, longest: &ReviewLength{Item: MediaCard{ID: 3, Title: "Alpha", Type: "game"}, WordCount: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for range 2 {
				got := buildReviewLengths(test.records)
				if got == nil || !reflect.DeepEqual(got.Shortest, test.shortest) || !reflect.DeepEqual(got.Longest, test.longest) {
					t.Fatalf("review lengths: got %#v, want shortest %#v and longest %#v", got, test.shortest, test.longest)
				}
				slices.Reverse(test.records)
			}
		})
	}
}

func TestReviewLengthsMediaPeriodAndPrivacy(t *testing.T) {
	for _, media := range []string{"movie", "tv", "game"} {
		t.Run(media, func(t *testing.T) {
			db := testutil.SetupDB(t)
			private := false
			owner := entity.User{Username: "review-owner", Password: "password", UserSettings: entity.UserSettings{Private: &private, PrivateThoughts: &private}}
			viewer := entity.User{Username: "review-viewer", Password: "password"}
			for _, user := range []*entity.User{&owner, &viewer} {
				if err := db.Create(user).Error; err != nil {
					t.Fatal(err)
				}
			}
			add := func(user uint, id int, kind, thoughts, played string) entity.Watched {
				t.Helper()
				status := entity.FINISHED
				if played == "" {
					status = entity.PLANNED
				}
				var w entity.Watched
				if kind == "game" {
					w = createStatsGame(t, db, user, id, status, 8, nil)
					if err := db.Model(&w).Update("thoughts", thoughts).Error; err != nil {
						t.Fatal(err)
					}
				} else {
					content := entity.Content{TmdbID: id, Title: fmt.Sprintf("Title %03d", id), Type: entity.ContentType(kind)}
					if err := db.Create(&content).Error; err != nil {
						t.Fatal(err)
					}
					w = entity.Watched{UserID: user, ContentID: &content.ID, Status: status, Rating: 8, Thoughts: thoughts}
					if err := db.Create(&w).Error; err != nil {
						t.Fatal(err)
					}
				}
				if played != "" {
					addGameStatsActivity(t, db, w, entity.STATUS_CHANGED, `"FINISHED"`, played, true)
				}
				return w
			}
			short := add(owner.ID, 101, media, "Bravo review", "2025-01-02")
			secret := "REVIEW TEXT MUST NEVER BE INCLUDED"
			long := add(owner.ID, 102, media, secret, "2025-01-03")
			add(owner.ID, 103, media, strings.Repeat("Prior ", 20), "2024-12-01")
			add(owner.ID, 104, media, strings.Repeat("Planned ", 50), "")
			add(owner.ID, 105, media, " \n\u00a0\u2003", "2025-02-01")
			other := "movie"
			if media == "movie" {
				other = "tv"
			}
			add(owner.ID, 106, other, strings.Repeat("Other ", 60), "2025-01-02")
			add(viewer.ID, 107, media, strings.Repeat("Viewer ", 70), "2025-01-02")
			addGameStatsActivity(t, db, short, entity.STATUS_CHANGED, `"FINISHED"`, "2025-03-01", true)
			addGameStatsActivity(t, db, long, entity.THOUGHTS_CHANGED, secret, "2026-01-01", false)
			service := NewService(db, nil)
			for _, expected := range []struct {
				query    Query
				shortest int
				longest  int
			}{
				{Query{Scope: ScopeYear, Year: 2025, Media: media}, 101, 102},
				{Query{Scope: ScopeYear, Year: 2024, Media: media}, 103, 103},
				{Query{Scope: ScopeLifetime, Media: media}, 101, 103},
			} {
				got, err := service.GetStats(owner.ID, expected.query)
				if err != nil {
					t.Fatal(err)
				}
				if got.ReviewLengths == nil || got.ReviewLengths.Shortest == nil || got.ReviewLengths.Longest == nil || got.ReviewLengths.Shortest.Item.ID != expected.shortest || got.ReviewLengths.Longest.Item.ID != expected.longest {
					t.Fatalf("query %#v: unexpected review lengths %#v", expected.query, got.ReviewLengths)
				}
			}
			for _, hidden := range []bool{false, true} {
				got, err := service.GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2030, Media: media, HideReviews: hidden})
				if err != nil {
					t.Fatal(err)
				}
				body, err := json.Marshal(got)
				if err != nil {
					t.Fatal(err)
				}
				if hidden && strings.Contains(string(body), `"reviewLengths"`) || !hidden && !strings.Contains(string(body), `"reviewLengths":{}`) {
					t.Fatalf("empty/hidden response contract: %s", body)
				}
			}

			gin.SetMode(gin.TestMode)
			engine := gin.New()
			br := appRouter.NewBaseRouter(db, engine.Group("/api"), &config.ServerConfig{JWT_SECRET: "test"})
			NewRouter(br, service, watchedFeature.NewService(db, nil, nil, nil, nil)).AddRoutes()
			token := func(id uint) string {
				t.Helper()
				value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, entity.TokenClaims{UserID: id, RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte("test"))
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			ownerToken, viewerToken := token(owner.ID), token(viewer.ID)
			publicPath := fmt.Sprintf("/api/public/users/%d/review-owner/stats?media=%s&year=2025", owner.ID, media)
			privatePath := "/api/stats?media=" + media + "&year=2025"
			get := func(path, auth string) StatsResponse {
				t.Helper()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set("Authorization", auth)
				r := httptest.NewRecorder()
				engine.ServeHTTP(r, req)
				var got StatsResponse
				if r.Code != http.StatusOK || json.Unmarshal(r.Body.Bytes(), &got) != nil {
					t.Fatalf("stats request: %d %s", r.Code, r.Body.String())
				}
				if strings.Contains(r.Body.String(), secret) || strings.Contains(r.Body.String(), `"thoughts"`) || (!got.ReviewsVisible && strings.Contains(r.Body.String(), `"reviewLengths"`)) {
					t.Fatalf("review text or hidden lengths leaked: %s", r.Body.String())
				}
				return got
			}
			for _, route := range []struct{ path, auth string }{{privatePath, ownerToken}, {publicPath, ""}, {publicPath, viewerToken}} {
				got := get(route.path, route.auth)
				if got.Owner.ID != owner.ID || got.ReviewLengths == nil || got.ReviewLengths.Shortest == nil || got.ReviewLengths.Longest == nil || got.ReviewLengths.Shortest.Item.ID != 101 || got.ReviewLengths.Shortest.WordCount != 2 || got.ReviewLengths.Longest.Item.ID != 102 || got.ReviewLengths.Longest.WordCount != 6 {
					t.Fatalf("owner review lengths: %#v", got.ReviewLengths)
				}
			}
			got := get(privatePath, viewerToken)
			if got.Owner.ID != viewer.ID || got.ReviewLengths == nil || got.ReviewLengths.Shortest == nil || got.ReviewLengths.Shortest.Item.ID != 107 {
				t.Fatalf("viewer review lengths: %#v", got.ReviewLengths)
			}
			if err := db.Model(&owner).Update("private_thoughts", true).Error; err != nil {
				t.Fatal(err)
			}
			for _, auth := range []string{"", viewerToken} {
				got := get(publicPath, auth)
				if got.ReviewsVisible || got.ReviewLengths != nil {
					t.Fatalf("private review lengths must be hidden: %#v", got.ReviewLengths)
				}
			}
			got = get(privatePath, ownerToken)
			if !got.ReviewsVisible || got.ReviewLengths == nil || got.ReviewLengths.Shortest == nil {
				t.Fatal("owner must still see own private review lengths")
			}
		})
	}
}
