package dash0

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStripNotificationChannelServerFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()

	c := &NotificationChannelDefinition{
		Metadata: NotificationChannelMetadata{
			Annotations: &NotificationChannelAnnotations{
				Dash0ComcreatedAt: &createdAt,
				Dash0ComupdatedAt: &updatedAt,
			},
			Labels: &NotificationChannelLabels{
				Dash0Comid:     Ptr("keep-this"),
				Dash0Comorigin: Ptr("my-origin"),
			},
		},
	}

	StripNotificationChannelServerFields(c)

	if c.Metadata.Annotations.Dash0ComcreatedAt != nil {
		t.Error("Dash0ComcreatedAt should be nil")
	}
	if c.Metadata.Annotations.Dash0ComupdatedAt != nil {
		t.Error("Dash0ComupdatedAt should be nil")
	}
	if c.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
	if c.Metadata.Labels.Dash0Comorigin == nil || *c.Metadata.Labels.Dash0Comorigin != "my-origin" {
		t.Error("Dash0Comorigin should be preserved")
	}
}

func TestStripNotificationChannelServerFields_NilLabels(t *testing.T) {
	c := &NotificationChannelDefinition{}
	StripNotificationChannelServerFields(c) // should not panic
	if c.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
}

func TestStripNotificationChannelServerFields_Nil(t *testing.T) {
	StripNotificationChannelServerFields(nil) // should not panic
}

func TestClearNotificationChannelID(t *testing.T) {
	c := &NotificationChannelDefinition{Metadata: NotificationChannelMetadata{Labels: &NotificationChannelLabels{Dash0Comid: Ptr("nc-1")}}}
	ClearNotificationChannelID(c)
	if c.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearNotificationChannelID_NilLabels(t *testing.T) {
	c := &NotificationChannelDefinition{}
	ClearNotificationChannelID(c) // should not panic
}

func TestClearNotificationChannelID_NilChannel(t *testing.T) {
	ClearNotificationChannelID(nil) // should not panic
}

func TestGetNotificationChannelID(t *testing.T) {
	tests := []struct {
		name    string
		channel *NotificationChannelDefinition
		want    string
	}{
		{"nil channel", nil, ""},
		{"nil labels", &NotificationChannelDefinition{}, ""},
		{"nil ID", &NotificationChannelDefinition{Metadata: NotificationChannelMetadata{Labels: &NotificationChannelLabels{}}}, ""},
		{"with ID", &NotificationChannelDefinition{Metadata: NotificationChannelMetadata{Labels: &NotificationChannelLabels{Dash0Comid: Ptr("nc-123")}}}, "nc-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNotificationChannelID(tt.channel); got != tt.want {
				t.Errorf("GetNotificationChannelID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetNotificationChannelID(t *testing.T) {
	c := &NotificationChannelDefinition{}
	SetNotificationChannelID(c, "new-id")
	if c.Metadata.Labels == nil || c.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *c.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *c.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetNotificationChannelID_Overwrites(t *testing.T) {
	c := &NotificationChannelDefinition{
		Metadata: NotificationChannelMetadata{Labels: &NotificationChannelLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetNotificationChannelID(c, "new-id")
	if *c.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *c.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetNotificationChannelID_Nil(t *testing.T) {
	SetNotificationChannelID(nil, "new-id") // should not panic
}

func TestSetNotificationChannelIDIfAbsent(t *testing.T) {
	c := &NotificationChannelDefinition{}
	SetNotificationChannelIDIfAbsent(c, "new-id")
	if c.Metadata.Labels == nil || c.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *c.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *c.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetNotificationChannelIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	c := &NotificationChannelDefinition{
		Metadata: NotificationChannelMetadata{Labels: &NotificationChannelLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetNotificationChannelIDIfAbsent(c, "new-id")
	if *c.Metadata.Labels.Dash0Comid != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", *c.Metadata.Labels.Dash0Comid, "existing-id")
	}
}

func TestSetNotificationChannelIDIfAbsent_Nil(t *testing.T) {
	SetNotificationChannelIDIfAbsent(nil, "new-id") // should not panic
}

func TestGetNotificationChannelName(t *testing.T) {
	tests := []struct {
		name    string
		channel *NotificationChannelDefinition
		want    string
	}{
		{"nil channel", nil, ""},
		{"empty name", &NotificationChannelDefinition{}, ""},
		{"with name", &NotificationChannelDefinition{Metadata: NotificationChannelMetadata{Name: "Slack Alerts"}}, "Slack Alerts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNotificationChannelName(tt.channel); got != tt.want {
				t.Errorf("GetNotificationChannelName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetNotificationChannelOrigin(t *testing.T) {
	tests := []struct {
		name    string
		channel *NotificationChannelDefinition
		want    string
	}{
		{"nil channel", nil, ""},
		{"nil labels", &NotificationChannelDefinition{}, ""},
		{"nil origin", &NotificationChannelDefinition{Metadata: NotificationChannelMetadata{Labels: &NotificationChannelLabels{}}}, ""},
		{"with origin", &NotificationChannelDefinition{Metadata: NotificationChannelMetadata{Labels: &NotificationChannelLabels{Dash0Comorigin: Ptr("terraform")}}}, "terraform"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNotificationChannelOrigin(tt.channel); got != tt.want {
				t.Errorf("GetNotificationChannelOrigin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetNotificationChannelOrigin(t *testing.T) {
	c := &NotificationChannelDefinition{}
	SetNotificationChannelOrigin(c, "terraform")
	if c.Metadata.Labels == nil || c.Metadata.Labels.Dash0Comorigin == nil {
		t.Fatal("expected origin to be set")
	}
	if *c.Metadata.Labels.Dash0Comorigin != "terraform" {
		t.Errorf("Origin = %q, want %q", *c.Metadata.Labels.Dash0Comorigin, "terraform")
	}
}

func TestSetNotificationChannelOrigin_Nil(t *testing.T) {
	SetNotificationChannelOrigin(nil, "terraform") // should not panic
}

func newTestNotificationChannel() NotificationChannelDefinition {
	return NotificationChannelDefinition{
		Kind: Dash0NotificationChannel,
		Metadata: NotificationChannelMetadata{
			Name: "Slack Alerts",
			Labels: &NotificationChannelLabels{
				Dash0Comid: Ptr("nc-123"),
			},
		},
	}
}

func TestNotificationChannels_Integration(t *testing.T) {
	t.Run("ListNotificationChannels returns channels", func(t *testing.T) {
		channel := newTestNotificationChannel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/notification-channels" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]NotificationChannelDefinition{channel})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		channels, err := client.ListNotificationChannels(context.Background())
		if err != nil {
			t.Fatalf("ListNotificationChannels failed: %v", err)
		}

		if len(channels) != 1 {
			t.Fatalf("expected 1 channel, got %d", len(channels))
		}
		if channels[0].Metadata.Name != "Slack Alerts" {
			t.Errorf("expected name %q, got %q", "Slack Alerts", channels[0].Metadata.Name)
		}
	})

	t.Run("ListNotificationChannels returns empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]NotificationChannelDefinition{})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		channels, err := client.ListNotificationChannels(context.Background())
		if err != nil {
			t.Fatalf("ListNotificationChannels failed: %v", err)
		}

		if len(channels) != 0 {
			t.Errorf("expected empty list, got %d channels", len(channels))
		}
	})

	t.Run("GetNotificationChannel returns channel", func(t *testing.T) {
		channel := newTestNotificationChannel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/notification-channels/nc-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(channel)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.GetNotificationChannel(context.Background(), "nc-123")
		if err != nil {
			t.Fatalf("GetNotificationChannel failed: %v", err)
		}

		if got.Metadata.Name != "Slack Alerts" {
			t.Errorf("expected name %q, got %q", "Slack Alerts", got.Metadata.Name)
		}
	})

	t.Run("GetNotificationChannel handles 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Notification channel not found",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetNotificationChannel(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error for 404 response")
		}

		if !IsNotFound(err) {
			t.Errorf("expected IsNotFound to return true")
		}
	})

	t.Run("CreateNotificationChannel succeeds with 200", func(t *testing.T) {
		channel := newTestNotificationChannel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/notification-channels" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			var reqChannel NotificationChannelDefinition
			if err := json.Unmarshal(body, &reqChannel); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if reqChannel.Metadata.Name != "Slack Alerts" {
				t.Errorf("expected request name %q, got %q", "Slack Alerts", reqChannel.Metadata.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(channel)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.CreateNotificationChannel(context.Background(), &channel)
		if err != nil {
			t.Fatalf("CreateNotificationChannel failed: %v", err)
		}

		if got.Metadata.Name != "Slack Alerts" {
			t.Errorf("expected name %q, got %q", "Slack Alerts", got.Metadata.Name)
		}
	})

	t.Run("CreateNotificationChannel succeeds with 201", func(t *testing.T) {
		channel := newTestNotificationChannel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(channel)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.CreateNotificationChannel(context.Background(), &channel)
		if err != nil {
			t.Fatalf("CreateNotificationChannel failed: %v", err)
		}

		if got.Metadata.Name != "Slack Alerts" {
			t.Errorf("expected name %q, got %q", "Slack Alerts", got.Metadata.Name)
		}
		if GetNotificationChannelID(got) != "nc-123" {
			t.Errorf("expected ID %q, got %q", "nc-123", GetNotificationChannelID(got))
		}
	})

	t.Run("CreateNotificationChannel handles error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Invalid channel definition",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		channel := newTestNotificationChannel()
		_, err = client.CreateNotificationChannel(context.Background(), &channel)
		if err == nil {
			t.Fatal("expected error for 400 response")
		}

		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d", http.StatusBadRequest, apiErr.StatusCode)
		}
	})

	t.Run("UpdateNotificationChannel succeeds", func(t *testing.T) {
		channel := newTestNotificationChannel()
		channel.Metadata.Name = "Updated Alerts"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/notification-channels/nc-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPut {
				t.Errorf("unexpected method: %s", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			var reqChannel NotificationChannelDefinition
			if err := json.Unmarshal(body, &reqChannel); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if reqChannel.Metadata.Name != "Updated Alerts" {
				t.Errorf("expected request name %q, got %q", "Updated Alerts", reqChannel.Metadata.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(channel)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.UpdateNotificationChannel(context.Background(), "nc-123", &channel)
		if err != nil {
			t.Fatalf("UpdateNotificationChannel failed: %v", err)
		}

		if got.Metadata.Name != "Updated Alerts" {
			t.Errorf("expected name %q, got %q", "Updated Alerts", got.Metadata.Name)
		}
	})

	t.Run("DeleteNotificationChannel succeeds with 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/notification-channels/nc-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodDelete {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.DeleteNotificationChannel(context.Background(), "nc-123")
		if err != nil {
			t.Fatalf("DeleteNotificationChannel failed: %v", err)
		}
	})

	t.Run("DeleteNotificationChannel succeeds with 204", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.DeleteNotificationChannel(context.Background(), "nc-123")
		if err != nil {
			t.Fatalf("DeleteNotificationChannel failed: %v", err)
		}
	})

	t.Run("DeleteNotificationChannel handles error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Forbidden",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.DeleteNotificationChannel(context.Background(), "nc-123")
		if err == nil {
			t.Fatal("expected error for 403 response")
		}

		if !IsForbidden(err) {
			t.Errorf("expected IsForbidden to return true")
		}
	})

	t.Run("ListNotificationChannelsIter iterates channels", func(t *testing.T) {
		channels := []NotificationChannelDefinition{
			newTestNotificationChannel(),
			{
				Kind:     Dash0NotificationChannel,
				Metadata: NotificationChannelMetadata{Name: "Email Alerts"},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(channels)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		iter := client.ListNotificationChannelsIter(context.Background())
		var names []string
		for iter.Next() {
			names = append(names, iter.Current().Metadata.Name)
		}
		if iter.Err() != nil {
			t.Fatalf("iterator error: %v", iter.Err())
		}

		if len(names) != 2 {
			t.Fatalf("expected 2 channels, got %d", len(names))
		}
		if names[0] != "Slack Alerts" {
			t.Errorf("expected first channel %q, got %q", "Slack Alerts", names[0])
		}
		if names[1] != "Email Alerts" {
			t.Errorf("expected second channel %q, got %q", "Email Alerts", names[1])
		}
	})

	t.Run("verifies authorization header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer auth_test123" {
				t.Errorf("unexpected Authorization header: %s", auth)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]NotificationChannelDefinition{})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListNotificationChannels(context.Background())
		if err != nil {
			t.Fatalf("ListNotificationChannels failed: %v", err)
		}
	})
}
