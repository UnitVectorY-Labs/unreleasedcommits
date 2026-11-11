package main

import (
	"bytes"
	"encoding/json/v2"
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
