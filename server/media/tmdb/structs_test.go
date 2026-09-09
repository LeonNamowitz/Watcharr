package tmdb

import (
	"encoding/json"
	"testing"
)

func TestAppendedSuggestionMetadataResponseShapes(t *testing.T) {
	var movie MovieDetails
	if err := json.Unmarshal([]byte(`{
		"keywords":{"keywords":[{"id":4344,"name":"musical"}]},
		"credits":{"crew":[{"id":1,"name":"Hans Zimmer","job":"Original Music Composer"}]},
		"belongs_to_collection":{"id":119,"name":"Example Collection"}
	}`), &movie); err != nil {
		t.Fatal(err)
	}
	if len(movie.Keywords.Keywords) != 1 || movie.Keywords.Keywords[0].ID != 4344 || len(movie.Credits.Crew) != 1 || movie.Credits.Crew[0].Job != "Original Music Composer" || movie.BelongsToCollection == nil || movie.BelongsToCollection.ID != 119 {
		t.Fatalf("movie details = %#v", movie)
	}

	var show ShowDetails
	if err := json.Unmarshal([]byte(`{
		"keywords":{"results":[{"id":4379,"name":"time travel"}]},
		"aggregate_credits":{"crew":[{"id":3,"name":"Ludwig Göransson","jobs":[{"job":"Original Music Composer","episode_count":10}]}]}
	}`), &show); err != nil {
		t.Fatal(err)
	}
	if len(show.Keywords.Results) != 1 || show.Keywords.Results[0].ID != 4379 || len(show.AggregateCredits.Crew) != 1 || len(show.AggregateCredits.Crew[0].Jobs) != 1 {
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
