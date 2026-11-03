package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const telemetryTransformationRuleJSON = `{
  "kind": "Dash0TelemetryTransformationRule",
  "metadata": {
    "name": "redact-checkout",
    "labels": {
      "dash0.com/id": "rule-71",
      "dash0.com/origin": "terraform-checkout-19",
      "dash0.com/dataset": "body-dataset",
      "dash0.com/source": "terraform"
    },
    "annotations": {
      "dash0.com/created-at": "2026-10-25T10:00:00Z",
      "dash0.com/updated-at": "2026-10-27T14:03:08.123456789Z",
      "dash0.com/deleted-at": "2026-10-26T12:00:00Z"
    }
  },
  "spec": {
    "enabled": true,
    "signal": "logs",
    "context": "log",
    "priority": 100,
    "display": {"name": "Redact checkout secrets"},
    "conditions": ["resource.attributes[\"service.name\"] == \"checkout\"", "severity_number > 9"],
    "transformations": [
      {"name": "redact", "statements": ["delete_key(attributes, \"password\")", "set(attributes[\"redacted\"], true)"]},
      {"name": "tag", "statements": ["set(attributes[\"team\"], \"payments\")"]}
    ],
    "examples": [{
      "name": "checkout warning",
      "otlp": {"resourceLogs": [{
        "resource": {"attributes": [{"key": "service.name", "value": {"stringValue": "checkout"}}]},
        "scopeLogs": [{"scope": {"name": "checkout-sdk"}, "logRecords": [{
          "timeUnixNano": "1793109788123456789",
          "severityNumber": 13,
          "body": {"stringValue": "Payment accepted"},
          "attributes": [{"key": "order.id", "value": {"intValue": "9007199254740993"}}]
        }]}]
      }]}
    }]
  }
}`

func newTestTelemetryTransformationRule(t *testing.T) *TelemetryTransformationRuleDefinition {
	t.Helper()
	var rule TelemetryTransformationRuleDefinition
	if err := json.Unmarshal([]byte(telemetryTransformationRuleJSON), &rule); err != nil {
		t.Fatal(err)
	}
	return &rule
}

func assertTelemetryTransformationRuleJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotJSON, wantJSON any
	if err := json.Unmarshal(got, &gotJSON); err != nil {
		t.Errorf("invalid JSON response or request: %v", err)
		return
	}
	if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Errorf("invalid expected JSON: %v", err)
		return
	}
	if diff := cmp.Diff(wantJSON, gotJSON); diff != "" {
		t.Errorf("JSON mismatch (-want +got):\n%s", diff)
	}
}

func TestTelemetryTransformationRules_Create(t *testing.T) {
	rule := newTestTelemetryTransformationRule(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodPost)
		assertEqual(t, "path", r.URL.Path, "/api/telemetry-transformation-rules")
		assertEqual(t, "dataset", r.URL.Query().Get("dataset"), "production")
		assertEqual(t, "Authorization", r.Header.Get("Authorization"), "Bearer auth_test123")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		assertTelemetryTransformationRuleJSON(t, body, telemetryTransformationRuleJSON)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, telemetryTransformationRuleJSON)
	}))
	defer server.Close()

	created, err := newTestClient(t, server.URL).CreateTelemetryTransformationRule(context.Background(), rule, Ptr("production"))
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(created)
	if err != nil {
		t.Fatal(err)
	}
	assertTelemetryTransformationRuleJSON(t, body, telemetryTransformationRuleJSON)
}

func TestTelemetryTransformationRules_ReadModifyWrite(t *testing.T) {
	updatedJSON := strings.Replace(telemetryTransformationRuleJSON, `"enabled": true`, `"enabled": false`, 1)
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		assertEqual(t, "path", r.URL.Path, "/api/telemetry-transformation-rules/terraform-checkout-19")
		assertEqual(t, "dataset", r.URL.Query().Get("dataset"), "production")
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, telemetryTransformationRuleJSON)
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read update: %v", err)
				return
			}
			assertTelemetryTransformationRuleJSON(t, body, updatedJSON)
			_, _ = io.WriteString(w, updatedJSON)
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	rule, err := c.GetTelemetryTransformationRule(context.Background(), "terraform-checkout-19", Ptr("production"))
	if err != nil {
		t.Fatal(err)
	}
	rule.Spec.Enabled = false
	updated, err := c.UpdateTelemetryTransformationRule(context.Background(), "terraform-checkout-19", rule, Ptr("production"))
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(updated)
	if err != nil {
		t.Fatal(err)
	}
	assertTelemetryTransformationRuleJSON(t, body, updatedJSON)
	if diff := cmp.Diff([]string{http.MethodGet, http.MethodPut}, methods); diff != "" {
		t.Errorf("HTTP methods (-want +got):\n%s", diff)
	}
}

func TestTelemetryTransformationRules_ListAndIter(t *testing.T) {
	const listJSON = `{"telemetryTransformationRules":[
		{"kind":"Dash0TelemetryTransformationRule","metadata":{"name":"redact"},"spec":{"enabled":true,"signal":"logs","context":"log","transformations":[{"name":"redact","statements":["delete_key(attributes, \"password\")"]}]}},
		{"kind":"Dash0TelemetryTransformationRule","metadata":{"name":"tag"},"spec":{"enabled":false,"signal":"metrics","context":"datapoint","priority":300,"transformations":[{"name":"tag","statements":["set(attributes[\"team\"], \"payments\")"]}]}}
	]}`
	for _, tt := range []struct {
		name    string
		dataset *string
		body    string
		want    []string
	}{
		{"default dataset", nil, listJSON, []string{"redact", "tag"}},
		{"explicit dataset", Ptr("production"), listJSON, []string{"redact", "tag"}},
		{"empty list", Ptr("production"), `{"telemetryTransformationRules":[]}`, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertEqual(t, "method", r.Method, http.MethodGet)
				assertEqual(t, "path", r.URL.Path, "/api/telemetry-transformation-rules")
				if tt.dataset == nil {
					if r.URL.Query().Has("dataset") {
						t.Error("nil dataset must not set a dataset query parameter")
					}
				} else {
					assertEqual(t, "dataset", r.URL.Query().Get("dataset"), *tt.dataset)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			c := newTestClient(t, server.URL)
			rules, err := c.ListTelemetryTransformationRules(context.Background(), tt.dataset)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, rule := range rules {
				names = append(names, rule.Metadata.Name)
				if rule.Spec.Examples != nil {
					t.Error("list must not manufacture examples")
				}
			}
			if diff := cmp.Diff(tt.want, names); diff != "" {
				t.Errorf("listed names (-want +got):\n%s", diff)
			}
			iter := c.ListTelemetryTransformationRulesIter(context.Background(), tt.dataset)
			names = nil
			for iter.Next() {
				names = append(names, iter.Current().Metadata.Name)
			}
			if err := iter.Err(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, names); diff != "" {
				t.Errorf("iterated names (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTelemetryTransformationRules_APIError(t *testing.T) {
	rule := newTestTelemetryTransformationRule(t)
	for _, tt := range []struct {
		name   string
		status int
		body   string
		header string
		trace  string
		call   func(Client) error
	}{
		{"list forbidden", 403, `{"error":{"code":403,"message":"forbidden","traceId":"body-trace-41"}}`, "header-trace-7", "header-trace-7", func(c Client) error {
			_, err := c.ListTelemetryTransformationRules(context.Background(), Ptr("production"))
			return err
		}},
		{"get missing", 404, `{"error":{"code":404,"message":"not found","traceId":"body-trace-41"}}`, "", "body-trace-41", func(c Client) error {
			_, err := c.GetTelemetryTransformationRule(context.Background(), "missing", Ptr("production"))
			return err
		}},
		{"create invalid rule", 400, `{"error":{"code":400,"type":"invalid_rule","message":"rule rejected","traceId":"body-trace-41","violations":[{"kind":"protected","attribute":"dash0.secret","message":"protected attribute"}]}}`, "", "body-trace-41", func(c Client) error {
			_, err := c.CreateTelemetryTransformationRule(context.Background(), rule, Ptr("production"))
			return err
		}},
		{"create conflict", 409, `{"error":{"code":409,"message":"origin exists","traceId":"body-trace-41"}}`, "", "body-trace-41", func(c Client) error {
			_, err := c.CreateTelemetryTransformationRule(context.Background(), rule, Ptr("production"))
			return err
		}},
		{"update stale", 400, `{"error":{"code":400,"type":"superseded_version","message":"rule changed","traceId":"body-trace-41"}}`, "", "body-trace-41", func(c Client) error {
			_, err := c.UpdateTelemetryTransformationRule(context.Background(), "rule-71", rule, Ptr("production"))
			return err
		}},
		{"update missing is not upsert", 404, `{"error":{"code":404,"message":"not found","traceId":"body-trace-41"}}`, "", "body-trace-41", func(c Client) error {
			_, err := c.UpdateTelemetryTransformationRule(context.Background(), "missing", rule, Ptr("production"))
			return err
		}},
		{"delete forbidden", 403, `{"error":{"code":403,"message":"forbidden","traceId":"body-trace-41"}}`, "", "body-trace-41", func(c Client) error {
			return c.DeleteTelemetryTransformationRule(context.Background(), "rule-71", Ptr("production"))
		}},
		{"iterator server error", 500, `{"error":{"code":500,"message":"unavailable","traceId":"body-trace-41"}}`, "", "body-trace-41", func(c Client) error {
			iter := c.ListTelemetryTransformationRulesIter(context.Background(), Ptr("production"))
			if iter.Next() {
				t.Error("error iterator must not yield a rule")
			}
			return iter.Err()
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assertEqual(t, "dataset", r.URL.Query().Get("dataset"), "production")
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("x-trace-id", tt.header)
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			c, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"), WithTransport(NewTransport(WithTransportMaxRetries(0))))
			if err != nil {
				t.Fatal(err)
			}
			err = tt.call(c)
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected APIError, got %T: %v", err, err)
			}
			if apiErr.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", apiErr.StatusCode, tt.status)
			}
			assertEqual(t, "body", apiErr.Body, tt.body)
			assertEqual(t, "trace", apiErr.TraceID, tt.trace)
			if apiErr.Message == "" {
				t.Error("APIError must retain the server message")
			}
			if requests.Load() != 1 {
				t.Errorf("expected a single failed request, got %d", requests.Load())
			}
			body, err := json.Marshal(rule)
			if err != nil {
				t.Fatal(err)
			}
			assertTelemetryTransformationRuleJSON(t, body, telemetryTransformationRuleJSON)
		})
	}
}

func TestTelemetryTransformationRules_RequestRouting(t *testing.T) {
	rule := newTestTelemetryTransformationRule(t)
	for _, tt := range []struct {
		method string
		path   string
		call   func(Client, *string) error
	}{
		{http.MethodGet, "", func(c Client, dataset *string) error {
			_, err := c.ListTelemetryTransformationRules(context.Background(), dataset)
			return err
		}},
		{http.MethodGet, "/terraform-checkout-19", func(c Client, dataset *string) error {
			_, err := c.GetTelemetryTransformationRule(context.Background(), "terraform-checkout-19", dataset)
			return err
		}},
		{http.MethodPost, "", func(c Client, dataset *string) error {
			_, err := c.CreateTelemetryTransformationRule(context.Background(), rule, dataset)
			return err
		}},
		{http.MethodPut, "/terraform-checkout-19", func(c Client, dataset *string) error {
			_, err := c.UpdateTelemetryTransformationRule(context.Background(), "terraform-checkout-19", rule, dataset)
			return err
		}},
		{http.MethodDelete, "/terraform-checkout-19", func(c Client, dataset *string) error {
			return c.DeleteTelemetryTransformationRule(context.Background(), "terraform-checkout-19", dataset)
		}},
	} {
		for _, dataset := range []*string{nil, Ptr("non-default")} {
			name := "omitted dataset"
			if dataset != nil {
				name = *dataset
			}
			t.Run(tt.method+tt.path+"/"+name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assertEqual(t, "method", r.Method, tt.method)
					assertEqual(t, "path", r.URL.Path, "/api/telemetry-transformation-rules"+tt.path)
					if dataset == nil {
						if r.URL.Query().Has("dataset") {
							t.Error("nil dataset must leave routing to the server")
						}
					} else {
						assertEqual(t, "dataset", r.URL.Query().Get("dataset"), "non-default")
					}
					if r.URL.Query().Has("updatedAt") {
						t.Error("unconditional operations must not invent an updatedAt parameter")
					}
					w.Header().Set("Content-Type", "application/json")
					if tt.method == http.MethodDelete {
						w.WriteHeader(http.StatusOK)
						return
					}
					if tt.method == http.MethodGet && tt.path == "" {
						_, _ = io.WriteString(w, `{"telemetryTransformationRules":[]}`)
						return
					}
					_, _ = io.WriteString(w, telemetryTransformationRuleJSON)
				}))
				defer server.Close()
				if err := tt.call(newTestClient(t, server.URL), dataset); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestTelemetryTransformationRules_UpdateWithoutExamples(t *testing.T) {
	for _, tt := range []struct {
		name     string
		examples *[]TelemetryTransformationRuleExample
	}{
		{"omitted", nil},
		{"explicitly empty", Ptr([]TelemetryTransformationRuleExample{})},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rule := newTestTelemetryTransformationRule(t)
			rule.Spec.Examples = tt.examples
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assertEqual(t, "method", r.Method, http.MethodPut)
				var body struct {
					Spec map[string]json.RawMessage `json:"spec"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode update: %v", err)
					return
				}
				if tt.examples == nil {
					if _, present := body.Spec["examples"]; present {
						t.Error("omitted examples must not be fetched or merged into the update")
					}
				} else {
					assertEqual(t, "empty examples", string(body.Spec["examples"]), "[]")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"kind":"Dash0TelemetryTransformationRule","metadata":{"name":"redact-checkout"},"spec":{"enabled":true,"signal":"logs","context":"log","transformations":[]}}`)
			}))
			defer server.Close()
			updated, err := newTestClient(t, server.URL).UpdateTelemetryTransformationRule(context.Background(), "rule-71", rule, nil)
			if err != nil {
				t.Fatal(err)
			}
			if updated.Spec.Examples != nil {
				t.Error("the response must not inherit examples from the request or a cached rule")
			}
			if requests.Load() != 1 {
				t.Errorf("replacement must send only PUT, got %d requests", requests.Load())
			}
		})
	}
}

func TestTelemetryTransformationRules_CreateLostResponse(t *testing.T) {
	var creates atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodPost)
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("read create: %v", err)
			return
		}
		creates.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack response: %v", err)
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()
	c, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"), WithTransport(NewTransport(
		WithTransportMaxRetries(3),
		WithTransportRetryWaitMin(time.Millisecond),
		WithTransportRetryWaitMax(time.Millisecond),
		WithTransportRetryOnConflict(),
	)))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateTelemetryTransformationRule(context.Background(), newTestTelemetryTransformationRule(t), nil)
	if err == nil {
		t.Fatal("a lost response must return an error, not an invented successful rule")
	}
	if creates.Load() != 1 {
		t.Errorf("ambiguous POST must not be replayed, got %d creates", creates.Load())
	}
}

func TestTelemetryTransformationRules_StripServerFields(t *testing.T) {
	rule := newTestTelemetryTransformationRule(t)
	StripTelemetryTransformationRuleServerFields(rule)
	want := strings.Replace(telemetryTransformationRuleJSON, `"dash0.com/dataset": "body-dataset",`, `"dash0.com/dataset": "body-dataset"`, 1)
	want = strings.Replace(want, `"dash0.com/source": "terraform"`, "", 1)
	want = strings.Replace(want, `"dash0.com/created-at": "2026-10-25T10:00:00Z",`, "", 1)
	want = strings.Replace(want, `"dash0.com/updated-at": "2026-10-27T14:03:08.123456789Z",`, `"dash0.com/updated-at": "2026-10-27T14:03:08.123456789Z"`, 1)
	want = strings.Replace(want, `"dash0.com/deleted-at": "2026-10-26T12:00:00Z"`, "", 1)
	body, err := json.Marshal(rule)
	if err != nil {
		t.Fatal(err)
	}
	assertTelemetryTransformationRuleJSON(t, body, want)
	ClearTelemetryTransformationRuleID(rule)
	want = strings.Replace(want, `"dash0.com/id": "rule-71",`, "", 1)
	body, err = json.Marshal(rule)
	if err != nil {
		t.Fatal(err)
	}
	assertTelemetryTransformationRuleJSON(t, body, want)
}

func TestTelemetryTransformationRules_UnparsedSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "not a JSON rule")
	}))
	defer server.Close()
	c := newTestClient(t, server.URL)
	rule := newTestTelemetryTransformationRule(t)
	for name, call := range map[string]func() error{
		"list": func() error {
			_, err := c.ListTelemetryTransformationRules(context.Background(), nil)
			return err
		},
		"get": func() error {
			_, err := c.GetTelemetryTransformationRule(context.Background(), "rule-71", nil)
			return err
		},
		"create": func() error {
			_, err := c.CreateTelemetryTransformationRule(context.Background(), rule, nil)
			return err
		},
		"update": func() error {
			_, err := c.UpdateTelemetryTransformationRule(context.Background(), "rule-71", rule, nil)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil {
				t.Error("a non-JSON 200 must return an error, not nil success")
			}
		})
	}
}

func TestTelemetryTransformationRules_APINotConfigured(t *testing.T) {
	c, err := NewClient(WithOtlpEndpoint(OtlpEncodingJson, "https://otlp.example.com"), WithAuthToken("auth_test123"))
	if err != nil {
		t.Fatal(err)
	}
	rule := newTestTelemetryTransformationRule(t)
	ctx := context.Background()
	if _, err := c.ListTelemetryTransformationRules(ctx, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("list error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.GetTelemetryTransformationRule(ctx, "rule-71", nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("get error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.CreateTelemetryTransformationRule(ctx, rule, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("create error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.UpdateTelemetryTransformationRule(ctx, "rule-71", rule, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("update error = %v, want ErrAPINotConfigured", err)
	}
	if err := c.DeleteTelemetryTransformationRule(ctx, "rule-71", nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("delete error = %v, want ErrAPINotConfigured", err)
	}
	iter := c.ListTelemetryTransformationRulesIter(ctx, nil)
	if iter.Next() || !errors.Is(iter.Err(), ErrAPINotConfigured) {
		t.Errorf("iterator error = %v, want ErrAPINotConfigured without results", iter.Err())
	}
}

func TestTelemetryTransformationRules_NilInput(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	c := newTestClient(t, server.URL)
	if _, err := c.CreateTelemetryTransformationRule(context.Background(), nil, nil); err == nil {
		t.Error("create must reject nil input")
	}
	if _, err := c.UpdateTelemetryTransformationRule(context.Background(), "rule-71", nil, nil); err == nil {
		t.Error("update must reject nil input")
	}
	if requests.Load() != 0 {
		t.Errorf("nil input must not reach the server, got %d requests", requests.Load())
	}
}
