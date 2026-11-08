package tracker

import (
	"RocketRankBot/services/commander/internal/metrics"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const profile = `{"data":{"platformInfo":{"platformUserHandle":"A&B"},"segments":[{"type":"playlist","attributes":{"playlistId":11},"stats":{"rating":{"value":1234},"tier":{"value":17},"division":{"value":2}}},{"type":"playlist","attributes":{"playlistId":63},"stats":{}},{"type":"playlist","attributes":{"playlistId":999}},{"type":"overview","attributes":{"playlistId":10}}]}}`

func TestParse(t *testing.T) {
	decimalProfile := strings.NewReplacer("1234", "1234.0", "17}", "17.0}", "2}", "2.0}").Replace(profile)
	for _, body := range []string{profile, decimalProfile, "<html><head><title>JSON</title></head><body><pre>" + html.EscapeString(decimalProfile) + "</pre></body></html>"} {
		result, err := parse(body)
		if err != nil || result.DisplayName != "A&B" || len(result.Ranks) != 2 || *result.Ranks[0] != (PlayerRank{Playlist: Ranked2v2, Mmr: 1234, Rank: 17, Division: 2}) || result.Ranks[1].Playlist != Heatseeker {
			t.Fatalf("result = %+v, error = %v", result, err)
		}
	}
	for _, check := range []struct {
		body     string
		expected error
	}{
		{"You are being rate limited", ErrRateLimited},
		{"You are making too many requests", ErrRateLimited},
		{`{"errors":[{"message":"CollectorResultStatus::NotFound"}]}`, ErrNotFound},
	} {
		if _, err := parse(check.body); !errors.Is(err, check.expected) {
			t.Fatalf("error = %v, want %v", err, check.expected)
		}
	}
	for _, body := range []string{"not JSON", "{}", `{"data":{}}`} {
		if _, err := parse(body); err == nil {
			t.Fatalf("accepted %q", body)
		}
	}
	challenge := `{"errors":[{"code":"Warden::Challenge","message":"You are making too many requests and must solve a captcha to continue.","data":{}}]}`
	for _, body := range []string{challenge, "<html><body><pre>" + html.EscapeString(challenge) + "</pre></body></html>"} {
		_, err := parse(body)
		if !errors.Is(err, ErrChallenge) || errors.Is(err, ErrRateLimited) {
			t.Fatalf("captcha response misclassified: %v", err)
		}
	}
}

func TestSessions(t *testing.T) {
	initialPrewarmed := metricSnapshot(t, metrics.GaugeTrackerPrewarmedSessions).GetGauge().GetValue()
	var mutex sync.Mutex
	created, warmed, destroyed := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		var request struct {
			Cmd, Session, URL string
			EmbeddedTurnstile any
		}
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		switch request.Cmd {
		case "sessions.create":
			created++
			fmt.Fprintf(writer, `{"status":"ok","session":"session-%d"}`, created)
		case "sessions.destroy":
			destroyed++
			fmt.Fprint(writer, `{"status":"ok"}`)
		case "request.get":
			body := profile
			if request.URL == homeURL {
				warmed++
				body = "<html><body>Tracker</body></html>"
				if request.EmbeddedTurnstile == nil {
					t.Error("prewarm omitted embedded Turnstile handling")
				}
			} else if request.EmbeddedTurnstile != nil {
				t.Error("normal rank request enabled embedded Turnstile handling")
			}
			json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "solution": map[string]any{"status": 200, "response": body}})
		default:
			t.Errorf("unexpected command %q", request.Cmd)
		}
	}))
	defer server.Close()
	client := New(context.Background(), server.URL, time.Hour)
	waitForSessions(t, client, 2, 3*time.Second)
	if got := metricSnapshot(t, metrics.GaugeTrackerPrewarmedSessions).GetGauge().GetValue(); got != initialPrewarmed+2 {
		t.Fatalf("prewarmed session gauge = %v, want %v", got, initialPrewarmed+2)
	}
	if created != 2 || warmed != 2 {
		t.Fatalf("created %d, warmed %d", created, warmed)
	}
	first := <-client.sessions
	second := <-client.sessions
	if _, err := client.GetRanks(context.Background(), "epic", "player"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("busy sessions returned %v", err)
	}
	client.sessions <- first
	client.sessions <- second
	for index := 0; index < 4; index++ {
		if _, err := client.GetRanks(context.Background(), "epic", "A / B"); err != nil {
			t.Fatal(err)
		}
	}
	client.Close()
	if got := metricSnapshot(t, metrics.GaugeTrackerPrewarmedSessions).GetGauge().GetValue(); got != initialPrewarmed {
		t.Fatalf("prewarmed session gauge after close = %v, want %v", got, initialPrewarmed)
	}
	if destroyed != 2 {
		t.Fatalf("destroyed %d sessions", destroyed)
	}
}

func TestWardenRecovery(t *testing.T) {
	for _, outcome := range []string{"success", "not_found", "still_challenged", "solver_error"} {
		t.Run(outcome, func(t *testing.T) {
			challenge := `{"errors":[{"code":"Warden::Challenge","message":"Solve captcha"}]}`
			apiCalls, recoveryCalls := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				var request struct {
					Cmd, Session, URL string
					EmbeddedTurnstile *struct {
						ProbeURL       string `json:"probeUrl"`
						DismissConsent bool
					}
				}
				json.NewDecoder(req.Body).Decode(&request)
				if request.Session != "same-session" {
					t.Errorf("replaced challenged session: %q", request.Session)
				}
				body := profile
				if request.EmbeddedTurnstile != nil {
					recoveryCalls++
					if request.URL != homeURL+"rocket-league/profile/psn/A%20%2F%20B/overview" || request.EmbeddedTurnstile.ProbeURL != profileURL+"psn/A%20%2F%20B" || !request.EmbeddedTurnstile.DismissConsent {
						t.Errorf("unexpected recovery request: %+v", request)
					}
					if outcome == "solver_error" {
						fmt.Fprint(writer, `{"status":"error","message":"unresolved Turnstile"}`)
						return
					}
					body = "<html><body>Tracker</body></html>"
				} else {
					apiCalls++
					if apiCalls == 1 || outcome == "still_challenged" {
						body = challenge
					} else if outcome == "not_found" {
						body = `{"errors":[{"message":"CollectorResultStatus::NotFound"}]}`
					}
				}
				json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "solution": map[string]any{"status": 200, "response": body}})
			}))
			defer server.Close()
			client := &Client{endpoint: server.URL, http: server.Client(), maxAge: time.Hour, sessions: make(chan *session, 1), wake: make(chan struct{}, 1), done: make(chan struct{})}
			current := &session{id: "same-session", created: time.Now(), requests: 1}
			defer current.clearPrewarmed()
			client.sessions <- current
			if _, err := client.GetRanks(context.Background(), "ps", "A / B"); !errors.Is(err, ErrChallenge) {
				t.Fatalf("initial request = %v, want challenge", err)
			}
			if _, err := client.GetRanks(context.Background(), "ps", "A / B"); !errors.Is(err, ErrUnavailable) || apiCalls != 1 {
				t.Fatalf("pending recovery served a rank request: %v", err)
			}
			current = <-client.sessions
			err := client.recover(context.Background(), current)
			if recoveryCalls != 1 {
				t.Fatalf("recovery attempts = %d", recoveryCalls)
			}
			if outcome == "success" || outcome == "not_found" {
				if err != nil || current.recoveryURL != "" || current.failures != 0 || !current.prewarmed || current.requests != 4 {
					t.Fatalf("recovery failed: %+v, %v", current, err)
				}
			} else if err == nil || current.recoveryURL == "" || current.failures != 2 || current.prewarmed {
				t.Fatalf("failed recovery released session: %+v, %v", current, err)
			}
		})
	}
}

func TestWardenFallsBackToOtherReadySession(t *testing.T) {
	var apiCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		var request struct{ Cmd, Session string }
		json.NewDecoder(req.Body).Decode(&request)
		body := profile
		if request.Cmd == "request.get" {
			apiCalls.Add(1)
			if request.Session == "challenged" {
				body = `{"errors":[{"code":"Warden::Challenge","message":"Solve captcha"}]}`
			}
		}
		json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "solution": map[string]any{"status": 200, "response": body}})
	}))
	defer server.Close()
	client := &Client{endpoint: server.URL, http: server.Client(), maxAge: time.Hour, sessions: make(chan *session, 2), wake: make(chan struct{}, 1), done: make(chan struct{})}
	challenged := &session{id: "challenged", created: time.Now()}
	client.sessions <- challenged
	client.sessions <- &session{id: "ready", created: time.Now()}
	defer client.destroyAll()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ranks, err := client.GetRanks(ctx, "epic", "player")
	if err != nil || ranks == nil || apiCalls.Load() != 2 {
		t.Fatalf("did not fail over to ready session: ranks=%+v, calls=%d, error=%v", ranks, apiCalls.Load(), err)
	}
	if challenged.recoveryURL == "" || challenged.failures != 1 {
		t.Fatalf("challenged session was not quarantined: %+v", challenged)
	}
}

func TestRotationThresholds(t *testing.T) {
	client := &Client{maxAge: 2 * time.Hour}
	for _, check := range []struct {
		name    string
		current session
		expired bool
	}{
		{"healthy", session{id: "id", created: time.Now(), requests: 100, failures: 2}, false},
		{"requests", session{id: "id", created: time.Now(), requests: 101}, true},
		{"failures", session{id: "id", created: time.Now(), failures: 3}, true},
		{"age", session{id: "id", created: time.Now().Add(-2*time.Hour - time.Second)}, true},
		{"empty", session{}, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			if got := client.expired(&check.current); got != check.expired {
				t.Fatalf("expired = %v", got)
			}
		})
	}
}

func TestRequestRotation(t *testing.T) {
	for _, reason := range []string{"requests", "age", "failures"} {
		t.Run(reason, func(t *testing.T) {
			created, destroyed, warmed := 0, 0, 0
			fail := false
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				var request struct{ Cmd, Session, URL string }
				json.NewDecoder(req.Body).Decode(&request)
				switch request.Cmd {
				case "sessions.create":
					created++
					fmt.Fprintf(writer, `{"status":"ok","session":"id-%d"}`, created)
				case "sessions.destroy":
					destroyed++
					fmt.Fprint(writer, `{"status":"ok"}`)
				case "request.get":
					body := profile
					if request.URL == homeURL {
						warmed++
						body = "<html><body>Tracker</body></html>"
					} else {
						if !strings.HasSuffix(request.URL, "/psn/A%20%2F%20B") {
							t.Errorf("unexpected URL %q", request.URL)
						}
						if fail {
							body = "invalid JSON"
						}
					}
					json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "solution": map[string]any{"status": 200, "response": body}})
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithCancel(context.Background())
			client := &Client{endpoint: server.URL, http: server.Client(), maxAge: time.Hour, sessions: make(chan *session, 2), wake: make(chan struct{}, 1), done: make(chan struct{}), cancel: cancel}
			current := &session{}
			if err := client.replace(context.Background(), current); err != nil {
				t.Fatal(err)
			}
			client.sessions <- current
			fetch := func() error { _, err := client.GetRanks(context.Background(), "ps", "A / B"); return err }
			switch reason {
			case "requests":
				current.requests = 100
				if err := fetch(); err != nil {
					t.Fatal(err)
				}
				if created != 1 {
					t.Fatal("rotated before serving over 100 requests")
				}
			case "age":
				current.created = time.Now().Add(-time.Hour - time.Second)
			case "failures":
				fail = true
				for attempt := 0; attempt < 2; attempt++ {
					if fetch() == nil {
						t.Fatal("expected parsing failure")
					}
				}
				fail = false
				if err := fetch(); err != nil {
					t.Fatal(err)
				}
				if current.failures != 0 {
					t.Fatal("success did not reset consecutive failures")
				}
				fail = true
				for attempt := 0; attempt < 3; attempt++ {
					if fetch() == nil {
						t.Fatal("expected parsing failure")
					}
				}
				if created != 1 {
					t.Fatal("rotated before three consecutive failures")
				}
				fail = false
			}
			if err := fetch(); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("expired session returned %v", err)
			}
			replacedRequests := current.requests
			beforeReplacement := metricSnapshot(t, metrics.HistogramTrackerSessionRequests).GetHistogram()
			go client.maintain(ctx)
			waitForSessions(t, client, 1, 3*time.Second)
			if err := fetch(); err != nil {
				t.Fatal(err)
			}
			client.Close()
			afterReplacement := metricSnapshot(t, metrics.HistogramTrackerSessionRequests).GetHistogram()
			if afterReplacement.GetSampleCount() != beforeReplacement.GetSampleCount()+1 || afterReplacement.GetSampleSum()-beforeReplacement.GetSampleSum() != float64(replacedRequests) {
				t.Fatalf("replacement histogram did not record %d requests", replacedRequests)
			}
			if created != 2 || warmed != 2 || destroyed != 2 {
				t.Fatalf("created %d, warmed %d, destroyed %d", created, warmed, destroyed)
			}
		})
	}
}

func TestFailedPrewarm(t *testing.T) {
	var created, destroyed atomic.Int32
	var recovered atomic.Bool
	failed := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		var request struct{ Cmd, Session, URL string }
		json.NewDecoder(req.Body).Decode(&request)
		switch request.Cmd {
		case "sessions.create":
			fmt.Fprintf(writer, `{"status":"ok","session":"id-%d"}`, created.Add(1))
		case "sessions.destroy":
			destroyed.Add(1)
			fmt.Fprint(writer, `{"status":"ok"}`)
			if !recovered.Load() {
				select {
				case failed <- struct{}{}:
				default:
				}
			}
		case "request.get":
			status := 200
			body := profile
			if request.URL == homeURL && request.Session != "id-1" && !recovered.Load() {
				status = 403
			}
			json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "solution": map[string]any{"status": status, "response": body}})
		}
	}))
	defer server.Close()
	client := New(context.Background(), server.URL, time.Hour)
	defer client.Close()
	select {
	case <-failed:
	case <-time.After(3 * time.Second):
		t.Fatal("failed prewarm was not cleaned up")
	}
	waitForSessions(t, client, 1, 3*time.Second)
	if _, err := client.GetRanks(context.Background(), "epic", "player"); err != nil {
		t.Fatalf("healthy session unavailable: %v", err)
	}
	recovered.Store(true)
	client.wake <- struct{}{}
	waitForSessions(t, client, 2, 3*time.Second)
}

func TestFlareSolverrIntegration(t *testing.T) {
	endpoint := os.Getenv("FLARESOLVERR_URL")
	if endpoint == "" {
		t.Skip("FLARESOLVERR_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	client := New(ctx, endpoint, 2*time.Hour)
	defer client.Close()
	waitForSessions(t, client, 2, 2*time.Minute)
	result, err := client.GetRanks(ctx, "epic", "yannismate")
	if err != nil {
		t.Fatalf("Tracker live request failed (%s): %v", requestErrorCode(err), err)
	}
	if !strings.EqualFold(result.DisplayName, "yannismate") || len(result.Ranks) == 0 {
		t.Fatalf("expected yannismate profile with playlist data, got %+v", result)
	}
	t.Logf("Tracker profile %q: %d playlists", result.DisplayName, len(result.Ranks))
}

func waitForSessions(t *testing.T, client *Client, count int, timeout time.Duration) {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		ready := 0
		for remaining := len(client.sessions); remaining > 0; remaining-- {
			select {
			case current := <-client.sessions:
				if !client.expired(current) && current.recoveryURL == "" {
					ready++
				}
				client.sessions <- current
			default:
			}
		}
		if ready == count {
			return
		}
		select {
		case <-timer.C:
			t.Fatalf("wanted %d ready sessions, found %d", count, ready)
		case <-ticker.C:
		}
	}
}

func TestUnavailableAndRecovery(t *testing.T) {
	var recovered atomic.Bool
	var created atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		if !recovered.Load() {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var request struct{ Cmd string }
		json.NewDecoder(req.Body).Decode(&request)
		switch request.Cmd {
		case "sessions.create":
			fmt.Fprintf(writer, `{"status":"ok","session":"id-%d"}`, created.Add(1))
		case "request.get":
			json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "solution": map[string]any{"status": 200, "response": profile}})
		case "sessions.destroy":
			fmt.Fprint(writer, `{"status":"ok"}`)
		}
	}))
	defer server.Close()
	client := New(context.Background(), server.URL, time.Hour)
	defer client.Close()
	result := make(chan error, 1)
	go func() {
		_, err := client.GetRanks(context.Background(), "epic", "player")
		result <- err
	}()
	select {
	case err := <-result:
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("rank lookup blocked while sessions were unavailable")
	}
	recovered.Store(true)
	select {
	case client.wake <- struct{}{}:
	default:
	}
	waitForSessions(t, client, 2, 3*time.Second)
	if _, err := client.GetRanks(context.Background(), "epic", "player"); err != nil {
		t.Fatal(err)
	}
}

func TestBlockedPrewarm(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		var request struct{ Cmd string }
		json.NewDecoder(req.Body).Decode(&request)
		switch request.Cmd {
		case "sessions.create":
			fmt.Fprint(writer, `{"status":"ok","session":"warming"}`)
		case "request.get":
			started <- struct{}{}
			<-req.Context().Done()
		case "sessions.destroy":
			fmt.Fprint(writer, `{"status":"ok"}`)
		}
	}))
	defer server.Close()
	client := New(context.Background(), server.URL, time.Hour)
	defer client.Close()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("prewarm did not start")
	}
	result := make(chan error, 1)
	go func() {
		_, err := client.GetRanks(context.Background(), "epic", "player")
		result <- err
	}()
	select {
	case err := <-result:
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("rank lookup waited for prewarm")
	}
}

func TestMaintenanceDoesNotBlockOtherSession(t *testing.T) {
	started := make(chan struct{}, 1)
	var created atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		var request struct{ Cmd, Session string }
		json.NewDecoder(req.Body).Decode(&request)
		switch request.Cmd {
		case "sessions.create":
			fmt.Fprintf(writer, `{"status":"ok","session":"id-%d"}`, created.Add(1))
		case "request.get":
			if request.Session == "id-1" {
				started <- struct{}{}
				<-req.Context().Done()
				return
			}
			json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "solution": map[string]any{"status": 200, "response": profile}})
		case "sessions.destroy":
			fmt.Fprint(writer, `{"status":"ok"}`)
		}
	}))
	defer server.Close()
	client := New(context.Background(), server.URL, time.Hour)
	defer client.Close()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("blocked session did not start prewarming")
	}
	waitForSessions(t, client, 1, 3*time.Second)
	if _, err := client.GetRanks(context.Background(), "epic", "player"); err != nil {
		t.Fatalf("blocked maintenance prevented the other session from serving ranks: %v", err)
	}
}

func metricSnapshot(t *testing.T, metric prometheus.Metric) *dto.Metric {
	t.Helper()
	value := &dto.Metric{}
	if err := metric.Write(value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestRequestMetrics(t *testing.T) {
	challenge := `{"errors":[{"code":"Warden::Challenge","message":"You are making too many requests and must solve a captcha to continue.","data":{}}]}`
	for _, check := range []struct {
		name         string
		status       int
		body         string
		code         string
		solverStatus string
	}{
		{name: "success", status: 200, body: profile, code: "none", solverStatus: "ok"},
		{name: "not found", status: 404, body: `{"errors":[{"code":"CollectorResultStatus::NotFound"}]}`, code: "not_found", solverStatus: "ok"},
		{name: "captcha", status: 200, body: challenge, code: "captcha_required", solverStatus: "ok"},
		{name: "captcha with 429", status: 429, body: challenge, code: "captcha_required", solverStatus: "ok"},
		{name: "rate limited", status: 429, body: `{"message":"slow down"}`, code: "rate_limited", solverStatus: "ok"},
		{name: "parsing", status: 200, body: "not JSON", code: "parsing_error", solverStatus: "ok"},
		{name: "http error", status: 503, body: "unavailable", code: "http_error", solverStatus: "ok"},
		{name: "api error", status: 200, body: `{"errors":[{"code":"Unexpected","message":"failed"}]}`, code: "api_error", solverStatus: "ok"},
		{name: "solver error", code: "solver_error", solverStatus: "error"},
		{name: "no session", code: "unavailable"},
	} {
		t.Run(check.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				json.NewEncoder(writer).Encode(map[string]any{"status": check.solverStatus, "message": "test failure", "solution": map[string]any{"status": check.status, "response": check.body}})
			}))
			defer server.Close()
			client := &Client{endpoint: server.URL, http: server.Client(), maxAge: time.Hour, sessions: make(chan *session, 1), wake: make(chan struct{}, 1), done: make(chan struct{})}
			if check.name != "no session" {
				client.sessions <- &session{id: "test", created: time.Now()}
			}
			outcome := "error"
			if check.code == "none" {
				outcome = "success"
			}
			counter := metrics.CounterTrackerRequests.WithLabelValues(outcome, check.code)
			beforeCount := metricSnapshot(t, counter).GetCounter().GetValue()
			beforeTime := metricSnapshot(t, metrics.HistogramTrackerRequestTime).GetHistogram().GetSampleCount()
			_, err := client.GetRanks(context.Background(), "epic", "player")
			if got := requestErrorCode(err); got != check.code {
				t.Fatalf("error code = %q, want %q (%v)", got, check.code, err)
			}
			if got := metricSnapshot(t, counter).GetCounter().GetValue(); got != beforeCount+1 {
				t.Fatalf("request counter = %v, want %v", got, beforeCount+1)
			}
			if got := metricSnapshot(t, metrics.HistogramTrackerRequestTime).GetHistogram().GetSampleCount(); got != beforeTime+1 {
				t.Fatalf("duration sample count = %d, want %d", got, beforeTime+1)
			}
		})
	}
}

func TestRequestErrorCodes(t *testing.T) {
	for _, check := range []struct {
		err  error
		code string
	}{
		{ErrNotFound, "not_found"},
		{ErrChallenge, "captcha_required"},
		{ErrRateLimited, "rate_limited"},
		{context.DeadlineExceeded, "timeout"},
		{context.Canceled, "canceled"},
		{errTransport, "transport_error"},
		{errSolver, "solver_error"},
		{errHTTP, "http_error"},
		{errParsing, "parsing_error"},
		{errAPI, "api_error"},
		{ErrUnavailable, "unavailable"},
	} {
		t.Run(check.code, func(t *testing.T) {
			if got := requestErrorCode(errors.Join(ErrUnavailable, check.err)); got != check.code {
				t.Fatalf("error code = %q, want %q", got, check.code)
			}
		})
	}
}
