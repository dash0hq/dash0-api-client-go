package yaml

import (
	"testing"
	"time"
)

func TestPromDuration_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		json string
		want time.Duration
	}{
		{"string duration", `"5m0s"`, 5 * time.Minute},
		{"string short", `"30s"`, 30 * time.Second},
		{"numeric zero", `0`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d PromDuration
			if err := d.UnmarshalJSON([]byte(tt.json)); err != nil {
				t.Fatalf("UnmarshalJSON(%s) error: %v", tt.json, err)
			}
			if time.Duration(d) != tt.want {
				t.Errorf("got %v, want %v", time.Duration(d), tt.want)
			}
		})
	}
}

func TestPromDuration_MarshalJSON(t *testing.T) {
	d := PromDuration(2*time.Minute + 30*time.Second)
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2m30s"` {
		t.Errorf("MarshalJSON() = %s, want %q", b, "2m30s")
	}
}

func TestPromDuration_InvalidType(t *testing.T) {
	var d PromDuration
	err := d.UnmarshalJSON([]byte(`true`))
	if err == nil {
		t.Error("expected error for bool type")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0s"},
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours and minutes", 2*time.Hour + 30*time.Minute, "2h30m"},
		{"all components", 1*time.Hour + 2*time.Minute + 3*time.Second, "1h2m3s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
