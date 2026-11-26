package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"encoding/json/v2"
	"github.com/google/go-github/v62/github"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRepositoryJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo.json")
	original := RepositoryData{Owner: "example", Name: "repo", LatestReleaseTime: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)}
	if err := writeJSON(path, original); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// V2 writes nil slices as empty arrays, including repositories without commits.
	if !bytes.Contains(data, []byte(`"unreleased_commits": []`)) || !bytes.HasSuffix(data, []byte("\n")) {
		t.Fatalf("unexpected JSON output: %s", data)
	}
	var got RepositoryData
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Owner != original.Owner || got.Name != original.Name || !got.LatestReleaseTime.Equal(original.LatestReleaseTime) || len(got.UnreleasedCommits) != 0 {
		t.Fatalf("unexpected round trip: %+v", got)
	}
	// Existing v1 output containing a null slice remains readable.
	if err := json.Unmarshal([]byte(`{"unreleased_commits":null}`), &got); err != nil {
		t.Fatal(err)
	}
}

func TestJSONStrictInput(t *testing.T) {
	for _, input := range [][]byte{
		[]byte(`{"name":"first","name":"second"}`),
		append([]byte(`{"name":"`), 'a', 0xff, '"', '}'),
	} {
		var repo RepositoryData
		if err := json.Unmarshal(input, &repo); err == nil {
			t.Errorf("accepted invalid input %q", input)
		}
	}
	var repo RepositoryData
	if err := json.Unmarshal([]byte(`{"Name":"wrong case"}`), &repo); err != nil || repo.Name != "" {
		t.Fatalf("field names should match case exactly: %+v, %v", repo, err)
	}
}

func TestLoadLastCrawlTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timestamp.json")
	want := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := writeJSON(path, TimestampData{LastCrawled: want}); err != nil {
		t.Fatal(err)
	}
	got, err := loadLastCrawlTimestamp(path)
	if err != nil || !got.Equal(want) {
		t.Fatalf("got %v, %v; want %v", got, err, want)
	}
	if err := os.WriteFile(path, []byte(`{"last_crawled":"2025-01-02T03:04:05Z","last_crawled":"2025-01-02T03:04:05Z"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadLastCrawlTimestamp(path); err == nil {
		t.Fatal("accepted duplicate timestamp")
	}
}

func TestArchivedRepositories(t *testing.T) {
	for _, limit := range []int{0, 1} {
		t.Run(fmt.Sprintf("limit_%d", limit), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/orgs/example/repos" || r.URL.Query().Get("type") != "public" {
					t.Errorf("unexpected request: %s", r.URL)
				}
				if r.URL.Query().Get("page") == "2" {
					fmt.Fprint(w, `[{"name":"active","archived":false}]`)
					return
				}
				w.Header().Set("Link", fmt.Sprintf(`<http://%s/orgs/example/repos?page=2>; rel="next"`, r.Host))
				fmt.Fprint(w, `[{"name":"archived","archived":true}]`)
			}))
			defer server.Close()
			client, err := github.NewClient(server.Client()).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
			if err != nil {
				t.Fatal(err)
			}
			// Use the test server directly, without the enterprise API path prefix.
			client.BaseURL.Path = "/"
			repos, archived, err := listPublicRepos(context.Background(), client, "example", limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(repos) != 1 || repos[0].GetName() != "active" || len(archived) != 1 || archived[0].GetName() != "archived" {
				t.Fatalf("unexpected repositories: active=%v archived=%v", repos, archived)
			}
			dataDir, outputDir := t.TempDir(), t.TempDir()
			for _, name := range []string{"active", "archived"} {
				if err := writeJSON(filepath.Join(dataDir, name+".json"), RepositoryData{Name: name}); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(outputDir, name+".html"), []byte(name), 0600); err != nil {
					t.Fatal(err)
				}
			}
			for _, repo := range archived {
				if err := removeArchivedRepoFiles(dataDir, outputDir, repo.GetName()); err != nil {
					t.Fatal(err)
				}
			}
			for _, path := range []string{filepath.Join(dataDir, "archived.json"), filepath.Join(outputDir, "archived.html")} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("archived file still exists: %s (%v)", path, err)
				}
			}
			for _, path := range []string{filepath.Join(dataDir, "active.json"), filepath.Join(outputDir, "active.html")} {
				if _, err := os.Stat(path); err != nil {
					t.Error(err)
				}
			}
			if err := removeArchivedRepoFiles(dataDir, outputDir, "archived"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
