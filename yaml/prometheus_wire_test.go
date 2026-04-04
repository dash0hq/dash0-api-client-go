package yaml

import (
	"testing"
	"time"
)

func TestPrometheusDuration_JSONRoundTrip(t *testing.T) {
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
			var d prometheusDuration
			if err := d.UnmarshalJSON([]byte(tt.json)); err != nil {
				t.Fatalf("UnmarshalJSON(%s) error: %v", tt.json, err)
			}
			if time.Duration(d) != tt.want {
				t.Errorf("got %v, want %v", time.Duration(d), tt.want)
			}
		})
	}
}

func TestPrometheusDuration_MarshalJSON(t *testing.T) {
	d := prometheusDuration(2*time.Minute + 30*time.Second)
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2m30s"` {
		t.Errorf("MarshalJSON() = %s, want %q", b, "2m30s")
	}
}

func TestPrometheusDuration_InvalidType(t *testing.T) {
	var d prometheusDuration
	err := d.UnmarshalJSON([]byte(`true`))
	if err == nil {
		t.Error("expected error for bool type")
	}
}
