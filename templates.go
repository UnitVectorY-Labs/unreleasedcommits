package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"time"
)

// interpolateColor blends between two RGB colors based on factor (0-1)
func interpolateColor(r1, g1, b1, r2, g2, b2 int, factor float64) (int, int, int) {
	r := int(math.Round(float64(r1) + factor*float64(r2-r1)))
	g := int(math.Round(float64(g1) + factor*float64(g2-g1)))
	b := int(math.Round(float64(b1) + factor*float64(b2-b1)))
	return r, g, b
}

// getColorForValue returns a hex color from green to yellow to red based on normalized value (0-1)
func getColorForValue(normalizedValue float64) string {
	// Green RGB: 16, 185, 129
	// Yellow RGB: 251, 191, 36
	// Red RGB: 239, 68, 68

	var r, g, b int
	if normalizedValue < 0.5 {
		// Green to Yellow
		r, g, b = interpolateColor(16, 185, 129, 251, 191, 36, normalizedValue*2)
	} else {
		// Yellow to Red
		r, g, b = interpolateColor(251, 191, 36, 239, 68, 68, (normalizedValue-0.5)*2)
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// getTextColor returns white or black text color based on background brightness
func getTextColor(normalizedValue float64) string {
	if normalizedValue > 0.6 {
		return "#ffffff"
	}
	return "#000000"
}

func generateIndexPage(outputDir string, repos []RepositoryData, lastUpdated string) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("failed to parse index template: %w", err)
	}

	var summaries []SummaryData
	totalCommits := 0
	reposWithCommits := 0

	// Track min/max values for color scaling
	minCommits := -1
	maxCommits := 0
	minDaysBehind := -1
	maxDaysBehind := 0
	minDaysSinceRelease := -1
	maxDaysSinceRelease := 0

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

		// Update min/max values
		if minCommits == -1 || commitCount < minCommits {
			minCommits = commitCount
		}
		if commitCount > maxCommits {
			maxCommits = commitCount
		}

		if minDaysBehind == -1 || daysBehind < minDaysBehind {
			minDaysBehind = daysBehind
		}
		if daysBehind > maxDaysBehind {
			maxDaysBehind = daysBehind
		}

		if minDaysSinceRelease == -1 || daysSinceRelease < minDaysSinceRelease {
			minDaysSinceRelease = daysSinceRelease
		}
		if daysSinceRelease > maxDaysSinceRelease {
			maxDaysSinceRelease = daysSinceRelease
		}

		summaries = append(summaries, SummaryData{
			Name:             repo.Name,
			CommitCount:      commitCount,
			DaysBehind:       daysBehind,
			DaysSinceRelease: daysSinceRelease,
			LatestRelease:    repo.LatestReleaseTag,
			URL:              fmt.Sprintf("%s.html", repo.Name),
			RepositoryURL:    repo.RepositoryURL,
			DefaultBranch:    repo.DefaultBranch,
		})
	}

	// Set defaults if no data
	if minCommits == -1 {
		minCommits = 0
	}
	if minDaysBehind == -1 {
		minDaysBehind = 0
	}
	if minDaysSinceRelease == -1 {
		minDaysSinceRelease = 0
	}

	// Compute colors for each summary
	for i := range summaries {
		// Compute commit count color
		commitRange := maxCommits - minCommits
		if commitRange > 0 {
			normalized := float64(summaries[i].CommitCount-minCommits) / float64(commitRange)
			summaries[i].CommitCountBgColor = getColorForValue(normalized)
			summaries[i].CommitCountTextColor = getTextColor(normalized)
		} else {
			summaries[i].CommitCountBgColor = getColorForValue(0)
			summaries[i].CommitCountTextColor = getTextColor(0)
		}

		// Compute days behind color
		daysBehindRange := maxDaysBehind - minDaysBehind
		if daysBehindRange > 0 {
			normalized := float64(summaries[i].DaysBehind-minDaysBehind) / float64(daysBehindRange)
			summaries[i].DaysBehindBgColor = getColorForValue(normalized)
			summaries[i].DaysBehindTextColor = getTextColor(normalized)
		} else {
			summaries[i].DaysBehindBgColor = getColorForValue(0)
			summaries[i].DaysBehindTextColor = getTextColor(0)
		}

		// Compute days since release color
		daysSinceRange := maxDaysSinceRelease - minDaysSinceRelease
		if daysSinceRange > 0 {
			normalized := float64(summaries[i].DaysSinceRelease-minDaysSinceRelease) / float64(daysSinceRange)
			summaries[i].DaysSinceBgColor = getColorForValue(normalized)
			summaries[i].DaysSinceTextColor = getTextColor(normalized)
		} else {
			summaries[i].DaysSinceBgColor = getColorForValue(0)
			summaries[i].DaysSinceTextColor = getTextColor(0)
		}
	}

	file, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		return err
	}
	defer file.Close()

	// Extract owner from the first repository (all repos have the same owner)
	owner := ""
	if len(repos) > 0 {
		owner = repos[0].Owner
	}

	data := struct {
		Owner               string
		TotalRepos          int
		TotalCommits        int
		ReposWithCommits    int
		Repos               []SummaryData
		MinCommits          int
		MaxCommits          int
		MinDaysBehind       int
		MaxDaysBehind       int
		MinDaysSinceRelease int
		MaxDaysSinceRelease int
		LastUpdated         string
	}{
		Owner:               owner,
		TotalRepos:          len(repos),
		TotalCommits:        totalCommits,
		ReposWithCommits:    reposWithCommits,
		Repos:               summaries,
		MinCommits:          minCommits,
		MaxCommits:          maxCommits,
		MinDaysBehind:       minDaysBehind,
		MaxDaysBehind:       maxDaysBehind,
		MinDaysSinceRelease: minDaysSinceRelease,
		MaxDaysSinceRelease: maxDaysSinceRelease,
		LastUpdated:         lastUpdated,
	}

	return tmpl.ExecuteTemplate(file, "index.html", data)
}

func generateRepoPage(outputDir string, repo RepositoryData, lastUpdated string) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("failed to parse repo template: %w", err)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s.html", repo.Name))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Calculate DaysBehind and DaysSinceRelease
	daysBehind := 0
	commitCount := len(repo.UnreleasedCommits)
	if commitCount > 0 && !repo.LatestReleaseTime.IsZero() {
		// Since commits are ordered with newest first (reversed in main.go)
		latestCommitTime := repo.UnreleasedCommits[0].Timestamp
		daysBehind = int(latestCommitTime.Sub(repo.LatestReleaseTime).Hours() / 24)
	}

	daysSinceRelease := 0
	if !repo.LatestReleaseTime.IsZero() {
		daysSinceRelease = int(time.Since(repo.LatestReleaseTime).Hours() / 24)
	}

	// Create a data struct with the calculated fields
	data := struct {
		RepositoryData
		DaysBehind       int
		DaysSinceRelease int
		LastUpdated      string
	}{
		RepositoryData:   repo,
		DaysBehind:       daysBehind,
		DaysSinceRelease: daysSinceRelease,
		LastUpdated:      lastUpdated,
	}

	return tmpl.ExecuteTemplate(file, "repo.html", data)
}

func generateCSS(outputDir string) error {
	return copyEmbeddedFile(templateFS, "templates/style.css", filepath.Join(outputDir, "style.css"))
}

// loadTemplates loads templates from the embedded filesystem,
// or from disk if TEMPLATE_PATH environment variable is set (for development).
func loadTemplates() (*template.Template, error) {
	// Dev-time override: load from disk if TEMPLATE_PATH is set
	if dir := os.Getenv("TEMPLATE_PATH"); dir != "" {
		fmt.Printf("Loading templates from disk: %s\n", dir)
		return template.ParseGlob(filepath.Join(dir, "*.html"))
	}
	// Production: load from embedded filesystem
	return template.ParseFS(templateFS, "templates/*.html")
}

// copyEmbeddedFile copies a file from the embedded filesystem to the destination path.
func copyEmbeddedFile(fsys fs.FS, src, dst string) error {
	// Dev-time override: copy from disk if TEMPLATE_PATH is set
	if dir := os.Getenv("TEMPLATE_PATH"); dir != "" {
		// Extract filename from src path
		filename := filepath.Base(src)
		srcPath := filepath.Join(dir, filename)
		fmt.Printf("Copying file from disk: %s\n", srcPath)
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read file from disk: %w", err)
		}
		return os.WriteFile(dst, content, 0644)
	}
	// Production: read from embedded filesystem
	content, err := fs.ReadFile(fsys, src)
	if err != nil {
		return fmt.Errorf("failed to read embedded file: %w", err)
	}
	return os.WriteFile(dst, content, 0644)
}
