# healthepro-slack-bot

A Slack bot that fetches a daily school lunch menu from the
[Health-e Pro](https://www.healthepro.com/) API (the backend behind the "My School Menus" app)
and posts it each school-day morning.

## How it works

Many school districts publish menus via Health-e Pro at `menus.healthepro.com`. The API is
public and requires no authentication. Each day's entry has two fields: `setting` (the
current/overwritten version) and `setting_original` (the original). When a district's data entry
produces a blank `current_display`, the bot falls back to `setting_original` so you still get
the menu.

## Configuration

### Finding your IDs (`cmd/setup`)

Run the interactive setup tool to search for your district, pick your school, and pick your
menu — it writes `config.yaml` for you:

```bash
go run ./cmd/setup
```

Example session:

```
Search for your school district: diwa

Found 1 organization(s):
  1) Diwa Kitchen (Texas)
Pick an organization [1-1]: 1

Sites for Diwa Kitchen (14):
  1) Diwa Kitchen Irving Blvd
  ...
  12) Valor North Austin
  ...
Pick a site [1-14]: 12

Menus for Valor North Austin (5):
  1) 6-8 Lunch
  ...
  4) K-5 Lunch
  ...
Pick a menu [1-5]: 4

Wrote config.yaml
  org_id:  1375  (Diwa Kitchen)
  menu_id: 108888  (K-5 Lunch)
```

### Static config (`config.yaml` — not checked into git)

```yaml
org_id: 0      # Health-e Pro organization ID
menu_id: 0     # Health-e Pro menu ID
evening_cron: "0 19 * * 0-4"      # night-before preview (default: 7pm Sun–Thu)
morning_cron: "0 6 * * 1-5"       # morning re-check (default: 6am Mon–Fri)
db_path: "/data/menus.db"          # inside the container; bind-mount ./data/ from the host
timezone: "America/Chicago"        # cron timezone
```

### Posting behavior

- **Evening (7pm):** always fetches and posts *tomorrow's* menu to Slack.
- **Morning (6am):** fetches *today's* menu and posts only if it changed since the evening post.
  No message is sent if the menu is unchanged.

### Secrets (`.env` — not checked into git)

```
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
```

## Running locally

```bash
go run ./cmd/setup          # generates config.yaml interactively
cp .env.example .env        # add your Slack webhook URL
docker-compose up
```

## Metrics

The bot exposes Prometheus metrics on `:9090/metrics`:

- `healthepro_requests_total{status_code}` — requests to the Health-e Pro API
- `slack_requests_total{status_code}` — requests to the Slack webhook

## Development

```bash
go test ./...
golangci-lint run
```

## Planned improvements

**TODO: Proactive 14-day background monitor**
Every 15 minutes, wake up and re-fetch the next 14 days of menus. For any day where the entree
changed since the last fetch, immediately send a Slack message summarising the diff — how many
days changed and what the entree switched from/to. The existing evening/morning cron posts (full
menu details) would continue unchanged alongside this.

> **NOTE:** This will almost certainly create too much noise over time, but it'll be invaluable
> early on for validating that the fallback logic and change-detection are actually working
> correctly against real data.

TODO: fix failing golang-ci linter

TODO: fix handling of secrets in docker-compose.yml

TODO: change default port, scrape w/ prometheus, validate metrics

## License

MIT — see [LICENSE](LICENSE).
