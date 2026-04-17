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

### Static config (`config.yaml` — not checked into git)

```yaml
org_id: 0     # Health-e Pro organization ID
site_id: 0    # Health-e Pro site (school) ID
menu_id: 0    # Health-e Pro menu ID
evening_cron: "0 19 * * 0-4"  # night-before preview (default: 7pm Sun–Thu)
morning_cron: "0 6 * * 1-5"   # morning re-check (default: 6am Mon–Fri)
db_path: "/data/menus.db"      # inside the container; bind-mount ./data/ from the host
```

To find your IDs, open the "My School Menus" app, select your district/school/menu, then look
up the matching org at `menus.healthepro.com/api/organizations`.

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
cp config.yaml.example config.yaml   # fill in your IDs
cp .env.example .env                  # add your Slack webhook URL
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

**TODO: ID discovery utility**
Finding the right `org_id`, `site_id`, and `menu_id` currently requires manually poking the API.
A `cmd/setup` interactive CLI would let new users search for their district, pick their school,
pick their menu, and write out a ready-to-use `config.yaml` — no API spelunking required.

**TODO: Proactive 14-day background monitor**
Every 15 minutes, wake up and re-fetch the next 14 days of menus. For any day where the entree
changed since the last fetch, immediately send a Slack message summarising the diff — how many
days changed and what the entree switched from/to. The existing evening/morning cron posts (full
menu details) would continue unchanged alongside this.

> **NOTE:** This will almost certainly create too much noise over time, but it'll be invaluable
> early on for validating that the fallback logic and change-detection are actually working
> correctly against real data.

## License

MIT — see [LICENSE](LICENSE).
