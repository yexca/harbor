package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const maxServices = 500

var errNotFound = errors.New("service not found")

type service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Hidden      bool   `json:"hidden"`
}

type database struct {
	Version  int       `json:"version"`
	Services []service `json:"services"`
}

type store struct {
	mu   sync.RWMutex
	path string
	data database
}

func openStore(dir string) (*store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s := &store{path: filepath.Join(dir, "services.json"), data: database{Version: 1, Services: []service{}}}
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, fmt.Errorf("invalid services.json (the file has been preserved): %w", err)
	}
	if s.data.Version != 1 || len(s.data.Services) > maxServices {
		return nil, errors.New("unsupported or oversized services.json")
	}
	ids := make(map[string]bool)
	for i := range s.data.Services {
		item := &s.data.Services[i]
		if item.ID == "" || ids[item.ID] {
			return nil, errors.New("services.json contains missing or duplicate IDs")
		}
		ids[item.ID] = true
		if err := validateService(item); err != nil {
			return nil, fmt.Errorf("invalid saved service %s: %w", item.ID, err)
		}
	}
	return s, nil
}

func (s *store) list() []service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]service{}, s.data.Services...)
}

// Persist a complete snapshot before exposing the change in memory. A failed
// write leaves the previous configuration intact.
func (s *store) commit(items []service) error {
	next := database{Version: 1, Services: items}
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".services-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp.Name(), s.path); err != nil {
		return err
	}
	s.data = next
	return nil
}

func (s *store) add(item service) (service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data.Services) >= maxServices {
		return service{}, fmt.Errorf("maximum of %d services reached", maxServices)
	}
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return service{}, err
	}
	item.ID = hex.EncodeToString(id)
	items := append(append([]service{}, s.data.Services...), item)
	return item, s.commit(items)
}

func (s *store) update(id string, item service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := append([]service{}, s.data.Services...)
	for i := range items {
		if items[i].ID == id {
			item.ID = id
			items[i] = item
			return s.commit(items)
		}
	}
	return errNotFound
}

func (s *store) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := append([]service{}, s.data.Services...)
	for i := range items {
		if items[i].ID == id {
			return s.commit(append(items[:i], items[i+1:]...))
		}
	}
	return errNotFound
}

// A missing hidden field in older configurations means the service stays on
// the home screen. Visibility updates do not overwrite the rest of a card.
func (s *store) setVisibility(id string, hidden bool) (service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := append([]service{}, s.data.Services...)
	for i := range items {
		if items[i].ID == id {
			items[i].Hidden = hidden
			return items[i], s.commit(items)
		}
	}
	return service{}, errNotFound
}
