package store

import (
	"os"
	"path/filepath"
	"testing"
)

import ytdlp "nameless/internal/ytdlp"

func setupTestDB(t *testing.T) *DB {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})
	return db
}

func TestCreateProfile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	profile, err := db.CreateProfile("https://tiktok.com/@test", "Test Profile")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}
	if profile.ID != 1 {
		t.Errorf("expected profile ID 1, got %d", profile.ID)
	}
	if profile.URL != "https://tiktok.com/@test" {
		t.Errorf("expected url https://tiktok.com/@test, got %s", profile.URL)
	}
	if profile.Name != "Test Profile" {
		t.Errorf("expected name Test Profile, got %s", profile.Name)
	}
	if profile.VideoCount != 0 {
		t.Errorf("expected video_count 0, got %d", profile.VideoCount)
	}
	if !profile.FetchedAt.IsZero() {
		t.Errorf("expected zero FetchedAt, got %v", profile.FetchedAt)
	}
}

func TestUpsertVideos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a profile first
	_, err := db.CreateProfile("https://tiktok.com/@test", "Test Profile")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Upsert some videos
	vids := []ytdlp.VideoMeta{
		{ID: "vid1", URL: "https://example.com/video1", Title: "Video 1"},
		{ID: "vid2", URL: "https://example.com/video2", Title: "Video 2"},
		{ID: "vid1", URL: "https://example.com/video1", Title: "Video 1 dup"}, // duplicate
	}
	n, err := db.UpsertVideos(1, vids)
	if err != nil {
		t.Fatalf("UpsertVideos failed: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 new videos, got %d", n)
	}

	// Upsert again - should update existing, not increase count
	n2, err := db.UpsertVideos(1, []ytdlp.VideoMeta{
		{ID: "vid3", URL: "https://example.com/video3", Title: "Video 3"},
	})
	if err != nil {
		t.Fatalf("UpsertVideos second batch failed: %v", err)
	}
	if n2 != 1 {
		t.Errorf("expected 1 new video on second batch, got %d", n2)
	}

	// Verify profile video count
	profiles, err := db.Profiles()
	if err != nil {
		t.Fatalf("Profiles failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].VideoCount != 3 {
		t.Errorf("expected video_count=3, got %d", profiles[0].VideoCount)
	}
}

func TestVideos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create profile and videos
	db.CreateProfile("https://tiktok.com/@test", "Test Profile")
	db.UpsertVideos(1, []ytdlp.VideoMeta{
		{ID: "v1", URL: "https://example.com/1", Title: "Video 1"},
		{ID: "v2", URL: "https://example.com/2", Title: "Video 2"},
	})

	vids, err := db.Videos(1)
	if err != nil {
		t.Fatalf("Videos failed: %v", err)
	}
	if len(vids) != 2 {
		t.Errorf("expected 2 videos, got %d", len(vids))
	}
}

func TestFailedVideos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create profile and insert a video first
	db.CreateProfile("https://tiktok.com/@test", "Test Profile")
	db.UpsertVideos(1, []ytdlp.VideoMeta{
		{ID: "v1", URL: "https://example.com/1", Title: "Video 1"},
	})
	db.SetVideoFailed(1, "some error", "/path/to/video")

	failed, err := db.FailedVideos(1)
	if err != nil {
		t.Fatalf("FailedVideos failed: %v", err)
	}
	if len(failed) != 1 {
		t.Errorf("expected 1 failed video, got %d", len(failed))
	}
	if failed[0].Status != "failed" {
		t.Errorf("expected status failed, got %s", failed[0].Status)
	}
}

func TestResetInterrupted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create profile and insert a video first
	db.CreateProfile("https://tiktok.com/@test", "Test Profile")
	db.UpsertVideos(1, []ytdlp.VideoMeta{
		{ID: "v1", URL: "https://example.com/1", Title: "Video 1"},
	})
	db.SetVideoFailed(1, "some error", "/path/to/video")
	db.SetVideoStatus(1, "paused")

	// Reset interrupted
	err := db.ResetInterrupted()
	if err != nil {
		t.Fatalf("ResetInterrupted failed: %v", err)
	}

	// Check that videos are now pending
	failed, err := db.FailedVideos(1)
	if err != nil {
		t.Fatalf("FailedVideos after reset failed: %v", err)
	}
	if len(failed) != 0 {
		t.Errorf("expected 0 failed videos after reset, got %d", len(failed))
	}

	// Check status is now pending
	vids, err := db.Videos(1)
	if err != nil {
		t.Fatalf("Videos after reset failed: %v", err)
	}
	if len(vids) != 1 {
		t.Errorf("expected 1 video, got %d", len(vids))
	}
	if vids[0].Status != "pending" {
		t.Errorf("expected status pending after reset, got %s", vids[0].Status)
	}
}

func TestSetVideoStatusAndProgress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	db.CreateProfile("https://tiktok.com/@test", "Test Profile")
	db.UpsertVideos(1, []ytdlp.VideoMeta{
		{ID: "v1", URL: "https://example.com/1", Title: "Video 1"},
	})

	// Set status
	err := db.SetVideoStatus(1, "downloading")
	if err != nil {
		t.Fatalf("SetVideoStatus failed: %v", err)
	}

	// Set progress
	err = db.SetVideoProgress(1, 0.5, 512, 1024)
	if err != nil {
		t.Fatalf("SetVideoProgress failed: %v", err)
	}

	// Verify
	vids, err := db.Videos(1)
	if err != nil {
		t.Fatalf("Videos failed: %v", err)
	}
	if vids[0].Status != "downloading" {
		t.Errorf("expected status downloading, got %s", vids[0].Status)
	}
	if vids[0].Pct != 0.5 {
		t.Errorf("expected pct 0.5, got %f", vids[0].Pct)
	}
	if vids[0].Downloaded != 512 {
		t.Errorf("expected downloaded 512, got %d", vids[0].Downloaded)
	}
	if vids[0].Total != 1024 {
		t.Errorf("expected total 1024, got %d", vids[0].Total)
	}
}

func TestDeleteProfile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create profile with videos
	db.CreateProfile("https://tiktok.com/@test1", "Profile 1")
	db.CreateProfile("https://tiktok.com/@test2", "Profile 2")
	db.UpsertVideos(1, []ytdlp.VideoMeta{
		{ID: "v1", URL: "https://example.com/1", Title: "Video 1"},
	})

	// Delete profile 1
	err := db.DeleteProfile(1)
	if err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}

	// Profile 1 should be gone
	profiles, err := db.Profiles()
	if err != nil {
		t.Fatalf("Profiles failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile after deletion, got %d", len(profiles))
	}
	if profiles[0].Name != "Profile 2" {
		t.Errorf("expected profile 2, got %s", profiles[0].Name)
	}

	// Videos of profile 1 should be deleted too
	vids, err := db.Videos(1)
	if err != nil {
		t.Fatalf("Videos failed: %v", err)
	}
	if len(vids) != 0 {
		t.Errorf("expected 0 videos for deleted profile, got %d", len(vids))
	}
}
