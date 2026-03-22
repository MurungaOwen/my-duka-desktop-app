package syncengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"inventory-desktop/internal/backend/store"
)

type LocalStore interface {
	ListPendingSyncRecords(limit int64) ([]store.SyncRecord, error)
	MarkSyncRecordsSynced(recordIDs []string) error
	ApplyIncomingMutations(sourceDeviceID string, mutations []store.SyncMutation) (store.SyncPushResponse, error)
}

type Config struct {
	BaseURL    string
	DeviceID   string
	BatchLimit int64
	Interval   time.Duration
	HTTPClient *http.Client
	OnCycle    func(StepResult, error, int)
}

type Engine struct {
	store      LocalStore
	client     *http.Client
	baseURL    string
	deviceID   string
	batchLimit int64
	interval   time.Duration

	lastCursor string
	onCycle    func(StepResult, error, int)
}

type StepResult struct {
	Pushed int `json:"pushed"`
	Pulled int `json:"pulled"`
}

func New(store LocalStore, cfg Config) (*Engine, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	deviceID := strings.TrimSpace(cfg.DeviceID)
	if baseURL == "" {
		return nil, fmt.Errorf("base url is required")
	}
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}
	if cfg.BatchLimit <= 0 || cfg.BatchLimit > 1000 {
		cfg.BatchLimit = 200
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Engine{
		store:      store,
		client:     cfg.HTTPClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		deviceID:   deviceID,
		batchLimit: cfg.BatchLimit,
		interval:   cfg.Interval,
		onCycle:    cfg.OnCycle,
	}, nil
}

func (e *Engine) RunOnce(ctx context.Context) (StepResult, error) {
	var result StepResult

	pushed, err := e.push(ctx)
	if err != nil {
		return result, err
	}
	result.Pushed = pushed

	pulled, err := e.pull(ctx)
	if err != nil {
		return result, err
	}
	result.Pulled = pulled

	return result, nil
}

func (e *Engine) Start(ctx context.Context) error {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	failures := 0
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		stepResult, err := e.RunOnce(ctx)
		if err != nil {
			failures++
			if e.onCycle != nil {
				e.onCycle(StepResult{}, err, failures)
			}
			backoff := BackoffForFailures(failures)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		failures = 0
		if e.onCycle != nil {
			e.onCycle(stepResult, nil, failures)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (e *Engine) push(ctx context.Context) (int, error) {
	pending, err := e.store.ListPendingSyncRecords(e.batchLimit)
	if err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}

	mutations := make([]store.SyncMutation, 0, len(pending))
	recordIDs := make([]string, 0, len(pending))
	for _, rec := range pending {
		mutations = append(mutations, store.SyncMutation{
			MutationID: rec.ID,
			TableName:  rec.TableName,
			RecordID:   rec.RecordID,
			Operation:  rec.Operation,
			Payload:    rec.Payload,
			CreatedAt:  rec.CreatedAt,
		})
		recordIDs = append(recordIDs, rec.ID)
	}

	reqBody, err := json.Marshal(store.SyncPushRequest{
		DeviceID:  e.deviceID,
		Mutations: mutations,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal push payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/sync/push", bytes.NewReader(reqBody))
	if err != nil {
		return 0, fmt.Errorf("build push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("push request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return 0, fmt.Errorf("push failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var pushResp store.SyncPushResponse
	if err := json.NewDecoder(resp.Body).Decode(&pushResp); err != nil {
		return 0, fmt.Errorf("decode push response: %w", err)
	}

	if err := e.store.MarkSyncRecordsSynced(recordIDs); err != nil {
		return 0, err
	}

	return pushResp.Applied + pushResp.Skipped, nil
}

func (e *Engine) pull(ctx context.Context) (int, error) {
	endpoint, err := url.Parse(e.baseURL + "/sync/pull")
	if err != nil {
		return 0, fmt.Errorf("parse pull endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("device_id", e.deviceID)
	query.Set("since", e.lastCursor)
	query.Set("limit", fmt.Sprintf("%d", e.batchLimit))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("build pull request: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("pull request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return 0, fmt.Errorf("pull failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var pullResp store.SyncPullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		return 0, fmt.Errorf("decode pull response: %w", err)
	}
	if len(pullResp.Mutations) == 0 {
		return 0, nil
	}

	if _, err := e.store.ApplyIncomingMutations("remote", pullResp.Mutations); err != nil {
		return 0, err
	}

	for _, m := range pullResp.Mutations {
		if m.CreatedAt > e.lastCursor {
			e.lastCursor = m.CreatedAt
		}
	}

	return len(pullResp.Mutations), nil
}

func BackoffForFailures(failures int) time.Duration {
	switch {
	case failures <= 1:
		return 5 * time.Second
	case failures == 2:
		return 15 * time.Second
	case failures == 3:
		return 30 * time.Second
	default:
		return 60 * time.Second
	}
}
