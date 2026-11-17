package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type friendNote struct {
	PUUID string `json:"puuid,omitempty"`
	Name  string `json:"name"`
	Tag   string `json:"tag"`
	When  string `json:"when,omitempty"`
}

type nameChange struct {
	PUUID   string `json:"puuid,omitempty"`
	From    string `json:"from"`
	FromTag string `json:"fromTag,omitempty"`
	To      string `json:"to"`
	ToTag   string `json:"toTag,omitempty"`
	When    string `json:"when"`
}

type socialBook struct {
	Seen      map[string]friendNote `json:"seen"`
	Ready     bool                  `json:"ready"`
	Unfriends []friendNote          `json:"unfriends"`
	Renames   []nameChange          `json:"renames"`
	Watch     []friendNote          `json:"watch"`
}

var socialMu sync.Mutex

func brandFile(name string) string {
	dirs := []string{}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd, filepath.Join(wd, "brand"))
	}
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		dirs = append(dirs, filepath.Join(base, "Velvet"), filepath.Join(base, "Valvet"))
	}
	for _, dir := range dirs {
		for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".svg"} {
			path := filepath.Join(dir, name+ext)
			info, err := os.Stat(path)
			if err == nil && !info.IsDir() {
				return path
			}
		}
	}
	return ""
}

func socialPath() string {
	return filepath.Join(velvetBase(), "social.json")
}

func loadSocial() socialBook {
	book := socialBook{Seen: map[string]friendNote{}, Unfriends: []friendNote{}, Renames: []nameChange{}, Watch: []friendNote{}}
	raw, err := os.ReadFile(socialPath())
	if err != nil {
		return book
	}
	_ = json.Unmarshal(raw, &book)
	if book.Seen == nil {
		book.Seen = map[string]friendNote{}
	}
	if book.Unfriends == nil {
		book.Unfriends = []friendNote{}
	}
	if book.Watch == nil {
		book.Watch = []friendNote{}
	}
	if book.Renames == nil {
		book.Renames = []nameChange{}
	}
	return book
}

func saveSocial(book socialBook) {
	raw, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		return
	}
	path := socialPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, raw, 0o644)
}

func diffFriends(prev map[string]friendNote, ready bool, current map[string]friendNote) (map[string]friendNote, []friendNote, bool) {
	if current == nil {
		current = map[string]friendNote{}
	}
	if !ready {
		return current, nil, true
	}
	if len(current) == 0 && len(prev) > 0 {
		return prev, nil, true
	}
	next := map[string]friendNote{}
	for id, note := range current {
		if old, ok := prev[id]; ok {
			if note.Name == "" {
				note.Name = old.Name
			}
			if note.Tag == "" {
				note.Tag = old.Tag
			}
		}
		next[id] = note
	}
	gone := []friendNote{}
	now := time.Now().UTC().Format(time.RFC3339)
	for id, note := range prev {
		if _, ok := current[id]; ok {
			continue
		}
		note.When = now
		gone = append(gone, note)
	}
	return next, gone, true
}

func renamedFriends(prev, next map[string]friendNote) []nameChange {
	if len(prev) == 0 || len(next) == 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	out := []nameChange{}
	for id, note := range next {
		old, ok := prev[id]
		if !ok || blankName(old.Name) || blankName(note.Name) {
			continue
		}
		if strings.EqualFold(old.Name, note.Name) && strings.EqualFold(old.Tag, note.Tag) {
			continue
		}
		out = append(out, nameChange{
			PUUID: id, From: old.Name, FromTag: old.Tag, To: note.Name, ToTag: note.Tag, When: now,
		})
	}
	return out
}

func (c *riotClient) observeFriends(current map[string]friendNote) {
	socialMu.Lock()
	defer socialMu.Unlock()
	book := loadSocial()
	next, gone, ready := diffFriends(book.Seen, book.Ready, current)
	renamed := []nameChange{}
	if book.Ready {
		renamed = renamedFriends(book.Seen, next)
	}
	same := ready == book.Ready && len(gone) == 0 && len(renamed) == 0 && len(next) == len(book.Seen)
	if same {
		for id := range next {
			if _, ok := book.Seen[id]; !ok {
				same = false
				break
			}
		}
	}
	ids := make([]string, 0, len(next))
	for id := range next {
		ids = append(ids, id)
	}
	c.mu.Lock()
	c.friendIDs = ids
	c.mu.Unlock()
	if same {
		return
	}
	book.Seen = next
	book.Ready = ready
	if len(gone) > 0 {
		book.Unfriends = append(gone, book.Unfriends...)
		if len(book.Unfriends) > 80 {
			book.Unfriends = book.Unfriends[:80]
		}
	}
	if len(renamed) > 0 {
		book.Renames = append(renamed, book.Renames...)
		if len(book.Renames) > 80 {
			book.Renames = book.Renames[:80]
		}
	}
	saveSocial(book)
}

func rememberWatchPUUID(name, tag, puuid string) {
	if puuid == "" {
		return
	}
	socialMu.Lock()
	defer socialMu.Unlock()
	book := loadSocial()
	changed := false
	for i := range book.Watch {
		if strings.EqualFold(book.Watch[i].Name, name) && strings.EqualFold(book.Watch[i].Tag, tag) && book.Watch[i].PUUID == "" {
			book.Watch[i].PUUID = puuid
			changed = true
		}
	}
	if changed {
		saveSocial(book)
	}
}

func (c *riotClient) socialView() socialBook {
	socialMu.Lock()
	defer socialMu.Unlock()
	book := loadSocial()
	book.Seen = nil
	return book
}

func (c *riotClient) addWatch(name, tag string) socialBook {
	name, tag = strings.TrimSpace(name), strings.TrimSpace(tag)
	socialMu.Lock()
	defer socialMu.Unlock()
	book := loadSocial()
	if name == "" || tag == "" {
		book.Seen = nil
		return book
	}
	for _, item := range book.Watch {
		if strings.EqualFold(item.Name, name) && strings.EqualFold(item.Tag, tag) {
			book.Seen = nil
			return book
		}
	}
	book.Watch = append([]friendNote{{Name: name, Tag: tag, When: time.Now().UTC().Format(time.RFC3339)}}, book.Watch...)
	if len(book.Watch) > 40 {
		book.Watch = book.Watch[:40]
	}
	saveSocial(book)
	book.Seen = nil
	return book
}

func (c *riotClient) removeWatch(name, tag string) socialBook {
	socialMu.Lock()
	defer socialMu.Unlock()
	book := loadSocial()
	next := []friendNote{}
	for _, item := range book.Watch {
		if strings.EqualFold(item.Name, name) && strings.EqualFold(item.Tag, tag) {
			continue
		}
		next = append(next, item)
	}
	if next == nil {
		next = []friendNote{}
	}
	book.Watch = next
	saveSocial(book)
	book.Seen = nil
	return book
}
