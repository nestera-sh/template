package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ErrPushClient — Push Service ответил 4xx: запрос некорректен, повторять бессмысленно.
var ErrPushClient = errors.New("push service client error")

// ErrPushServer — Push Service ответил 5xx/429/503: временная проблема, можно повторить.
var ErrPushServer = errors.New("push service server error")

type Client struct {
	httpClient *http.Client
	baseURL    string
	tracer     trace.Tracer
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		tracer:     otel.Tracer("push-gateway"),
	}
}

type pushRequest struct {
	RequestID   string            `json:"request_id"`
	RecipientID string            `json:"recipient_id"`
	Kind        string            `json:"kind"`
	Data        map[string]string `json:"data"`
}

func (c *Client) RequestPosition(ctx context.Context, tripID, driverID string) error {
	ctx, span := c.tracer.Start(ctx, "push.RequestPosition",
		trace.WithAttributes(
			attribute.String("push.trip_id", tripID),
			attribute.String("push.driver_id", driverID),
		),
	)
	defer span.End()

	reqBody := pushRequest{
		RequestID:   uuid.New().String(),
		RecipientID: driverID,
		Kind:        "REQUEST_POSITION",
		Data:        map[string]string{"trip_id": tripID},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/api/v1/push"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 500:
		err := fmt.Errorf("%w: status %d", ErrPushServer, resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	case resp.StatusCode == http.StatusTooManyRequests:
		err := fmt.Errorf("%w: status %d (retry-after=%s)", ErrPushServer, resp.StatusCode, resp.Header.Get("Retry-After"))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	case resp.StatusCode >= 400:
		err := fmt.Errorf("%w: status %d", ErrPushClient, resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
