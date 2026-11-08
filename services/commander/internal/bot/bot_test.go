package bot

import (
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/tracker"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type rankCache struct {
	db.CacheDB
	cached *tracker.Ranks
}

func (cache *rankCache) FindCachedRank(context.Context, db.RLPlatform, string) (*tracker.Ranks, bool, error) {
	return cache.cached, cache.cached != nil, nil
}

func TestRankMessageWithoutSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := tracker.New(context.Background(), server.URL, time.Hour)
	defer client.Close()
	for _, check := range []struct {
		name   string
		cached *tracker.Ranks
		want   string
	}{
		{"uncached", nil, "Player ranks are temporarily unavailable. Please try again shortly."},
		{"cached", &tracker.Ranks{DisplayName: "Player"}, "Player"},
	} {
		t.Run(check.name, func(t *testing.T) {
			instance := &bot{tracker: client, cacheDB: &rankCache{cached: check.cached}}
			if got := instance.getRankMessage(context.Background(), db.RLPlatformEpic, "player", "$(name)"); got != check.want {
				t.Fatalf("message = %q, want %q", got, check.want)
			}
		})
	}
}

func TestRankTimeoutLeavesReplyTime(t *testing.T) {
	var block atomic.Bool
	var created atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		var request struct{ Cmd, URL string }
		json.NewDecoder(req.Body).Decode(&request)
		switch request.Cmd {
		case "sessions.create":
			fmt.Fprintf(writer, `{"status":"ok","session":"id-%d"}`, created.Add(1))
		case "request.get":
			if request.URL != "https://rocketleague.tracker.network/" && block.Load() {
				<-req.Context().Done()
				return
			}
			json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "solution": map[string]any{"status": 200, "response": `{"data":{"platformInfo":{"platformUserHandle":"Player"},"segments":[]}}`}})
		case "sessions.destroy":
			fmt.Fprint(writer, `{"status":"ok"}`)
		}
	}))
	defer server.Close()
	client := tracker.New(context.Background(), server.URL, time.Hour)
	defer client.Close()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		_, err := client.GetRanks(context.Background(), "epic", "player")
		if err == nil {
			break
		}
		if !errors.Is(err, tracker.ErrUnavailable) {
			t.Fatal(err)
		}
		select {
		case <-timer.C:
			t.Fatal("session did not become ready")
		case <-ticker.C:
		}
	}
	block.Store(true)
	instance := &bot{tracker: client, cacheDB: &rankCache{}}
	ctx, cancel := context.WithTimeout(context.Background(), 2200*time.Millisecond)
	defer cancel()
	if got := instance.getRankMessage(ctx, db.RLPlatformEpic, "player", "$(name)"); got != messageRanksUnavailable {
		t.Fatalf("message = %q", got)
	}
	deadline, _ := ctx.Deadline()
	if ctx.Err() != nil || time.Until(deadline) < time.Second {
		t.Fatal("rank request consumed the time needed to send a reply")
	}
}
