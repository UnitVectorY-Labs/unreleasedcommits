[![License](https://img.shields.io/badge/license-MIT-blue)](https://opensource.org/licenses/MIT) [![Active](https://img.shields.io/badge/Status-Active-green)](https://guide.unitvectorylabs.com/bestpractices/status/#active) [![Go Report Card](https://goreportcard.com/badge/github.com/UnitVectorY-Labs/unreleasedcommits)](https://goreportcard.com/report/github.com/UnitVectorY-Labs/unreleasedcommits)

# unreleasedcommits

Generates reports that identify commits on the default branch which haven’t yet been included in the latest GitHub release.

## Features

- **Crawl Command**: Fetches unreleased commits from GitHub repositories and saves results as JSON
- **Generate Command**: Creates static HTML pages from crawl data with visual indicators
- **Color-Coded Metrics**: Heat map visualization for commit counts, days behind, and days since release
- **Automatic Sorting**: Repositories sorted alphabetically with newest commits first
- **Timestamp Tracking**: Records last crawl time for reference

## Automation

This repository is designed to automatically update commit data for a GitHub organization's repositories. The tool identifies commits on the default branch that haven't been included in the latest release, helping track which repositories need new releases.

## Building

```bash
go build -o unreleasedcommits main.go
```

## Usage

### Crawl Command

Fetches unreleased commits from GitHub repositories:

```bash
./unreleasedcommits -crawl -owner <organization> [flags]
```

**Flags:**
- `-owner <name>`: GitHub owner/organization name (required)
- `-limit <int>`: Limit number of repositories to process (default: 0 = no limit)

**Requirements:**
- Requires the `GITHUB_TOKEN` environment variable with a valid GitHub personal access token

**Example:**
```bash
export GITHUB_TOKEN=your_token_here
./unreleasedcommits -crawl -owner UnitVectorY-Labs
```

### Generate Command

Creates static HTML pages from crawl JSON data:

```bash
./unreleasedcommits -generate
```

**Input:** JSON files from `data/` directory  
**Output:** HTML files in `output/` directory

**Example:**
```bash
./unreleasedcommits -generate
```

## Output Format

### JSON Output (from crawl)

Each repository gets a JSON file in `data/` with the following structure:

```json
{
  "owner": "UnitVectorY-Labs",
  "name": "example-repo",
  "default_branch": "main",
  "latest_release_tag": "v1.2.3",
  "latest_release_time": "2025-01-15T10:30:00Z",
  "unreleased_commits": [
    {
      "sha": "abc123...",
      "author": "username",
      "message": "Fix bug in feature X",
      "timestamp": "2025-02-01T14:20:00Z",
      "url": "https://github.com/..."
    }
  ],
  "repository_url": "https://github.com/UnitVectorY-Labs/example-repo"
}
```

Additionally, a `timestamp.json` file is created:

```json
{
  "last_crawled": "2025-02-10T15:30:00Z"
}
```

### HTML Output (from generate)

- `index.html`: Summary table with metrics for all repositories
- `<repo>.html`: Detailed page for each repository showing commit history
- `style.css`: Responsive stylesheet copied from `templates/`

## Requirements

- Latest version of Go
- `templates/` directory with `index.html`, `repo.html`, and `style.css`
- GitHub personal access token with repository read permissions
- Standard library dependencies: `github.com/google/go-github/v62` and `golang.org/x/oauth2`

## Repository Processing

The tool processes public repositories from the specified organization:

- Skips repositories without releases
- Compares the default branch against the latest release tag
- Captures all commits between the release and branch HEAD
- Records commit metadata (SHA, author, message, timestamp, URL)

## Metrics

The tool calculates three key metrics visualized with color coding:

- **Unreleased Commits**: Count of commits not included in the latest release
- **Days Behind**: Days between the latest release and the most recent commit
- **Days Since Release**: Days since the latest release was published

Colors range from green (low values) through yellow to red (high values).
