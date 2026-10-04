package stats

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/media/igdb"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"gorm.io/gorm"
)

const (
	ScopeYear       = "year"
	ScopeLifetime   = "lifetime"
	metadataWorkers = 6
	// Personal ratings use a 10-point scale: 9 is 90/100.
	highestRatedEpisodeMinimum = 9
)

type Query struct {
	Scope       string
	Year        int
	Media       string
	HideReviews bool
}

type TMDBProvider interface {
	MovieDetails(tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error)
	ShowDetails(tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error)
}

type IGDBProvider interface {
	GameStatsDetails([]int) (map[int]igdb.GameStatsDetails, error)
}

type Service struct {
	db   *gorm.DB
	tmdb TMDBProvider
	igdb IGDBProvider
}

func NewService(db *gorm.DB, provider TMDBProvider, games ...IGDBProvider) *Service {
	s := &Service{db: db, tmdb: provider}
	if len(games) > 0 {
		s.igdb = games[0]
	}
	return s
}

type StatsResponse struct {
	Library              *LibraryStats     `json:"library,omitempty"`
	Calendar             []DailyStat       `json:"calendar,omitempty"`
	Games                *GameStats        `json:"games,omitempty"`
	Scope                string            `json:"scope"`
	Media                string            `json:"media"`
	Owner                entity.PublicUser `json:"owner"`
	ReviewsVisible       bool              `json:"reviewsVisible"`
	Languages            []BarStat         `json:"languages"`
	Studios              []PersonStat      `json:"studios"`
	Year                 int               `json:"year,omitempty"`
	AvailableYears       []int             `json:"availableYears"`
	Summary              Summary           `json:"summary"`
	History              []HistoryPoint    `json:"history"`
	Decades              []DecadeStat      `json:"decades"`
	Episodes             []MediaCard       `json:"episodes"`
	HighestRatedEpisodes HighestRated      `json:"highestRatedEpisodes"`
	HighestRated         HighestRated      `json:"highestRated"`
	Activity             ActivityStats     `json:"activity"`
	Milestones           Milestones        `json:"milestones"`
	Genres               []BarStat         `json:"genres"`
	Countries            []BarStat         `json:"countries"`
	Breakdown            Breakdown         `json:"breakdown"`
	People               PeopleStats       `json:"people"`
	Crew                 []CrewDepartment  `json:"crew"`
	HighsLows            HighsLows         `json:"highsLows"`
	RatingDifferences    RatingDifferences `json:"ratingDifferences"`
	Posters              []MediaCard       `json:"posters"`
	Watchlist            []MediaCard       `json:"watchlist"`
	Metadata             MetadataStatus    `json:"metadata"`
}

type Summary struct {
	Hours         *float64 `json:"hours,omitempty"`
	Games         int      `json:"games,omitempty"`
	Completed     int      `json:"completed,omitempty"`
	Titles        int      `json:"titles"`
	Movies        int      `json:"movies"`
	Shows         int      `json:"shows"`
	Plays         int      `json:"plays"`
	AverageRating float64  `json:"averageRating"`
}

type HistoryPoint struct {
	Games              int         `json:"games,omitempty"`
	Completed          int         `json:"completed,omitempty"`
	CompletedTitleKeys []string    `json:"completedTitleKeys,omitempty"`
	Items              []MediaCard `json:"items"`
	ReviewedTitleKeys  []string    `json:"reviewedTitleKeys,omitempty"`
	Year               int         `json:"year"`
	Movies             int         `json:"movies"`
	Shows              int         `json:"shows"`
	Reviewed           *int        `json:"reviewed,omitempty"`
	Titles             int         `json:"titles"`
	AverageRating      float64     `json:"averageRating"`
}

type DecadeStat struct {
	Decade        int         `json:"decade"`
	Titles        int         `json:"titles"`
	AverageRating float64     `json:"averageRating"`
	Items         []MediaCard `json:"items"`
}

type MediaCard struct {
	CommunityRating float64 `json:"communityRating,omitempty"`
	CoverID         string  `json:"coverId,omitempty"`
	PlaytimeHours   *uint   `json:"playtimeHours,omitempty"`
	EpisodeName     string  `json:"episodeName,omitempty"`
	StillPath       string  `json:"stillPath,omitempty"`
	SeasonNumber    int     `json:"seasonNumber,omitempty"`
	EpisodeNumber   int     `json:"episodeNumber,omitempty"`
	ID              int     `json:"id"`
	Type            string  `json:"type"`
	Title           string  `json:"title"`
	PosterPath      string  `json:"posterPath,omitempty"`
	ReleaseYear     int     `json:"releaseYear,omitempty"`
	Rating          float64 `json:"rating,omitempty"`
	TMDBRating      float64 `json:"tmdbRating,omitempty"`
	VoteCount       uint32  `json:"voteCount,omitempty"`
	Plays           int     `json:"plays,omitempty"`
	Runtime         int     `json:"runtime,omitempty"`
	Date            string  `json:"date,omitempty"`
}

type HighestRated struct {
	Current []MediaCard `json:"current"`
	Older   []MediaCard `json:"older"`
	Unknown []MediaCard `json:"unknown,omitempty"`
}

type WeekStat struct {
	Items         []MediaCard `json:"items"`
	Titles        []string    `json:"titles"`
	Start         string      `json:"start"`
	Plays         int         `json:"plays"`
	UniqueTitles  int         `json:"uniqueTitles"`
	AverageRating float64     `json:"averageRating"`
}

type MonthStat struct {
	Items         []MediaCard `json:"items"`
	Month         string      `json:"month"`
	Plays         int         `json:"plays"`
	AverageRating float64     `json:"averageRating"`
}

type ActivityStats struct {
	Total           int         `json:"total"`
	AveragePerWeek  float64     `json:"averagePerWeek"`
	AveragePerMonth float64     `json:"averagePerMonth"`
	Weeks           []WeekStat  `json:"weeks"`
	Months          []MonthStat `json:"months"`
}

type Milestones struct {
	First       *MediaCard  `json:"first,omitempty"`
	Last        *MediaCard  `json:"last,omitempty"`
	MostWatched []MediaCard `json:"mostWatched"`
}

type BarStat struct {
	TitleKeys     []string `json:"titleKeys"`
	Label         string   `json:"label"`
	Count         int      `json:"count"`
	AverageRating float64  `json:"averageRating"`
}

type PieStat struct {
	TitleKeys []string `json:"titleKeys"`
	Label     string   `json:"label"`
	Count     int      `json:"count"`
}

type RatingBucket struct {
	Rating float64 `json:"rating"`
	Count  int     `json:"count"`
}

type Breakdown struct {
	Release            []PieStat      `json:"release"`
	Plays              []PieStat      `json:"plays"`
	Reviews            []PieStat      `json:"reviews"`
	RatingDistribution []RatingBucket `json:"ratingDistribution"`
	WatchlistAdditions int            `json:"watchlistAdditions"`
	WatchlistTitles    []MediaCard    `json:"watchlistTitles"`
}

type PersonStat struct {
	TitleKeys     []string `json:"titleKeys"`
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	ProfilePath   string   `json:"profilePath,omitempty"`
	Titles        int      `json:"titles"`
	AverageRating float64  `json:"averageRating"`
}

type PeopleStats struct {
	Cast      []PersonStat `json:"cast"`
	Directors []PersonStat `json:"directors"`
}

type CrewJob struct {
	Job    string       `json:"job"`
	People []PersonStat `json:"people"`
}

type CrewDepartment struct {
	Department string    `json:"department"`
	Jobs       []CrewJob `json:"jobs"`
}

type HighsLows struct {
	HighestCommunityRated *MediaCard `json:"highestCommunityRated,omitempty"`
	MostPlaytime          *MediaCard `json:"mostPlaytime,omitempty"`
	LeastPlaytime         *MediaCard `json:"leastPlaytime,omitempty"`
	HighestTMDBRated      *MediaCard `json:"highestTMDBRated,omitempty"`
	LowestRated           *MediaCard `json:"lowestRated,omitempty"`
	MostVoted             *MediaCard `json:"mostVoted,omitempty"`
	LeastVoted            *MediaCard `json:"leastVoted,omitempty"`
	Newest                *MediaCard `json:"newest,omitempty"`
	Oldest                *MediaCard `json:"oldest,omitempty"`
	Longest               *MediaCard `json:"longest,omitempty"`
	Shortest              *MediaCard `json:"shortest,omitempty"`
}

type RatingDifferences struct {
	HigherAverage float64     `json:"higherAverage"`
	Higher        []MediaCard `json:"higher"`
	LowerAverage  float64     `json:"lowerAverage"`
	Lower         []MediaCard `json:"lower"`
	Average       float64     `json:"average"`
}

type MetadataStatus struct {
	Partial      bool     `json:"partial"`
	FailedTitles []string `json:"failedTitles"`
}

type watchedRecord struct {
	watched     entity.Watched
	content     *entity.Content
	plays       []time.Time
	firstPlay   *time.Time
	firstPlayID uint
	addDates    []time.Time
	episode     *entity.WatchedEpisode
	parent      *watchedRecord
	episodeName string
	stillPath   string
}

type playEvent struct {
	record *watchedRecord
	date   time.Time
}

type personCredit struct {
	id          int
	name        string
	profilePath string
	department  string
	job         string
}

type contentMetadata struct {
	genres    []string
	countries []string
	languages []string
	studios   []personCredit
	card      *entity.Content
	cast      []personCredit
	crew      []personCredit
}

func (s *Service) GetStats(userID uint, q Query) (StatsResponse, error) {
	if q.Scope != ScopeLifetime && q.Scope != ScopeYear {
		return StatsResponse{}, errors.New("invalid stats scope")
	}
	if q.Scope == ScopeYear && q.Year == 0 {
		q.Year = time.Now().UTC().Year()
	}

	if q.Media == "game" {
		return s.getGameStats(userID, q)
	}

	var watched []entity.Watched
	if err := s.db.Where("user_id = ? AND content_id IS NOT NULL", userID).
		Preload("Content").Preload("Activity", "user_id = ?", userID).Find(&watched).Error; err != nil {
		return StatsResponse{}, errors.New("failed to load watched items")
	}

	if q.Media == "" {
		q.Media = "movie"
	}
	if q.Media != "movie" && q.Media != "tv" {
		return StatsResponse{}, errors.New("invalid stats media")
	}
	var owner entity.User
	if err := s.db.Preload("Avatar").First(&owner, userID).Error; err != nil {
		return StatsResponse{}, errors.New("failed to load stats owner")
	}
	records := make([]*watchedRecord, 0, len(watched))
	availableYears := map[int]bool{}
	for i := range watched {
		if watched[i].Content == nil ||
			(watched[i].Content.Type != entity.MOVIE && watched[i].Content.Type != entity.SHOW) {
			continue
		}
		r := &watchedRecord{watched: watched[i], content: watched[i].Content}
		for _, activity := range watched[i].Activity {
			date := effectiveDate(activity.CreatedAt, activity.CustomDate)
			if topLevelStatus(activity) != "" {
				availableYears[date.Year()] = true
			}
			if activity.CountAsPlay {
				r.plays = append(r.plays, date)
				availableYears[date.Year()] = true
			}
			if (isAddActivity(activity.Type) || activity.Type == entity.STATUS_CHANGED || activity.Type == entity.STATUS_CHANGED_AUTO) && isPlannedAdd(activity.Data) {
				r.addDates = append(r.addDates, date)

			}
		}
		sort.Slice(r.plays, func(i, j int) bool { return r.plays[i].Before(r.plays[j]) })
		if len(r.plays) > 0 {
			first := r.plays[0]
			r.firstPlay = &first
		}
		if hasLegacyFinish(r) {
			availableYears[r.watched.CreatedAt.UTC().Year()] = true
		}
		records = append(records, r)
	}

	availableYears[time.Now().UTC().Year()] = true
	allRecords := records
	records = filterMedia(records, q.Media)
	years := make([]int, 0, len(availableYears))
	for year := range availableYears {
		years = append(years, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	scopeRecords := filterRecordsWithPlays(records)
	if q.Scope == ScopeYear {
		scopeRecords = filterRecordsForYear(records, q.Year)
	}

	watchlist := filterWatchlist(records)
	watchlistAdditions := recordsWithWatchlistAdditions(records, q)
	episodeRecords, err := s.loadEpisodes(records, q, availableYears)
	if err != nil {
		return StatsResponse{}, err
	}
	allEpisodes := episodeRecords
	if q.Media == "tv" && q.Scope == ScopeYear {
		allEpisodes, err = s.loadEpisodes(records, Query{Scope: ScopeLifetime, Media: "tv"}, availableYears)
		if err != nil {
			return StatsResponse{}, err
		}
	}
	library := buildLibrary(records, allEpisodes, q)
	for year := range availableYears {
		if !containsYear(years, year) {
			years = append(years, year)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	hoursRecords := recordsForHours(records, q)
	enrichmentRecords := append([]*watchedRecord{}, scopeRecords...)
	enrichmentRecords = append(enrichmentRecords, hoursRecords...)
	for _, ep := range episodeRecords {
		enrichmentRecords = append(enrichmentRecords, ep.parent)
	}
	enrichmentRecords = append(enrichmentRecords, watchlist...)
	enrichmentRecords = append(enrichmentRecords, watchlistAdditions...)
	metadata, failures := s.enrich(enrichmentRecords)
	for _, r := range enrichmentRecords {
		if c := metadata[contentKey(r.content)].card; c != nil {
			r.content = c
		}
	}
	metadataStatus := MetadataStatus{FailedTitles: uniqueStrings(failures)}
	metadataStatus.Partial = len(metadataStatus.FailedTitles) > 0

	decades := make([]DecadeStat, 0)
	if q.Scope == ScopeLifetime {
		decades = buildDecades(scopeRecords, metadata)
	}
	response := StatsResponse{
		Scope: q.Scope,
		Media: q.Media, Owner: owner.GetSafe(), ReviewsVisible: !q.HideReviews,
		Languages:      buildLanguageBars(scopeRecords, metadata),
		Studios:        buildStudios(scopeRecords, metadata),
		Year:           q.Year,
		AvailableYears: years,
		History:        buildHistory(allRecords, q.HideReviews),
		Decades:        decades,
		Genres:         buildBars(scopeRecords, metadata, true),
		Countries:      buildBars(scopeRecords, metadata, false),
		People:         buildPeople(scopeRecords, metadata),
		Crew:           buildCrew(scopeRecords, metadata),
		Posters:        uniqueCards(scopeRecords, metadata),
		Watchlist:      topWatchlist(watchlist, userID, q),
		Metadata:       metadataStatus,
	}
	response.Library = &library
	response.Summary = buildSummary(scopeRecords)
	response.HighestRated = buildHighestRated(scopeRecords, q.Year, metadata)
	activityRecords := scopeRecords
	if q.Media == "tv" {
		activityRecords = episodeRecords
		s.enrichEpisodes(episodeRecords, metadata, &response.Metadata)
		response.HighestRatedEpisodes = buildHighestRated(episodeRecords, q.Year, metadata)
		response.Episodes = uniqueCards(episodeRecords, metadata)
		response.People.Cast = buildEpisodeCast(episodeRecords, metadata)
	}
	response.Summary.Hours = estimatedHours(hoursRecords, episodeRecords)
	response.Calendar = buildCalendar(activityRecords)
	response.Activity = buildActivity(activityRecords)
	if q.Scope == ScopeYear {
		response.Activity = fillActivity(response.Activity, activityRecords, q.Year, time.Now().UTC())
	}
	response.Milestones = buildMilestones(scopeRecords, metadata)
	response.Breakdown = buildBreakdown(scopeRecords, records, q, metadata)
	response.HighsLows = buildHighsLowsWithShowRuntime(
		scopeRecords,
		metadata,
		q.Scope == ScopeLifetime && q.Media == "tv",
	)
	response.RatingDifferences = buildRatingDifferences(scopeRecords, metadata)
	response.Metadata.FailedTitles = uniqueStrings(response.Metadata.FailedTitles)
	response.Metadata.Partial = len(response.Metadata.FailedTitles) > 0

	if q.HideReviews {
		response.Breakdown.Reviews = nil
	}
	return response, nil
}

func effectiveDate(created time.Time, custom *time.Time) time.Time {
	if custom != nil {
		return custom.UTC()
	}
	return created.UTC()
}

func isAddActivity(activityType entity.ActivityType) bool {
	switch activityType {
	case entity.ADDED_WATCHED, entity.IMPORTED_ADDED_WATCHED,
		entity.IMPORTED_ADDED_WATCHED_JF, entity.IMPORTED_ADDED_WATCHED_PLEX,
		entity.IMPORTED_WATCHED, entity.IMPORTED_WATCHED_JF, entity.IMPORTED_WATCHED_PLEX:
		return true
	default:
		return false
	}
}

func isPlannedAdd(data string) bool {
	var payload struct {
		Status entity.WatchedStatus `json:"status"`
	}
	if json.Unmarshal([]byte(data), &payload) == nil && payload.Status != "" {
		return payload.Status == entity.PLANNED
	}
	return strings.EqualFold(strings.Trim(strings.TrimSpace(data), `"`), string(entity.PLANNED))
}

func filterRecordsForYear(records []*watchedRecord, year int) []*watchedRecord {
	result := make([]*watchedRecord, 0)
	for _, record := range records {
		plays := make([]time.Time, 0)
		for _, date := range record.plays {
			if date.Year() == year {
				plays = append(plays, date)
			}
		}
		if len(plays) > 0 {
			clone := *record
			clone.plays = plays
			result = append(result, &clone)
		}
	}
	return result
}

func filterRecordsWithPlays(records []*watchedRecord) []*watchedRecord {
	result := make([]*watchedRecord, 0, len(records))
	for _, record := range records {
		if len(record.plays) > 0 {
			result = append(result, record)
		}
	}
	return result
}

func filterWatchlist(records []*watchedRecord) []*watchedRecord {
	result := make([]*watchedRecord, 0)
	for _, record := range records {
		if record.watched.Status == entity.PLANNED && len(record.plays) == 0 {
			result = append(result, record)
		}
	}
	return result
}

func (s *Service) enrich(records []*watchedRecord) (map[string]contentMetadata, []string) {
	metadata := make(map[string]contentMetadata)
	if s.tmdb == nil || len(records) == 0 {
		return metadata, nil
	}
	unique := make(map[string]*watchedRecord)
	for _, record := range records {
		key := contentKey(record.content)
		unique[key] = record
	}
	type result struct {
		key  string
		data contentMetadata
		err  error
	}
	jobs := make(chan *watchedRecord)
	results := make(chan result, len(unique))
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for record := range jobs {
			key := contentKey(record.content)
			data, err := s.fetchMetadata(record.content)
			results <- result{key: key, data: data, err: err}
		}
	}
	workers := metadataWorkers
	if len(unique) < workers {
		workers = len(unique)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}
	go func() {
		for _, record := range unique {
			jobs <- record
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	failed := make([]string, 0)
	for item := range results {
		if item.err != nil {
			if record := unique[item.key]; record != nil && record.content != nil {
				failed = append(failed, record.content.Title)
			} else {
				failed = append(failed, item.key)
			}
			continue
		}
		metadata[item.key] = item.data
	}
	sort.Strings(failed)
	return metadata, failed
}

func (s *Service) fetchMetadata(content *entity.Content) (contentMetadata, error) {
	result := contentMetadata{}
	if content == nil {
		return result, errors.New("missing content")
	}
	switch content.Type {
	case entity.MOVIE:
		details, err := s.tmdb.MovieDetails(tmdb.MovieDetailsOptions{
			ID:             strconv.Itoa(content.TmdbID),
			Params:         map[string]string{"append_to_response": "credits"},
			DontRunDBCache: true,
		})
		if err != nil {
			return result, err
		}
		copyContent := *content
		copyContent.VoteAverage = details.VoteAverage
		copyContent.VoteCount = details.VoteCount
		result.card = &copyContent
		for _, language := range details.SpokenLanguages {
			result.languages = append(result.languages, language.EnglishName)
		}
		for _, company := range details.ProductionCompanies {
			result.studios = append(result.studios, personCredit{id: company.ID, name: company.Name, profilePath: company.LogoPath})
		}
		for _, genre := range details.Genres {
			result.genres = append(result.genres, genre.Name)
		}
		for _, country := range details.ProductionCountries {
			result.countries = append(result.countries, country.Name)
		}
		copyContent.Runtime = details.Runtime
		if d, err := time.Parse("2006-01-02", details.ReleaseDate); err == nil {
			copyContent.ReleaseDate = &d
		}
		for _, cast := range details.Credits.Cast {
			result.cast = append(result.cast, personCredit{
				id: cast.ID, name: cast.Name, profilePath: cast.ProfilePath,
			})
		}
		for _, crew := range details.Credits.Crew {
			result.crew = append(result.crew, personCredit{
				id: crew.ID, name: crew.Name, profilePath: crew.ProfilePath,
				department: crew.Department, job: crew.Job,
			})
		}
	case entity.SHOW:
		details, err := s.tmdb.ShowDetails(tmdb.ShowDetailsOptions{
			ID:             strconv.Itoa(content.TmdbID),
			Params:         map[string]string{"append_to_response": "aggregate_credits,credits"},
			DontRunDBCache: true,
		})
		if err != nil {
			return result, err
		}
		copyContent := *content
		copyContent.VoteAverage = details.VoteAverage
		copyContent.VoteCount = details.VoteCount
		result.card = &copyContent
		for _, language := range details.SpokenLanguages {
			result.languages = append(result.languages, language.EnglishName)
		}
		for _, company := range details.ProductionCompanies {
			result.studios = append(result.studios, personCredit{id: company.ID, name: company.Name, profilePath: company.LogoPath})
		}
		for _, genre := range details.Genres {
			result.genres = append(result.genres, genre.Name)
		}
		for _, country := range details.ProductionCountries {
			result.countries = append(result.countries, country.Name)
		}
		if details.NumberOfEpisodes > 0 {
			copyContent.NumberOfEpisodes = details.NumberOfEpisodes
		}
		if len(details.EpisodeRunTime) > 0 && details.EpisodeRunTime[0] > 0 {
			copyContent.Runtime = uint32(details.EpisodeRunTime[0])
		}
		if d, err := time.Parse("2006-01-02", details.FirstAirDate); err == nil {
			copyContent.ReleaseDate = &d
		}
		for _, creator := range details.CreatedBy {
			result.crew = append(result.crew, personCredit{
				id: creator.ID, name: creator.Name, profilePath: creator.ProfilePath,
				department: "Creation", job: "Creator",
			})
		}
		for _, crew := range details.AggregateCredits.Crew {
			for _, job := range crew.Jobs {
				result.crew = append(result.crew, personCredit{
					id: crew.ID, name: crew.Name, profilePath: crew.ProfilePath, department: crew.Department, job: job.Job,
				})
			}
		}
		for _, cast := range details.AggregateCredits.Cast {
			result.cast = append(result.cast, personCredit{id: cast.ID, name: cast.Name, profilePath: cast.ProfilePath})
		}
		credits := details.Credits
		for _, cast := range credits.Cast {
			result.cast = append(result.cast, personCredit{
				id: cast.ID, name: cast.Name, profilePath: cast.ProfilePath,
			})
		}
		for _, crew := range credits.Crew {
			result.crew = append(result.crew, personCredit{
				id: crew.ID, name: crew.Name, profilePath: crew.ProfilePath,
				department: crew.Department, job: crew.Job,
			})
		}
	}
	return result, nil
}

func contentKey(content *entity.Content) string {
	if content == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", content.Type, content.TmdbID)
}

func buildSummary(records []*watchedRecord) Summary {
	result := Summary{}
	var ratingTotal float64
	rated := 0
	for _, record := range records {
		result.Titles++
		if record.content.Type == entity.MOVIE {
			result.Movies++
		} else {
			result.Shows++
		}
		result.Plays += len(record.plays)
		if record.watched.Rating > 0 {
			ratingTotal += record.watched.Rating
			rated++
		}
	}
	if rated > 0 {
		result.AverageRating = ratingTotal / float64(rated)
	}
	return result
}

func buildHistory(records []*watchedRecord, hideReviews bool) []HistoryPoint {
	byYear := map[int][]*watchedRecord{}
	for _, r := range records {
		seen := map[int]bool{}
		for _, d := range r.plays {
			if !seen[d.Year()] {
				byYear[d.Year()] = append(byYear[d.Year()], r)
				seen[d.Year()] = true
			}
		}
	}
	years := []int{}
	for y := range byYear {
		years = append(years, y)
	}
	sort.Ints(years)
	result := []HistoryPoint{}
	if len(years) == 0 {
		return result
	}
	for y := years[0]; y <= years[len(years)-1]; y++ {
		summary := buildSummary(byYear[y])
		p := HistoryPoint{Items: uniqueCards(byYear[y], nil), Year: y, Titles: summary.Titles, Movies: summary.Movies, Shows: summary.Shows, AverageRating: summary.AverageRating}
		if !hideReviews {
			n := 0
			for _, r := range byYear[y] {
				if strings.TrimSpace(r.watched.Thoughts) != "" {
					p.ReviewedTitleKeys = append(p.ReviewedTitleKeys, contentKey(r.content))
					n++
				}
			}
			p.Reviewed = &n
		}
		result = append(result, p)
	}
	return result
}

func buildDecades(records []*watchedRecord, metadata map[string]contentMetadata) []DecadeStat {
	type aggregate struct {
		records []*watchedRecord
		total   float64
		rated   int
	}
	byDecade := make(map[int]*aggregate)
	for _, record := range records {
		year := releaseYear(record.content)
		if year == 0 {
			continue
		}
		decade := year - year%10
		value := byDecade[decade]
		if value == nil {
			value = &aggregate{}
			byDecade[decade] = value
		}
		value.records = append(value.records, record)
		if record.watched.Rating > 0 {
			value.total += record.watched.Rating
			value.rated++
		}
	}
	result := make([]DecadeStat, 0, len(byDecade))
	for decade, value := range byDecade {
		sort.SliceStable(value.records, func(i, j int) bool {
			if value.records[i].watched.Rating == value.records[j].watched.Rating {
				return value.records[i].content.Title < value.records[j].content.Title
			}
			return value.records[i].watched.Rating > value.records[j].watched.Rating
		})
		if value.rated < 2 {
			continue
		}
		favorites := make([]*watchedRecord, 0)
		for _, record := range value.records {
			if record.watched.Rating > 8 {
				favorites = append(favorites, record)
			}
		}
		items := cards(favorites, metadata, len(favorites))
		result = append(result, DecadeStat{
			Decade:        decade,
			Titles:        len(value.records),
			AverageRating: average(value.total, value.rated),
			Items:         items,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
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

func buildHighestRated(records []*watchedRecord, year int, metadata map[string]contentMetadata) HighestRated {
	current := make([]*watchedRecord, 0)
	older := make([]*watchedRecord, 0)
	unknown := make([]*watchedRecord, 0)
	for _, record := range records {
		if record.episode != nil && record.watched.Rating < highestRatedEpisodeMinimum {
			continue
		}
		if record.watched.Rating <= 0 || (year == 0 && record.watched.Rating <= 8) {
			continue
		}
		if year == 0 || releaseYear(record.content) == year {
			current = append(current, record)
		} else if releaseYear(record.content) > 0 && releaseYear(record.content) < year {
			older = append(older, record)
		} else if record.episode != nil && releaseYear(record.content) == 0 {
			unknown = append(unknown, record)
		}
	}
	byRating := func(items []*watchedRecord) {
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].watched.Rating == items[j].watched.Rating {
				return items[i].content.Title < items[j].content.Title
			}
			return items[i].watched.Rating > items[j].watched.Rating
		})
	}
	byRating(current)
	byRating(older)
	byRating(unknown)
	limit := 5
	if year == 0 {
		limit = len(current)
	} else {
		qualified := 0
		for _, record := range current {
			if record.watched.Rating >= 8 {
				qualified++
			}
		}
		if qualified > limit {
			limit = qualified
		}
		if limit > len(current) {
			limit = len(current)
		}
	}
	olderLimit := 5
	// Episode sections paginate in the UI; retain every rated episode in each tab.
	if len(records) > 0 && records[0].episode != nil {
		limit = len(current)
		olderLimit = len(older)
	}
	return HighestRated{Current: cards(current, metadata, limit), Older: cards(older, metadata, olderLimit), Unknown: cards(unknown, metadata, len(unknown))}
}

func buildActivity(records []*watchedRecord) ActivityStats {
	type aggregate struct {
		plays  int
		titles map[uint]bool
		rating float64
		rated  int
		items  []MediaCard
	}
	weeks := make(map[string]*aggregate)
	months := make(map[string]*aggregate)
	for _, record := range records {
		for _, date := range record.plays {
			week := startOfWeek(date).Format("2006-01-02")
			month := date.Format("2006-01")
			for key, target := range map[string]map[string]*aggregate{"week": weeks, "month": months} {
				var bucket string
				if key == "week" {
					bucket = week
				} else {
					bucket = month
				}
				a := target[bucket]
				if a == nil {
					a = &aggregate{titles: make(map[uint]bool)}
					target[bucket] = a
				}
				a.plays++
				resultCard := mediaCard(record, nil)
				if !a.titles[record.watched.ID] {
					a.items = append(a.items, resultCard)
				}
				if !a.titles[record.watched.ID] && record.watched.Rating > 0 {
					a.rating += record.watched.Rating
					a.rated++
				}
				a.titles[record.watched.ID] = true
			}
		}
	}
	weekKeys := make([]string, 0, len(weeks))
	for key := range weeks {
		weekKeys = append(weekKeys, key)
	}
	sort.Strings(weekKeys)
	monthKeys := make([]string, 0, len(months))
	for key := range months {
		monthKeys = append(monthKeys, key)
	}
	sort.Strings(monthKeys)
	result := ActivityStats{Weeks: make([]WeekStat, 0, len(weekKeys)), Months: make([]MonthStat, 0, len(monthKeys))}
	for _, key := range weekKeys {
		a := weeks[key]
		result.Weeks = append(result.Weeks, WeekStat{Items: a.items, Start: key, Plays: a.plays, UniqueTitles: len(a.titles), AverageRating: average(a.rating, a.rated)})
	}
	for _, key := range monthKeys {
		a := months[key]
		result.Months = append(result.Months, MonthStat{Items: a.items, Month: key, Plays: a.plays, AverageRating: average(a.rating, a.rated)})
	}
	for _, w := range result.Weeks {
		result.Total += w.Plays
	}
	return result
}

func startOfWeek(date time.Time) time.Time {
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	day := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -day)
}

func buildMilestones(records []*watchedRecord, metadata map[string]contentMetadata) Milestones {
	events := make([]playEvent, 0)
	for _, record := range records {
		for _, date := range record.plays {
			events = append(events, playEvent{record: record, date: date})
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].date.Before(events[j].date) })
	result := Milestones{MostWatched: make([]MediaCard, 0)}
	if len(events) > 0 {
		result.First = pointerCard(events[0].record, metadata, events[0].date)
		result.Last = pointerCard(events[len(events)-1].record, metadata, events[len(events)-1].date)
	}
	counts := make(map[uint]int)
	for _, event := range events {
		counts[event.record.watched.ID]++
	}
	most := make([]*watchedRecord, 0)
	for _, record := range records {
		if counts[record.watched.ID] >= 2 {
			most = append(most, record)
		}
	}
	sort.SliceStable(most, func(i, j int) bool {
		if counts[most[i].watched.ID] == counts[most[j].watched.ID] {
			return most[i].content.Title < most[j].content.Title
		}
		return counts[most[i].watched.ID] > counts[most[j].watched.ID]
	})
	for _, record := range most {
		card := mediaCard(record, metadata)
		card.Plays = counts[record.watched.ID]
		result.MostWatched = append(result.MostWatched, card)
	}
	return result
}

func buildBars(records []*watchedRecord, metadata map[string]contentMetadata, genres bool) []BarStat {
	type aggregate struct {
		keys   map[string]bool
		count  int
		rating float64
		rated  int
	}
	values := make(map[string]*aggregate)
	for _, record := range records {
		data := metadata[contentKey(record.content)]
		items := data.countries
		if genres {
			items = data.genres
		}
		seen := make(map[string]bool)
		for _, label := range items {
			if label == "" || seen[label] {
				continue
			}
			seen[label] = true
			a := values[label]
			if a == nil {
				a = &aggregate{keys: map[string]bool{}}
				values[label] = a
			}
			key := contentKey(record.content)
			if a.keys[key] {
				continue
			}
			a.keys[key] = true
			a.count++
			if record.watched.Rating > 0 {
				a.rating += record.watched.Rating
				a.rated++
			}
		}
	}
	result := make([]BarStat, 0, len(values))
	for label, value := range values {
		keys := make([]string, 0, len(value.keys))
		for key := range value.keys {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		result = append(result, BarStat{Label: label, Count: value.count, AverageRating: average(value.rating, value.rated), TitleKeys: keys})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Label < result[j].Label
		}
		return result[i].Count > result[j].Count
	})
	return result
}

func buildBreakdown(records, allRecords []*watchedRecord, q Query, metadata map[string]contentMetadata) Breakdown {
	members := map[string][]string{}
	current, older := 0, 0
	reviews, notReviewed := 0, 0
	ratingCounts := make([]int, 101)
	plays := 0
	for _, record := range records {
		if releaseYear(record.content) > 0 && releaseYear(record.content) == q.Year {
			current++
			members["Selected year"] = append(members["Selected year"], contentKey(record.content))
		} else if releaseYear(record.content) > 0 && releaseYear(record.content) < q.Year {
			older++
			members["Older"] = append(members["Older"], contentKey(record.content))
		}
		if strings.TrimSpace(record.watched.Thoughts) == "" {
			notReviewed++
			members["Not reviewed"] = append(members["Not reviewed"], contentKey(record.content))
		} else {
			reviews++
			members["Reviewed"] = append(members["Reviewed"], contentKey(record.content))
		}
		bucket := int(record.watched.Rating*10 + 0.5)
		if bucket < 0 || bucket > 100 {
			bucket = 0
		}
		ratingCounts[bucket]++
		plays += len(record.plays)
	}
	titles := 0
	for _, record := range records {
		if record.firstPlay != nil && (q.Scope == ScopeLifetime || record.firstPlay.Year() == q.Year) {
			titles++
			members["First watches"] = append(members["First watches"], contentKey(record.content))
		}
		firstInScope := record.firstPlay != nil && (q.Scope == ScopeLifetime || record.firstPlay.Year() == q.Year)
		if len(record.plays) > 1 || (!firstInScope && len(record.plays) > 0) {
			members["Rewatches"] = append(members["Rewatches"], contentKey(record.content))
		}
	}
	rewatches := plays - titles
	if rewatches < 0 {
		rewatches = 0
	}
	distribution := make([]RatingBucket, 0, len(ratingCounts))
	for rating, count := range ratingCounts {
		distribution = append(distribution, RatingBucket{Rating: float64(rating) / 10, Count: count})
	}
	watchlistTitles := uniqueCards(recordsWithWatchlistAdditions(allRecords, q), metadata)
	return Breakdown{
		Release:            []PieStat{{Label: "Selected year", TitleKeys: members["Selected year"], Count: current}, {Label: "Older", TitleKeys: members["Older"], Count: older}},
		Plays:              []PieStat{{Label: "First watches", TitleKeys: members["First watches"], Count: titles}, {Label: "Rewatches", TitleKeys: members["Rewatches"], Count: rewatches}},
		Reviews:            []PieStat{{Label: "Reviewed", TitleKeys: members["Reviewed"], Count: reviews}, {Label: "Not reviewed", TitleKeys: members["Not reviewed"], Count: notReviewed}},
		RatingDistribution: distribution,
		WatchlistAdditions: len(watchlistTitles),
		WatchlistTitles:    watchlistTitles,
	}
}

func recordsWithWatchlistAdditions(records []*watchedRecord, q Query) []*watchedRecord {
	result := make([]*watchedRecord, 0)
	for _, record := range records {
		for _, date := range record.addDates {
			if q.Scope == ScopeLifetime || date.Year() == q.Year {
				result = append(result, record)
				break
			}
		}
	}
	return result
}

func buildPeople(records []*watchedRecord, metadata map[string]contentMetadata) PeopleStats {
	cast := make(map[int]*personAggregate)
	directors := make(map[int]*personAggregate)
	for _, record := range records {
		data := metadata[contentKey(record.content)]
		seenCast := make(map[int]bool)
		seenDirector := make(map[int]bool)
		for _, credit := range data.cast {
			if seenCast[credit.id] {
				continue
			}
			seenCast[credit.id] = true
			addPerson(cast, credit, record)
		}
		for _, credit := range data.crew {
			if !isDirector(credit) || seenDirector[credit.id] {
				continue
			}
			seenDirector[credit.id] = true
			addPerson(directors, credit, record)
		}
	}
	return PeopleStats{Cast: peopleFromAggregates(cast), Directors: peopleFromAggregates(directors)}
}

type personAggregate struct {
	keys  map[string]bool
	stat  PersonStat
	total float64
	rated int
}

func addPerson(values map[int]*personAggregate, credit personCredit, record *watchedRecord) {
	value := values[credit.id]
	if value == nil {
		value = &personAggregate{keys: map[string]bool{}, stat: PersonStat{ID: credit.id, Name: credit.name, ProfilePath: credit.profilePath, TitleKeys: []string{}}}
		values[credit.id] = value
	}
	key := recordKey(record)
	if value.keys[key] {
		return
	}
	value.keys[key] = true
	value.stat.TitleKeys = append(value.stat.TitleKeys, key)
	value.stat.Titles++
	if record.watched.Rating > 0 {
		value.total += record.watched.Rating
		value.rated++
	}
	value.stat.AverageRating = average(value.total, value.rated)
}

func isDirector(credit personCredit) bool {
	return strings.EqualFold(credit.job, "director") ||
		strings.EqualFold(credit.job, "creator") ||
		strings.EqualFold(credit.department, "creation")
}

func peopleFromAggregates(values map[int]*personAggregate) []PersonStat {
	result := make([]PersonStat, 0, len(values))
	for _, value := range values {
		if value.stat.Titles >= 2 {
			sort.Strings(value.stat.TitleKeys)
			result = append(result, value.stat)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Titles == result[j].Titles {
			return result[i].Name < result[j].Name
		}
		return result[i].Titles > result[j].Titles
	})
	return result
}

func buildCrew(records []*watchedRecord, metadata map[string]contentMetadata) []CrewDepartment {
	departments := make(map[string]map[string]map[int]*personAggregate)
	for _, record := range records {
		seen := make(map[string]bool)
		for _, credit := range metadata[contentKey(record.content)].crew {
			department := credit.department
			if department == "" {
				department = "Other"
			}
			job := credit.job
			if job == "" {
				job = "Contributor"
			}
			key := fmt.Sprintf("%s:%d", job, credit.id)
			if seen[key] {
				continue
			}
			seen[key] = true
			if departments[department] == nil {
				departments[department] = make(map[string]map[int]*personAggregate)
			}
			if departments[department][job] == nil {
				departments[department][job] = make(map[int]*personAggregate)
			}
			addPerson(departments[department][job], credit, record)
		}
	}
	departmentsResult := make([]CrewDepartment, 0, len(departments))
	for department, jobs := range departments {
		departmentResult := CrewDepartment{Department: department, Jobs: make([]CrewJob, 0, len(jobs))}
		for job, people := range jobs {
			if list := peopleFromAggregates(people); len(list) > 0 {
				departmentResult.Jobs = append(departmentResult.Jobs, CrewJob{Job: job, People: list})
			}
		}
		sort.Slice(departmentResult.Jobs, func(i, j int) bool { return departmentResult.Jobs[i].Job < departmentResult.Jobs[j].Job })

		if len(departmentResult.Jobs) > 0 {
			departmentsResult = append(departmentsResult, departmentResult)
		}
	}
	sort.Slice(departmentsResult, func(i, j int) bool { return departmentsResult[i].Department < departmentsResult[j].Department })
	return departmentsResult
}

func buildHighsLows(records []*watchedRecord, metadata map[string]contentMetadata) HighsLows {
	return buildHighsLowsWithShowRuntime(records, metadata, false)
}

func buildHighsLowsWithShowRuntime(
	records []*watchedRecord,
	metadata map[string]contentMetadata,
	totalShowRuntime bool,
) HighsLows {
	result := HighsLows{}
	for _, record := range records {
		card := mediaCard(record, metadata)
		if card.TMDBRating > 0 && (result.HighestTMDBRated == nil || card.TMDBRating > result.HighestTMDBRated.TMDBRating) {
			copyCard := card
			result.HighestTMDBRated = &copyCard
		}
		if card.TMDBRating > 0 && (result.LowestRated == nil || card.TMDBRating < result.LowestRated.TMDBRating) {
			copyCard := card
			result.LowestRated = &copyCard
		}
		if card.VoteCount > 0 && (result.MostVoted == nil || card.VoteCount > result.MostVoted.VoteCount) {
			copyCard := card
			result.MostVoted = &copyCard
		}
		if card.VoteCount > 0 && (result.LeastVoted == nil || card.VoteCount < result.LeastVoted.VoteCount) {
			copyCard := card
			result.LeastVoted = &copyCard
		}
		if record.content.ReleaseDate != nil {
			if result.Newest == nil || record.content.ReleaseDate.After(parseCardDate(result.Newest.Date)) {
				copyCard := card
				result.Newest = &copyCard
			}
			if result.Oldest == nil || record.content.ReleaseDate.Before(parseCardDate(result.Oldest.Date)) {
				copyCard := card
				result.Oldest = &copyCard
			}
		}
		runtime := estimatedRuntime(record.content)
		if totalShowRuntime {
			runtime = highLowRuntime(record.content)
		}
		if runtime > 0 && (result.Longest == nil || runtime > result.Longest.Runtime) {
			copyCard := card
			copyCard.Runtime = runtime
			result.Longest = &copyCard
		}
		if runtime > 0 && (result.Shortest == nil || runtime < result.Shortest.Runtime) {
			copyCard := card
			copyCard.Runtime = runtime
			result.Shortest = &copyCard
		}
	}
	return result
}

func buildRatingDifferences(records []*watchedRecord, metadata map[string]contentMetadata) RatingDifferences {
	result := RatingDifferences{}
	var total float64
	count := 0
	for _, record := range records {
		if record.watched.Rating <= 0 || record.content.VoteAverage <= 0 {
			continue
		}
		difference := record.watched.Rating - float64(record.content.VoteAverage)
		total += difference
		count++
		card := mediaCard(record, metadata)
		if difference >= 1-1e-6 {
			result.Higher = append(result.Higher, card)
		} else if difference <= -1+1e-6 {
			result.Lower = append(result.Lower, card)
		}
	}
	sort.SliceStable(result.Higher, func(i, j int) bool {
		return result.Higher[i].Rating-result.Higher[i].TMDBRating > result.Higher[j].Rating-result.Higher[j].TMDBRating
	})
	sort.SliceStable(result.Lower, func(i, j int) bool {
		return result.Lower[i].Rating-result.Lower[i].TMDBRating < result.Lower[j].Rating-result.Lower[j].TMDBRating
	})

	if count > 0 {
		result.Average = total / float64(count)
	}
	for _, card := range result.Higher {
		result.HigherAverage += card.Rating - card.TMDBRating
	}
	if len(result.Higher) > 0 {
		result.HigherAverage /= float64(len(result.Higher))
	}
	for _, card := range result.Lower {
		result.LowerAverage += card.Rating - card.TMDBRating
	}
	if len(result.Lower) > 0 {
		result.LowerAverage /= float64(len(result.Lower))
	}
	return result
}

func uniqueCards(records []*watchedRecord, metadata map[string]contentMetadata) []MediaCard {
	result := make([]MediaCard, 0, len(records))
	seen := make(map[uint]bool)
	for _, record := range records {
		if seen[record.watched.ID] {
			continue
		}
		seen[record.watched.ID] = true
		result = append(result, mediaCard(record, metadata))
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Title < result[j].Title })
	return result
}

// Picks are stable for the owner, period and media while eligible watchlist data is unchanged.
func topWatchlist(records []*watchedRecord, ownerID uint, q Query) []MediaCard {
	all := uniqueCards(filterWatchlist(records), nil)
	result := make([]MediaCard, 0, len(all))
	for _, c := range all {
		if c.TMDBRating > 0 && c.VoteCount > 0 {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TMDBRating != result[j].TMDBRating {
			return result[i].TMDBRating > result[j].TMDBRating
		}
		if result[i].VoteCount != result[j].VoteCount {
			return result[i].VoteCount > result[j].VoteCount
		}
		return result[i].ID < result[j].ID
	})
	if len(result) > 30 {
		result = result[:30]
	}
	period := strconv.Itoa(q.Year)
	if q.Scope == ScopeLifetime {
		period = "all"
	}
	scores := make(map[string]string, len(result))
	for _, c := range result {
		key := fmt.Sprintf("%s:%d", c.Type, c.ID)
		scores[key] = fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s:%s", ownerID, period, q.Media, key))))
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := fmt.Sprintf("%s:%d", result[i].Type, result[i].ID), fmt.Sprintf("%s:%d", result[j].Type, result[j].ID)
		if scores[a] == scores[b] {
			return a < b
		}
		return scores[a] < scores[b]
	})
	if len(result) > 5 {
		result = result[:5]
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func cards(records []*watchedRecord, metadata map[string]contentMetadata, limit int) []MediaCard {
	result := make([]MediaCard, 0, min(len(records), limit))
	for _, record := range records {
		result = append(result, mediaCard(record, metadata))
		if len(result) == limit {
			break
		}
	}
	return result
}

func pointerCard(record *watchedRecord, metadata map[string]contentMetadata, date time.Time) *MediaCard {
	card := mediaCard(record, metadata)
	card.Date = date.Format("2006-01-02")
	return &card
}

func mediaCard(record *watchedRecord, metadata map[string]contentMetadata) MediaCard {
	card := MediaCard{
		ID:          record.content.TmdbID,
		Type:        string(record.content.Type),
		Title:       record.content.Title,
		PosterPath:  record.content.PosterPath,
		ReleaseYear: releaseYear(record.content),
		Rating:      record.watched.Rating,
		TMDBRating:  float64(record.content.VoteAverage),
		VoteCount:   record.content.VoteCount,
		Plays:       len(record.plays),
		Runtime:     estimatedRuntime(record.content),
	}
	if record.content.ReleaseDate != nil {
		card.Date = record.content.ReleaseDate.UTC().Format("2006-01-02")
	}
	if record.episode != nil {
		card.EpisodeName = record.episodeName
		card.StillPath = record.stillPath
		card.SeasonNumber = record.episode.SeasonNumber
		card.EpisodeNumber = record.episode.EpisodeNumber
	}
	return card
}

func releaseYear(content *entity.Content) int {
	if content == nil || content.ReleaseDate == nil {
		return 0
	}
	return content.ReleaseDate.UTC().Year()
}

func estimatedRuntime(content *entity.Content) int {
	if content == nil {
		return 0
	}
	return int(content.Runtime)
}

func highLowRuntime(content *entity.Content) int {
	runtime := estimatedRuntime(content)
	if content == nil || content.Type != entity.SHOW {
		return runtime
	}
	if content.NumberOfEpisodes == 0 {
		return 0
	}
	return runtime * int(content.NumberOfEpisodes)
}

func parseCardDate(value string) time.Time {
	date, _ := time.Parse("2006-01-02", value)
	return date
}

func average(total float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
