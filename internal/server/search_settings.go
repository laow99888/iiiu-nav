package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

const searchEnginesSettingKey = "search_engines"

var supportedSearchEngineIDs = map[string]struct{}{
	"google":     {},
	"baidu":      {},
	"bing":       {},
	"duckduckgo": {},
}

var defaultSearchEngineConfig = []searchEngineConfig{
	{ID: "google", Enabled: true},
	{ID: "baidu", Enabled: true},
	{ID: "bing", Enabled: true},
	{ID: "duckduckgo", Enabled: true},
}

type SettingsStore interface {
	Setting(ctx context.Context, key string) (json.RawMessage, bool, error)
	PutSetting(ctx context.Context, key string, value json.RawMessage) error
}

type searchEngineConfig struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type searchEngineSettingsRequest struct {
	Engines []searchEngineConfig `json:"engines"`
}

type searchSettingsHandler struct {
	store SettingsStore
}

func registerSearchSettingsRoutes(
	mux *http.ServeMux,
	authenticator Authenticator,
	store SettingsStore,
) {
	handler := &searchSettingsHandler{store: store}
	mux.Handle("PUT /api/settings/search-engines", RequireAdmin(
		authenticator,
		http.HandlerFunc(handler.update),
	))
}

func (handler *searchSettingsHandler) update(writer http.ResponseWriter, request *http.Request) {
	var body searchEngineSettingsRequest
	if err := decodeJSON(writer, request, &body); err != nil || validateSearchEngineConfig(body.Engines) != nil {
		writeError(writer, http.StatusBadRequest, "invalid_search_engines")
		return
	}
	value, err := json.Marshal(body.Engines)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "search_engines_encode_failed")
		return
	}
	if err := handler.store.PutSetting(request.Context(), searchEnginesSettingKey, value); err != nil {
		writeError(writer, http.StatusInternalServerError, "search_engines_update_failed")
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func readSearchEngineConfig(ctx context.Context, store SettingsStore) ([]searchEngineConfig, error) {
	if store == nil {
		return cloneSearchEngineConfig(defaultSearchEngineConfig), nil
	}
	value, found, err := store.Setting(ctx, searchEnginesSettingKey)
	if err != nil {
		return nil, err
	}
	if !found {
		return cloneSearchEngineConfig(defaultSearchEngineConfig), nil
	}
	var config []searchEngineConfig
	if err := json.Unmarshal(value, &config); err != nil {
		return nil, err
	}
	if err := validateSearchEngineConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

func validateSearchEngineConfig(config []searchEngineConfig) error {
	if len(config) != len(supportedSearchEngineIDs) {
		return errors.New("search engine config must contain every supported engine")
	}
	seen := make(map[string]struct{}, len(config))
	enabled := 0
	for _, engine := range config {
		if _, supported := supportedSearchEngineIDs[engine.ID]; !supported {
			return errors.New("unsupported search engine")
		}
		if _, duplicate := seen[engine.ID]; duplicate {
			return errors.New("duplicate search engine")
		}
		seen[engine.ID] = struct{}{}
		if engine.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return errors.New("at least one search engine must be enabled")
	}
	return nil
}

func cloneSearchEngineConfig(config []searchEngineConfig) []searchEngineConfig {
	return append([]searchEngineConfig(nil), config...)
}
