package tracker

import (
	"RocketRankBot/services/commander/internal/metrics"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

const (
	homeURL                = "https://rocketleague.tracker.network/"
	profileURL             = "https://api.tracker.gg/api/v2/rocket-league/standard/profile/"
	interactionsURL        = "https://api.tracker.gg/api/v1/rocket-league/wrapper/interactions"
	sessionCount           = 2
	defaultSessionMaxAge   = 2 * time.Hour
	maxSessionRequests     = 100
	maxConsecutiveFailures = 3
	requestTimeout         = 60 * time.Second
	httpTimeout            = 70 * time.Second
	maintenanceInterval    = time.Second
)

type Playlist int

const (
	Unranked Playlist = iota
	Ranked1v1
	Ranked2v2
	Ranked3v3
	Hoops
	Rumble
	Dropshot
	Snowday
	Tournaments
	Ranked4v4
	Heatseeker
)

var (
	ErrNotFound     = errors.New("player not found")
	ErrRateLimited  = errors.New("rate limited by Cloudflare")
	ErrChallenge    = errors.New("Tracker captcha required")
	ErrUnavailable  = errors.New("rank lookup temporarily unavailable")
	errTransport    = errors.New("FlareSolverr transport error")
	errSolver       = errors.New("FlareSolverr response error")
	errHTTP         = errors.New("upstream HTTP error")
	errParsing      = errors.New("Tracker parsing error")
	errAPI          = errors.New("Tracker API error")
	playlistMapping = map[int]Playlist{
		0:  Unranked,
		10: Ranked1v1,
		11: Ranked2v2,
		13: Ranked3v3,
		61: Ranked4v4,
		27: Hoops,
		28: Rumble,
		29: Dropshot,
		30: Snowday,
		63: Heatseeker,
		34: Tournaments,
	}
)

type Ranks struct {
	DisplayName string        `json:"displayName,omitempty"`
	Ranks       []*PlayerRank `json:"ranks,omitempty"`
}

type PlayerRank struct {
	Playlist Playlist `json:"playlist,omitempty"`
	Mmr      int      `json:"mmr,omitempty"`
	Rank     int      `json:"rank,omitempty"`
	Division int      `json:"division,omitempty"`
}

type session struct {
	id          string
	created     time.Time
	requests    int
	failures    int
	prewarmed   bool
	recoveryURL string
}

type Client struct {
	endpoint string
	http     *http.Client
	maxAge   time.Duration
	sessions chan *session
	wake     chan struct{}
	cancel   context.CancelFunc
	done     chan struct{}
	mutex    sync.Mutex
	active   sync.WaitGroup
	closed   bool
}

type response struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Session  string `json:"session"`
	Solution struct {
		Status   int    `json:"status"`
		Response string `json:"response"`
	} `json:"solution"`
}

func New(ctx context.Context, endpoint string, maxAge time.Duration) *Client {
	if maxAge <= 0 {
		maxAge = defaultSessionMaxAge
	}
	ctx, cancel := context.WithCancel(ctx)
	client := &Client{
		endpoint: strings.TrimRight(endpoint, "/") + "/v1",
		http:     &http.Client{Timeout: httpTimeout},
		maxAge:   maxAge,
		sessions: make(chan *session, sessionCount),
		wake:     make(chan struct{}, 1),
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	for index := 0; index < sessionCount; index++ {
		client.sessions <- &session{}
	}
	client.wake <- struct{}{}
	go client.maintain(ctx)
	return client
}

func (client *Client) Close() {
	client.stopRequests()
	<-client.done
	client.active.Wait()
	client.destroyAll()
}

func (client *Client) stopRequests() {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	client.closed = true
	client.cancel()
}

func (client *Client) startRequest() bool {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	if client.closed {
		return false
	}
	client.active.Add(1)
	return true
}

func (current *session) clearPrewarmed() {
	if current.prewarmed {
		metrics.GaugeTrackerPrewarmedSessions.Dec()
		current.prewarmed = false
	}
}

func (client *Client) destroyAll() {
	for len(client.sessions) > 0 {
		current := <-client.sessions
		current.clearPrewarmed()
		client.destroy(current.id)
	}
}

func (client *Client) destroy(id string) {
	if id == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	_, _ = client.call(ctx, map[string]any{"cmd": "sessions.destroy", "session": id})
}

func (client *Client) expired(current *session) bool {
	return current.id == "" || time.Since(current.created) > client.maxAge || current.requests > maxSessionRequests || current.failures >= maxConsecutiveFailures
}

func (client *Client) replace(ctx context.Context, current *session) error {
	if current.id != "" {
		metrics.HistogramTrackerSessionRequests.Observe(float64(current.requests))
	}
	current.clearPrewarmed()
	client.destroy(current.id)
	*current = session{}
	createdAt := time.Now()
	created, err := client.call(ctx, map[string]any{"cmd": "sessions.create"})
	if err != nil {
		return err
	}
	if created.Session == "" {
		return errors.New("FlareSolverr returned an empty session ID")
	}
	warm, err := client.getEmbedded(ctx, created.Session, homeURL, interactionsURL)
	if err == nil {
		err = pageError(warm)
	}
	if err != nil {
		client.destroy(created.Session)
		return fmt.Errorf("prewarm FlareSolverr session: %w", err)
	}
	*current = session{id: created.Session, created: createdAt, requests: 1, prewarmed: true}
	metrics.GaugeTrackerPrewarmedSessions.Inc()
	return nil
}

func (client *Client) recover(ctx context.Context, current *session) error {
	target := current.recoveryURL
	pageURL := homeURL + "rocket-league/profile/" + strings.TrimPrefix(target, profileURL) + "/overview"
	current.requests++
	page, err := client.getEmbedded(ctx, current.id, pageURL, target)
	if err == nil {
		err = pageError(page)
	}
	if err == nil {
		current.requests++
		page, err = client.get(ctx, current.id, target)
		if err == nil {
			_, err = ranksFromPage(page)
		}
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		current.failures++
		return fmt.Errorf("recover FlareSolverr session: %w", err)
	}
	current.failures = 0
	current.recoveryURL = ""
	if !current.prewarmed && !client.expired(current) {
		current.prewarmed = true
		metrics.GaugeTrackerPrewarmedSessions.Inc()
	}
	return nil
}

func (client *Client) maintain(ctx context.Context) {
	defer close(client.done)
	ticker := time.NewTicker(maintenanceInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-client.wake:
		}
		for remaining := len(client.sessions); remaining > 0; remaining-- {
			select {
			case current := <-client.sessions:
				if client.expired(current) {
					if err := client.replace(ctx, current); err != nil && ctx.Err() == nil {
						log.Ctx(ctx).Error().Err(err).Msg("Could not replace FlareSolverr session")
					}
				} else if current.recoveryURL != "" {
					if err := client.recover(ctx, current); err != nil && ctx.Err() == nil {
						log.Ctx(ctx).Error().Err(err).Msg("Could not recover FlareSolverr session")
					}
				}
				client.sessions <- current
			default:
			}
		}
	}
}

func (client *Client) GetRanks(ctx context.Context, platform, identifier string) (ranks *Ranks, err error) {
	started := time.Now()
	defer func() {
		metrics.HistogramTrackerRequestTime.Observe(time.Since(started).Seconds())
		outcome := "success"
		if err != nil {
			outcome = "error"
		}
		metrics.CounterTrackerRequests.WithLabelValues(outcome, requestErrorCode(err)).Inc()
	}()
	if !client.startRequest() {
		return nil, ErrUnavailable
	}
	defer client.active.Done()
	switch platform {
	case "ps":
		platform = "psn"
	case "xbox":
		platform = "xbl"
	}
	var current *session
	for remaining := len(client.sessions); remaining > 0; remaining-- {
		select {
		case <-ctx.Done():
			return nil, errors.Join(ErrUnavailable, ctx.Err())
		case <-client.done:
			return nil, ErrUnavailable
		case candidate := <-client.sessions:
			if client.expired(candidate) || candidate.recoveryURL != "" {
				candidate.clearPrewarmed()
				client.sessions <- candidate
				continue
			}
			current = candidate
		default:
		}
		if current != nil {
			break
		}
	}
	if current == nil {
		return nil, ErrUnavailable
	}
	defer func() {
		if client.expired(current) {
			current.clearPrewarmed()
		}
		client.sessions <- current
		select {
		case client.wake <- struct{}{}:
		default:
		}
	}()
	current.requests++
	target := profileURL + url.PathEscape(platform) + "/" + url.PathEscape(identifier)
	page, err := client.get(ctx, current.id, target)
	if err == nil {
		ranks, err = ranksFromPage(page)
	}
	if errors.Is(err, ErrChallenge) {
		current.recoveryURL = target
		current.clearPrewarmed()
	}
	if err == nil || errors.Is(err, ErrNotFound) {
		current.failures = 0
	} else {
		current.failures++
		if !errors.Is(err, ErrRateLimited) {
			err = errors.Join(ErrUnavailable, err)
		}
	}
	return ranks, err
}

func requestErrorCode(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrChallenge):
		return "captcha_required"
	case errors.Is(err, ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, errTransport):
		return "transport_error"
	case errors.Is(err, errSolver):
		return "solver_error"
	case errors.Is(err, errHTTP):
		return "http_error"
	case errors.Is(err, errParsing):
		return "parsing_error"
	case errors.Is(err, errAPI):
		return "api_error"
	default:
		return "unavailable"
	}
}

func (client *Client) get(ctx context.Context, sessionID, target string) (*response, error) {
	return client.call(ctx, map[string]any{"cmd": "request.get", "session": sessionID, "url": target, "maxTimeout": requestTimeout.Milliseconds()})
}

func (client *Client) getEmbedded(ctx context.Context, sessionID, target, probeURL string) (*response, error) {
	return client.call(ctx, map[string]any{
		"cmd": "request.get", "session": sessionID, "url": target, "maxTimeout": requestTimeout.Milliseconds(),
		"embeddedTurnstile": map[string]any{"probeUrl": probeURL, "dismissConsent": true},
	})
}

func (client *Client) call(ctx context.Context, payload map[string]any) (*response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errTransport, err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errTransport, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: FlareSolverr HTTP status %d", errHTTP, res.StatusCode)
	}
	var result response
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: %w", errSolver, err)
	}
	if result.Status != "ok" {
		return nil, fmt.Errorf("%w: %s", errSolver, result.Message)
	}
	return &result, nil
}

func ranksFromPage(page *response) (*Ranks, error) {
	ranks, err := parse(page.Solution.Response)
	if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrChallenge) && !errors.Is(err, errAPI) {
		if statusErr := pageError(page); statusErr != nil {
			err = statusErr
		}
	}
	return ranks, err
}

func pageError(page *response) error {
	if page.Solution.Status == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	if page.Solution.Status < 200 || page.Solution.Status >= 300 || strings.TrimSpace(page.Solution.Response) == "" {
		return fmt.Errorf("%w: Tracker HTTP status %d", errHTTP, page.Solution.Status)
	}
	return nil
}

func parse(text string) (*Ranks, error) {
	if strings.HasPrefix(strings.TrimSpace(text), "<") {
		tokenizer := html.NewTokenizer(strings.NewReader(text))
		var body strings.Builder
		inBody := false
		for {
			tokenType := tokenizer.Next()
			if tokenType == html.ErrorToken {
				if tokenizer.Err() != io.EOF {
					return nil, tokenizer.Err()
				}
				break
			}
			token := tokenizer.Token()
			if tokenType == html.StartTagToken && token.Data == "body" {
				inBody = true
			} else if tokenType == html.EndTagToken && token.Data == "body" {
				inBody = false
			} else if tokenType == html.TextToken && inBody {
				body.WriteString(token.Data)
			}
		}
		text = body.String()
	}
	if strings.Contains(text, "CollectorResultStatus::NotFound") {
		return nil, ErrNotFound
	}
	var payload struct {
		Errors []struct{ Code, Message string }
		Data   *struct {
			PlatformInfo *struct{ PlatformUserHandle string }
			Segments     []struct {
				Type       string
				Attributes struct{ PlaylistID *int }
				Stats      struct {
					Rating   struct{ Value float64 }
					Tier     struct{ Value float64 }
					Division struct{ Value float64 }
				}
			}
		}
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		if strings.Contains(text, "You are being rate limited") || strings.Contains(text, "You are making too many requests") {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("%w: %w", errParsing, err)
	}
	for _, apiError := range payload.Errors {
		if apiError.Code == "Warden::Challenge" {
			return nil, fmt.Errorf("%w: %s (%s)", ErrChallenge, apiError.Message, apiError.Code)
		}
		return nil, fmt.Errorf("%w %s: %s", errAPI, apiError.Code, apiError.Message)
	}
	if payload.Data == nil || payload.Data.PlatformInfo == nil {
		return nil, fmt.Errorf("%w: missing Tracker profile data", errParsing)
	}
	result := &Ranks{DisplayName: payload.Data.PlatformInfo.PlatformUserHandle, Ranks: []*PlayerRank{}}
	for _, segment := range payload.Data.Segments {
		if segment.Type != "playlist" || segment.Attributes.PlaylistID == nil {
			continue
		}
		playlist, known := playlistMapping[*segment.Attributes.PlaylistID]
		if !known {
			continue
		}
		result.Ranks = append(result.Ranks, &PlayerRank{Playlist: playlist, Mmr: int(segment.Stats.Rating.Value), Rank: int(segment.Stats.Tier.Value), Division: int(segment.Stats.Division.Value)})
	}
	return result, nil
}
