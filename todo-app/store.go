package main

import (
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
	CreatedAt time.Time `json:"createdAt"`
}

// Stats summarizes the list.
type Stats struct {
	Total  int `json:"total"`
	Active int `json:"active"`
	Done   int `json:"done"`
}

// Todos stores the list as JSON in the user data directory.
type Todos struct {
	mu    sync.Mutex
	path  string
	items []Todo
	next  int64
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
		if err := json.Unmarshal(data, &t.items); err != nil {
			return nil, err
		}
	}
	for _, it := range t.items {
		t.next = max(t.next, it.ID)
	}
	return t, nil
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
// synchronous.
func (t *Todos) Save() {
	t.mu.Lock()
	data, err := json.MarshalIndent(t.items, "", "  ")
	t.mu.Unlock()
	if err != nil {
		return
	}
	_ = os.WriteFile(t.path, data, 0o644)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
