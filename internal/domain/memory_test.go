package domain

import (
	"testing"
	"time"
)

func TestInMemoryStore_GetSetDelete(t *testing.T) {
	s := NewInMemoryStore()

	if _, ok, _ := s.Get("example.com"); ok {
		t.Fatal("expected no entry before Set")
	}

	entry := Entry{
		Domain:     "example.com",
		ExpireTime: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := s.Set(entry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, ok, err := s.Get("example.com")
	if err != nil || !ok {
		t.Fatalf("Get failed: ok=%v err=%v", ok, err)
	}
	t.Logf("got entry: %+v", got)

	if err := s.Delete("example.com"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, ok, _ := s.Get("example.com"); ok {
		t.Fatal("expected no entry after Delete")
	}
}

func TestInMemoryStore_List(t *testing.T) {
	s := NewInMemoryStore()
	if err := s.Set(Entry{Domain: "a.com"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(Entry{Domain: "b.com"}); err != nil {
		t.Fatal(err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
}
