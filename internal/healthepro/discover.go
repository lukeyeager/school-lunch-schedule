package healthepro

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Org is an organization returned by the discovery API.
type Org struct {
	ID    int
	Name  string
	State string
}

// Site is a school site within an organization.
type Site struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Menu is a menu available at a site.
type Menu struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// DiscoveryClient queries the Health-e Pro API to find org/site/menu IDs.
type DiscoveryClient struct {
	baseURL string
	http    *http.Client
}

// NewDiscoveryClient creates a DiscoveryClient using the public Health-e Pro API.
func NewDiscoveryClient() *DiscoveryClient {
	return newDiscoveryClient(baseURL)
}

func newDiscoveryClient(base string) *DiscoveryClient {
	return &DiscoveryClient{baseURL: base, http: &http.Client{}}
}

// SearchOrgs fetches all organizations and returns those whose name contains query
// (case-insensitive). Pass an empty string to return all organizations.
func (c *DiscoveryClient) SearchOrgs(query string) ([]Org, error) {
	body, err := c.get(c.baseURL + "/api/organizations")
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Data []struct {
			Name          string `json:"name"`
			Organizations []struct {
				ID    int    `json:"id"`
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"organizations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing organizations: %w", err)
	}
	stateGroups := wrapper.Data

	lower := strings.ToLower(query)
	var results []Org
	for _, sg := range stateGroups {
		for _, o := range sg.Organizations {
			if query == "" || strings.Contains(strings.ToLower(o.Name), lower) {
				results = append(results, Org{
					ID:    o.ID,
					Name:  o.Name,
					State: sg.Name,
				})
			}
		}
	}
	return results, nil
}

// ListSites returns all sites (schools) for the given organization.
func (c *DiscoveryClient) ListSites(orgID int) ([]Site, error) {
	url := fmt.Sprintf("%s/api/organizations/%d/sites/list", c.baseURL, orgID)
	body, err := c.get(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []Site `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing sites: %w", err)
	}
	return resp.Data, nil
}

// ListMenus returns all menus for the given site within an organization.
func (c *DiscoveryClient) ListMenus(orgID, siteID int) ([]Menu, error) {
	url := fmt.Sprintf("%s/api/organizations/%d/sites/%d/menus/", c.baseURL, orgID, siteID)
	body, err := c.get(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []Menu `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing menus: %w", err)
	}
	return resp.Data, nil
}

func (c *DiscoveryClient) get(url string) ([]byte, error) {
	resp, err := c.http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return body, nil
}
