//go:build darwin

package local_testing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// do makes an authenticated request to the local backend and returns the response.
// The caller is responsible for closing resp.Body.
func do(t *testing.T, token, method, path string, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, BaseURL+path, body)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// playerState fetches the current player state. Returns nil and skips the test if
// no active device is found (204 No Content).
func playerState(t *testing.T, token string) map[string]any {
	t.Helper()
	resp := do(t, token, "GET", "/api/player/state", nil)
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		t.Skip("no active Spotify device — open Spotify and play something first")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /api/player/state: %d %s", resp.StatusCode, body)
	}
	var state map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatalf("decode player state: %v", err)
	}
	return state
}

// deviceID extracts the active device ID from a player state response.
func deviceID(state map[string]any) string {
	dev, _ := state["device"].(map[string]any)
	id, _ := dev["id"].(string)
	return id
}

// currentTrackSpotifyID extracts the bare Spotify track ID (without "spotify:track:" prefix)
// from the current item in a player state response.
func currentTrackSpotifyID(state map[string]any) string {
	item, _ := state["item"].(map[string]any)
	uri, _ := item["uri"].(string) // e.g. "spotify:track:4iV5W9uYEdYUVa79Axb7Rh"
	return strings.TrimPrefix(uri, "spotify:track:")
}

// expectOK reads and closes the body, failing the test if the status is unexpected.
func expectOK(t *testing.T, resp *http.Response, wantStatuses ...int) {
	t.Helper()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	for _, s := range wantStatuses {
		if resp.StatusCode == s {
			return
		}
	}
	// premium_required is a soft skip — Connect endpoints need Premium
	var parsed map[string]any
	if json.Unmarshal(body, &parsed) == nil {
		if e, _ := parsed["error"].(string); e == "premium_required" {
			t.Skipf("Spotify Premium required: %s", parsed["message"])
		}
	}
	t.Fatalf("unexpected status %d: %s", resp.StatusCode, body)
}

// ---- tests ----

func TestPlayback_Pause(t *testing.T) {
	token := MustGetToken()
	state := playerState(t, token)
	devID := deviceID(state)

	path := "/api/player/pause"
	if devID != "" {
		path += "?device_id=" + devID
	}
	resp := do(t, token, "PUT", path, nil)
	expectOK(t, resp, http.StatusNoContent, http.StatusOK)
	t.Log("paused")
}

func TestPlayback_Play(t *testing.T) {
	token := MustGetToken()
	state := playerState(t, token)
	devID := deviceID(state)

	payload := fmt.Sprintf(`{"device_id":%q}`, devID)
	resp := do(t, token, "PUT", "/api/player/play", strings.NewReader(payload))
	expectOK(t, resp, http.StatusNoContent, http.StatusOK)
	t.Log("resumed")
}

func TestPlayback_Seek(t *testing.T) {
	timestamp := 12000 // ms
	token := MustGetToken()
	playerState(t, token)

	path := fmt.Sprintf("/api/spotify/me/player/seek?position_ms=%d", timestamp)
	resp := do(t, token, "PUT", path, nil)
	expectOK(t, resp, http.StatusNoContent, http.StatusOK)
	t.Logf("seeked to %dms", timestamp)
}

func TestPlayback_Previous(t *testing.T) {
	token := MustGetToken()
	playerState(t, token)

	resp := do(t, token, "POST", "/api/player/previous", nil)
	expectOK(t, resp, http.StatusNoContent, http.StatusOK)
	t.Log("skipped to previous")
}

func TestPlayback_Next(t *testing.T) {
	token := MustGetToken()
	playerState(t, token)

	resp := do(t, token, "POST", "/api/player/next", nil)
	expectOK(t, resp, http.StatusNoContent, http.StatusOK)
	t.Log("skipped to next")
}

func TestPlayback_SetVolume(t *testing.T) {
	volume_pct := 20
	token := MustGetToken()
	playerState(t, token)

	path := fmt.Sprintf("/api/spotify/me/player/volume?volume_percent=%d", volume_pct)
	resp := do(t, token, "PUT", path, nil)
	expectOK(t, resp, http.StatusNoContent, http.StatusOK)
	t.Logf("volume set to %d%%", volume_pct)
}

func TestPlayback_Like(t *testing.T) {
	token := MustGetToken()
	state := playerState(t, token)
	trackID := currentTrackSpotifyID(state)
	if trackID == "" {
		t.Skip("no current track")
	}

	resp := do(t, token, "PUT", "/api/spotify/me/tracks?ids="+trackID, nil)
	expectOK(t, resp, http.StatusOK)
	t.Logf("liked track %s", trackID)
}

func TestPlayback_Unlike(t *testing.T) {
	token := MustGetToken()
	state := playerState(t, token)
	trackID := currentTrackSpotifyID(state)
	if trackID == "" {
		t.Skip("no current track")
	}

	resp := do(t, token, "DELETE", "/api/spotify/me/tracks?ids="+trackID, nil)
	expectOK(t, resp, http.StatusOK)
	t.Logf("unliked track %s", trackID)
}

func TestPlayback_ListQueue(t *testing.T) {
	token := MustGetToken()
	playerState(t, token)

	resp := do(t, token, "GET", "/api/spotify/me/player/queue", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /api/spotify/me/player/queue: %d %s", resp.StatusCode, body)
	}

	var queue struct {
		CurrentlyPlaying map[string]any   `json:"currently_playing"`
		Queue            []map[string]any `json:"queue"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&queue); err != nil {
		t.Fatalf("decode queue: %v", err)
	}

	if item := queue.CurrentlyPlaying; item != nil {
		name, _ := item["name"].(string)
		t.Logf("now playing: %s", name)
	}
	t.Logf("queue has %d tracks", len(queue.Queue))
	for i, track := range queue.Queue {
		name, _ := track["name"].(string)
		t.Logf("  [%d] %s", i+1, name)
	}
}
