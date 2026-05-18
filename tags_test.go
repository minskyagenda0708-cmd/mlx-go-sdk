package mlx

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/minskyagenda0708-cmd/mlx-go-sdk/internal/testutil"
)

func TestTagsCreate(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/tag/create" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Tags created"},"data":{"tags":[{"id":"tag-1","name":"DemoTag","color":"red","created_at":"2026-05-18T00:00:00Z","updated_at":"2026-05-18T00:00:00Z","created_by":"test@example.com","in_use_count":0}]}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Tags.Create(context.Background(), &CreateTagsRequest{
		Tags: []CreateTagItem{{Name: "DemoTag", Color: "red"}},
	})
	if err != nil {
		t.Fatalf("Tags.Create returned error: %v", err)
	}
	if len(resp.Data.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(resp.Data.Tags))
	}
	if resp.Data.Tags[0].Name != "DemoTag" {
		t.Fatalf("unexpected tag name: %s", resp.Data.Tags[0].Name)
	}
}

func TestTagsUpdate(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tag/update" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Tags updated"},"data":{"tags":[{"id":"tag-1","name":"UpdatedTag","color":"blue","created_at":"2026-05-18T00:00:00Z","updated_at":"2026-05-18T01:00:00Z","created_by":"test@example.com","in_use_count":1}]}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Tags.Update(context.Background(), &UpdateTagsRequest{
		Tags: []UpdateTagItem{{ID: "tag-1", Name: "UpdatedTag", Color: "blue"}},
	})
	if err != nil {
		t.Fatalf("Tags.Update returned error: %v", err)
	}
	if resp.Data.Tags[0].Color != "blue" {
		t.Fatalf("unexpected tag color: %s", resp.Data.Tags[0].Color)
	}
}

func TestTagsRemove(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tag/remove" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Tags removed"},"data":{}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, _, err = client.Tags.Remove(context.Background(), &RemoveTagsRequest{IDs: []string{"tag-1"}})
	if err != nil {
		t.Fatalf("Tags.Remove returned error: %v", err)
	}
}

func TestTagsAssignToProfiles(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tag/assign_to_profiles" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Tags assigned"},"data":{}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, _, err = client.Tags.AssignToProfiles(context.Background(), &AssignTagsRequest{
		TagIDs:     []string{"tag-1"},
		ProfileIDs: []string{"profile-1"},
	})
	if err != nil {
		t.Fatalf("Tags.AssignToProfiles returned error: %v", err)
	}
}

func TestTagsSearch(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tag/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Tags search done"},"data":{"tags":[{"id":"tag-1","name":"SearchTag","color":"green","created_at":"2026-05-18T00:00:00Z","updated_at":"2026-05-18T00:00:00Z","created_by":"test@example.com","in_use_count":2}],"total_count":1}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Tags.Search(context.Background(), &SearchTagsRequest{
		SearchText: "SearchTag",
		Limit:      10,
		Offset:     0,
	})
	if err != nil {
		t.Fatalf("Tags.Search returned error: %v", err)
	}
	if len(resp.Data.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(resp.Data.Tags))
	}
	if resp.Data.TotalCount != 1 {
		t.Fatalf("expected total_count 1, got %d", resp.Data.TotalCount)
	}
}
