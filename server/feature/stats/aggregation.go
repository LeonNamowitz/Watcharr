package stats

import (
	"sort"
	"time"
)

func filterMedia(records []*watchedRecord, media string) []*watchedRecord {
	result := make([]*watchedRecord, 0)
	for _, r := range records {
		if string(r.content.Type) == media {
			result = append(result, r)
		}
	}
	return result
}

func buildLanguageBars(records []*watchedRecord, metadata map[string]contentMetadata) []BarStat {
	copyMetadata := map[string]contentMetadata{}
	for key, value := range metadata {
		value.countries = value.languages
		copyMetadata[key] = value
	}
	return buildBars(records, copyMetadata, false)
}

func buildStudios(records []*watchedRecord, metadata map[string]contentMetadata) []PersonStat {
	values := map[int]*personAggregate{}
	for _, r := range records {
		seen := map[int]bool{}
		for _, c := range metadata[contentKey(r.content)].studios {
			if !seen[c.id] {
				addPerson(values, c, r)
				seen[c.id] = true
			}
		}
	}
	return peopleFromAggregates(values)
}

// Calendar weeks include the partial weeks at both year boundaries.
func fillActivity(data ActivityStats, records []*watchedRecord, year int, now time.Time) ActivityStats {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)
	weeks := map[string]WeekStat{}
	months := map[string]MonthStat{}
	for _, w := range data.Weeks {
		weeks[w.Start] = w
	}
	for _, m := range data.Months {
		months[m.Month] = m
	}
	data.Weeks = make([]WeekStat, 0)
	data.Months = make([]MonthStat, 0)
	for d := startOfWeek(start); d.Before(end); d = d.AddDate(0, 0, 7) {
		key := d.Format("2006-01-02")
		w := weeks[key]
		w.Start = key
		titles := []string{}
		for _, r := range records {
			count := 0
			for _, played := range r.plays {
				if startOfWeek(played).Equal(d) {
					count++
				}
			}
			if count > 0 {
				title := r.content.Title
				if count > 1 {
					title += " (repeat watches)"
				}
				titles = append(titles, title)
			}
		}
		sort.Strings(titles)
		w.Titles = titles
		data.Weeks = append(data.Weeks, w)
	}
	for d := start; d.Before(end); d = d.AddDate(0, 1, 0) {
		key := d.Format("2006-01")
		m := months[key]
		m.Month = key
		data.Months = append(data.Months, m)
	}
	elapsed := end.Sub(start).Hours() / 24
	monthCount := 12.0
	if now.Year() == year {
		elapsed = now.Sub(start).Hours() / 24
		monthStart := time.Date(year, now.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := monthStart.AddDate(0, 1, 0)
		monthCount = float64(now.Month()-1) + float64(now.Sub(monthStart))/float64(monthEnd.Sub(monthStart))
	}
	if elapsed < 1 {
		elapsed = 1
	}
	if monthCount < 1.0/31 {
		monthCount = 1.0 / 31
	}
	total := data.Total
	data.AveragePerWeek = float64(total) / (elapsed / 7)
	data.AveragePerMonth = float64(total) / monthCount
	return data
}
