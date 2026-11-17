package main

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

var ownedNameState struct {
	mu    sync.Mutex
	at    time.Time
	ttl   time.Duration
	names []string
}

func (c *riotClient) ownedSkinNames(ctx context.Context) []string {
	ownedNameState.mu.Lock()
	if ownedNameState.names != nil && time.Since(ownedNameState.at) < ownedNameState.ttl {
		out := append([]string(nil), ownedNameState.names...)
		ownedNameState.mu.Unlock()
		return out
	}
	ownedNameState.mu.Unlock()
	c.ensureSkins(ctx)
	c.ensureCosmetics(ctx)
	c.mu.Lock()
	sess := c.session
	skins := c.skins
	look := c.skinLook
	buddies := c.buddies
	flair := c.flair
	rewards := c.rewards
	c.mu.Unlock()
	if sess.puuid == "" || sess.shard == "" {
		return []string{}
	}
	names := []string{}
	seen := map[string]bool{}
	add := func(name string) {
		name = skinWishName(name)
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			return
		}
		seen[key] = true
		names = append(names, name)
	}
	for _, ids := range c.allOwned(ctx, sess) {
		for id := range ids {
			if info, ok := skins[id]; ok {
				add(info.Name)
			}
			if info, ok := look[id]; ok {
				add(info.Name)
			}
			if info, ok := buddies[id]; ok {
				add(info.Name)
			}
			if info, ok := flair[id]; ok {
				add(info.Name)
			}
			if info, ok := rewards[id]; ok {
				add(info.Name)
			}
		}
	}
	sort.Strings(names)
	ttl := 10 * time.Minute
	if len(names) == 0 {
		ttl = 20 * time.Second
	}
	ownedNameState.mu.Lock()
	ownedNameState.at = time.Now()
	ownedNameState.ttl = ttl
	ownedNameState.names = names
	ownedNameState.mu.Unlock()
	return names
}

func forgetOwnedCache() {
	ownedNameState.mu.Lock()
	ownedNameState.at = time.Time{}
	ownedNameState.mu.Unlock()
}
