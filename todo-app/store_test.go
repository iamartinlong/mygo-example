package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type savedTodo struct {
	Done bool `json:"done"`
}

func waitSaved(t *testing.T, path string, wantDone bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			var got []savedTodo
			if json.Unmarshal(data, &got) == nil && len(got) > 0 && got[0].Done == wantDone {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("store never saved wantDone=%v", wantDone)
}

func openTestStore(t *testing.T) (*Todos, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "todos.json")
	s, err := openStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	return s, path
}

func waitEmpty(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			var got []savedTodo
			if json.Unmarshal(data, &got) == nil && len(got) == 0 {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("store never saved the empty list")
}

func TestStoreRoundTrip(t *testing.T) {
	s, path := openTestStore(t)

	a, err := s.Add(" write the report ")
	if err != nil || a.Title != "write the report" {
		t.Fatalf("Add: %v %+v", err, a)
	}
	if _, err := s.Add(""); err == nil {
		t.Fatal("empty title should fail")
	}
	s.SetDone(a.ID, true)

	waitSaved(t, path, true)

	reopened, err := openStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Snapshot()
	if len(got) != 1 || got[0].Title != "write the report" || !got[0].Done {
		t.Fatalf("reopened store wrong: %+v", got)
	}
}

func TestRemoveAndClearDone(t *testing.T) {
	s, _ := openTestStore(t)
	a, _ := s.Add("one")
	b, _ := s.Add("two")

	s.Remove(a.ID)
	if got := s.Snapshot(); len(got) != 1 || got[0].ID != b.ID {
		t.Fatalf("Remove left %+v", got)
	}

	s.SetDone(b.ID, true)
	if n := s.ClearDone(); n != 1 {
		t.Fatalf("ClearDone removed %d", n)
	}
	if got := s.Snapshot(); len(got) != 0 {
		t.Fatalf("ClearDone left %+v", got)
	}
	st := s.Stats()
	if st.Total != 0 || st.Done != 0 || st.Active != 0 {
		t.Fatalf("Stats wrong: %+v", st)
	}
	waitEmpty(t, s.path)
}
