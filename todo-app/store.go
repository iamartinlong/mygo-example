package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// Todo is an item of the list.
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	Due       string    `json:"due,omitempty"` // date as 2006-01-02, empty for none
	CreatedAt time.Time `json:"createdAt"`
}

// Stats summarizes the list.
type Stats struct {
	Total  int `json:"total"`
	Active int `json:"active"`
	Done   int `json:"done"`
}

// storeVersion is the on-disk format version. Version 1 was a bare JSON
// array of Todo; 2 wraps it in storeFile with the version field.
const storeVersion = 2

// storeFile is the versioned envelope of todos.json.
type storeFile struct {
	Version int    `json:"version"`
	Items   []Todo `json:"items"`
}

// Todos stores the list as JSON in the user data directory.
type Todos struct {
	mu        sync.Mutex
	path      string
	items     []Todo
	next      int64
	observers []func()
}

// OpenStore loads the saved todos.
func OpenStore() (*Todos, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return nil, err
	}
	return openStoreAt(filepath.Join(dir, "todos.json"))
}

func openStoreAt(path string) (*Todos, error) {
	t := &Todos{path: path}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(data) > 0 {
		if err := loadStore(data, t); err != nil {
			return nil, err
		}
	}
	for _, it := range t.items {
		t.next = max(t.next, it.ID)
	}
	return t, nil
}

func loadStore(data []byte, t *Todos) error {
	// Windows editors write a UTF-8 BOM, which json.Unmarshal rejects.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if b := data[0]; b == '[' { // v1: bare array
		return json.Unmarshal(data, &t.items)
	}
	var f storeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	if f.Version > storeVersion {
		return errors.New("todos.json was written by a newer app version")
	}
	t.items = f.Items
	return nil
}

// Snapshot returns a copy of the list, done last.
func (t *Todos) Snapshot() []Todo {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := slices.Clone(t.items)
	slices.SortStableFunc(out, func(a, b Todo) int {
		return boolToInt(a.Done) - boolToInt(b.Done)
	})
	return out
}

// Add creates a todo.
func (t *Todos) Add(title string) (Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Todo{}, errors.New("a todo needs a title")
	}
	t.mu.Lock()
	t.next++
	todo := Todo{ID: t.next, Title: title, CreatedAt: time.Now()}
	t.items = append(t.items, todo)
	t.mu.Unlock()
	t.Save()
	return todo, nil
}

// SetTitle renames a todo. Empty titles are rejected and keep the old one.
func (t *Todos) SetTitle(id int64, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("a todo needs a title")
	}
	t.mu.Lock()
	i := slices.IndexFunc(t.items, func(it Todo) bool { return it.ID == id })
	if i < 0 {
		t.mu.Unlock()
		return nil
	}
	t.items[i].Title = title
	t.mu.Unlock()
	t.Save()
	return nil
}

// SetDue sets or clears a due date, as a 2006-01-02 date; empty clears it.
func (t *Todos) SetDue(id int64, due string) {
	t.mu.Lock()
	i := slices.IndexFunc(t.items, func(it Todo) bool { return it.ID == id })
	if i < 0 {
		t.mu.Unlock()
		return
	}
	t.items[i].Due = due
	t.mu.Unlock()
	t.Save()
}

// SetDone marks a todo done or not done.
func (t *Todos) SetDone(id int64, done bool) {
	t.mu.Lock()
	i := slices.IndexFunc(t.items, func(it Todo) bool { return it.ID == id })
	if i < 0 {
		t.mu.Unlock()
		return
	}
	t.items[i].Done = done
	t.mu.Unlock()
	t.Save()
}

// Remove deletes a todo by ID.
func (t *Todos) Remove(id int64) {
	t.mu.Lock()
	t.items = slices.DeleteFunc(t.items, func(it Todo) bool { return it.ID == id })
	t.mu.Unlock()
	t.Save()
}

// ClearDone removes completed todos and returns how many were removed.
func (t *Todos) ClearDone() int {
	n := 0
	t.mu.Lock()
	before := len(t.items)
	t.items = slices.DeleteFunc(t.items, func(it Todo) bool { return it.Done })
	n = before - len(t.items)
	t.mu.Unlock()
	if n > 0 {
		t.Save()
	}
	return n
}

// OnChange registers fn to run after each saved change. The callback runs
// on whatever thread made the change; a window it touches must schedule
// its own redraw through Window.Update.
func (t *Todos) OnChange(fn func()) {
	t.mu.Lock()
	t.observers = append(t.observers, fn)
	t.mu.Unlock()
}

// Stats returns counters for the footer.
func (t *Todos) Stats() Stats {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := Stats{Total: len(t.items)}
	for _, it := range t.items {
		if it.Done {
			s.Done++
		} else {
			s.Active++
		}
	}
	return s
}

// Save persists the list. Writes must not run concurrently: out-of-order
// writes would let an older list overwrite a newer one, so this is
// synchronous. The write is atomic (temp file + rename), so a crash mid
// write leaves the previous file intact.
func (t *Todos) Save() {
	t.mu.Lock()
	data, err := json.MarshalIndent(storeFile{Version: storeVersion, Items: t.items}, "", "  ")
	observers := slices.Clone(t.observers)
	t.mu.Unlock()
	if err != nil {
		return
	}
	t.atomicWrite(data)
	for _, fn := range observers {
		fn()
	}
}

func (t *Todos) atomicWrite(data []byte) {
	tmp := t.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	if _, err := f.Write(data); err == nil {
		err = f.Sync()
	}
	if err == nil {
		err = f.Close()
	} else {
		f.Close()
	}
	if err != nil {
		os.Remove(tmp)
		return
	}
	if err := os.Rename(tmp, t.path); err != nil {
		os.Remove(tmp)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
