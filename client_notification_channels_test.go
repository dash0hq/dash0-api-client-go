package dash0

import (
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
