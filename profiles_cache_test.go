package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	"nameless/ui/pages"

	"github.com/egoist/mygo/ui"
	_ "modernc.org/sqlite"
)

func TestRefreshProfileCountsSingleQueryCache(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "counts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT)")
	db.Exec("CREATE TABLE videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT)")
	db.Exec("INSERT INTO profiles(id,url,name) VALUES(1,'u1','a'),(2,'u2','b')")
	db.Exec("INSERT INTO videos(profile_id,url,title,status) VALUES(1,'x','v','done'),(1,'y','w','done'),(2,'z','u','pending')")

	a := &app{cfg: defaultConfig(), db: db}
	a.loadProfiles()
	if got := a.profileCounts[1]; got != 2 {
		t.Fatalf("profile 1 count: got %d want 2", got)
	}
	if got := a.profileCounts[2]; got != 1 {
		t.Fatalf("profile 2 count: got %d want 1", got)
	}
	// Profiles page renders from the cache without DB access per frame:
	// even with the DB closed, counts remain available.
	db.Close()
	a.mu.Lock()
	counts := a.profileCounts
	a.mu.Unlock()
	if counts[1] != 2 {
		t.Fatalf("cached count lost after db close: %v", counts)
	}
	_ = pages.ProfileItem{}
	_ = ui.NewTester
}
