package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

const maxServices = 500

var errNotFound = errors.New("service not found")

// Background files are named by the store, so a saved name never reaches the
// filesystem unless it matches this pattern.
var backgroundName = regexp.MustCompile(`^background-[0-9a-f]{24}\.(jpg|png|gif|webp)$`)

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

// Site identity and the custom background image are shared by every device.
// An empty title or icon means the built-in default.
type siteSettings struct {
	Title      string `json:"title"`
	Icon       string `json:"icon"`
	Background string `json:"background"`
}

type siteFile struct {
	Version int `json:"version"`
	siteSettings
}

type store struct {
	mu       sync.RWMutex
	path     string
	data     database
	sitePath string
	site     siteSettings
}

func openStore(dir string) (*store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s := &store{path: filepath.Join(dir, "services.json"), data: database{Version: 1, Services: []service{}}, sitePath: filepath.Join(dir, "site.json")}
	if err := s.loadSite(); err != nil {
		return nil, err
	}
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

// An absent site.json keeps the defaults. Invalid saved settings fail startup
// without replacing the file, like services.json.
func (s *store) loadSite() error {
	raw, err := os.ReadFile(s.sitePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var saved siteFile
	if err := json.Unmarshal(raw, &saved); err != nil {
		return fmt.Errorf("invalid site.json (the file has been preserved): %w", err)
	}
	if saved.Version != 1 {
		return errors.New("unsupported site.json version")
	}
	if err := validateSite(&saved.siteSettings); err != nil {
		return fmt.Errorf("invalid site.json: %w", err)
	}
	if saved.Background != "" {
		if _, err := os.Stat(filepath.Join(filepath.Dir(s.sitePath), saved.Background)); err != nil {
			// Fall back to the built-in wallpapers instead of refusing to start.
			log.Printf("custom background unavailable: %v", err)
			saved.Background = ""
		}
	}
	s.site = saved.siteSettings
	return nil
}

func validateSite(item *siteSettings) error {
	if err := validateTitle(&item.Title); err != nil {
		return err
	}
	if item.Icon != "" {
		if err := validateIconData(item.Icon); err != nil {
			return err
		}
	}
	if item.Background != "" && !backgroundName.MatchString(item.Background) {
		return errors.New("invalid background file name")
	}
	return nil
}

// Write a complete file beside its destination and rename it into place, so a
// failure leaves the previous file intact.
func writeAtomic(path, pattern string, raw []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(raw); err != nil {
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
	return os.Rename(tmp.Name(), path)
}

// Persist a complete snapshot before exposing the change in memory. A failed
// write leaves the previous configuration intact.
func (s *store) commit(items []service) error {
	next := database{Version: 1, Services: items}
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err = writeAtomic(s.path, ".services-*", append(raw, '\n')); err != nil {
		return err
	}
	s.data = next
	return nil
}

func (s *store) commitSite(next siteSettings) error {
	raw, err := json.MarshalIndent(siteFile{Version: 1, siteSettings: next}, "", "  ")
	if err != nil {
		return err
	}
	if err = writeAtomic(s.sitePath, ".site-*", append(raw, '\n')); err != nil {
		return err
	}
	s.site = next
	return nil
}

func (s *store) siteSettings() siteSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.site
}

// A nil icon keeps the saved icon; an empty one restores the default.
func (s *store) setIdentity(title string, icon *string) (siteSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.site
	next.Title = title
	if icon != nil {
		next.Icon = *icon
	}
	if err := s.commitSite(next); err != nil {
		return siteSettings{}, err
	}
	return next, nil
}

// Store the image under a fresh name before switching site.json to it, then
// remove the previous image. A failure keeps the previous background.
func (s *store) setBackground(raw []byte, extension string) (siteSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return siteSettings{}, err
	}
	name := "background-" + hex.EncodeToString(id) + "." + extension
	path := filepath.Join(filepath.Dir(s.sitePath), name)
	if err := writeAtomic(path, ".background-*", raw); err != nil {
		return siteSettings{}, err
	}
	previous := s.site.Background
	next := s.site
	next.Background = name
	if err := s.commitSite(next); err != nil {
		os.Remove(path)
		return siteSettings{}, err
	}
	s.removeBackgroundFile(previous)
	return next, nil
}

func (s *store) clearBackground() (siteSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.site.Background
	next := s.site
	next.Background = ""
	if previous == "" {
		return next, nil
	}
	if err := s.commitSite(next); err != nil {
		return siteSettings{}, err
	}
	s.removeBackgroundFile(previous)
	return next, nil
}

// A leftover file only costs disk space, so removal failures are logged.
func (s *store) removeBackgroundFile(name string) {
	if name == "" {
		return
	}
	if err := os.Remove(filepath.Join(filepath.Dir(s.sitePath), name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("remove previous background: %v", err)
	}
}

func (s *store) openBackground(name string) (*os.File, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if name == "" || name != s.site.Background {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(filepath.Dir(s.sitePath), name))
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
