package stats

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/media/tmdb"
)

func containsYear(years []int, year int) bool {
	for _, y := range years {
		if y == year {
			return true
		}
	}
	return false
}

// Episode dates follow completion activity, including imported and custom dates.
// Rows without completion activity retain their original recorded date.
func (s *Service) loadEpisodes(records []*watchedRecord, q Query, years map[int]bool) ([]*watchedRecord, error) {
	result := []*watchedRecord{}
	if q.Media != "tv" || len(records) == 0 {
		return result, nil
	}
	parents := map[uint]*watchedRecord{}
	ids := []uint{}
	for _, r := range records {
		parents[r.watched.ID] = r
		ids = append(ids, r.watched.ID)
	}
	var episodes []entity.WatchedEpisode
	if err := s.db.Where("watched_id IN ? AND user_id = ?", ids, records[0].watched.UserID).Find(&episodes).Error; err != nil {
		return nil, errors.New("failed to load watched episodes")
	}
	for i := range episodes {
		ep := &episodes[i]
		parent := parents[ep.WatchedID]
		dates := []time.Time{}
		ratedDate := ep.CreatedAt.UTC()
		for _, activity := range parent.watched.Activity {
			var payload struct {
				Season  int    `json:"season"`
				Episode int    `json:"episode"`
				Status  string `json:"status"`
			}
			if json.Unmarshal([]byte(activity.Data), &payload) != nil || payload.Season != ep.SeasonNumber || payload.Episode != ep.EpisodeNumber {
				continue
			}
			d := effectiveDate(activity.CreatedAt, activity.CustomDate)
			added := activity.Type == entity.EPISODE_ADDED || activity.Type == entity.EPISODE_ADDED_JF || activity.Type == entity.EPISODE_ADDED_PLEX
			if added {
				ratedDate = d
			}
			if (added || activity.Type == entity.EPISODE_STATUS_CHANGED) && (payload.Status == string(entity.FINISHED) || (added && payload.Status == "" && ep.Status == entity.FINISHED)) {
				dates = append(dates, d)
			}
		}
		if len(dates) == 0 && ep.Status == entity.FINISHED {
			dates = append(dates, ep.CreatedAt.UTC())
		}
		for _, d := range dates {
			years[d.Year()] = true
		}
		if ep.Rating > 0 {
			years[ratedDate.Year()] = true
		}
		scoped := []time.Time{}
		for _, d := range dates {
			if q.Scope == ScopeLifetime || d.Year() == q.Year {
				scoped = append(scoped, d)
			}
		}
		rating := float64(ep.Rating)
		if q.Scope == ScopeYear && len(scoped) == 0 && ratedDate.Year() != q.Year {
			rating = 0
		}
		if len(scoped) == 0 && rating == 0 {
			continue
		}
		c := *parent.content
		c.Title = fmt.Sprintf("%s · S%dE%d", c.Title, ep.SeasonNumber, ep.EpisodeNumber)
		c.ReleaseDate = nil
		r := &watchedRecord{content: &c, episode: ep, parent: parent, watched: entity.Watched{UserID: ep.UserID, Rating: rating}, plays: scoped}
		r.watched.ID = ep.ID
		result = append(result, r)
	}
	return result, nil
}

type episodeCreditsProvider interface {
	EpisodeCredits(string, string, string) (tmdb.ContentCredits, error)
}

type seasonProvider interface {
	SeasonDetails(string, string) (tmdb.SeasonDetails, error)
}

func (s *Service) enrichEpisodes(records []*watchedRecord, metadata map[string]contentMetadata, status *MetadataStatus) {
	for _, r := range records {
		r.content.Runtime = r.parent.content.Runtime
	}
	provider, ok := s.tmdb.(seasonProvider)
	if !ok {
		return
	}
	var metadataMu sync.Mutex
	type job struct {
		id      int
		season  int
		records []*watchedRecord
	}
	groups := map[string]*job{}
	for _, r := range records {
		key := fmt.Sprintf("%d:%d", r.content.TmdbID, r.episode.SeasonNumber)
		if groups[key] == nil {
			groups[key] = &job{id: r.content.TmdbID, season: r.episode.SeasonNumber}
		}
		groups[key].records = append(groups[key].records, r)
	}
	// Limit concurrent provider calls, as with title enrichment.
	jobs := make(chan *job)
	results := make(chan string, len(groups))
	for i := 0; i < min(metadataWorkers, len(groups)); i++ {
		go func() {
			for j := range jobs {
				details, err := provider.SeasonDetails(strconv.Itoa(j.id), strconv.Itoa(j.season))
				if err != nil {
					results <- fmt.Sprintf("%s (season %d)", strings.Split(j.records[0].content.Title, " · ")[0], j.season)
					continue
				}
				for _, r := range j.records {
					for _, ep := range details.Episodes {
						if ep.EpisodeNumber != r.episode.EpisodeNumber {
							continue
						}
						if ep.Runtime > 0 {
							r.content.Runtime = uint32(ep.Runtime)
						}
						r.episodeName = ep.Name
						r.stillPath = ep.StillPath
						if ep.Name != "" {
							r.content.Title += " · " + ep.Name
						}
						if d, err := time.Parse("2006-01-02", ep.AirDate); err == nil {
							r.content.ReleaseDate = &d
						}
						r.content.VoteAverage = float32(ep.VoteAverage)
						credits := []personCredit{}
						for _, c := range ep.GuestStars {
							credits = append(credits, personCredit{id: c.ID, name: c.Name, profilePath: c.ProfilePath})
						}
						if creditProvider, ok := s.tmdb.(episodeCreditsProvider); ok && len(r.plays) > 0 {
							full, err := creditProvider.EpisodeCredits(strconv.Itoa(j.id), strconv.Itoa(j.season), strconv.Itoa(ep.EpisodeNumber))
							if err != nil {
								metadataMu.Lock()
								status.FailedTitles = append(status.FailedTitles, r.content.Title+" (cast)")
								metadataMu.Unlock()
							} else {
								for _, c := range full.Cast {
									credits = append(credits, personCredit{id: c.ID, name: c.Name, profilePath: c.ProfilePath})
								}
								for _, c := range full.GuestStars {
									credits = append(credits, personCredit{id: c.ID, name: c.Name, profilePath: c.ProfilePath})
								}
							}
						}
						metadataMu.Lock()
						metadata[episodeKey(r)] = contentMetadata{cast: credits}
						metadataMu.Unlock()
					}
				}
				// Signal completion after updating this season's episode records.
				results <- ""
			}
		}()
	}
	go func() {
		for _, j := range groups {
			jobs <- j
		}
		close(jobs)
	}()
	for range groups {
		if failed := <-results; failed != "" {
			metadataMu.Lock()
			status.FailedTitles = append(status.FailedTitles, failed)
			metadataMu.Unlock()
		}
	}
}

func buildEpisodeCast(episodes []*watchedRecord, metadata map[string]contentMetadata) []PersonStat {
	values := map[int]*personAggregate{}
	// Episode credits are added by enrichment under episode-specific keys.
	for _, r := range episodes {
		if len(r.plays) == 0 {
			continue
		}
		key := episodeKey(r)
		for _, c := range metadata[key].cast {
			addPerson(values, c, r)
		}
	}
	return peopleFromAggregates(values)
}

func episodeKey(r *watchedRecord) string {
	return fmt.Sprintf("tv:%d:%d:%d", r.content.TmdbID, r.episode.SeasonNumber, r.episode.EpisodeNumber)
}

func recordKey(r *watchedRecord) string {
	if r.episode != nil {
		return episodeKey(r)
	}
	return contentKey(r.content)
}
