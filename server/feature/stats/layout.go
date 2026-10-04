package stats

import (
	"encoding/json"
	"errors"

	"github.com/sbondCo/Watcharr/database/entity"
	"gorm.io/gorm"
)

// Keep these stable IDs and their default order in sync with the frontend registry.
func defaultSectionOrder(media string) []string {
	order := []string{"library-status", "library-momentum", "library-waiting", "history", "decades", "highest-rated"}
	if media == "tv" {
		order = append(order, "highest-rated-episodes")
	}
	if media == "game" {
		order = append(order, "playtime")
	}
	order = append(order, "activity", "calendar", "milestones", "categories", "breakdown")
	if media == "game" {
		order = append(order, "companies")
	} else {
		order = append(order, "people")
	}
	return append(order, "highs-lows", "rated-higher", "rated-lower", "titles", "watchlist")
}

func normalizeSectionOrder(media string, saved []string) []string {
	defaults := defaultSectionOrder(media)
	allowed := make(map[string]bool, len(defaults))
	for _, id := range defaults {
		allowed[id] = true
	}
	order := make([]string, 0, len(defaults))
	for _, id := range append(append([]string{}, saved...), defaults...) {
		if allowed[id] {
			order = append(order, id)
			delete(allowed, id)
		}
	}
	return order
}

func validateSectionOrder(media string, order []string) ([]string, error) {
	if media != "movie" && media != "tv" && media != "game" {
		return nil, errors.New("invalid stats media")
	}
	allowed := map[string]bool{}
	for _, id := range defaultSectionOrder(media) {
		allowed[id] = true
	}
	for _, id := range order {
		if !allowed[id] {
			return nil, errors.New("stats layout contains an unknown, incompatible or duplicate section")
		}
		delete(allowed, id)
	}
	return normalizeSectionOrder(media, order), nil
}

func sectionOrderForOwner(owner entity.User, media string) []string {
	layout := map[string][]string{}
	if owner.StatsLayout != nil {
		// A missing or unreadable preference leaves the default layout intact.
		if err := json.Unmarshal([]byte(*owner.StatsLayout), &layout); err != nil {
			layout = nil
		}
	}
	return normalizeSectionOrder(media, layout[media])
}

func (s *Service) saveSectionOrder(userID uint, media string, order []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var owner entity.User
		if err := tx.Select("id", "stats_layout").First(&owner, userID).Error; err != nil {
			return err
		}
		layout := map[string][]string{}
		if owner.StatsLayout != nil {
			if err := json.Unmarshal([]byte(*owner.StatsLayout), &layout); err != nil {
				return err
			}
		}
		if layout == nil {
			layout = map[string][]string{}
		}
		layout[media] = order
		encoded, err := json.Marshal(layout)
		if err != nil {
			return err
		}
		return tx.Model(&entity.User{}).Where("id = ?", userID).Update("stats_layout", string(encoded)).Error
	})
}
