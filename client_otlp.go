package dash0

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// ErrOTLPNotConfigured is returned when SendTraces, SendMetrics, or SendLogs
// is called on a client created without WithOtlpEndpoint.
var ErrOTLPNotConfigured = errors.New("dash0: OTLP endpoint not configured (use WithOtlpEndpoint)")

// sendOTLP sends a marshaled OTLP payload to the configured endpoint at the
// given signal path (e.g. "/v1/traces"). It uses the shared HTTP client with
// the retry and rate-limit transport stack. The request is marked as idempotent
// so the retry transport can retry on transient failures.
func (c *client) sendOTLP(ctx context.Context, path string, body []byte, dataset *string) error {
	url := strings.TrimRight(c.config.otlpEndpoint, "/") + path

	req, err := http.NewRequestWithContext(
		withIdempotent(ctx),
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("dash0: failed to create OTLP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// The Authorization header is set by authTransport, which wraps
	// c.httpClient's transport stack.
	req.Header.Set("User-Agent", c.config.userAgent)
	if dataset != nil && *dataset != "" {
		req.Header.Set("Dash0-Dataset", *dataset)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dash0: OTLP request failed: %w", err)
	}

	respBody, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return fmt.Errorf("dash0: failed to read OTLP response: %w", readErr)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return newAPIErrorWithBody(resp, respBody)
}

// SendTraces marshals trace data to OTLP/JSON and sends it to the
// configured OTLP endpoint. If dataset is non-nil, it is sent via the
// Dash0-Dataset HTTP header; otherwise the organization's default dataset is used.
func (c *client) SendTraces(ctx context.Context, traces ptrace.Traces, dataset *string) error {
	if c.config.otlpEndpoint == "" {
		return ErrOTLPNotConfigured
	}
	marshaler := &ptrace.JSONMarshaler{}
	body, err := marshaler.MarshalTraces(traces)
	if err != nil {
		return fmt.Errorf("dash0: failed to marshal traces: %w", err)
	}
	return c.sendOTLP(ctx, "/v1/traces", body, dataset)
}

// SendMetrics marshals metric data to OTLP/JSON and sends it to the
// configured OTLP endpoint. If dataset is non-nil, it is sent via the
// Dash0-Dataset HTTP header; otherwise the organization's default dataset is used.
func (c *client) SendMetrics(ctx context.Context, metrics pmetric.Metrics, dataset *string) error {
	if c.config.otlpEndpoint == "" {
		return ErrOTLPNotConfigured
	}
	marshaler := &pmetric.JSONMarshaler{}
	body, err := marshaler.MarshalMetrics(metrics)
	if err != nil {
		return fmt.Errorf("dash0: failed to marshal metrics: %w", err)
	}
	return c.sendOTLP(ctx, "/v1/metrics", body, dataset)
}

// SendLogs marshals log data to OTLP/JSON and sends it to the
// configured OTLP endpoint. If dataset is non-nil, it is sent via the
// Dash0-Dataset HTTP header; otherwise the organization's default dataset is used.
func (c *client) SendLogs(ctx context.Context, logs plog.Logs, dataset *string) error {
	if c.config.otlpEndpoint == "" {
		return ErrOTLPNotConfigured
	}
	marshaler := &plog.JSONMarshaler{}
	body, err := marshaler.MarshalLogs(logs)
	if err != nil {
		return fmt.Errorf("dash0: failed to marshal logs: %w", err)
	}
	return c.sendOTLP(ctx, "/v1/logs", body, dataset)
}

// Close is a no-op in the current implementation. The underlying HTTP client
// is shared between the REST API and OTLP and does not require explicit
// shutdown.
func (c *client) Close(_ context.Context) error {
	return nil
}
