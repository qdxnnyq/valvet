package main

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type patchNote struct {
	Title string `json:"title"`
	Date  string `json:"date"`
	Blurb string `json:"blurb"`
	Image string `json:"image,omitempty"`
	Path  string `json:"path"`
	At    string `json:"at,omitempty"`
}

var (
	patchMu    sync.Mutex
	patchCache []patchNote
	patchWhen  time.Time
)

func esportName(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "vct") || strings.Contains(n, "champions") || strings.Contains(n, "esport") || strings.Contains(n, "team capsule") || strings.Contains(n, "masters")
}

func patchPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexAny(raw, "?#"); i >= 0 {
		raw = raw[:i]
	}
	if !strings.HasPrefix(raw, "/en-us/news/") || strings.Contains(raw, "..") || strings.Contains(raw, "://") || strings.Contains(raw, "\\") {
		return ""
	}
	return raw
}

func (c *riotClient) publicGet(ctx context.Context, raw string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Valvet")
	res, err := c.remote.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, statusErr(res.StatusCode)
	}
	return body, nil
}

type statusErr int

func (e statusErr) Error() string { return "page " + itoa(int(e)) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func nextJSON(raw []byte) map[string]any {
	const mark = `<script id="__NEXT_DATA__"`
	i := bytes.Index(raw, []byte(mark))
	if i < 0 {
		return nil
	}
	rest := raw[i:]
	open := bytes.IndexByte(rest, '>')
	close := bytes.Index(rest, []byte("</script>"))
	if open < 0 || close < 0 || close <= open {
		return nil
	}
	var data map[string]any
	if json.Unmarshal(rest[open+1:close], &data) != nil {
		return nil
	}
	return data
}

func pageBlades(data map[string]any) []any {
	props := asMap(data["props"])
	pageProps := asMap(props["pageProps"])
	page := asMap(pageProps["page"])
	return asArr(page["blades"])
}

func (c *riotClient) patchNotes(ctx context.Context) ([]patchNote, string) {
	patchMu.Lock()
	if len(patchCache) > 0 && time.Since(patchWhen) < 6*time.Hour {
		out := append([]patchNote(nil), patchCache...)
		patchMu.Unlock()
		return out, ""
	}
	patchMu.Unlock()
	var disk []patchNote
	if age, ok := readCache("patches", &disk); ok && len(disk) > 0 {
		patchMu.Lock()
		patchCache = disk
		patchWhen = time.Now().Add(-age)
		out := append([]patchNote(nil), disk...)
		patchMu.Unlock()
		if age > 6*time.Hour {
			cacheOnce("patches", func() { c.refreshPatches() })
		}
		return out, ""
	}
	if notes := c.refreshPatches(); len(notes) > 0 {
		return notes, ""
	}
	return nil, "Couldn't load patch notes."
}

func (c *riotClient) refreshPatches() []patchNote {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	raw, err := c.publicGet(ctx, "https://playvalorant.com/en-us/news/tags/patch-notes/")
	if err != nil {
		return nil
	}
	notes := parsePatchIndex(nextJSON(raw))
	if len(notes) == 0 {
		return nil
	}
	patchMu.Lock()
	patchCache = notes
	patchWhen = time.Now()
	patchMu.Unlock()
	writeCache("patches", notes)
	return notes
}

func parsePatchIndex(data map[string]any) []patchNote {
	out := []patchNote{}
	for _, blade := range pageBlades(data) {
		row := asMap(blade)
		if str(row["type"]) != "articleCardGrid" {
			continue
		}
		for _, item := range asArr(row["items"]) {
			card := asMap(item)
			action := asMap(card["action"])
			payload := asMap(action["payload"])
			path := patchPath(str(payload["url"]))
			title := str(card["title"])
			if path == "" || title == "" {
				continue
			}
			media := asMap(card["media"])
			desc := asMap(card["description"])
			out = append(out, patchNote{
				Title: title,
				Date:  patchDate(str(card["publishedAt"])),
				Blurb: htmlText(str(desc["body"])),
				Image: str(media["url"]),
				Path:  path,
				At:    str(card["publishedAt"]),
			})
		}
	}
	return out
}

func newsKind(text string) (string, bool) {
	n := strings.ToLower(text)
	switch {
	case strings.Contains(n, "bundle"):
		return "Bundle", true
	case strings.Contains(n, "patch"):
		return "Patch", false
	case strings.Contains(n, "agent"):
		return "Agent", false
	default:
		return "News", false
	}
}

type bulletin struct {
	Title  string `json:"title"`
	Date   string `json:"date"`
	Blurb  string `json:"blurb"`
	Image  string `json:"image,omitempty"`
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Bundle bool   `json:"bundle,omitempty"`
	At     string `json:"at,omitempty"`
}

var (
	newsMu    sync.Mutex
	newsCache []bulletin
	newsWhen  time.Time
)

func (c *riotClient) bulletin(ctx context.Context) ([]bulletin, string) {
	newsMu.Lock()
	if len(newsCache) > 0 && time.Since(newsWhen) < 30*time.Minute {
		out := append([]bulletin(nil), newsCache...)
		newsMu.Unlock()
		return out, ""
	}
	newsMu.Unlock()
	seen := map[string]bool{}
	out := []bulletin{}
	for _, page := range []string{
		"https://playvalorant.com/en-us/news/",
		"https://playvalorant.com/en-us/news/game-updates/",
		"https://playvalorant.com/en-us/news/announcements/",
	} {
		raw, err := c.publicGet(ctx, page)
		if err != nil {
			continue
		}
		for _, note := range parsePatchIndex(nextJSON(raw)) {
			if note.Path == "" || seen[note.Path] || strings.Contains(note.Path, "/community/") {
				continue
			}
			seen[note.Path] = true
			kind, bundle := newsKind(note.Title)
			out = append(out, bulletin{
				Title:  note.Title,
				Date:   note.Date,
				Blurb:  note.Blurb,
				Image:  note.Image,
				Path:   note.Path,
				Kind:   kind,
				Bundle: bundle,
				At:     note.At,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At > out[j].At })
	fresh := make([]bulletin, 0, len(out))
	cutoff := time.Now().Add(-75 * 24 * time.Hour)
	for _, note := range out {
		t, err := time.Parse(time.RFC3339Nano, note.At)
		if err != nil || t.Before(cutoff) {
			continue
		}
		fresh = append(fresh, note)
	}
	if len(fresh) > 0 {
		out = fresh
	}
	if len(out) == 0 {
		return nil, "Couldn't load Valorant news."
	}
	if len(out) > 16 {
		out = out[:16]
	}
	newsMu.Lock()
	newsCache = out
	newsWhen = time.Now()
	newsMu.Unlock()
	return append([]bulletin(nil), out...), ""
}

func patchDate(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return ""
	}
	return t.Format("Jan 2, 2006")
}

func (c *riotClient) patchBody(ctx context.Context, path string) (string, string, string) {
	path = patchPath(path)
	if path == "" {
		return "", "", "That note isn't on the list."
	}
	raw, err := c.publicGet(ctx, "https://playvalorant.com"+path)
	if err != nil {
		return "", "", "Couldn't open that patch."
	}
	title := ""
	var parts []string
	for _, blade := range pageBlades(nextJSON(raw)) {
		row := asMap(blade)
		switch str(row["type"]) {
		case "articleMasthead":
			if title == "" {
				title = str(asMap(row["header"])["title"])
				if title == "" {
					title = str(row["title"])
				}
			}
		case "articleRichText":
			text := htmlText(str(asMap(row["richText"])["body"]))
			if strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
	}
	if len(parts) == 0 {
		return title, "", "That patch didn't include the notes."
	}
	return title, strings.Join(parts, "\n\n"), ""
}

var (
	tagRE = regexp.MustCompile(`<[^>]+>`)
	gapRE = regexp.MustCompile(`\n{3,}`)
)

func htmlText(raw string) string {
	raw = strings.ReplaceAll(raw, "\r", "")
	replacer := strings.NewReplacer(
		"<br>", "\n", "<br/>", "\n", "<br />", "\n",
		"</p>", "\n", "</P>", "\n",
		"<h1>", "\n§ ", "<h2>", "\n§ ", "<h3>", "\n§ ", "<h4>", "\n§ ",
		"</h1>", "\n", "</h2>", "\n", "</h3>", "\n", "</h4>", "\n",
		"</li>", "\n", "</LI>", "\n",
		"<li>", "• ", "<li ", "• <",
	)
	raw = replacer.Replace(raw)
	raw = tagRE.ReplaceAllString(raw, "")
	raw = html.UnescapeString(raw)
	lines := strings.Split(raw, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(gapRE.ReplaceAllString(strings.Join(kept, "\n"), "\n\n"))
}

func (c *riotClient) esportsOffers(body map[string]any) []bundleView {
	out := []bundleView{}
	seen := map[string]bool{}
	for _, plugin := range asArr(pick(body, "PluginStores", "pluginStores")) {
		row := asMap(plugin)
		offers := asArr(pick(row, "Offers", "offers"))
		for _, raw := range offers {
			offer := asMap(raw)
			id := strings.ToLower(str(pick(offer, "OfferID", "offerId")))
			if id != "" && seen[id] {
				continue
			}
			if id != "" {
				seen[id] = true
			}
			price := vpCost(asMap(pick(offer, "Cost", "cost")))
			was := 0
			if discounted := vpCost(asMap(pick(offer, "DiscountedCost", "discountedCost"))); discounted > 0 && (price == 0 || discounted < price) {
				was = price
				price = discounted
			}
			items := []storeItem{}
			name, icon := "", ""
			for _, reward := range asArr(pick(offer, "Rewards", "rewards")) {
				piece := asMap(reward)
				itemID := strings.ToLower(str(pick(piece, "ItemID", "itemId")))
				n, ic := c.itemLook(itemID)
				if n == "" {
					continue
				}
				if name == "" {
					name, icon = n, ic
				}
				items = append(items, storeItem{Name: n, Icon: ic})
			}
			if name == "" {
				name = "Team capsule"
			}
			left := resetLabel(num(pick(offer, "DurationRemainingInSeconds", "durationRemainingInSeconds")))
			if left == "" {
				left = untilLabel(str(pick(offer, "EndDate", "endDate")))
			}
			if left == "" {
				left = untilLabel(str(pick(row, "StorefrontExpiry", "storefrontExpiry")))
			}
			out = append(out, bundleView{Name: name, Icon: icon, Price: price, Was: was, Left: left, Items: items, Offer: id})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (c *riotClient) esportsCatalog(ctx context.Context) []bundleView {
	c.ensureBundles(ctx)
	c.mu.Lock()
	art := c.bundleArt
	notes := c.bundleNote
	c.mu.Unlock()
	out := []bundleView{}
	for id, info := range art {
		if info.Name == "" || !esportName(info.Name) {
			continue
		}
		icon := info.Icon
		if icon == "" {
			icon = "https://media.valorant-api.com/bundles/" + id + "/displayicon.png"
		}
		out = append(out, bundleView{Name: info.Name, Icon: icon, Desc: notes[id]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (c *riotClient) esportsView(ctx context.Context) map[string]any {
	if page, age, ok := loadEsportPage(); ok {
		if age > 3*time.Minute {
			cacheOnce("esports", func() { c.refreshEsports() })
		}
		return esportMap(page)
	}
	return esportMap(c.refreshEsports())
}

type esportPage struct {
	Offers  []bundleView `json:"offers"`
	Catalog []bundleView `json:"catalog"`
	Events  []sportEvent `json:"events"`
	Error   string       `json:"error,omitempty"`
}

var (
	esportPageMu  sync.Mutex
	esportPageMem esportPage
	esportPageAt  time.Time
)

func esportMap(page esportPage) map[string]any {
	offers, catalog, events := page.Offers, page.Catalog, page.Events
	if offers == nil {
		offers = []bundleView{}
	}
	if catalog == nil {
		catalog = []bundleView{}
	}
	if events == nil {
		events = []sportEvent{}
	}
	return map[string]any{"offers": offers, "catalog": catalog, "events": events, "error": page.Error}
}

func loadEsportPage() (esportPage, time.Duration, bool) {
	esportPageMu.Lock()
	if !esportPageAt.IsZero() && esportPageFilled(esportPageMem) {
		page := esportPageMem
		age := time.Since(esportPageAt)
		esportPageMu.Unlock()
		return page, age, true
	}
	esportPageMu.Unlock()
	var disk esportPage
	age, ok := readCache("esports", &disk)
	if !ok || !esportPageFilled(disk) {
		return esportPage{}, 0, false
	}
	esportPageMu.Lock()
	esportPageMem = disk
	esportPageAt = time.Now().Add(-age)
	esportPageMu.Unlock()
	return disk, age, true
}

func esportPageFilled(page esportPage) bool {
	return len(page.Catalog)+len(page.Offers)+len(page.Events) > 0
}

func (c *riotClient) refreshEsports() esportPage {
	ctx, cancel := context.WithTimeout(context.Background(), 22*time.Second)
	defer cancel()
	c.mu.Lock()
	sess := c.session
	offers := append([]bundleView(nil), c.shop.Esports...)
	shopAge := time.Since(c.shopAt)
	c.mu.Unlock()
	if sess.puuid != "" && (len(offers) == 0 || shopAge > 10*time.Minute) {
		shop := c.dailyStore(ctx, sess)
		if len(shop.Esports) > 0 {
			offers = shop.Esports
		}
	}
	catalog := c.esportsCatalog(ctx)
	events := c.esportsEvents(ctx)
	page := esportPage{Offers: offers, Catalog: catalog, Events: events}
	if !esportPageFilled(page) {
		return page
	}
	esportPageMu.Lock()
	esportPageMem = page
	esportPageAt = time.Now()
	esportPageMu.Unlock()
	writeCache("esports", page)
	return page
}

type sportEvent struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Dates  string `json:"dates"`
	Prize  string `json:"prize,omitempty"`
	Image  string `json:"image,omitempty"`
	URL    string `json:"url,omitempty"`
}

var (
	eventMu    sync.Mutex
	eventCache []sportEvent
	eventWhen  time.Time
	themeMu    sync.Mutex
	themeCache []storeItem
	themeWhen  time.Time
)

func (c *riotClient) esportsEvents(ctx context.Context) []sportEvent {
	eventMu.Lock()
	if len(eventCache) > 0 && eventCache[0].URL != "" && time.Since(eventWhen) < 6*time.Hour {
		out := sortEvents(append([]sportEvent(nil), eventCache...))
		eventMu.Unlock()
		return out
	}
	eventMu.Unlock()
	var disk []sportEvent
	if age, ok := readCache("events", &disk); ok && len(disk) > 0 && disk[0].URL != "" {
		eventMu.Lock()
		eventCache = disk
		eventWhen = time.Now().Add(-age)
		out := sortEvents(append([]sportEvent(nil), disk...))
		eventMu.Unlock()
		if age > 6*time.Hour {
			cacheOnce("events", func() { c.refreshEvents() })
		}
		return out
	}
	if out := c.refreshEvents(); len(out) > 0 {
		return out
	}
	return []sportEvent{}
}

func (c *riotClient) refreshEvents() []sportEvent {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	raw, err := c.publicGet(ctx, "https://vlr.orlandomm.net/api/v1/events")
	if err != nil {
		return nil
	}
	var body map[string]any
	if json.Unmarshal(raw, &body) != nil {
		return nil
	}
	out := []sportEvent{}
	for _, item := range asArr(body["data"]) {
		row := asMap(item)
		name := str(row["name"])
		if name == "" {
			continue
		}
		id := str(row["id"])
		url := ""
		if id != "" {
			url = "https://www.vlr.gg/event/" + id
		}
		out = append(out, sportEvent{
			Name:   name,
			Status: str(row["status"]),
			Dates:  str(row["dates"]),
			Prize:  str(row["prizepool"]),
			Image:  str(row["img"]),
			URL:    url,
		})
	}
	out = sortEvents(out)
	if len(out) == 0 {
		return nil
	}
	eventMu.Lock()
	eventCache = out
	eventWhen = time.Now()
	eventMu.Unlock()
	writeCache("events", out)
	return out
}

func prizeAmount(raw string) int {
	n := 0
	seen := false
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			seen = true
			n = n*10 + int(r-'0')
			continue
		}
		if r == ',' || r == ' ' || r == '$' || r == '€' || r == '£' {
			continue
		}
		if seen {
			break
		}
	}
	return n
}

func eventLive(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return false
	}
	if strings.Contains(s, "complet") || strings.Contains(s, "finish") || strings.Contains(s, "cancel") || strings.Contains(s, "past") {
		return false
	}
	switch s {
	case "ongoing", "live", "active", "upcoming", "soon", "not started":
		return true
	}
	return strings.Contains(s, "ongo") || strings.Contains(s, "live") || strings.Contains(s, "upcom") || strings.Contains(s, "soon") || strings.Contains(s, "progress") || strings.Contains(s, "running")
}

func eventPhase(status string) int {
	s := strings.ToLower(status)
	if strings.Contains(s, "ongo") || strings.Contains(s, "live") || strings.Contains(s, "active") || strings.Contains(s, "progress") || strings.Contains(s, "running") {
		return 0
	}
	return 1
}

func sortEvents(in []sportEvent) []sportEvent {
	if len(in) == 0 {
		return in
	}
	out := []sportEvent{}
	for _, event := range in {
		if eventLive(event.Status) {
			out = append(out, event)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		pi, pj := eventPhase(out[i].Status), eventPhase(out[j].Status)
		if pi != pj {
			return pi < pj
		}
		ai, aj := prizeAmount(out[i].Prize), prizeAmount(out[j].Prize)
		if ai != aj {
			return ai > aj
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > 16 {
		out = out[:16]
	}
	return out
}

func themeTokens(name string) []string {
	skip := map[string]bool{"vct": true, "capsule": true, "bundle": true, "team": true, "the": true, "and": true, "esports": true, "champions": true, "collection": true, "classic": true}
	var out []string
	for _, word := range strings.Fields(strings.ToLower(name)) {
		word = strings.Trim(word, ".,:-")
		if len(word) < 4 || skip[word] {
			continue
		}
		if _, err := time.Parse("2006", word); err == nil {
			continue
		}
		out = append(out, word)
	}
	return out
}

func (c *riotClient) themeItems(ctx context.Context) []storeItem {
	themeMu.Lock()
	if len(themeCache) > 0 && time.Since(themeWhen) < 12*time.Hour {
		out := append([]storeItem(nil), themeCache...)
		themeMu.Unlock()
		return out
	}
	themeMu.Unlock()
	out := []storeItem{}
	for _, pair := range []struct{ url, kind string }{
		{"https://valorant-api.com/v1/weapons/skins", "Skin"},
		{"https://valorant-api.com/v1/buddies", "Buddy"},
		{"https://valorant-api.com/v1/sprays", "Spray"},
	} {
		raw, err := c.publicGet(ctx, pair.url)
		if err != nil {
			continue
		}
		var body map[string]any
		if json.Unmarshal(raw, &body) != nil {
			continue
		}
		for _, item := range asArr(body["data"]) {
			row := asMap(item)
			name := str(pick(row, "displayName"))
			if name == "" || strings.EqualFold(name, "Standard") {
				continue
			}
			icon := str(pick(row, "displayIcon"))
			if icon == "" {
				icon = str(pick(row, "fullTransparentIcon"))
			}
			out = append(out, storeItem{Name: name, Icon: icon, Currency: pair.kind})
		}
	}
	if len(out) == 0 {
		return []storeItem{}
	}
	themeMu.Lock()
	themeCache = out
	themeWhen = time.Now()
	themeMu.Unlock()
	return out
}

func (c *riotClient) bundleContents(ctx context.Context, name string) []storeItem {
	tokens := themeTokens(name)
	if len(tokens) == 0 {
		return []storeItem{}
	}
	out := []storeItem{}
	for _, item := range c.themeItems(ctx) {
		n := strings.ToLower(item.Name)
		ok := false
		for _, token := range tokens {
			if strings.Contains(n, token) {
				ok = true
				break
			}
		}
		if !ok {
			continue
		}
		out = append(out, item)
		if len(out) >= 24 {
			break
		}
	}
	return out
}
