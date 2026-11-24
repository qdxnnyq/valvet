package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func validPUUID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
	}
	return true
}

func main() {
	if wantsUninstall() {
		uninstallValvet()
		return
	}
	migrateAppDirs()
	if !claimSingleInstance() {
		return
	}
	registerInstalled()
	app := newRiotClient()
	mux := http.NewServeMux()
	mux.Handle("/pack/", packHandler())
	mux.HandleFunc("/api/art", artHandler)
	mux.HandleFunc("/api/replays", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost {
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(io.LimitReader(r.Body, 2000)).Decode(&body)
			if path, ok := replayPath(body.Name); ok {
				if err := openReplayFile(path); err != nil {
					_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error(), "game": valorantOpen(), "replays": loadReplayIndex()})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "game": valorantOpen(), "replays": loadReplayIndex()})
				return
			}
		}
		rows := archiveReplays()
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "game": valorantOpen(), "replays": rows})
	})
	mux.HandleFunc("/api/log", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": readAppLog()})
	})
	mux.HandleFunc("/sounds/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.Trim(strings.TrimPrefix(r.URL.Path, "/sounds/"), "/")
		w.Header().Set("Cache-Control", "no-store")
		if !writeMedia(w, r, name) {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/wait.gif", func(w http.ResponseWriter, r *http.Request) {
		writeBytes(w, r, "wait.gif", "image/gif", waitGif)
	})
	mux.HandleFunc("/yippee.gif", func(w http.ResponseWriter, r *http.Request) {
		writeBytes(w, r, "yippee.gif", "image/gif", yippeeGif)
	})
	mux.HandleFunc("/cry.gif", func(w http.ResponseWriter, r *http.Request) {
		writeBytes(w, r, "cry.gif", "image/gif", cryGif)
	})
	mux.HandleFunc("/api/media", func(w http.ResponseWriter, r *http.Request) {
		kind := mediaKind(r.URL.Query().Get("kind"))
		if kind == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost {
			limit := int64(2 << 20)
			if kind == "background" {
				limit = 6 << 20
			}
			if err := saveMedia(kind, http.MaxBytesReader(w, r.Body, limit), limit); err != nil {
				http.Error(w, "no", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodDelete {
			if path := mediaPath(kind); path != "" {
				_ = os.Remove(path)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !writeMedia(w, r, kind) {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/brand/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.Trim(strings.TrimPrefix(r.URL.Path, "/brand/"), "/")
		if name != "riot" && name != "league" {
			http.NotFound(w, r)
			return
		}
		if name == "league" && len(leagueLogo) > 0 {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(leagueLogo)
			return
		}
		if name == "riot" && len(riotLogo) > 0 {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(riotLogo)
			return
		}
		path := brandFile(name)
		if path == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, path)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/broadcast" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write([]byte(broadcastHTML))
			return
		}
		if r.URL.Path == "/app.css" {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(appCSS)
			return
		}
		if r.URL.Path == "/app-live.css" {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(appLiveCSS)
			return
		}
		if r.URL.Path == "/app-history.css" {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(appHistoryCSS)
			return
		}
		if r.URL.Path == "/app-maps.css" {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(appMapsCSS)
			return
		}
		if r.URL.Path == "/app.js" {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(appJS)
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("/api/pause", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost {
			var body struct {
				On bool `json:"on"`
			}
			_ = json.NewDecoder(io.LimitReader(r.Body, 1<<12)).Decode(&body)
			setLookupPause(body.On)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"on": lookupsPaused()})
	})
	mux.HandleFunc("/api/prefs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost {
			var bag map[string]string
			if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&bag) == nil {
				_ = writePrefs(bag)
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		_ = json.NewEncoder(w).Encode(readPrefs())
	})
	mux.HandleFunc("/api/rank", func(w http.ResponseWriter, r *http.Request) {
		id := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("id")))
		if !validPUUID(id) {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		openBrowser("https://vtl.lol/id/" + id)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/player", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(app.profile(ctx, r.URL.Query().Get("id")))
	})
	mux.HandleFunc("/api/replay", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		row := app.replayMatch(ctx, r.URL.Query().Get("id"))
		if row == nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "edges": row.Edges, "players": row.Players, "rounds": row.Rounds, "buys": row.Buys})
	})
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		queue := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("queue")))
		rows := app.matches(ctx, queue)
		_ = json.NewEncoder(w).Encode(map[string]any{"matches": rows})
	})
	mux.HandleFunc("/api/loadout", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost {
			var req struct {
				Weapon   string `json:"weapon"`
				Skin     string `json:"skin"`
				Level    string `json:"level"`
				Chroma   string `json:"chroma"`
				Buddy    string `json:"buddy"`
				Instance string `json:"instance"`
				Kind     string `json:"kind"`
				Slot     int    `json:"slot"`
				Type     string `json:"type"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Kind == "spray" || req.Kind == "flex" {
				_ = json.NewEncoder(w).Encode(app.equipFlair(ctx, req.Slot, req.Skin, req.Kind, req.Type))
				return
			}
			_ = json.NewEncoder(w).Encode(app.equip(ctx, req.Weapon, req.Skin, req.Level, req.Chroma, req.Buddy, req.Instance))
			return
		}
		if weapon := strings.TrimSpace(r.URL.Query().Get("weapon")); weapon == "flair" {
			_ = json.NewEncoder(w).Encode(app.flairOptions(ctx))
			return
		}
		if weapon := strings.TrimSpace(r.URL.Query().Get("weapon")); weapon != "" {
			_ = json.NewEncoder(w).Encode(app.wardrobe(ctx, weapon))
			return
		}
		_ = json.NewEncoder(w).Encode(app.loadout(ctx))
	})
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(app.lookupPlayer(ctx, r.URL.Query().Get("q")))
	})
	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{"agents": app.playableAgents(ctx)})
	})
	mux.HandleFunc("/api/gear", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		app.mu.Lock()
		sess := app.session
		app.mu.Unlock()
		rows, msg := app.masteryFor(ctx, sess, r.URL.Query().Get("id"))
		if rows == nil {
			rows = []masteryAgent{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"agents": rows, "error": msg})
	})
	mux.HandleFunc("/api/card", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(app.cardProfile(ctx))
	})
	mux.HandleFunc("/api/rr", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(app.rankSeries(ctx))
	})
	mux.HandleFunc("/api/news", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		notes, errText := app.bulletin(ctx)
		if notes == nil {
			notes = []bulletin{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"notes": notes, "error": errText})
	})
	mux.HandleFunc("/api/patches", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		notes, errText := app.patchNotes(ctx)
		if notes == nil {
			notes = []patchNote{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"notes": notes, "error": errText})
	})
	mux.HandleFunc("/api/patch", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		title, body, errText := app.patchBody(ctx, r.URL.Query().Get("path"))
		_ = json.NewEncoder(w).Encode(map[string]any{"title": title, "body": body, "error": errText})
	})
	mux.HandleFunc("/api/premier", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(app.premierTeam(ctx))
	})
	mux.HandleFunc("/api/acts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		rows := loadPastAct()
		if rows == nil {
			rows = []recentMatch{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"matches": rows})
	})
	mux.HandleFunc("/api/esports", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(app.esportsView(ctx))
	})
	mux.HandleFunc("/api/esports/items", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		items := app.bundleContents(ctx, r.URL.Query().Get("name"))
		if items == nil {
			items = []storeItem{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	})
	mux.HandleFunc("/api/buy", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			Offer    string `json:"offer"`
			Price    int    `json:"price"`
			Currency string `json:"currency"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		msg := app.buy(ctx, req.Offer, req.Price, req.Currency)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": msg == "", "error": msg})
	})
	mux.HandleFunc("/api/kc", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		items := app.kcCatalog(ctx)
		if items == nil {
			items = []storeItem{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	})
	mux.HandleFunc("/api/pass", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(app.battlepass(ctx))
	})
	mux.HandleFunc("/api/game", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(app.gameSettings(ctx))
	})
	mux.HandleFunc("/api/social", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		name, tag, _ := splitRiotID(q)
		switch r.Method {
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(app.addWatch(name, tag))
		case http.MethodDelete:
			_ = json.NewEncoder(w).Encode(app.removeWatch(name, tag))
		default:
			_ = json.NewEncoder(w).Encode(app.socialView())
		}
	})
	mux.HandleFunc("/api/crosshairs", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		rows, note := app.gameCrosshairs(ctx)
		_ = json.NewEncoder(w).Encode(map[string]any{"crosshairs": rows, "error": note})
	})
	mux.HandleFunc("/api/notify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "no", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Title string `json:"title"`
			Text  string `json:"text"`
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 2000)).Decode(&body)
		title := strings.TrimSpace(body.Title)
		text := strings.TrimSpace(body.Text)
		if title == "" {
			title = "Valvet"
		}
		if text == "" {
			text = "A wish is in the store"
		}
		desktopNotify(title, text)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/open", func(w http.ResponseWriter, r *http.Request) {
		parsed, err := url.Parse(r.URL.Query().Get("url"))
		host := ""
		if err == nil {
			host = parsed.Hostname()
		}
		if err != nil || parsed.Scheme != "https" || (host != "vtl.lol" && host != "tracker.gg" && host != "www.tracker.gg" && host != "x.com" && host != "twitter.com" && host != "www.x.com" && host != "www.twitter.com" && host != "playvalorant.com" && host != "www.playvalorant.com" && host != "mail.google.com") {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}
		openBrowser(parsed.String())
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/folder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		which := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("which")))
		dir := velvetBase()
		if which == "rounds" || which == "videos" {
			if videos, err := roundVideoDir(); err == nil {
				dir = videos
			}
		}
		revealFolder(dir)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/storage", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		var view storageView
		if r.Method == http.MethodPost {
			view = clearOldStorage()
		} else {
			view = storageNow()
		}
		_ = json.NewEncoder(w).Encode(view)
	})
	mux.HandleFunc("/api/feedback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		var body struct {
			Text string `json:"text"`
			Riot string `json:"riot"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&body) != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "Could not read that."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		riot := app.signedInRiot()
		if riot == "" {
			riot = cleanRiotID(body.Riot)
		}
		if err := postFeedback(ctx, body.Text, riot); err != nil {
			msg := "Couldn't send. Try again in a moment."
			switch err.Error() {
			case "short":
				msg = "Write a bit more first."
			case "wait":
				msg = "Wait a few seconds, then send again."
			case "busy":
				msg = "The report service is busy. Try again in a minute."
			case "activate":
				msg = "Not delivered yet. The inbox still has to confirm the first report. Try again after that link is opened."
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/discord", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		var body struct {
			URL      string `json:"url"`
			Title    string `json:"title"`
			Text     string `json:"text"`
			Status   bool   `json:"status"`
			Message  string `json:"message"`
			ClientID string `json:"clientId"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "Bad request."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		id, err := sendDiscord(ctx, body.URL, body.Title, body.Text, body.Message, body.Status)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "message": id})
	})
	mux.HandleFunc("/api/presence", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		var body struct {
			ClientID string `json:"clientId"`
			Details  string `json:"details"`
			State    string `json:"state"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "Bad request."})
			return
		}
		if err := setPresence(body.ClientID, body.Details, body.State); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/update", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		offer, err := fetchUpdate(ctx, r.URL.Query().Get("fresh") == "1")
		if r.Method != http.MethodPost {
			if err != nil && !offer.Update {
				appLog("update failed")
				offer.Error = "Couldn't check for an update. Quit Valvet and open it again."
				offer.Update = false
			}
			_ = json.NewEncoder(w).Encode(offer)
			return
		}
		if err != nil || !offer.Update {
			if err != nil {
				appLog("update failed")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "Valvet is already up to date."})
			return
		}
		if err := installUpdate(offer.URL); err != nil {
			appLog("update failed")
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		go func() {
			time.Sleep(400 * time.Millisecond)
			os.Exit(0)
		}()
	})
	mux.HandleFunc("/api/launch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost {
			var body struct {
				Mode string `json:"mode"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]string{"mode": setLaunchMode(body.Mode)})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"mode": launchMode()})
	})
	mux.HandleFunc("/api/owned", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{"names": app.ownedSkinNames(ctx)})
	})
	mux.HandleFunc("/api/room", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodPost {
			var person roomPerson
			if json.NewDecoder(io.LimitReader(r.Body, 8000)).Decode(&person) != nil {
				http.Error(w, "bad", http.StatusBadRequest)
				return
			}
			person = cleanPerson(person)
			okRoom := false
			if topic, ok := roomTopic(person.Room); ok {
				okRoom = publishRoom(ctx, topic, person) == nil
			}
			okLFG := false
			if person.Looking || person.Board {
				okLFG = publishRoom(ctx, ntfyTopic("lfg"), person) == nil
			}
			if !okRoom && !okLFG {
				http.Error(w, "bad room", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": okRoom || okLFG})
			return
		}
		code := r.URL.Query().Get("code")
		if code == "lfg" {
			_ = json.NewEncoder(w).Encode(map[string]any{"people": readBoard(ctx), "rates": readRates(ctx)})
			return
		}
		topic, ok := roomTopic(code)
		if !ok {
			http.Error(w, "bad room", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"people": readRoom(ctx, topic)})
	})
	mux.HandleFunc("/api/rate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		var rate boardRate
		if json.NewDecoder(io.LimitReader(r.Body, 2000)).Decode(&rate) != nil {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		if err := publishRate(ctx, rate); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/picture", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Name  string `json:"name"`
			Data  string `json:"data"`
			Video bool   `json:"video"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 48<<20)).Decode(&body) != nil || body.Data == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "Nothing to save."})
			return
		}
		path, err := saveUserFile(body.Name, body.Data, body.Video)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "path": path})
	})
	mux.HandleFunc("/api/lights", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Kind string `json:"kind"`
			Open string `json:"open"`
			WLED string `json:"wled"`
			R    int    `json:"r"`
			G    int    `json:"g"`
			B    int    `json:"b"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body) != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false})
			return
		}
		note := pushLights(body.Kind, body.Open, body.WLED, body.R, body.G, body.B)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "note": note})
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		app.setLiveOnly(r.URL.Query().Get("live") == "1")
		app.noteIdle(r.URL.Query().Get("asleep") == "1", r.URL.Query().Get("store") == "0", r.URL.Query().Get("book") == "0")
		if r.URL.Query().Get("asleep") != "1" && !app.busyPhase() {
			maybeArchiveReplays()
		}
		snap := app.state(ctx)
		if tab := takeTrayTab(); tab != "" {
			snap.Tray = tab
		}
		snap.PauseLookups = lookupsPaused()
		_ = json.NewEncoder(w).Encode(snap)
	})

	port := 47823
	var listener net.Listener
	var err error
	for i := 0; i < 8; i++ {
		listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port+i))
		if err == nil {
			port += i
			break
		}
	}
	if listener == nil {
		maybeWaitForRiot()
		openApp("http://127.0.0.1:47823/")
		return
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	go func() {
		server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
		_ = server.Serve(listener)
	}()
	maybeWaitForRiot()
	openApp(url)
}
