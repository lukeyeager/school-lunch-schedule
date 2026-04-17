// cmd/setup is an interactive CLI that helps you find the org_id and menu_id
// needed for config.yaml by browsing the public Health-e Pro API.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lukeyeager/healthepro-slack-bot/internal/healthepro"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	client := healthepro.NewDiscoveryClient()

	// 1. Search for an organization.
	query, _ := prompt(scanner, "Search for your school district: ")
	orgs, err := client.SearchOrgs(query)
	if err != nil {
		fatalf("searching organizations: %v", err)
	}
	if len(orgs) == 0 {
		fatalf("no organizations found matching %q", query)
	}

	fmt.Printf("\nFound %d organization(s):\n", len(orgs))
	for i, o := range orgs {
		fmt.Printf("  %d) %s (%s)\n", i+1, o.Name, o.State)
	}
	orgIdx := pickIndex(scanner, "Pick an organization", len(orgs))
	org := orgs[orgIdx]

	// 2. List sites for the chosen org.
	sites, err := client.ListSites(org.ID)
	if err != nil {
		fatalf("listing sites for %q: %v", org.Name, err)
	}
	if len(sites) == 0 {
		fatalf("no sites found for %q", org.Name)
	}

	fmt.Printf("\nSites for %s (%d):\n", org.Name, len(sites))
	for i, s := range sites {
		fmt.Printf("  %d) %s\n", i+1, s.Name)
	}
	siteIdx := pickIndex(scanner, "Pick a site", len(sites))
	site := sites[siteIdx]

	// 3. List menus for the chosen site.
	menus, err := client.ListMenus(org.ID, site.ID)
	if err != nil {
		fatalf("listing menus for %q: %v", site.Name, err)
	}
	if len(menus) == 0 {
		fatalf("no menus found for %q", site.Name)
	}

	fmt.Printf("\nMenus for %s (%d):\n", site.Name, len(menus))
	for i, m := range menus {
		fmt.Printf("  %d) %s\n", i+1, m.Name)
	}
	menuIdx := pickIndex(scanner, "Pick a menu", len(menus))
	menu := menus[menuIdx]

	// 4. Write config.yaml.
	const configTemplate = `org_id: %d     # Health-e Pro organization ID (%s)
menu_id: %d    # Health-e Pro menu ID (%s)
evening_cron: "0 19 * * 0-4"      # night-before preview (Sun–Thu)
morning_cron: "0 6 * * 1-5"       # morning re-check (Mon–Fri)
db_path: "/data/menus.db"          # inside container; bind-mount ./data/ from host
timezone: "America/Chicago"        # cron timezone
`
	content := fmt.Sprintf(configTemplate, org.ID, org.Name, menu.ID, menu.Name)

	const outPath = "config.yaml"
	if err := os.WriteFile(outPath, []byte(content), 0o600); err != nil {
		fatalf("writing %s: %v", outPath, err)
	}

	fmt.Printf("\nWrote %s\n", outPath)
	fmt.Printf("  org_id:  %d  (%s)\n", org.ID, org.Name)
	fmt.Printf("  site_id: %d  (%s)  — for reference only\n", site.ID, site.Name)
	fmt.Printf("  menu_id: %d  (%s)\n", menu.ID, menu.Name)
	fmt.Println("\nNext: edit timezone if needed, then run: docker-compose up")
}

// prompt prints a message and reads a line from the scanner. Returns "" on EOF.
func prompt(scanner *bufio.Scanner, msg string) (string, bool) {
	fmt.Print(msg)
	if !scanner.Scan() {
		return "", false
	}
	return strings.TrimSpace(scanner.Text()), true
}

// pickIndex prints a prompt and reads a 1-based index, returning the 0-based index.
func pickIndex(scanner *bufio.Scanner, label string, max int) int {
	for {
		raw, ok := prompt(scanner, fmt.Sprintf("%s [1-%d]: ", label, max))
		if !ok {
			fatalf("unexpected end of input")
		}
		n, err := strconv.Atoi(raw)
		if err == nil && n >= 1 && n <= max {
			return n - 1
		}
		fmt.Fprintf(os.Stderr, "  please enter a number between 1 and %d\n", max)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
