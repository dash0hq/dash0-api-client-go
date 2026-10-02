package dash0

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListMembersRoles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/members" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"kind":"Dash0Member","metadata":{"name":"alice","labels":{"dash0.com/id":"alice","dash0.com/role":"admin"}},"spec":{"display":{"email":"alice@example.com"}}},
			{"kind":"Dash0Member","metadata":{"name":"bob","labels":{"dash0.com/id":"bob","dash0.com/role":"basic_member"}},"spec":{"display":{"email":"bob@example.com"}}},
			{"kind":"Dash0Member","metadata":{"name":"legacy","labels":{"dash0.com/id":"legacy"}},"spec":{"display":{"email":"legacy@example.com"}}},
			{"kind":"Dash0Member","metadata":{"name":"future","labels":{"dash0.com/id":"future","dash0.com/role":"future_role"}},"spec":{"display":{"email":"future@example.com"}}}
		]`)
	}))
	defer server.Close()

	client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
	if err != nil {
		t.Fatal(err)
	}
	members, err := client.ListMembers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantRoles := []*string{Ptr("admin"), Ptr("basic_member"), nil, Ptr("future_role")}
	if len(members) != len(wantRoles) {
		t.Fatalf("got %d members, want %d", len(members), len(wantRoles))
	}
	for i, want := range wantRoles {
		if members[i].Metadata.Labels == nil {
			t.Fatalf("member %d: missing labels", i)
		}
		got := members[i].Metadata.Labels.Dash0Comrole
		if want == nil {
			if got != nil {
				t.Errorf("member %d: missing role decoded as %q", i, *got)
			}
			data, err := json.Marshal(members[i].Metadata.Labels)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			if _, exists := fields["dash0.com/role"]; exists {
				t.Error("absent role should remain omitted when marshaled")
			}
		} else if got == nil || *got != *want {
			t.Errorf("member %d: got role %v, want %q", i, got, *want)
		}
	}
}
