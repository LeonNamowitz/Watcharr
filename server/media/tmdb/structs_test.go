package tmdb

import (
	"encoding/json"
	"testing"
)

func TestAppendedKeywordResponseShapesAndMovieCollection(t *testing.T) {
	var movie MovieDetails
	if err := json.Unmarshal([]byte(`{
		"keywords":{"keywords":[{"id":4344,"name":"musical"}]},
		"belongs_to_collection":{"id":119,"name":"Example Collection"}
	}`), &movie); err != nil {
		t.Fatal(err)
	}
	if len(movie.Keywords.Keywords) != 1 || movie.Keywords.Keywords[0].ID != 4344 || movie.BelongsToCollection == nil || movie.BelongsToCollection.ID != 119 {
		t.Fatalf("movie details = %#v", movie)
	}

	var show ShowDetails
	if err := json.Unmarshal([]byte(`{"keywords":{"results":[{"id":4379,"name":"time travel"}]}}`), &show); err != nil {
		t.Fatal(err)
	}
	if len(show.Keywords.Results) != 1 || show.Keywords.Results[0].ID != 4379 {
		t.Fatalf("show details = %#v", show)
	}

	var standalone MovieDetails
	if err := json.Unmarshal([]byte(`{"belongs_to_collection":null}`), &standalone); err != nil {
		t.Fatal(err)
	}
	if standalone.BelongsToCollection != nil {
		t.Fatalf("standalone movie collection = %#v", standalone.BelongsToCollection)
	}
}
