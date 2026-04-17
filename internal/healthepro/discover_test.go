package healthepro

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// orgsResponse builds a fake /api/organizations payload.
func orgsResponse(t *testing.T, states []struct {
	Name  string
	Orgs  []struct{ ID int; Name, State string }
}) string {
	t.Helper()
	type orgJSON struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		State string `json:"state"`
	}
	type stateJSON struct {
		Name          string    `json:"name"`
		Organizations []orgJSON `json:"organizations"`
	}
	var groups []stateJSON
	for _, s := range states {
		g := stateJSON{Name: s.Name}
		for _, o := range s.Orgs {
			g.Organizations = append(g.Organizations, orgJSON{ID: o.ID, Name: o.Name, State: o.State})
		}
		groups = append(groups, g)
	}
	wrapped := map[string]any{"data": groups}
	b, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("marshal orgs: %v", err)
	}
	return string(b)
}

func TestSearchOrgs_Filter(t *testing.T) {
	payload := orgsResponse(t, []struct {
		Name string
		Orgs []struct{ ID int; Name, State string }
	}{
		{
			Name: "Texas",
			Orgs: []struct{ ID int; Name, State string }{
				{ID: 1, Name: "Austin ISD", State: "Texas"},
				{ID: 2, Name: "Diwa Kitchen", State: "Texas"},
			},
		},
		{
			Name: "California",
			Orgs: []struct{ ID int; Name, State string }{
				{ID: 3, Name: "Oakland USD", State: "California"},
			},
		},
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer ts.Close()

	client := newDiscoveryClient(ts.URL)

	results, err := client.SearchOrgs("diwa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != 2 || results[0].Name != "Diwa Kitchen" || results[0].State != "Texas" {
		t.Errorf("unexpected result: %+v", results[0])
	}
}

func TestSearchOrgs_EmptyQuery(t *testing.T) {
	payload := orgsResponse(t, []struct {
		Name string
		Orgs []struct{ ID int; Name, State string }
	}{
		{
			Name: "Texas",
			Orgs: []struct{ ID int; Name, State string }{
				{ID: 1, Name: "Austin ISD", State: "Texas"},
				{ID: 2, Name: "Diwa Kitchen", State: "Texas"},
			},
		},
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer ts.Close()

	client := newDiscoveryClient(ts.URL)

	results, err := client.SearchOrgs("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for empty query, got %d", len(results))
	}
}

func TestSearchOrgs_CaseInsensitive(t *testing.T) {
	payload := orgsResponse(t, []struct {
		Name string
		Orgs []struct{ ID int; Name, State string }
	}{
		{
			Name: "Texas",
			Orgs: []struct{ ID int; Name, State string }{
				{ID: 1, Name: "Diwa Kitchen", State: "Texas"},
			},
		},
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer ts.Close()

	client := newDiscoveryClient(ts.URL)

	for _, query := range []string{"DIWA", "Diwa", "diwa", "DiWa KiTcHeN"} {
		results, err := client.SearchOrgs(query)
		if err != nil {
			t.Fatalf("query %q: unexpected error: %v", query, err)
		}
		if len(results) != 1 {
			t.Errorf("query %q: expected 1 result, got %d", query, len(results))
		}
	}
}

func TestListSites(t *testing.T) {
	payload := `{"data":[{"id":10,"name":"Elm Elementary"},{"id":11,"name":"Oak Middle School"}]}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/organizations/42/sites/list" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer ts.Close()

	client := newDiscoveryClient(ts.URL)

	sites, err := client.ListSites(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sites) != 2 {
		t.Fatalf("expected 2 sites, got %d", len(sites))
	}
	if sites[0].ID != 10 || sites[0].Name != "Elm Elementary" {
		t.Errorf("unexpected site[0]: %+v", sites[0])
	}
	if sites[1].ID != 11 || sites[1].Name != "Oak Middle School" {
		t.Errorf("unexpected site[1]: %+v", sites[1])
	}
}

func TestListMenus(t *testing.T) {
	payload := `{"data":[{"id":200,"name":"Lunch Menu 25-26"},{"id":201,"name":"Breakfast Menu 25-26"}]}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/organizations/42/sites/10/menus/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer ts.Close()

	client := newDiscoveryClient(ts.URL)

	menus, err := client.ListMenus(42, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(menus) != 2 {
		t.Fatalf("expected 2 menus, got %d", len(menus))
	}
	if menus[0].ID != 200 || menus[0].Name != "Lunch Menu 25-26" {
		t.Errorf("unexpected menu[0]: %+v", menus[0])
	}
}

func TestDiscoveryClient_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := newDiscoveryClient(ts.URL)

	if _, err := client.SearchOrgs("anything"); err == nil {
		t.Error("expected error for SearchOrgs 500, got nil")
	}
	if _, err := client.ListSites(1); err == nil {
		t.Error("expected error for ListSites 500, got nil")
	}
	if _, err := client.ListMenus(1, 1); err == nil {
		t.Error("expected error for ListMenus 500, got nil")
	}
}
