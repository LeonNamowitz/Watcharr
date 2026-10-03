package stats

import (
	"sort"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
)

// LibraryStats uses all saved titles, independently of watched-only aggregates.
type LibraryStats struct {
	Statuses []StatusGroup   `json:"statuses"`
	Momentum []MomentumPoint `json:"momentum"`
	Planned  int             `json:"planned"`
	Watched  int             `json:"watched"`
	Waiting  WaitingStats    `json:"waiting"`
}
type StatusGroup struct {
	Status entity.WatchedStatus `json:"status"`
	Label  string               `json:"label"`
	Count  int                  `json:"count"`
	Items  []MediaCard          `json:"items"`
}
type MomentumPoint struct {
	Period  string      `json:"period"`
	Planned []MediaCard `json:"planned"`
	Watched []MediaCard `json:"watched"`
}
type WaitingStats struct {
	MedianDays *float64        `json:"medianDays"`
	Excluded   int             `json:"excluded"`
	Buckets    []WaitingBucket `json:"buckets"`
	Longest    []WaitingTitle  `json:"longest"`
}
type WaitingBucket struct {
	Label string      `json:"label"`
	Items []MediaCard `json:"items"`
}
type WaitingTitle struct {
	Item        MediaCard `json:"item"`
	Days        int       `json:"days"`
	PlannedDate string    `json:"plannedDate"`
	WatchedDate string    `json:"watchedDate"`
}
type DailyStat struct {
	Date  string      `json:"date"`
	Plays int         `json:"plays"`
	Items []MediaCard `json:"items"`
}

func topLevelStatus(a entity.Activity) entity.WatchedStatus {
	if !isAddActivity(a.Type) && a.Type != entity.STATUS_CHANGED && a.Type != entity.STATUS_CHANGED_AUTO {
		return ""
	}
	return activityStatus(a.Data)
}

func sortedActivities(events []entity.Activity) []entity.Activity {
	activities := append([]entity.Activity(nil), events...)
	sort.SliceStable(activities, func(i, j int) bool {
		a, b := effectiveDate(activities[i].CreatedAt, activities[i].CustomDate), effectiveDate(activities[j].CreatedAt, activities[j].CustomDate)
		if a.Equal(b) {
			return activities[i].ID < activities[j].ID
		}
		return a.Before(b)
	})
	return activities
}

// savedMediaRecord lets title and game adapters share status and conversion rules.
type savedMediaRecord struct {
	watched         entity.Watched
	card            MediaCard
	firstCompletion *completion
}
type completion struct {
	date       time.Time
	activityID uint
}
type datedMediaRecord struct {
	key   string
	card  MediaCard
	dates []time.Time
}

func buildLibrary(records, episodes []*watchedRecord, q Query) LibraryStats {
	// Episode completion dates use the same import/custom-date rules as Activity.
	firstEpisodes := map[uint]completion{}
	for _, ep := range episodes {
		if ep.firstPlay == nil {
			continue
		}
		d, id := *ep.firstPlay, ep.firstPlayID
		if old, ok := firstEpisodes[ep.parent.watched.ID]; !ok || d.Before(old.date) || (d.Equal(old.date) && id != 0 && old.activityID != 0 && id < old.activityID) {
			firstEpisodes[ep.parent.watched.ID] = completion{date: d, activityID: id}
		}
	}

	media := make([]savedMediaRecord, 0, len(records))
	for _, record := range records {
		r := savedMediaRecord{watched: record.watched, card: mediaCard(record, nil)}
		if ep, ok := firstEpisodes[record.watched.ID]; ok {
			r.firstCompletion = &ep
		}
		media = append(media, r)
	}
	return buildMediaLibrary(media, q)
}

func buildMediaLibrary(records []savedMediaRecord, q Query) LibraryStats {
	result := LibraryStats{Statuses: []StatusGroup{}, Momentum: []MomentumPoint{}, Waiting: WaitingStats{Buckets: []WaitingBucket{}, Longest: []WaitingTitle{}}}
	for i, status := range []entity.WatchedStatus{entity.FINISHED, entity.WATCHING, entity.PLANNED, entity.HOLD, entity.DROPPED} {
		result.Statuses = append(result.Statuses, StatusGroup{Status: status, Label: []string{"Finished", "Watching", "Planned", "On hold", "Dropped"}[i], Items: []MediaCard{}})
	}
	for _, label := range []string{"Same day", "1–7 days", "8–30 days", "31–90 days", "91–365 days", "Over 365 days"} {
		result.Waiting.Buckets = append(result.Waiting.Buckets, WaitingBucket{Label: label, Items: []MediaCard{}})
	}

	momentum := map[string]*MomentumPoint{}
	point := func(d time.Time) *MomentumPoint {
		key := d.Format("2006")
		if q.Scope == ScopeYear {
			key = d.Format("2006-01")
		}
		if momentum[key] == nil {
			momentum[key] = &MomentumPoint{Period: key, Planned: []MediaCard{}, Watched: []MediaCard{}}
		}
		return momentum[key]
	}
	inScope := func(d time.Time) bool { return q.Scope == ScopeLifetime || d.Year() == q.Year }
	for _, record := range records {
		statuses := map[entity.WatchedStatus]bool{}
		var plan, firstWatch *time.Time
		var planID, watchID uint
		for _, a := range sortedActivities(record.watched.Activity) {
			d := effectiveDate(a.CreatedAt, a.CustomDate)
			status := topLevelStatus(a)
			if status != "" && inScope(d) {
				statuses[status] = true
			}
			if status == entity.PLANNED && plan == nil {
				copy := d
				plan = &copy
				planID = a.ID
			}
			if a.CountAsPlay && firstWatch == nil {
				copy := d
				firstWatch = &copy
				watchID = a.ID
			}
		}
		if ep := record.firstCompletion; ep != nil && (firstWatch == nil || ep.date.Before(*firstWatch) || (ep.date.Equal(*firstWatch) && ep.activityID != 0 && watchID != 0 && ep.activityID < watchID)) {
			d := ep.date
			firstWatch = &d
			watchID = ep.activityID
		}

		card := record.card
		for i := range result.Statuses {
			group := &result.Statuses[i]
			include := statuses[group.Status]
			if q.Scope == ScopeLifetime {
				include = record.watched.Status == group.Status
			}
			if include {
				group.Items = append(group.Items, card)
				group.Count++
			}
		}
		invalid := plan == nil || (firstWatch != nil && (firstWatch.Before(*plan) || (firstWatch.Equal(*plan) && watchID != 0 && watchID < planID)))
		if invalid {
			if firstWatch != nil && inScope(*firstWatch) {
				result.Waiting.Excluded++
			}
			continue
		}
		if inScope(*plan) {
			p := point(*plan)
			p.Planned = append(p.Planned, card)
			result.Planned++
		}
		if firstWatch == nil || !inScope(*firstWatch) {
			continue
		}
		card.Date = firstWatch.Format("2006-01-02")
		p := point(*firstWatch)
		p.Watched = append(p.Watched, card)
		result.Watched++
		// Calendar days, rather than elapsed 24-hour periods: late evening to next morning is one day.
		plannedDay, _ := time.Parse("2006-01-02", plan.Format("2006-01-02"))
		watchedDay, _ := time.Parse("2006-01-02", firstWatch.Format("2006-01-02"))
		days := int(watchedDay.Sub(plannedDay).Hours() / 24)
		bucket := 0
		for _, upper := range []int{0, 7, 30, 90, 365} {
			if days <= upper {
				break
			}
			bucket++
		}
		result.Waiting.Buckets[bucket].Items = append(result.Waiting.Buckets[bucket].Items, card)
		result.Waiting.Longest = append(result.Waiting.Longest, WaitingTitle{Item: card, Days: days, PlannedDate: plan.Format("2006-01-02"), WatchedDate: firstWatch.Format("2006-01-02")})
	}
	if q.Scope == ScopeYear {
		for month := 1; month <= 12; month++ {
			point(time.Date(q.Year, time.Month(month), 1, 0, 0, 0, 0, time.UTC))
		}
	}
	for _, p := range momentum {
		sortCardsByTitle(p.Planned)
		sortCardsByTitle(p.Watched)
		result.Momentum = append(result.Momentum, *p)
	}
	sort.Slice(result.Momentum, func(i, j int) bool { return result.Momentum[i].Period < result.Momentum[j].Period })
	for i := range result.Statuses {
		sortCardsByTitle(result.Statuses[i].Items)
	}
	for i := range result.Waiting.Buckets {
		sortCardsByTitle(result.Waiting.Buckets[i].Items)
	}
	sort.Slice(result.Waiting.Longest, func(i, j int) bool {
		a, b := result.Waiting.Longest[i], result.Waiting.Longest[j]
		if a.Days != b.Days {
			return a.Days > b.Days
		}
		if a.Item.Title != b.Item.Title {
			return a.Item.Title < b.Item.Title
		}
		return a.Item.ID < b.Item.ID
	})
	if n := len(result.Waiting.Longest); n > 0 {
		values := make([]int, n)
		for i, v := range result.Waiting.Longest {
			values[i] = v.Days
		}
		sort.Ints(values)
		median := float64(values[n/2])
		if n%2 == 0 {
			median = (median + float64(values[n/2-1])) / 2
		}
		result.Waiting.MedianDays = &median
	}
	return result
}

func sortCardsByTitle(cards []MediaCard) {
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Title != cards[j].Title {
			return cards[i].Title < cards[j].Title
		}
		if cards[i].Type != cards[j].Type {
			return cards[i].Type < cards[j].Type
		}
		return cards[i].ID < cards[j].ID
	})
}

// Sparse days keep lifetime payloads bounded; the UI fills empty calendar dates.
func buildCalendar(records []*watchedRecord) []DailyStat {
	media := make([]datedMediaRecord, 0, len(records))
	for _, r := range records {
		media = append(media, datedMediaRecord{key: recordKey(r), card: mediaCard(r, nil), dates: r.plays})
	}
	return buildMediaCalendar(media)
}

func buildMediaCalendar(records []datedMediaRecord) []DailyStat {
	days := map[string]*DailyStat{}
	indexes := map[string]map[string]int{}
	for _, record := range records {
		for _, d := range record.dates {
			day, key := d.UTC().Format("2006-01-02"), record.key
			if days[day] == nil {
				days[day] = &DailyStat{Date: day, Items: []MediaCard{}}
				indexes[day] = map[string]int{}
			}
			days[day].Plays++
			if index, ok := indexes[day][key]; ok {
				days[day].Items[index].Plays++
			} else {
				card := record.card
				card.Date = day
				card.Plays = 1
				indexes[day][key] = len(days[day].Items)
				days[day].Items = append(days[day].Items, card)
			}
		}
	}
	result := make([]DailyStat, 0, len(days))
	for _, day := range days {
		sortCardsByTitle(day.Items)
		result = append(result, *day)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	return result
}
