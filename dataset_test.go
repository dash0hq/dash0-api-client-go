package dash0

import "testing"

func TestDatasetPtr(t *testing.T) {
	tests := []struct {
		name    string
		dataset string
		wantNil bool
		want    string
	}{
		{"empty string", "", true, ""},
		{"default", "default", true, ""},
		{"custom dataset", "my-dataset", false, "my-dataset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DatasetPtr(tt.dataset)
			if tt.wantNil {
				if got != nil {
					t.Errorf("DatasetPtr(%q) = %q, want nil", tt.dataset, *got)
				}
			} else {
				if got == nil {
					t.Errorf("DatasetPtr(%q) = nil, want %q", tt.dataset, tt.want)
				} else if *got != tt.want {
					t.Errorf("DatasetPtr(%q) = %q, want %q", tt.dataset, *got, tt.want)
				}
			}
		})
	}
}
