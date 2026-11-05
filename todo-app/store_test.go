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

func readItems(t *testing.T, path string) []savedTodo {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f storeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil
	}
	out := make([]savedTodo, len(f.Items))
	for i, it := range f.Items {
		out[i] = savedTodo{Done: it.Done}
	}
	return out
}

func waitSaved(t *testing.T, path string, wantDone bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got := readItems(t, path)
		if len(got) > 0 && got[0].Done == wantDone {
			return
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
		got := readItems(t, path)
		if len(got) == 0 {
			return
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

func TestSetTitle(t *testing.T) {
	s, _ := openTestStore(t)
	a, _ := s.Add("old title")

	if err := s.SetTitle(a.ID, "  new title  "); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	if got := s.Snapshot()[0].Title; got != "new title" {
		t.Fatalf("title = %q", got)
	}

	if err := s.SetTitle(a.ID, "   "); err == nil {
		t.Fatal("empty title should fail")
	}
	if got := s.Snapshot()[0].Title; got != "new title" {
		t.Fatalf("failed SetTitle changed title to %q", got)
	}

	if err := s.SetTitle(999, "nope"); err != nil {
		t.Fatalf("unknown id: %v", err)
	}
}

func TestSetDue(t *testing.T) {
	s, path := openTestStore(t)
	a, _ := s.Add("dated")

	s.SetDue(a.ID, "2026-11-20")
	if got := s.Snapshot()[0].Due; got != "2026-11-20" {
		t.Fatalf("due = %q", got)
	}
	s.SetDue(a.ID, "")
	if got := s.Snapshot()[0].Due; got != "" {
		t.Fatalf("clear left %q", got)
	}
	s.SetDue(a.ID, "2026-11-21")
	s.SetDue(999, "2026-11-22")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Items []struct {
			ID  int64  `json:"id"`
			Due string `json:"due"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Items) != 1 || f.Items[0].Due != "2026-11-21" {
		t.Fatalf("saved dues wrong: %+v", f.Items)
	}
}

func TestLoadStoreBOM(t *testing.T) {
	s, path := openTestStore(t)
	if _, err := s.Add("bom item"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append([]byte("\xef\xbb\xbf"), data...), 0o644); err != nil {
		t.Fatal(err)
	}
	reopened, err := openStoreAt(path)
	if err != nil {
		t.Fatalf("BOM data should load: %v", err)
	}
	if got := reopened.Snapshot(); len(got) != 1 || got[0].Title != "bom item" {
		t.Fatalf("BOM load wrong: %+v", got)
	}
}

func TestOnChange(t *testing.T) {
	s, _ := openTestStore(t)
	n := 0
	s.OnChange(func() { n++ })

	s.Add("one")
	s.SetDone(s.Snapshot()[0].ID, true)
	if n != 2 {
		t.Fatalf("observer ran %d times after Add+SetDone", n)
	}
	if got := s.ClearDone(); got != 1 {
		t.Fatalf("ClearDone removed %d", got)
	}
	if n != 3 {
		t.Fatalf("observer ran %d times after ClearDone", n)
	}
	if got := s.ClearDone(); got != 0 {
		t.Fatalf("second ClearDone removed %d", got)
	}
	if n != 3 { // ClearDone with nothing done skips Save
		t.Fatalf("observer ran %d times after no-op ClearDone", n)
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

func TestV1Migration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	v1 := `[{"id":1,"title":"legacy","done":true,"createdAt":"2026-01-01T00:00:00Z"}]`
	if err := os.WriteFile(path, []byte(v1), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := openStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	got := s.Snapshot()
	if len(got) != 1 || got[0].Title != "legacy" || !got[0].Done {
		t.Fatalf("v1 load wrong: %+v", got)
	}
	if _, err := s.Add("new"); err != nil {
		t.Fatal(err)
	}

	// Save is synchronous: the file is already the v2 envelope here.
	var f storeFile
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("saved file is not a v2 envelope: %v", err)
	}
	if f.Version != storeVersion || len(f.Items) != 2 {
		t.Fatalf("envelope wrong: version=%d items=%d", f.Version, len(f.Items))
	}
}

func TestFutureVersionRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	fut := `{"version":99,"items":[{"id":1,"title":"x","done":false}]}`
	if err := os.WriteFile(path, []byte(fut), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := openStoreAt(path); err == nil {
		t.Fatal("a future version must be rejected, not silently downgraded")
	}
}

func TestAtomicWriteLeavesNoTmp(t *testing.T) {
	s, path := openTestStore(t)
	if _, err := s.Add("one"); err != nil {
		t.Fatal(err)
	}
	waitSaved(t, path, false)
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("tmp file left behind: %v", err)
	}
}
