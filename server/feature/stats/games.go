package stats

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/media/igdb"
)

type GameStats struct {
	Platforms            []BarStat     `json:"platforms"`
	Modes                []BarStat     `json:"modes"`
	Themes               []BarStat     `json:"themes"`
	Perspectives         []BarStat     `json:"perspectives"`
	Developers           []BarStat     `json:"developers"`
	Publishers           []BarStat     `json:"publishers"`
	Statuses             []PieStat     `json:"statuses"`
	Completion           []PieStat     `json:"completion"`
	CompletionPercentage float64       `json:"completionPercentage"`
	Completions          ActivityStats `json:"completions"`
	Playtime             *GamePlaytime `json:"playtime,omitempty"`
}

type GamePlaytime struct {
	TotalHours    uint64           `json:"totalHours"`
	AverageHours  float64          `json:"averageHours"`
	MedianHours   float64          `json:"medianHours"`
	RecordedGames int              `json:"recordedGames"`
	MostPlayed    []MediaCard      `json:"mostPlayed"`
	Distribution  []PieStat        `json:"distribution"`
	ByRating      []PlaytimeRating `json:"byRating"`
}

type PlaytimeRating struct {
	Rating float64     `json:"rating"`
	Hours  uint64      `json:"hours"`
	Items  []MediaCard `json:"items"`
}

type gameRecord struct {
	watched        entity.Watched
	card           MediaCard
	progress       []time.Time
	completions    []time.Time
	allCompletions []time.Time
	backlogDates   []time.Time
	categories     map[string][]string
}

func activityStatus(data string) entity.WatchedStatus {
	var payload struct {
		Status entity.WatchedStatus `json:"status"`
	}
	if json.Unmarshal([]byte(data), &payload) == nil && payload.Status.IsValid() {
		return payload.Status
	}
	status := entity.WatchedStatus(strings.ToUpper(strings.Trim(strings.TrimSpace(data), `"`)))
	if status.IsValid() {
		return status
	}
	return ""
}

func makeGameRecord(w entity.Watched, lifetime bool) *gameRecord {
	g := w.Game
	c := MediaCard{ID: g.IgdbID, Type: "game", Title: g.Name, CoverID: g.CoverID,
		Rating: w.Rating, CommunityRating: g.Rating / 10, VoteCount: uint32(max(g.RatingCount, 0))}
	if g.Poster != nil {
		c.PosterPath = g.Poster.Path
	}
	if g.ReleaseDate != nil && !g.ReleaseDate.IsZero() {
		c.Date = g.ReleaseDate.UTC().Format("2006-01-02")
		c.ReleaseYear = g.ReleaseDate.UTC().Year()
	}
	if lifetime {
		c.PlaytimeHours = w.PlaytimeHours
	}
	r := &gameRecord{watched: w, card: c, categories: map[string][]string{
		"genres": strings.Split(g.Genres, "|"), "platforms": strings.Split(g.Platforms, "|"), "modes": strings.Split(g.GameModes, "|"),
	}}
	progressDays := make(map[string]time.Time)
	for _, a := range w.Activity {
		d := effectiveDate(a.CreatedAt, a.CustomDate)
		status := activityStatus(a.Data)
		statusEvent := isAddActivity(a.Type) || a.Type == entity.STATUS_CHANGED || a.Type == entity.STATUS_CHANGED_AUTO
		if a.CountAsPlay {
			r.completions = append(r.completions, d)
		}
		if statusEvent && status == entity.PLANNED {
			r.backlogDates = append(r.backlogDates, d)
		}
		if a.CountAsPlay || (statusEvent && status != "" && status != entity.PLANNED) {
			key := d.Format("2006-01-02")
			if previous, exists := progressDays[key]; !exists || d.Before(previous) {
				progressDays[key] = d
			}
		}
	}
	if len(progressDays) == 0 && w.Status.IsValid() && w.Status != entity.PLANNED && !w.CreatedAt.IsZero() {
		d := w.CreatedAt.UTC()
		progressDays[d.Format("2006-01-02")] = d
	}
	for _, d := range progressDays {
		r.progress = append(r.progress, d)
	}
	sort.Slice(r.progress, func(i, j int) bool { return r.progress[i].Before(r.progress[j]) })
	sort.Slice(r.completions, func(i, j int) bool { return r.completions[i].Before(r.completions[j]) })
	r.allCompletions = append([]time.Time{}, r.completions...)
	c.Plays = len(r.completions)
	r.card.Plays = c.Plays
	return r
}

func gameDatesInScope(dates []time.Time, q Query) []time.Time {
	result := make([]time.Time, 0)
	for _, d := range dates {
		if q.Scope == ScopeLifetime || d.Year() == q.Year {
			result = append(result, d)
		}
	}
	return result
}

func gameCompleted(r *gameRecord, q Query) bool {
	return len(r.completions) > 0 || (q.Scope == ScopeLifetime && r.watched.Status == entity.FINISHED)
}

func (s *Service) getGameStats(userID uint, q Query) (StatsResponse, error) {
	var owner entity.User
	if err := s.db.Preload("Avatar").First(&owner, userID).Error; err != nil {
		return StatsResponse{}, errors.New("failed to load stats owner")
	}
	var watched []entity.Watched
	if err := s.db.Where("user_id = ? AND game_id IS NOT NULL", userID).Order("id ASC").
		Preload("Game.Poster").Preload("Activity", "user_id = ?", userID).Find(&watched).Error; err != nil {
		return StatsResponse{}, errors.New("failed to load games")
	}
	all, scoped, backlog, additions := []*gameRecord{}, []*gameRecord{}, []*gameRecord{}, []*gameRecord{}
	years := map[int]bool{time.Now().UTC().Year(): true}
	for _, w := range watched {
		if w.Game == nil {
			continue
		}
		r := makeGameRecord(w, q.Scope == ScopeLifetime)
		all = append(all, r)
		for _, d := range append(append([]time.Time{}, r.progress...), r.backlogDates...) {
			years[d.Year()] = true
		}
		if len(gameDatesInScope(r.backlogDates, q)) > 0 {
			additions = append(additions, r)
		}
		if w.Status == entity.PLANNED && len(r.progress) == 0 {
			backlog = append(backlog, r)
		}
		if len(gameDatesInScope(r.progress, q)) > 0 {
			scoped = append(scoped, r)
		}
	}
	// Enrich only cards needed by this view; history uses saved metadata.
	enrich := append(append(append([]*gameRecord{}, scoped...), backlog...), additions...)
	failures := s.enrichGames(enrich)
	history := gameHistory(all, q.HideReviews)
	for i, r := range scoped {
		clone := *r
		clone.progress = gameDatesInScope(r.progress, q)
		clone.completions = gameDatesInScope(r.completions, q)
		clone.card.Plays = len(clone.completions)
		scoped[i] = &clone
	}
	response := StatsResponse{Scope: q.Scope, Year: q.Year, Media: "game", Owner: owner.GetSafe(), ReviewsVisible: !q.HideReviews,
		History: history, Posters: gameCards(scoped), Watchlist: gameBacklogPicks(backlog, userID, q),
		Genres: gameBars(scoped, "genres"), Countries: []BarStat{}, Languages: []BarStat{}, Studios: []PersonStat{},
		People: PeopleStats{Cast: []PersonStat{}, Directors: []PersonStat{}}, Crew: []CrewDepartment{}, Episodes: []MediaCard{},
		Decades: []DecadeStat{}, HighestRatedEpisodes: HighestRated{Current: []MediaCard{}, Older: []MediaCard{}},
		Metadata: MetadataStatus{Partial: len(failures) > 0, FailedTitles: failures},
	}
	for y := range years {
		response.AvailableYears = append(response.AvailableYears, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(response.AvailableYears)))
	response.Summary = gameSummary(scoped, q)
	hours := float64(0)
	for _, r := range scoped {
		if q.Scope == ScopeYear && (len(r.allCompletions) == 0 || r.allCompletions[0].Year() != q.Year) {
			continue
		}
		if r.watched.PlaytimeHours != nil {
			hours += float64(*r.watched.PlaytimeHours)
		}
	}
	response.Summary.Hours = &hours
	response.HighestRated = gameHighestRated(scoped, q)
	if q.Scope == ScopeLifetime {
		response.Decades = gameDecades(scoped)
	}
	response.Activity = gameActivity(scoped, false, q)
	response.Milestones = gameMilestones(scoped)
	response.Breakdown = gameBreakdown(scoped, additions, q)
	response.HighsLows, response.RatingDifferences = gameHighsLows(scoped)
	games := &GameStats{Platforms: gameBars(scoped, "platforms"), Modes: gameBars(scoped, "modes"),
		Themes: gameBars(scoped, "themes"), Perspectives: gameBars(scoped, "perspectives"),
		Developers: gameBars(scoped, "developers"), Publishers: gameBars(scoped, "publishers"),
		Statuses: []PieStat{}, Completion: []PieStat{}, Completions: gameActivity(scoped, true, q)}
	statusKeys, completionKeys := map[string][]string{}, map[string][]string{}
	for _, r := range scoped {
		status := map[entity.WatchedStatus]string{entity.FINISHED: "Finished", entity.WATCHING: "Playing", entity.HOLD: "On hold", entity.DROPPED: "Dropped", entity.PLANNED: "Planned"}[r.watched.Status]
		statusKeys[status] = append(statusKeys[status], gameKey(r))
		label := "Not completed"
		if gameCompleted(r, q) {
			label = "Completed"
		}
		completionKeys[label] = append(completionKeys[label], gameKey(r))
	}
	for _, label := range []string{"Finished", "Playing", "On hold", "Dropped", "Planned"} {
		games.Statuses = append(games.Statuses, PieStat{Label: label, Count: len(statusKeys[label]), TitleKeys: statusKeys[label]})
	}
	for _, label := range []string{"Completed", "Not completed"} {
		games.Completion = append(games.Completion, PieStat{Label: label, Count: len(completionKeys[label]), TitleKeys: completionKeys[label]})
	}
	if len(scoped) > 0 {
		games.CompletionPercentage = 100 * float64(len(completionKeys["Completed"])) / float64(len(scoped))
	}
	if q.Scope == ScopeLifetime {
		games.Playtime = gamePlaytime(scoped)
		if n := len(games.Playtime.MostPlayed); n > 0 {
			most, least := games.Playtime.MostPlayed[0], games.Playtime.MostPlayed[n-1]
			response.HighsLows.MostPlaytime, response.HighsLows.LeastPlaytime = &most, &least
		}
	}
	response.Games = games
	return response, nil
}

func (s *Service) enrichGames(records []*gameRecord) []string {
	unique := make(map[int]*gameRecord)
	for _, r := range records {
		unique[r.card.ID] = r
	}
	ids := make([]int, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	metadata := map[int]igdb.GameStatsDetails{}
	if s.igdb != nil && len(ids) > 0 {
		metadata, _ = s.igdb.GameStatsDetails(ids)
	}
	failures := []string{}
	for _, id := range ids {
		if _, ok := metadata[id]; !ok {
			failures = append(failures, unique[id].card.Title)
		}
	}
	for _, r := range records {
		details, ok := metadata[r.card.ID]
		if !ok {
			continue
		}
		r.card.CommunityRating = details.Rating / 10
		r.card.VoteCount = uint32(max(details.RatingCount, 0))
		for key, values := range map[string][]igdb.StatsCategory{"genres": details.Genres, "platforms": details.Platforms, "modes": details.GameModes, "themes": details.Themes, "perspectives": details.Perspectives} {
			if len(values) == 0 {
				continue
			}
			labels := []string{}
			for _, value := range values {
				labels = append(labels, value.Name)
			}
			r.categories[key] = labels
		}
		r.categories["developers"], r.categories["publishers"] = []string{}, []string{}
		for _, company := range details.Companies {
			if company.Developer {
				r.categories["developers"] = append(r.categories["developers"], company.Company.Name)
			}
			if company.Publisher {
				r.categories["publishers"] = append(r.categories["publishers"], company.Company.Name)
			}
		}
	}
	return uniqueStrings(failures)
}

func gameKey(r *gameRecord) string { return fmt.Sprintf("game:%d", r.card.ID) }
func gameCards(records []*gameRecord) []MediaCard {
	cards := make([]MediaCard, 0, len(records))
	for _, r := range records {
		cards = append(cards, r.card)
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Title == cards[j].Title {
			return cards[i].ID < cards[j].ID
		}
		return cards[i].Title < cards[j].Title
	})
	return cards
}
func gameSummary(records []*gameRecord, q Query) Summary {
	result := Summary{Titles: len(records), Games: len(records)}
	var total float64
	rated := 0
	for _, r := range records {
		result.Plays += len(r.completions)
		if gameCompleted(r, q) {
			result.Completed++
		}
		if r.card.Rating > 0 {
			total += r.card.Rating
			rated++
		}
	}
	result.AverageRating = average(total, rated)
	return result
}
func gameHistory(records []*gameRecord, hideReviews bool) []HistoryPoint {
	byYear := map[int][]*gameRecord{}
	years := []int{}
	for _, r := range records {
		seen := map[int]bool{}
		for _, d := range r.progress {
			if !seen[d.Year()] {
				byYear[d.Year()] = append(byYear[d.Year()], r)
				seen[d.Year()] = true
			}
		}
	}
	for year := range byYear {
		years = append(years, year)
	}
	sort.Ints(years)
	result := []HistoryPoint{}
	if len(years) == 0 {
		return result
	}
	for year := years[0]; year <= years[len(years)-1]; year++ {
		q := Query{Scope: ScopeYear, Year: year}
		summary := gameSummary(byYear[year], q)
		p := HistoryPoint{Year: year, Titles: summary.Titles, Games: summary.Games, AverageRating: summary.AverageRating, Items: gameCards(byYear[year])}
		for _, r := range byYear[year] {
			if len(gameDatesInScope(r.completions, q)) > 0 {
				p.Completed++
				p.CompletedTitleKeys = append(p.CompletedTitleKeys, gameKey(r))
			}
		}
		if !hideReviews {
			for _, r := range byYear[year] {
				if strings.TrimSpace(r.watched.Thoughts) != "" {
					p.ReviewedTitleKeys = append(p.ReviewedTitleKeys, gameKey(r))
				}
			}
			count := len(p.ReviewedTitleKeys)
			p.Reviewed = &count
		}
		result = append(result, p)
	}
	return result
}
func gameBars(records []*gameRecord, category string) []BarStat {
	type aggregate struct {
		keys  []string
		total float64
		rated int
	}
	groups := map[string]*aggregate{}
	for _, r := range records {
		seen := map[string]bool{}
		for _, label := range r.categories[category] {
			label = strings.TrimSpace(label)
			if label == "" || seen[label] {
				continue
			}
			seen[label] = true
			if groups[label] == nil {
				groups[label] = &aggregate{}
			}
			a := groups[label]
			a.keys = append(a.keys, gameKey(r))
			if r.card.Rating > 0 {
				a.total += r.card.Rating
				a.rated++
			}
		}
	}
	result := []BarStat{}
	for label, a := range groups {
		sort.Strings(a.keys)
		result = append(result, BarStat{Label: label, Count: len(a.keys), TitleKeys: a.keys, AverageRating: average(a.total, a.rated)})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Label < result[j].Label
		}
		return result[i].Count > result[j].Count
	})
	return result
}
func gameHighestRated(records []*gameRecord, q Query) HighestRated {
	result := HighestRated{Current: []MediaCard{}, Older: []MediaCard{}}
	for _, r := range records {
		c := r.card
		if c.Rating <= 0 || (q.Scope == ScopeLifetime && c.Rating <= 8) {
			continue
		}
		if q.Scope == ScopeLifetime || c.ReleaseYear == q.Year {
			result.Current = append(result.Current, c)
		} else if c.ReleaseYear > 0 && c.ReleaseYear < q.Year {
			result.Older = append(result.Older, c)
		}
	}
	sortGameRatings(result.Current)
	sortGameRatings(result.Older)
	return result
}
func sortGameRatings(cards []MediaCard) {
	sort.SliceStable(cards, func(i, j int) bool {
		if cards[i].Rating == cards[j].Rating {
			return cards[i].Title < cards[j].Title
		}
		return cards[i].Rating > cards[j].Rating
	})
}
func gameDecades(records []*gameRecord) []DecadeStat {
	groups := map[int][]*gameRecord{}
	for _, r := range records {
		if r.card.ReleaseYear > 0 {
			d := r.card.ReleaseYear / 10 * 10
			groups[d] = append(groups[d], r)
		}
	}
	result := []DecadeStat{}
	for decade, rs := range groups {
		rated, total := 0, 0.0
		favorites := []MediaCard{}
		for _, r := range rs {
			if r.card.Rating > 0 {
				rated++
				total += r.card.Rating
			}
			if r.card.Rating > 8 {
				favorites = append(favorites, r.card)
			}
		}
		if rated < 2 {
			continue
		}
		sortGameRatings(favorites)
		result = append(result, DecadeStat{Decade: decade, Titles: len(rs), AverageRating: average(total, rated), Items: favorites})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].AverageRating == result[j].AverageRating {
			if result[i].Titles == result[j].Titles {
				return result[i].Decade < result[j].Decade
			}
			return result[i].Titles > result[j].Titles
		}
		return result[i].AverageRating > result[j].AverageRating
	})
	if len(result) > 3 {
		result = result[:3]
	}
	return result
}

func gameActivity(records []*gameRecord, completions bool, q Query) ActivityStats {
	type bucket struct {
		count int
		items map[int]MediaCard
	}
	weeks, months := map[string]*bucket{}, map[string]*bucket{}
	for _, r := range records {
		dates := r.progress
		if completions {
			dates = r.completions
		}
		for _, d := range dates {
			for key, groups := range map[string]map[string]*bucket{startOfWeek(d).Format("2006-01-02"): weeks, d.Format("2006-01"): months} {
				if groups[key] == nil {
					groups[key] = &bucket{items: map[int]MediaCard{}}
				}
				groups[key].count++
				groups[key].items[r.card.ID] = r.card
			}
		}
	}
	items := func(b *bucket) ([]MediaCard, float64) {
		cards := []MediaCard{}
		rated, total := 0, 0.0
		for _, c := range b.items {
			cards = append(cards, c)
			if c.Rating > 0 {
				rated++
				total += c.Rating
			}
		}
		sort.Slice(cards, func(i, j int) bool { return cards[i].Title < cards[j].Title })
		return cards, average(total, rated)
	}
	result := ActivityStats{Weeks: []WeekStat{}, Months: []MonthStat{}}
	for start, b := range weeks {
		cards, rating := items(b)
		result.Total += b.count
		result.Weeks = append(result.Weeks, WeekStat{Start: start, Plays: b.count, UniqueTitles: len(cards), Items: cards, AverageRating: rating})
	}
	for month, b := range months {
		cards, rating := items(b)
		result.Months = append(result.Months, MonthStat{Month: month, Plays: b.count, Items: cards, AverageRating: rating})
	}
	sort.Slice(result.Weeks, func(i, j int) bool { return result.Weeks[i].Start < result.Weeks[j].Start })
	sort.Slice(result.Months, func(i, j int) bool { return result.Months[i].Month < result.Months[j].Month })
	if q.Scope == ScopeYear {
		result = fillActivity(result, nil, q.Year, time.Now().UTC())
	}
	for i := range result.Weeks {
		result.Weeks[i].Titles = []string{}
		for _, c := range result.Weeks[i].Items {
			result.Weeks[i].Titles = append(result.Weeks[i].Titles, c.Title)
		}
	}
	return result
}
func gameMilestones(records []*gameRecord) Milestones {
	result := Milestones{MostWatched: []MediaCard{}}
	var first, last time.Time
	for _, r := range records {
		for _, d := range r.progress {
			if first.IsZero() || d.Before(first) {
				first = d
				c := r.card
				c.Date = d.Format("2006-01-02")
				result.First = &c
			}
			if last.IsZero() || d.After(last) {
				last = d
				c := r.card
				c.Date = d.Format("2006-01-02")
				result.Last = &c
			}
		}
		firstInScope := len(r.allCompletions) > 0 && len(r.completions) > 0 && r.allCompletions[0].Equal(r.completions[0])
		if len(r.completions) > 1 || (len(r.completions) > 0 && !firstInScope) {
			result.MostWatched = append(result.MostWatched, r.card)
		}
	}
	sort.Slice(result.MostWatched, func(i, j int) bool {
		if result.MostWatched[i].Plays == result.MostWatched[j].Plays {
			return result.MostWatched[i].Title < result.MostWatched[j].Title
		}
		return result.MostWatched[i].Plays > result.MostWatched[j].Plays
	})
	return result
}
func gameBreakdown(records, additions []*gameRecord, q Query) Breakdown {
	members := map[string][]string{}
	distribution := make([]RatingBucket, 101)
	for i := range distribution {
		distribution[i].Rating = float64(i) / 10
	}
	firsts, replays := 0, 0
	for _, r := range records {
		if r.card.ReleaseYear > 0 && r.card.ReleaseYear == q.Year {
			members["Selected year"] = append(members["Selected year"], gameKey(r))
		} else if r.card.ReleaseYear > 0 && r.card.ReleaseYear < q.Year {
			members["Older"] = append(members["Older"], gameKey(r))
		}
		label := "Not reviewed"
		if strings.TrimSpace(r.watched.Thoughts) != "" {
			label = "Reviewed"
		}
		members[label] = append(members[label], gameKey(r))
		bucket := int(r.card.Rating*10 + 0.5)
		if bucket < 0 || bucket > 100 {
			bucket = 0
		}
		distribution[bucket].Count++
		for i, d := range r.allCompletions {
			if q.Scope != ScopeLifetime && d.Year() != q.Year {
				continue
			}
			label := "Replays"
			if i == 0 {
				label = "First completions"
				firsts++
			} else {
				replays++
			}
			members[label] = append(members[label], gameKey(r))
		}
	}
	pie := func(label string) PieStat {
		return PieStat{Label: label, Count: len(members[label]), TitleKeys: uniqueStrings(members[label])}
	}
	result := Breakdown{Release: []PieStat{pie("Selected year"), pie("Older")}, Plays: []PieStat{pie("First completions"), pie("Replays")}, RatingDistribution: distribution, WatchlistAdditions: len(additions), WatchlistTitles: gameCards(additions)}
	result.Plays[0].Count, result.Plays[1].Count = firsts, replays
	if !q.HideReviews {
		result.Reviews = []PieStat{pie("Reviewed"), pie("Not reviewed")}
	}
	return result
}
func gameHighsLows(records []*gameRecord) (HighsLows, RatingDifferences) {
	highs := HighsLows{}
	differences := RatingDifferences{Higher: []MediaCard{}, Lower: []MediaCard{}}
	count, total := 0, 0.0
	for _, r := range records {
		c := r.card
		if c.CommunityRating > 0 {
			if highs.HighestCommunityRated == nil || c.CommunityRating > highs.HighestCommunityRated.CommunityRating {
				copy := c
				highs.HighestCommunityRated = &copy
			}
			if highs.LowestRated == nil || c.CommunityRating < highs.LowestRated.CommunityRating {
				copy := c
				highs.LowestRated = &copy
			}
			if c.Rating > 0 {
				d := c.Rating - c.CommunityRating
				total += d
				count++
				if d >= 1-1e-6 {
					differences.Higher = append(differences.Higher, c)
					differences.HigherAverage += d
				}
				if d <= -1+1e-6 {
					differences.Lower = append(differences.Lower, c)
					differences.LowerAverage += d
				}
			}
		}
		if c.VoteCount > 0 {
			if highs.MostVoted == nil || c.VoteCount > highs.MostVoted.VoteCount {
				copy := c
				highs.MostVoted = &copy
			}
			if highs.LeastVoted == nil || c.VoteCount < highs.LeastVoted.VoteCount {
				copy := c
				highs.LeastVoted = &copy
			}
		}
		if c.Date != "" {
			if highs.Newest == nil || c.Date > highs.Newest.Date {
				copy := c
				highs.Newest = &copy
			}
			if highs.Oldest == nil || c.Date < highs.Oldest.Date {
				copy := c
				highs.Oldest = &copy
			}
		}
	}
	differences.Average = average(total, count)
	differences.HigherAverage = average(differences.HigherAverage, len(differences.Higher))
	differences.LowerAverage = average(differences.LowerAverage, len(differences.Lower))
	sort.SliceStable(differences.Higher, func(i, j int) bool {
		a, b := differences.Higher[i], differences.Higher[j]
		if a.Rating-a.CommunityRating == b.Rating-b.CommunityRating {
			return a.Title < b.Title
		}
		return a.Rating-a.CommunityRating > b.Rating-b.CommunityRating
	})
	sort.SliceStable(differences.Lower, func(i, j int) bool {
		a, b := differences.Lower[i], differences.Lower[j]
		if a.Rating-a.CommunityRating == b.Rating-b.CommunityRating {
			return a.Title < b.Title
		}
		return a.Rating-a.CommunityRating < b.Rating-b.CommunityRating
	})
	return highs, differences
}
func gamePlaytime(records []*gameRecord) *GamePlaytime {
	result := &GamePlaytime{MostPlayed: []MediaCard{}, Distribution: []PieStat{}, ByRating: []PlaytimeRating{}}
	members := map[string][]string{}
	ratings := map[float64]*PlaytimeRating{}
	hours := []float64{}
	for _, r := range records {
		label := "Unrecorded"
		includeInDistribution := true
		if r.card.PlaytimeHours != nil {
			h := *r.card.PlaytimeHours
			result.TotalHours += uint64(h)
			result.RecordedGames++
			hours = append(hours, float64(h))
			result.MostPlayed = append(result.MostPlayed, r.card)
			switch {
			case h == 0:
				includeInDistribution = false
			case h < 10:
				label = "1–9 hours"
			case h < 25:
				label = "10–24 hours"
			case h < 50:
				label = "25–49 hours"
			case h < 100:
				label = "50–99 hours"
			case h < 500:
				label = "100–499 hours"
			default:
				label = "500+ hours"
			}
			if ratings[r.card.Rating] == nil {
				ratings[r.card.Rating] = &PlaytimeRating{Rating: r.card.Rating, Items: []MediaCard{}}
			}
			b := ratings[r.card.Rating]
			b.Hours += uint64(h)
			b.Items = append(b.Items, r.card)
		}
		if includeInDistribution {
			members[label] = append(members[label], gameKey(r))
		}
	}
	result.AverageHours = average(float64(result.TotalHours), result.RecordedGames)
	sort.Float64s(hours)
	if len(hours) > 0 {
		n := len(hours)
		result.MedianHours = (hours[(n-1)/2] + hours[n/2]) / 2
	}
	sort.Slice(result.MostPlayed, func(i, j int) bool {
		a, b := result.MostPlayed[i], result.MostPlayed[j]
		if *a.PlaytimeHours == *b.PlaytimeHours {
			return a.Title < b.Title
		}
		return *a.PlaytimeHours > *b.PlaytimeHours
	})
	for _, label := range []string{
		"Unrecorded",
		"1–9 hours",
		"10–24 hours",
		"25–49 hours",
		"50–99 hours",
		"100–499 hours",
		"500+ hours",
	} {
		result.Distribution = append(result.Distribution, PieStat{Label: label, Count: len(members[label]), TitleKeys: members[label]})
	}
	for _, b := range ratings {
		sort.Slice(b.Items, func(i, j int) bool { return b.Items[i].Title < b.Items[j].Title })
		result.ByRating = append(result.ByRating, *b)
	}
	sort.Slice(result.ByRating, func(i, j int) bool { return result.ByRating[i].Rating < result.ByRating[j].Rating })
	return result
}
func gameBacklogPicks(records []*gameRecord, owner uint, q Query) []MediaCard {
	cards := []MediaCard{}
	for _, r := range records {
		if r.card.CommunityRating > 0 && r.card.VoteCount > 0 {
			cards = append(cards, r.card)
		}
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].CommunityRating == cards[j].CommunityRating {
			if cards[i].VoteCount == cards[j].VoteCount {
				return cards[i].ID < cards[j].ID
			}
			return cards[i].VoteCount > cards[j].VoteCount
		}
		return cards[i].CommunityRating > cards[j].CommunityRating
	})
	if len(cards) > 30 {
		cards = cards[:30]
	}
	score := func(c MediaCard) string {
		return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%d:game:%d", owner, q.Scope, q.Year, c.ID))))
	}
	sort.Slice(cards, func(i, j int) bool { return score(cards[i]) < score(cards[j]) })
	if len(cards) > 5 {
		cards = cards[:5]
	}
	return cards
}
