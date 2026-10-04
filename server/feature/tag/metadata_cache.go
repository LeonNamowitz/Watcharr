package tag

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
)

const metadataCacheTTL = 24 * time.Hour

func validSuggestionKind(kind suggestionKind) bool {
	switch kind {
	case suggestionKindAll, suggestionKindGenre, suggestionKindKeyword, suggestionKindComposer,
		suggestionKindLanguage, suggestionKindCollection, suggestionKindFuture,
		suggestionKindGameGenre, suggestionKindGameMode, suggestionKindGameFuture, suggestionKindGameCategory:
		return true
	}
	return false
}

func metadataProfile(kind suggestionKind) string {
	switch kind {
	case suggestionKindGenre, suggestionKindLanguage, suggestionKindCollection:
		return "basic"
	case suggestionKindKeyword:
		return "keywords"
	case suggestionKindComposer:
		return "composers"
	case suggestionKindFuture, suggestionKindGameGenre, suggestionKindGameMode, suggestionKindGameFuture, suggestionKindGameCategory:
		return "local"
	default:
		return "all"
	}
}

func metadataParams(profile string, contentType entity.ContentType) map[string]string {
	credits := "credits"
	if contentType == entity.SHOW {
		credits = "aggregate_credits"
	}
	switch profile {
	case "keywords":
		return map[string]string{"append_to_response": "keywords"}
	case "composers":
		return map[string]string{"append_to_response": credits}
	case "all":
		return map[string]string{"append_to_response": "keywords," + credits}
	default:
		return nil
	}
}

type metadataCacheEntry struct {
	Version   int
	ExpiresAt time.Time
	Metadata  contentMetadata
}

// Cache public provider facets only. Ownership and tag membership are queried
// afresh by each endpoint; never persist an owner's watched data or candidates.
func (s *Service) cachedContentMetadata(item entity.Watched, profile string) (contentMetadata, error) {
	key := fmt.Sprintf("%s-%d-%s", item.Content.Type, item.Content.TmdbID, profile)
	result, err, _ := s.metadataRequests.Do(key, func() (any, error) {
		// A legacy full scan can satisfy every category without another lookup.
		fullKey := fmt.Sprintf("%s-%d-all", item.Content.Type, item.Content.TmdbID)
		if cached, ok := s.loadMetadata(fullKey); ok {
			return cached, nil
		}
		if profile != "all" {
			if cached, ok := s.loadMetadata(key); ok {
				return cached, nil
			}
		}
		metadata, err := s.fetchContentMetadata(item, profile)
		if err != nil {
			return contentMetadata{}, err
		}
		metadata.media = domain.Media{}
		s.metadataCache.Set(key, metadata, metadataCacheTTL)
		if s.metadataCacheDir != "" {
			entry := metadataCacheEntry{Version: 1, ExpiresAt: s.now().Add(metadataCacheTTL), Metadata: metadata}
			if err := s.saveMetadata(key, entry); err != nil {
				slog.Warn("Tag suggestions: failed saving metadata cache", "error", err)
			}
		}
		return metadata, nil
	})
	if err != nil {
		return contentMetadata{}, err
	}
	metadata := result.(contentMetadata)
	metadata.media = domain.NewMediaFromWatched(&item, ptr(domain.NewWatchedDtoForLists(&item)))
	return metadata, nil
}

func (s *Service) loadMetadata(key string) (contentMetadata, bool) {
	if cached, ok := s.metadataCache.Get(key); ok {
		return cached.(contentMetadata), true
	}
	if s.metadataCacheDir != "" {
		data, err := os.ReadFile(filepath.Join(s.metadataCacheDir, key+".json"))
		var entry metadataCacheEntry
		if err == nil && json.Unmarshal(data, &entry) == nil && entry.Version == 1 {
			if remaining := entry.ExpiresAt.Sub(s.now()); remaining > 0 {
				s.metadataCache.Set(key, entry.Metadata, remaining)
				return entry.Metadata, true
			}
		}
	}
	return contentMetadata{}, false
}

func (s *Service) saveMetadata(key string, entry metadataCacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.metadataCacheDir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(s.metadataCacheDir, ".metadata-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), filepath.Join(s.metadataCacheDir, key+".json"))
}
