package pages

import "testing"

func TestSortVideos(t *testing.T) {
	vids := []VideoItem{
		{ID: 1, Title: "Banana", Duration: "1:00", Status: "done"},
		{ID: 2, Title: "apple", Duration: "5:00", Status: "failed"},
		{ID: 3, Title: "Cherry", Duration: "2:30", Status: "pending"},
	}
	// Default preserves order and does not mutate.
	got := sortVideos(vids, "Default")
	if got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("default order changed: %v", got)
	}
	if got := sortVideos(vids, "Title A-Z"); got[0].Title != "apple" {
		t.Fatalf("title asc: %v", got)
	}
	if got := sortVideos(vids, "Newest first"); got[0].ID != 3 {
		t.Fatalf("newest: %v", got)
	}
	if got := sortVideos(vids, "Longest first"); got[0].ID != 2 {
		t.Fatalf("longest: %v", got)
	}
	if got := sortVideos(vids, "Needs attention"); got[0].Status != "failed" {
		t.Fatalf("needs attention should put failed first: %v", got)
	}
	// Input not mutated.
	if vids[0].ID != 1 {
		t.Fatalf("input mutated: %v", vids)
	}
	if parseDurationSec("1:02:03") != 3723 || parseDurationSec("--:--") != 0 {
		t.Fatalf("duration parse wrong")
	}
	// Sorted + paginated together: sort applies before paging.
	page, total := filterAndPaginate(vids, "", "Title A-Z", 1, 2)
	if total != 2 || len(page) != 2 || page[0].Title != "apple" {
		t.Fatalf("filterSortPaginate: page=%v total=%d", page, total)
	}
}
