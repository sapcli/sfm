package launchpad

import "testing"

func TestGetNewExpiryDate(t *testing.T) {
	v := GetNewExpiryDate(10)
	if len(v) < len("datetime'2006-01-02T00:00:00'") {
		t.Fatalf("unexpected date format: %s", v)
	}
}

func TestGetDateFromTimestamp(t *testing.T) {
	tm, err := GetDateFromTimestamp("/Date(1648530729000)/")
	if err != nil {
		t.Fatalf("GetDateFromTimestamp failed: %v", err)
	}
	if tm.Year() != 2022 {
		t.Fatalf("unexpected year: %d", tm.Year())
	}
}

func TestChunkUsers(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	chunks := ChunkUsers(items, 2)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
}
