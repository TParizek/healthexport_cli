---
name: healthexport
description: "Query health and fitness data from Apple Health via the HealthExport MCP server. MANDATORY TRIGGER: Use this skill whenever the user mentions 'healthexport', 'health export', 'HealthExport', 'Apple Health', or asks about ANY personal health metric including: steps, step count, heart rate, resting heart rate, HRV, heart rate variability, sleep, sleep analysis, workouts, exercise, calories, active energy, resting energy, walking distance, running distance, cycling distance, flights climbed, weight, BMI, body fat, blood pressure, blood oxygen, respiratory rate, body temperature, water intake, caffeine, nutrition, stand time, swimming, or exercise minutes. Also trigger when users say 'how many steps', 'my health data', 'fitness data', 'health stats', 'health trends', 'show my steps', 'track my sleep', 'my workouts', or any request to fetch, chart, graph, analyze, summarize, or export personal health or fitness metrics from their phone or watch."
---

# HealthExport — Apple Health Data Access

You have access to the **HealthExport MCP server** which reads data from Apple Health. Do NOT search the MCP registry, plugin marketplace, or the web for health data tools — they are already available as deferred tools in this session.

## Step 1: Fetch the tools

Use `ToolSearch` with query `select:mcp__HealthExport__fetch_health_data,mcp__HealthExport__list_health_types,mcp__HealthExport__health_export_status,mcp__HealthExport__list_workouts,mcp__HealthExport__fetch_workout_detail,mcp__HealthExport__fetch_workout_series` to load the tool schemas.

## Step 2: Check connectivity (optional, use if first query fails)

Call `mcp__HealthExport__health_export_status` to verify the MCP server is connected.

## Step 3: Discover available metrics

Call `mcp__HealthExport__list_health_types` to see all queryable metric names. Supports an optional `category` filter:
- `aggregated` — totals like steps, calories, distance
- `record` — readings like heart rate, blood pressure, sleep, weight
- `workout` — exercise sessions like running, cycling, swimming

## Step 4: Fetch the data

Call `mcp__HealthExport__fetch_health_data` with:
- `from` — start date in `YYYY-MM-DD` format
- `to` — end date in `YYYY-MM-DD` format
- `types` — array of metric names (use the exact names returned by `list_health_types`, e.g. `["Step count"]`)
- `aggregate` (optional) — `"day"`, `"week"`, `"month"`, or `"year"` for rolled-up trends

## Common Patterns

- **"Steps from last year"** → `list_health_types` (category: aggregated) → find "Step count" → `fetch_health_data` with from/to spanning the year and aggregate: "month"
- **"My sleep this week"** → `list_health_types` (category: record) → find "Sleep analysis" → `fetch_health_data` with this week's date range
- **"Heart rate for March"** → `fetch_health_data` with types: ["Heart rate"] and focused date ranges; heart rate is not a cumulative sum.
- **"Compare my workouts month over month"** → fetch workout summaries without aggregation, or use `list_workouts` for uploaded enriched workouts.
- **"Heart rate and splits for my run"** → `list_workouts` → choose the recording by ID/source → `fetch_workout_detail` and `fetch_workout_series` for `heartRate`.

## Visualization

When the user asks for a chart, graph, or visualization, use Python (matplotlib) to create the image and save it to the outputs folder. Always include:
- Clear title with metric name and date range
- Labeled axes with readable formatting (e.g., "250k" not "250000")
- Average line for context
- Total annotation when it makes sense
- Clean styling (hide top/right spines, subtle grid)

## Workout details

Use `list_workouts` with inclusive RFC3339 `start`/`end` (maximum 366 days, 10 workouts per page). Follow `nextOffset`. Then use its `workoutId` and `revision`:

- `fetch_workout_detail`: events plus `splitUnit` km/mi; 1–4 pages per section. Follow each section's continuation independently. Split distances remain meters.
- `fetch_workout_series`: explicitly select heartRate/distance/speed/elevation/power/cadence; 1–8 pages, 256 points each. Follow `nextOffset`; request unit conversion only when needed.

Check availability; missing data is not zero. Stored aggregates cannot recover raw peaks or exact effort durations. Retain point kind, extrema, sample count, coverage and gaps. A revision conflict means list again and restart. These tools do not sync HealthKit. Full CLI JSON exports use `he workouts export`; FIT/TCX/GPX are unsupported.
