package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

// CommitInfo represents a single commit with all relevant details
type CommitInfo struct {
	SHA       string    `json:"sha"`
	Author    string    `json:"author"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
}

// RepositoryData represents all data for a repository
type RepositoryData struct {
	Owner             string       `json:"owner"`
	Name              string       `json:"name"`
	DefaultBranch     string       `json:"default_branch"`
	LatestReleaseTag  string       `json:"latest_release_tag"`
	LatestReleaseTime time.Time    `json:"latest_release_time"`
	UnreleasedCommits []CommitInfo `json:"unreleased_commits"`
	RepositoryURL     string       `json:"repository_url"`
}

// SummaryData represents summary info for the index page
type SummaryData struct {
	Name                 string
	CommitCount          int
	DaysBehind           int
	DaysSinceRelease     int
	LatestRelease        string
	URL                  string
	RepositoryURL        string
	DefaultBranch        string
	CommitCountBgColor   string
	CommitCountTextColor string
	DaysBehindBgColor    string
	DaysBehindTextColor  string
	DaysSinceBgColor     string
	DaysSinceTextColor   string
}

// TimestampData captures when the crawl last ran
type TimestampData struct {
	LastCrawled time.Time `json:"last_crawled"`
}

func main() {
	crawlMode := flag.Bool("crawl", false, "Crawl GitHub API and generate JSON files")
	generateMode := flag.Bool("generate", false, "Generate HTML pages from JSON files")
	owner := flag.String("owner", "", "GitHub owner/organization name (required for -crawl)")
	limit := flag.Int("limit", 0, "Limit number of repositories to process (0 = no limit)")
	flag.Parse()

	if !*crawlMode && !*generateMode {
		log.Fatal("Please specify either -crawl or -generate mode")
	}

	if *crawlMode && *generateMode {
		log.Fatal("Please specify only one mode: -crawl or -generate")
	}

	if *crawlMode {
		if *owner == "" {
			log.Fatal("Owner is required when using -crawl mode. Use -owner flag to specify the GitHub owner/organization name")
		}
		runCrawl(*owner, *limit)
	} else if *generateMode {
		runGenerate()
	}
}

func runCrawl(owner string, limit int) {
	ctx := context.Background()

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, ts)
	client := github.NewClient(httpClient)

	fmt.Printf("Fetching repositories for organization: %s\n", owner)

	outputDir := "data"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	repos, err := listPublicRepos(ctx, client, owner, limit)
	if err != nil {
		log.Fatalf("Failed to list repositories: %v", err)
	}

	fmt.Printf("Found %d public repositories\n", len(repos))

	processedCount := 0
	for i, repo := range repos {
		repoName := repo.GetName()
		fmt.Printf("[%d/%d] Processing %s...\n", i+1, len(repos), repoName)

		hasRelease, releaseData := checkLatestRelease(ctx, client, owner, repoName)
		if !hasRelease {
			fmt.Printf("  ⏭️  Skipping %s (no releases)\n", repoName)
			continue
		}

		repoDetail, _, err := client.Repositories.Get(ctx, owner, repoName)
		if err != nil {
			fmt.Printf("  ❌ Error getting repo details: %v\n", err)
			continue
		}

		defaultBranch := repoDetail.GetDefaultBranch()
		tagName := releaseData.GetTagName()
		releaseTime := releaseData.GetPublishedAt().Time

		fmt.Printf("  Latest release: %s (%s)\n", tagName, releaseTime.Format("2006-01-02"))

		commits, err := compareAllCommits(ctx, client, owner, repoName, tagName, defaultBranch)
		if err != nil {
			fmt.Printf("  ❌ Error comparing commits: %v\n", err)
			continue
		}

		var commitInfos []CommitInfo
		for _, c := range commits {
			author := "unknown"
			if c.Author != nil && c.Author.GetLogin() != "" {
				author = c.Author.GetLogin()
			} else if c.Commit != nil && c.Commit.Author != nil && c.Commit.Author.GetName() != "" {
				author = c.Commit.Author.GetName()
			}

			commitInfos = append(commitInfos, CommitInfo{
				SHA:       c.GetSHA(),
				Author:    author,
				Message:   c.Commit.GetMessage(),
				Timestamp: c.Commit.Author.GetDate().Time,
				URL:       c.GetHTMLURL(),
			})
		}

		// Reverse the commits so newest are first
		for i, j := 0, len(commitInfos)-1; i < j; i, j = i+1, j-1 {
			commitInfos[i], commitInfos[j] = commitInfos[j], commitInfos[i]
		}

		repoData := RepositoryData{
			Owner:             owner,
			Name:              repoName,
			DefaultBranch:     defaultBranch,
			LatestReleaseTag:  tagName,
			LatestReleaseTime: releaseTime,
			UnreleasedCommits: commitInfos,
			RepositoryURL:     repoDetail.GetHTMLURL(),
		}

		filename := filepath.Join(outputDir, fmt.Sprintf("%s.json", repoName))
		if err := writeJSON(filename, repoData); err != nil {
			fmt.Printf("  ❌ Error writing JSON: %v\n", err)
			continue
		}

		fmt.Printf("  ✅ Saved %d unreleased commits to %s\n", len(commitInfos), filename)
		processedCount++
	}

	crawlTime := time.Now().UTC()
	timestampFile := filepath.Join(outputDir, "timestamp.json")
	if err := writeJSON(timestampFile, TimestampData{LastCrawled: crawlTime}); err != nil {
		log.Printf("⚠️  Failed to write crawl timestamp: %v", err)
	} else {
		fmt.Printf("\n🕒 Recorded crawl timestamp: %s\n", crawlTime.Format(time.RFC3339))
	}

	fmt.Printf("\n🎉 Crawl complete! Processed %d repositories with releases.\n", processedCount)
}

func runGenerate() {
	dataDir := "data"
	outputDir := "output"

	fmt.Println("Generating HTML pages...")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(dataDir, "*.json"))
	if err != nil {
		log.Fatalf("Failed to read data directory: %v", err)
	}

	lastUpdated := ""
	timestampPath := filepath.Join(dataDir, "timestamp.json")
	if ts, err := loadLastCrawlTimestamp(timestampPath); err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: could not load crawl timestamp: %v\n", err)
		}
	} else {
		lastUpdated = formatTimestampForFooter(ts)
	}

	var allRepos []RepositoryData
	for _, file := range files {
		if filepath.Base(file) == "timestamp.json" {
			continue
		}

		var repo RepositoryData
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file, err)
			continue
		}

		if err := json.Unmarshal(data, &repo); err != nil {
			fmt.Printf("Error parsing %s: %v\n", file, err)
			continue
		}

		allRepos = append(allRepos, repo)
	}

	if len(allRepos) == 0 {
		log.Fatal("No repository JSON files found in data directory. Run with -crawl first.")
	}

	sort.Slice(allRepos, func(i, j int) bool {
		return allRepos[i].Name < allRepos[j].Name
	})

	if err := generateIndexPage(outputDir, allRepos, lastUpdated); err != nil {
		log.Fatalf("Failed to generate index page: %v", err)
	}

	for _, repo := range allRepos {
		if err := generateRepoPage(outputDir, repo, lastUpdated); err != nil {
			fmt.Printf("Error generating page for %s: %v\n", repo.Name, err)
		}
	}

	if err := generateCSS(outputDir); err != nil {
		log.Fatalf("Failed to generate CSS: %v", err)
	}

	fmt.Printf("✅ Generated HTML pages in %s/ directory\n", outputDir)
	fmt.Printf("   Open %s/index.html in your browser\n", outputDir)
}

func listPublicRepos(ctx context.Context, client *github.Client, owner string, limit int) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opt := &github.RepositoryListByOrgOptions{
		Type:        "public",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, owner, opt)
		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		if limit > 0 && len(allRepos) >= limit {
			return allRepos[:limit], nil
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, nil
}

func checkLatestRelease(ctx context.Context, client *github.Client, owner, repo string) (bool, *github.RepositoryRelease) {
	rel, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil || rel == nil || rel.GetTagName() == "" {
		return false, nil
	}
	return true, rel
}

func compareAllCommits(ctx context.Context, client *github.Client, owner, repo, base, head string) ([]*github.RepositoryCommit, error) {
	var all []*github.RepositoryCommit
	page := 1
	perPage := 100

	for {
		comp, resp, err := client.Repositories.CompareCommits(ctx, owner, repo, base, head,
			&github.ListOptions{Page: page, PerPage: perPage})
		if err != nil {
			return nil, err
		}

		all = append(all, comp.Commits...)

		if resp.NextPage == 0 || len(comp.Commits) < perPage {
			break
		}
		page = resp.NextPage
	}

	return all, nil
}

func writeJSON(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func loadLastCrawlTimestamp(filename string) (time.Time, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return time.Time{}, err
	}

	var ts TimestampData
	if err := json.Unmarshal(data, &ts); err != nil {
		return time.Time{}, err
	}

	return ts.LastCrawled.UTC(), nil
}

func formatTimestampForFooter(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("January 2, 2006 15:04 UTC")
}
