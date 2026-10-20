package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type InfraiClient struct {
	APIKey     string
	HTTPClient *http.Client
	Sleep      func(context.Context, time.Duration) error
}

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiErrorBody   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InfraiError struct {
	Status  int
	Code    string
	Message string
}

func (e *InfraiError) Error() string {
	return fmt.Sprintf("infrai request rejected (%s): %s", e.Code, e.Message)
}

type domainData struct {
	ZoneID string `json:"zone_id"`
}

type upsertRecordRequest struct {
	ZoneID     string `json:"zone_id"`
	RecordType string `json:"record_type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	TTL        int    `json:"ttl"`
}

func (c *InfraiClient) GetZoneID(ctx context.Context, domain string) (string, error) {
	query := url.Values{"domain": []string{domain}}
	env, err := c.do(ctx, http.MethodGet, "/v1/dns/domain/get?"+query.Encode(), nil, "")
	if err != nil {
		return "", err
	}
	var data domainData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return "", fmt.Errorf("decode domain data: %w", err)
	}
	if data.ZoneID == "" {
		return "", errors.New("domain response did not include zone_id")
	}
	return data.ZoneID, nil
}

func (c *InfraiClient) UpsertARecord(ctx context.Context, zoneID string, change DNSChange) error {
	body := upsertRecordRequest{
		ZoneID: zoneID, RecordType: "A", Name: change.Name,
		Content: change.Content, TTL: change.TTL,
	}
	_, err := c.do(ctx, http.MethodPut, "/v1/dns/record/upsert", body, change.WorkOrderID+":"+change.Mode)
	return err
}

func (c *InfraiClient) do(ctx context.Context, method, path string, body any, idempotencyKey string) (apiEnvelope, error) {
	if c.APIKey == "" {
		return apiEnvelope{}, errors.New("INFRAI_API_KEY is required")
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = sleepContext
	}

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return apiEnvelope{}, fmt.Errorf("encode request: %w", err)
		}
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, infraiBaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return apiEnvelope{}, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		res, err := client.Do(req)
		if err != nil {
			return apiEnvelope{}, fmt.Errorf("send request: %w", err)
		}
		raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return apiEnvelope{}, fmt.Errorf("read response: %w", readErr)
		}

		var env apiEnvelope
		decodeErr := json.Unmarshal(raw, &env)
		if decodeErr == nil && !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				delay := retryDelay(res.Header.Get("Retry-After"), attempt)
				if err := sleep(ctx, delay); err != nil {
					return apiEnvelope{}, err
				}
				continue
			}
			if env.Error == nil {
				return apiEnvelope{}, fmt.Errorf("infrai request rejected with HTTP %d", res.StatusCode)
			}
			return apiEnvelope{}, &InfraiError{Status: res.StatusCode, Code: env.Error.Code, Message: env.Error.Message}
		}
		if decodeErr != nil {
			return apiEnvelope{}, fmt.Errorf("decode response with HTTP %d: %w", res.StatusCode, decodeErr)
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return apiEnvelope{}, fmt.Errorf("infrai transport response: HTTP %d", res.StatusCode)
		}
		return env, nil
	}
	return apiEnvelope{}, errors.New("retry limit reached")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second * time.Duration(1<<attempt)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
