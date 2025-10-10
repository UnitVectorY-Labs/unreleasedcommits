package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"
)

func generateIndexPage(outputDir string, repos []RepositoryData) error {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		return fmt.Errorf("failed to parse index template: %w", err)
	}

	var summaries []SummaryData
	totalCommits := 0
	reposWithCommits := 0

	for _, repo := range repos {
		commitCount := len(repo.UnreleasedCommits)
		totalCommits += commitCount
		if commitCount > 0 {
			reposWithCommits++
		}

		daysBehind := 0
		// Calculate days between the last release and the most recent commit
		if commitCount > 0 && len(repo.UnreleasedCommits) > 0 && !repo.LatestReleaseTime.IsZero() {
			// Since commits are ordered with newest first (reversed in main.go)
			latestCommitTime := repo.UnreleasedCommits[0].Timestamp
			daysBehind = int(latestCommitTime.Sub(repo.LatestReleaseTime).Hours() / 24)
		}

		daysSinceRelease := 0
		if !repo.LatestReleaseTime.IsZero() {
			daysSinceRelease = int(time.Since(repo.LatestReleaseTime).Hours() / 24)
		}

		summaries = append(summaries, SummaryData{
			Name:             repo.Name,
			CommitCount:      commitCount,
			DaysBehind:       daysBehind,
			DaysSinceRelease: daysSinceRelease,
			LatestRelease:    repo.LatestReleaseTag,
			URL:              fmt.Sprintf("%s.html", repo.Name),
		})
	}

	file, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		TotalRepos       int
		TotalCommits     int
		ReposWithCommits int
		Repos            []SummaryData
	}{
		TotalRepos:       len(repos),
		TotalCommits:     totalCommits,
		ReposWithCommits: reposWithCommits,
		Repos:            summaries,
	}

	return tmpl.Execute(file, data)
}

func generateRepoPage(outputDir string, repo RepositoryData) error {
	tmpl, err := template.ParseFiles("templates/repo.html")
	if err != nil {
		return fmt.Errorf("failed to parse repo template: %w", err)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s.html", repo.Name))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, repo)
}

func generateCSS(outputDir string) error {
	cssContent, err := os.ReadFile("templates/style.css")
	if err != nil {
		return fmt.Errorf("failed to read CSS template: %w", err)
	}

	filename := filepath.Join(outputDir, "style.css")
	return os.WriteFile(filename, cssContent, 0644)
}
