package stats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
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

func TestStatsLayoutNormalization(t *testing.T) {
	for _, media := range []string{"movie", "tv", "game"} {
		defaults := defaultSectionOrder(media)
		if got := normalizeSectionOrder(media, nil); !reflect.DeepEqual(got, defaults) {
			t.Fatalf("%s defaults changed: %v", media, got)
		}
		got := normalizeSectionOrder(media, []string{"titles", "retired-section", "titles", "history"})
		if len(got) != len(defaults) || got[0] != "titles" || got[1] != "history" || got[2] != defaults[0] {
			t.Fatalf("%s must remove stale/duplicate sections and append missing sections: %v", media, got)
		}
		for _, order := range [][]string{{"titles", "titles"}, {"unknown"}} {
			if _, err := validateSectionOrder(media, order); err == nil {
				t.Fatalf("%s accepted invalid order %v", media, order)
			}
		}
		broken := "invalid json"
		if got := sectionOrderForOwner(entity.User{StatsLayout: &broken}, media); !reflect.DeepEqual(got, defaults) {
			t.Fatal("unreadable stored settings must fall back to defaults")
		}
	}
	for _, input := range []struct{ media, section string }{{"movie", "companies"}, {"movie", "highest-rated-episodes"}, {"tv", "playtime"}, {"game", "people"}, {"invalid", "titles"}} {
		if _, err := validateSectionOrder(input.media, []string{input.section}); err == nil {
			t.Fatalf("accepted incompatible section: %#v", input)
		}
	}
}

func TestStatsLayoutPersistenceAndOwnerScope(t *testing.T) {
	db := testutil.SetupDB(t)
	owners := []entity.User{{Username: "layout-owner", Password: "password"}, {Username: "layout-viewer", Password: "password"}}
	for i := range owners {
		if err := db.Create(&owners[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	br := appRouter.NewBaseRouter(db, engine.Group("/api"), &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, NewService(db, nil), watchedFeature.NewService(db, nil, nil, nil, nil)).AddRoutes()
	token := func(id uint) string {
		t.Helper()
		value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, entity.TokenClaims{UserID: id, RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte("test"))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	ownerToken, viewerToken := token(owners[0].ID), token(owners[1].ID)
	call := func(method, path, body, auth string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		engine.ServeHTTP(r, req)
		return r
	}
	if r := call(http.MethodPut, "/api/stats/layout", `{"media":"movie","sectionOrder":["titles"]}`, ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("layout write requires auth: %d", r.Code)
	}
	for _, body := range []string{
		`{"media":"other","sectionOrder":[]}`,
		`{"media":"movie","sectionOrder":["titles","titles"]}`,
		`{"media":"movie","sectionOrder":["unknown"]}`,
		`{"media":"movie","sectionOrder":["companies"]}`,
		`{"media":"game","sectionOrder":["people"]}`,
		`{"media":"tv","sectionOrder":["playtime"]}`,
		`{"media":"movie"}`,
		`{"media":"movie","sectionOrder":null}`,
		`{"media":"movie","sectionOrder":"titles"}`,
	} {
		if r := call(http.MethodPut, "/api/stats/layout", body, ownerToken); r.Code != http.StatusBadRequest {
			t.Fatalf("invalid layout %s: %d %s", body, r.Code, r.Body.String())
		}
	}
	expected := map[string][]string{}
	for _, input := range []struct{ media, first string }{{"movie", "titles"}, {"tv", "highest-rated-episodes"}, {"game", "playtime"}} {
		body, _ := json.Marshal(map[string]any{"media": input.media, "sectionOrder": []string{input.first}})
		r := call(http.MethodPut, "/api/stats/layout", string(body), ownerToken)
		if r.Code != http.StatusOK {
			t.Fatalf("save %s: %d %s", input.media, r.Code, r.Body.String())
		}
		var response struct {
			SectionOrder []string `json:"sectionOrder"`
		}
		if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		expected[input.media] = normalizeSectionOrder(input.media, []string{input.first})
		if !reflect.DeepEqual(response.SectionOrder, expected[input.media]) {
			t.Fatalf("save must return normalized order: %v", response.SectionOrder)
		}
	}
	// A separate service and fresh database read demonstrate persisted state.
	for _, media := range []string{"movie", "tv", "game"} {
		for _, query := range []Query{{Scope: ScopeYear, Year: 2025, Media: media}, {Scope: ScopeLifetime, Media: media}} {
			response, err := NewService(db, nil).GetStats(owners[0].ID, query)
			if err != nil || !reflect.DeepEqual(response.SectionOrder, expected[media]) {
				t.Fatalf("persisted %s %s: %v %v", media, query.Scope, response.SectionOrder, err)
			}
		}
	}
	public := "/api/public/users/" + strconv.Itoa(int(owners[0].ID)) + "/layout-owner/stats"
	for _, media := range []string{"movie", "tv", "game"} {
		for _, auth := range []string{"", viewerToken} {
			r := call(http.MethodGet, public+"?media="+media+"&year=all", "", auth)
			var response StatsResponse
			if r.Code != http.StatusOK || json.Unmarshal(r.Body.Bytes(), &response) != nil || !reflect.DeepEqual(response.SectionOrder, expected[media]) || response.Owner.ID != owners[0].ID {
				t.Fatalf("public must use owner's layout: %d %s", r.Code, r.Body.String())
			}
		}
		r := call(http.MethodGet, "/api/stats?media="+media, "", viewerToken)
		var response StatsResponse
		if r.Code != http.StatusOK || json.Unmarshal(r.Body.Bytes(), &response) != nil || !reflect.DeepEqual(response.SectionOrder, defaultSectionOrder(media)) {
			t.Fatalf("viewer must retain own defaults: %d %s", r.Code, r.Body.String())
		}
	}
	if r := call(http.MethodPut, public+"/layout", `{"media":"movie","sectionOrder":[]}`, ownerToken); r.Code != http.StatusNotFound {
		t.Fatalf("public routes must not allow layout writes: %d", r.Code)
	}
	if r := call(http.MethodGet, strings.Replace(public, "/layout-owner/", "/wrong-name/", 1), "", ""); r.Code != http.StatusForbidden {
		t.Fatalf("wrong username must be rejected: %d", r.Code)
	}
	if err := db.Model(&entity.User{}).Where("id = ?", owners[0].ID).Update("private", true).Error; err != nil {
		t.Fatal(err)
	}
	if r := call(http.MethodGet, public, "", viewerToken); r.Code != http.StatusForbidden {
		t.Fatalf("private owner's layout must not be public: %d", r.Code)
	}
	if r := call(http.MethodPut, "/api/stats/layout", `{"media":"movie","sectionOrder":[]}`, ownerToken); r.Code != http.StatusOK {
		t.Fatalf("reset failed: %d %s", r.Code, r.Body.String())
	}
	var owner entity.User
	if err := db.First(&owner, owners[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sectionOrderForOwner(owner, "movie"), defaultSectionOrder("movie")) || !reflect.DeepEqual(sectionOrderForOwner(owner, "tv"), expected["tv"]) || !reflect.DeepEqual(sectionOrderForOwner(owner, "game"), expected["game"]) {
		t.Fatal("reset must preserve other media orders")
	}
	if err := db.Migrator().DropTable(&entity.User{}); err != nil {
		t.Fatal(err)
	}
	if r := call(http.MethodPut, "/api/stats/layout", `{"media":"movie","sectionOrder":[]}`, ownerToken); r.Code != http.StatusInternalServerError {
		t.Fatalf("storage failure must not report success: %d", r.Code)
	}
}
