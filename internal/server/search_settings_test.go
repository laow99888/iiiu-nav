package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"iiiu-nav/internal/auth"
)

func TestSearchEngineSettingsRequireAdminAndValidateFixedIDs(t *testing.T) {
	t.Parallel()

	authenticator := &fakeAuthenticator{authenticateErr: auth.ErrUnauthenticated}
	settings := &fakeSettingsStore{}
	handler := New(Config{Auth: authenticator, Settings: settings})

	anonymous := authRequest(
		http.MethodPut,
		"/api/settings/search-engines",
		`{"engines":[{"id":"google","enabled":true}]}`,
	)
	anonymousResponse := httptest.NewRecorder()
	handler.ServeHTTP(anonymousResponse, anonymous)
	if anonymousResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous status 401, got %d", anonymousResponse.Code)
	}
	authenticator.authenticateErr = nil

	invalid := authRequest(
		http.MethodPut,
		"/api/settings/search-engines",
		`{"engines":[{"id":"attacker","enabled":true},{"id":"baidu","enabled":true},{"id":"bing","enabled":true},{"id":"duckduckgo","enabled":true}]}`,
	)
	invalid.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusBadRequest || settings.putCalls != 0 {
		t.Fatalf("unexpected invalid update: status=%d calls=%d", invalidResponse.Code, settings.putCalls)
	}

	allDisabled := authRequest(
		http.MethodPut,
		"/api/settings/search-engines",
		`{"engines":[{"id":"google","enabled":false},{"id":"baidu","enabled":false},{"id":"bing","enabled":false},{"id":"duckduckgo","enabled":false}]}`,
	)
	allDisabled.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	allDisabledResponse := httptest.NewRecorder()
	handler.ServeHTTP(allDisabledResponse, allDisabled)
	if allDisabledResponse.Code != http.StatusBadRequest || settings.putCalls != 0 {
		t.Fatalf("unexpected all-disabled update: status=%d calls=%d", allDisabledResponse.Code, settings.putCalls)
	}

	valid := authRequest(
		http.MethodPut,
		"/api/settings/search-engines",
		`{"engines":[{"id":"bing","enabled":true},{"id":"google","enabled":false},{"id":"baidu","enabled":true},{"id":"duckduckgo","enabled":false}]}`,
	)
	valid.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	validResponse := httptest.NewRecorder()
	handler.ServeHTTP(validResponse, valid)
	if validResponse.Code != http.StatusNoContent || settings.putCalls != 1 {
		t.Fatalf("unexpected valid update: status=%d calls=%d", validResponse.Code, settings.putCalls)
	}
	if settings.key != searchEnginesSettingKey || string(settings.value) != `[{"id":"bing","enabled":true},{"id":"google","enabled":false},{"id":"baidu","enabled":true},{"id":"duckduckgo","enabled":false}]` {
		t.Fatalf("unexpected persisted setting: key=%q value=%s", settings.key, settings.value)
	}
}

func TestSearchEngineSettingsDefaultAndStoredOrder(t *testing.T) {
	t.Parallel()

	defaults, err := readSearchEngineConfig(context.Background(), &fakeSettingsStore{})
	if err != nil || len(defaults) != 4 || defaults[0].ID != "google" {
		t.Fatalf("unexpected defaults: config=%+v err=%v", defaults, err)
	}
	store := &fakeSettingsStore{
		found: true,
		value: json.RawMessage(`[{"id":"bing","enabled":true},{"id":"baidu","enabled":false},{"id":"google","enabled":true},{"id":"duckduckgo","enabled":true}]`),
	}
	configured, err := readSearchEngineConfig(context.Background(), store)
	if err != nil || configured[0].ID != "bing" || configured[1].Enabled {
		t.Fatalf("unexpected stored config: config=%+v err=%v", configured, err)
	}
}

type fakeSettingsStore struct {
	key      string
	value    json.RawMessage
	found    bool
	putCalls int
}

func (store *fakeSettingsStore) Setting(_ context.Context, _ string) (json.RawMessage, bool, error) {
	return store.value, store.found, nil
}

func (store *fakeSettingsStore) PutSetting(_ context.Context, key string, value json.RawMessage) error {
	store.key = key
	store.value = append(json.RawMessage(nil), value...)
	store.found = true
	store.putCalls++
	return nil
}

var _ SettingsStore = (*fakeSettingsStore)(nil)
