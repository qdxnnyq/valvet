      async function tickRR() {
        if (document.body.classList.contains("minimal")) return;
        try {
          const data = await (await fetch("/api/rr", { cache: "no-store" })).json();
          rrPoints = data.points || [];
          weekRR = Number(data.week) || 0;
          const today = document.getElementById("today");
          if (today) today.dataset.key = "";
          paintToday(history);
          recentKey = "";
          if (tab === "history") drawRecent(modeQueue ? modeMatches || [] : history);
          paintRR(document.getElementById("rr-slot"));
        } catch {}
      }
      function stateQuery() {
        const bits = [];
        if (document.body.classList.contains("minimal")) bits.push("live=1");
        if (document.hidden) bits.push("asleep=1");
        const hidden = hiddenTabs();
        if (["store", "bundle", "esports"].every((name) => hidden.indexOf(name) >= 0)) bits.push("store=0");
        if (["history", "session", "records", "maps", "deaths", "clutches", "queues", "profile", "team"].every((name) => hidden.indexOf(name) >= 0)) bits.push("book=0");
        return bits.length ? "?" + bits.join("&") : "";
      }
      async function tick() {
        try {
          const res = await fetch("/api/state" + stateQuery(), { cache: "no-store" });
          draw(await res.json());
        } catch {
          draw({ ok: false, error: "Close this window and open Valvet again.", recent: history });
        }
      }
      let tickGen = 0;
      function armTick() {
        const gen = ++tickGen;
        const phase = (latestState && latestState.phase) || "";
        const active = phase === "In game" || phase === "Agent select" || phase === "Loading lobby" || phase === "In queue" || phase === "In lobby";
        const ranksDue = ((latestState && latestState.players) || []).some((player) => player && player.puuid && (!player.rank || (player.rank === "Unranked" && !(Number(player.tier) >= 3))));
        const wait = document.hidden ? 60000 : active ? (ranksDue ? 2000 : 5000) : 16000;
        setTimeout(() => {
          if (gen !== tickGen) return;
          tick().finally(() => { if (gen === tickGen) armTick(); });
        }, wait);
      }
      document.addEventListener("visibilitychange", () => {
        tickGen += 1;
        if (document.hidden) armTick();
        else tick().finally(armTick);
      });
      async function tickMastery() {
        if (document.body.classList.contains("minimal")) return;
        try {
          const data = await (await fetch("/api/card", { cache: "no-store" })).json();
          if (data.level) knownLevel = data.level;
          if (data.name) paintIdentity(data.name + (data.tag ? "#" + data.tag : ""), data.level || knownLevel);
        } catch {}
      }
      document.getElementById("credit").onclick = () => {
        fetch("/api/open?url=" + encodeURIComponent("https://x.com/qdxnnyq"), { method: "POST" }).catch(() => {});
      };
      const youBox = document.getElementById("you");
      if (youBox) {
        youBox.title = "Your profile";
        youBox.onclick = () => setTab("profile");
      }
      async function installUpdate(button) {
        rememberPrefs();
        if (!button || button.dataset.busy === "1") return;
        button.dataset.busy = "1";
        button.disabled = true;
        const previous = button.textContent;
        button.textContent = "Updating…";
        const bar = document.getElementById("update-bar");
        if (bar) bar.dataset.busy = "1";
        try {
          const result = await (await fetch("/api/update", { method: "POST" })).json();
          if (!result.ok) {
            button.dataset.busy = "";
            button.disabled = false;
            button.textContent = result.error || previous;
            if (bar) bar.dataset.busy = "";
          }
        } catch {
          button.dataset.busy = "";
          button.disabled = false;
          button.textContent = "Update failed";
          if (bar) bar.dataset.busy = "";
        }
      }
      function semverNewer(remote, local) {
        const parse = (value) => String(value || "").replace(/^v/i, "").split(".").map((part) => parseInt(part, 10));
        const a = parse(remote);
        const b = parse(local);
        if (a.length < 3 || b.length < 3 || a.some((n) => Number.isNaN(n)) || b.some((n) => Number.isNaN(n))) return false;
        for (let i = 0; i < 3; i++) if (a[i] !== b[i]) return a[i] > b[i];
        return false;
      }
      let updateTicket = 0;
      function grabJSON(url, ms) {
        const ctrl = new AbortController();
        const timer = setTimeout(() => ctrl.abort(), ms);
        return fetch(url, { cache: "no-store", signal: ctrl.signal, headers: { Accept: "application/json" } })
          .then((res) => (res.ok ? res.json() : null))
          .catch(() => null)
          .finally(() => clearTimeout(timer));
      }
      function releaseName(rel) {
        const hit = String((rel && rel.name) || "").match(/\d+\.\d+\.\d+/);
        return hit ? hit[0] : "";
      }
      function hasUpdate(data) {
        if (!data) return false;
        const name = String(data.name || "");
        const label = String(data.label || NOTES_VER || "");
        if (name && label && semverNewer(name, label)) return true;
        if (name && NOTES_VER && semverNewer(name, NOTES_VER)) return true;
        const remote = Number(data.version) || 0;
        const local = Number(data.current) || 0;
        return remote > 0 && local > 0 && remote > local;
      }
      function mergeOffer(base, extra) {
        const out = Object.assign({ label: NOTES_VER, update: false }, base || {});
        const nextName = String((extra && extra.name) || "");
        const prevName = String(out.name || "");
        const nextWins = nextName && (!prevName || nextName === prevName || semverNewer(nextName, prevName));
        if (nextWins) {
          out.name = nextName;
          if (extra && extra.notes) out.notes = extra.notes;
        }
        out.version = Math.max(Number(out.version) || 0, Number(extra && extra.version) || 0);
        if (extra && Number(extra.current) > 0) out.current = Number(extra.current);
        if (!out.label) out.label = NOTES_VER;
        out.update = hasUpdate(out);
        return out;
      }
      function paintUpdate(data) {
        const button = document.getElementById("update");
        const bar = document.getElementById("update-bar");
        const chip = document.getElementById("build-chip");
        const line = document.getElementById("build-line");
        const build = (data && data.label) || NOTES_VER;
        const newer = hasUpdate(data);
        if (chip && chip.dataset.busy !== "1") chip.textContent = build;
        if (chip) {
          chip.classList.toggle("hot", newer);
          chip.title = newer ? "Update to " + ((data && data.name) || "the newest build") : "";
          if (!newer) chip.onclick = null;
        }
        if (line) {
          line.textContent = newer
            ? "Valvet " + build + ". Update to " + ((data && data.name) || "the newest build") + " is the pink bar, the sidebar button, and this version chip."
            : "Valvet " + build + ". This is the newest build.";
        }
        if (!newer) {
          if (bar && bar.dataset.busy !== "1") bar.hidden = true;
          if (button && button.dataset.busy !== "1") button.hidden = true;
          return;
        }
        const label = "Update to " + ((data && data.name) || "the newest build");
        if (button) {
          button.hidden = false;
          button.textContent = "Update";
          button.classList.add("pop");
          button.onclick = () => installUpdate(button);
        }
        if (bar) {
          bar.hidden = false;
          const text = document.getElementById("update-bar-text");
          if (text) text.textContent = label + (data.notes ? " · " + data.notes : "");
        }
        const barGo = document.getElementById("update-go");
        if (barGo) barGo.onclick = () => installUpdate(barGo);
        if (chip) {
          chip.textContent = "Update";
          chip.classList.add("hot");
          chip.title = label;
          chip.onclick = () => installUpdate(chip);
        }
      }
      async function checkUpdate(manual) {
        const ticket = ++updateTicket;
        const ask = document.getElementById("set-update");
        const note = document.getElementById("set-update-note");
        if (manual && ask) {
          ask.disabled = true;
          ask.textContent = "Checking…";
        }
        if (manual && note) note.textContent = "Checking GitHub…";
        const stamp = Date.now();
        const [file, rel, local] = await Promise.all([
          grabJSON("https://raw.githubusercontent.com/qdxnnyq/valvet/main/version.json?t=" + stamp, 15000),
          grabJSON("https://api.github.com/repos/qdxnnyq/valvet/releases/latest", 15000),
          grabJSON("/api/update?fresh=1&t=" + stamp, 15000),
        ]);
        if (ticket !== updateTicket) return;
        const remoteOk = !!(file && (file.name || file.version)) || !!releaseName(rel);
        if (!remoteOk && (!local || local.error)) {
          if (note) note.textContent = (local && local.error) || "Couldn't check GitHub. Try again.";
          if (manual && ask) {
            ask.disabled = false;
            ask.textContent = "Check for update";
          }
          return;
        }
        let data = local && typeof local === "object" ? local : { label: NOTES_VER, update: false };
        if (file && (file.name || file.version)) data = mergeOffer(data, { name: file.name || "", version: file.version || 0, notes: file.notes || "" });
        const named = releaseName(rel);
        if (named) data = mergeOffer(data, { name: named, version: parseInt(String(rel.tag_name || "").replace(/^v/i, ""), 10) || 0 });
        data.update = hasUpdate(data);
        paintUpdate(data);
        if (note) note.textContent = data.update ? "Update to " + (data.name || "the newest build") + " is ready." : "This is the newest build.";
        if (manual && ask) {
          ask.disabled = false;
          ask.textContent = "Check for update";
        }
      }
      const askUpdate = document.getElementById("set-update");
      if (askUpdate) askUpdate.onclick = () => checkUpdate(true);
      function playHello() {
        if (localStorage.getItem("valvet-sound") === "off") return;
        const audio = new Audio("/sounds/meow");
        audio.volume = 0.1;
        audio.play().catch(() => {});
      }
      playHello();
      checkUpdate(false);
      setTimeout(() => checkUpdate(false), 20000);
      tick();
      armTick();
      loadAct(true);
      tickMastery();
      setInterval(tickMastery, 20000);
      setInterval(() => checkUpdate(false), 180000);
      setInterval(loadSocial, 20000);
      setInterval(() => { if (!tabHidden("profile") && !tabHidden("history")) tickRR(); }, 60000);
      if (!tabHidden("profile") && !tabHidden("history")) tickRR();
      setInterval(() => { statusKey = ""; }, 90000);
      setInterval(() => { if (tab === "store" || tab === "pass") paintNightClock(); }, 1000);
      const TOUR = [
        { tab: "live", mark: "tab-live", kicker: "Start", title: "This is Valvet", body: "Valvet follows the Riot Client already open on this PC. It never asks for your password. Leave the client open and the pages fill in." },
        { tab: "live", mark: "board", kicker: "Live", title: "The match", body: "Your side is on the left and the other side is on the right. Each player shows name, agent, party, and rank. The header is the average rank of that side." },
        { tab: "live", mark: "post", kicker: "After the match", title: "Result", body: "When the match ends, this card shows the result. Copy card saves a picture." },
        { tab: "friends", mark: "tab-friends", kicker: "Friends", title: "Friends", body: "Valorant, League, and the Riot Client, from the friends on this PC. An empty list stays empty." },
        { tab: "history", mark: "tab-history", kicker: "History", title: "This act", body: "History keeps the games from this act. Each side is sorted by KDA. Highlight a name and the same person lights up in the stats. The next act replaces this one." },
        { tab: "profile", mark: "tab-profile", kicker: "Profile", title: "Your numbers", body: "Win rate with friends counts Riot friends, after two games together. Weapons show headshot percent. Maps show win rate. The chart is HS, ACS, and kills. A day starts at 6:00 am. Older days stay under View more." },
        { tab: "session", mark: "tab-session", kicker: "Session", title: "Time played", body: "The chart is the last 30 days. Click a day to see the matches from that day. The clock is time played since 6:00 am." },
        { tab: "maps", mark: "tab-maps", kicker: "Maps", title: "Maps", body: "Each map is the minimap. Bigger fits the picture in the window. Esc or Close leaves it. Death spots use the same map." },
        { tab: "seen", mark: "tab-seen", kicker: "Seen", title: "Who is around", body: "Green is on the client or in a game right now, with where they are. Yellow is AFK. The rest is the last time and place. Riot Mobile is ignored." },
        { tab: "news", mark: "tab-news", kicker: "News", title: "News", body: "Recent Valorant posts. Open a card to read it on playvalorant.com." },
        { tab: "alerts", mark: "watch-block", kicker: "Trace", title: "Watch list", body: "Add a Name#Tag. If they are in your lobby or your match, the row is marked and the watch sound can play. If they finish a game you were not in, Valvet tells you." },
        { tab: "game", mark: "tab-game", kicker: "Game", title: "Your settings", body: "Graphics and the sensitivity from the Valorant menu on this PC. Type your mouse DPI for eDPI, cm/360, and the sensitivity for a target eDPI." },
        { tab: "pass", mark: "tab-pass", kicker: "Pass", title: "Battle pass", body: "Level, XP, and the time left in the act. Reward tiles stay square. A free pass, if Riot ships one, shows on this page." },
        { tab: "settings", mark: "set-switch", kicker: "Sounds", title: "Sounds", body: "Sounds are a tab in Settings. Friend in your match, watch list, win, and loss each have their own sound. Win and loss have their own volume." },
        { tab: "settings", mark: "set-minimal", kicker: "Live only", title: "Live only", body: "Live only keeps the match and leaves the other pages off. Turn it off when you want the rest." },
        { tab: "settings", mark: "swatches", kicker: "Look", title: "Colour", body: "Pick a preset, or upload a picture and Valvet takes the colours from it." },
        { tab: "settings", mark: "feedback-card", kicker: "Contact", title: "Feedback", body: "Write what happened and press Send. The same card opens Twitter." },
        { tab: "crosshairs", mark: "tab-crosshairs", kicker: "Crosshairs", title: "Crosshairs", body: "Paste a code, name it, and keep a set on this PC. Export copies that set. Valvet does not read the crosshair out of Valorant." },
      ];
      let tourAt = 0;
      function tourList() {
        return TOUR.filter((step) => step && !tabHidden(step.tab) && document.getElementById(step.mark));
      }
      function renderTour() {
        const steps = tourList();
        if (!steps.length || tourAt >= steps.length) { closeTour(); return; }
        const step = steps[tourAt];
        document.getElementById("tour-kicker").textContent = (tourAt + 1) + " / " + steps.length + " · " + step.kicker;
        document.getElementById("tour-title").textContent = step.title;
        document.getElementById("tour-body").textContent = step.body;
        document.getElementById("tour-next").textContent = tourAt === steps.length - 1 ? "Done" : "Next";
        const dots = document.getElementById("tour-dots");
        dots.replaceChildren();
        steps.forEach((_, index) => {
          const dot = document.createElement("i");
          if (index === tourAt) dot.className = "on";
          dots.append(dot);
        });
        document.querySelectorAll(".tour-mark").forEach((node) => node.classList.remove("tour-mark"));
        quietNav = true;
        setTab(step.tab);
        quietNav = false;
        const mark = document.getElementById(step.mark);
        if (mark) {
          mark.classList.add("tour-mark");
          requestAnimationFrame(() => mark.scrollIntoView({ block: "start", inline: "nearest" }));
        }
      }
      function openTour() {
        tourAt = 0;
        document.body.classList.add("touring");
        document.getElementById("tour").hidden = false;
        renderTour();
      }
      function tourDone() {
        const v = localStorage.getItem("valvet-tour");
        return v === "43" || v === "done" || v === "seen";
      }
      function markNotes() {
        localStorage.setItem("valvet-notes", NOTES_VER);
      }
      function showWhatsNew() {
        if (localStorage.getItem("valvet-notes") === NOTES_VER) return;
        const body = document.getElementById("notes-body");
        const card = document.getElementById("whats-new");
        const title = document.getElementById("notes-title");
        if (!body || !card) return;
        const note = APP_NOTES[0];
        if (title) title.textContent = (note && note.ver) || NOTES_VER;
        body.replaceChildren();
        for (const line of (note && note.lines) || []) body.append(el("p", "sub", line));
        card.hidden = false;
      }
      function closeTour() {
        const replay = tourDone();
        localStorage.setItem("valvet-tour", "seen");
        markNotes();
        document.body.classList.remove("touring");
        document.getElementById("tour").hidden = true;
        hideSampleCard();
        document.querySelectorAll(".tour-mark").forEach((node) => node.classList.remove("tour-mark"));
        quietNav = true;
        setTab(replay ? "settings" : "live");
        quietNav = false;
      }
      document.getElementById("tour-skip").onclick = closeTour;
      document.getElementById("tour-next").onclick = () => {
        if (tourAt >= tourList().length - 1) closeTour();
        else { tourAt += 1; renderTour(); }
      };
      document.getElementById("tour-again").onclick = openTour;
      function openTwitter() {
        fetch("/api/open?url=" + encodeURIComponent("https://x.com/qdxnnyq")).catch(() => {});
      }
      const thanksX = document.getElementById("thanks-x");
      const thanksGo = document.getElementById("thanks-go");
      if (thanksX) thanksX.onclick = openTwitter;
      if (thanksGo) thanksGo.onclick = () => {
        document.getElementById("thanks").hidden = true;
        if (!tourDone()) openTour();
      };
      const feedbackSend = document.getElementById("set-feedback-send");
      const feedbackNote = document.getElementById("set-feedback-note");
      const feedbackBox = document.getElementById("set-feedback");
      if (feedbackSend) feedbackSend.onclick = async () => {
        const text = feedbackBox ? feedbackBox.value.trim() : "";
        if (feedbackNote) feedbackNote.textContent = "Sending…";
        try {
          const data = await (await fetch("/api/feedback", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ text }) })).json();
          if (data && data.ok) {
            if (feedbackNote) feedbackNote.textContent = "Sent.";
            if (feedbackBox) feedbackBox.value = "";
          } else if (feedbackNote) feedbackNote.textContent = (data && data.error) || "Couldn't send. Try again.";
        } catch {
          if (feedbackNote) feedbackNote.textContent = "Couldn't send. Try again.";
        }
      };
      const feedbackX = document.getElementById("set-feedback-x");
      if (feedbackX) feedbackX.onclick = openTwitter;
      const notesGo = document.getElementById("notes-go");
      if (notesGo) notesGo.onclick = () => {
        const card = document.getElementById("whats-new");
        if (card) card.hidden = true;
        markNotes();
      };
      const notesCardClick = document.getElementById("whats-new");
      if (notesCardClick) notesCardClick.onclick = () => {
        notesCardClick.hidden = true;
        markNotes();
      };
      const jumpSounds = document.getElementById("jump-sounds");
      if (jumpSounds) jumpSounds.onclick = () => setTab("sounds");
      function applySetTab(name) {
        const allowed = ["look", "sound", "window", "share", "discord", "backup"];
        const pick = allowed.indexOf(name) >= 0 ? name : (localStorage.getItem("valvet-set-tab") || "look");
        const use = allowed.indexOf(pick) >= 0 ? pick : "look";
        localStorage.setItem("valvet-set-tab", use);
        document.querySelectorAll("#set-switch button").forEach((btn) => btn.classList.toggle("on", btn.dataset.set === use));
        document.querySelectorAll("#pane-settings .set-card").forEach((card) => { card.hidden = (card.dataset.set || "look") !== use; });
      }
      const setSwitch = document.getElementById("set-switch");
      if (setSwitch && !setSwitch.dataset.bound) {
        setSwitch.dataset.bound = "1";
        setSwitch.addEventListener("click", (event) => {
          const btn = event.target.closest("button");
          if (!btn || !btn.dataset.set) return;
          applySetTab(btn.dataset.set);
        });
      }
      applySetTab(localStorage.getItem("valvet-set-tab") || "look");
      const frame = document.getElementById("app-frame");
      if (frame) {
        const frameMin = document.getElementById("frame-min");
        const frameMax = document.getElementById("frame-max");
        const frameClose = document.getElementById("frame-close");
        if (frameMin && window.valvetWin) frameMin.onclick = () => valvetWin("min");
        if (frameMax && window.valvetWin) frameMax.onclick = () => valvetWin("max");
        if (frameClose && window.valvetWin) frameClose.onclick = () => valvetWin("close");
        frame.addEventListener("pointerdown", (event) => {
          if (event.button !== 0 || event.target.closest("button")) return;
          if (window.valvetDrag) valvetDrag();
        });
        frame.addEventListener("dblclick", (event) => {
          if (event.target.closest("button") || !window.valvetWin) return;
          valvetWin("max");
        });
      }
      function paintHidePages() {
        const host = document.getElementById("hide-pages");
        if (!host || host.dataset.bound) return;
        host.dataset.bound = "1";
        for (const name of TABS) {
          if (name === "live" || name === "settings") continue;
          const button = document.getElementById("tab-" + name);
          if (!button) continue;
          const label = el("label");
          const box = document.createElement("input");
          box.type = "checkbox";
          box.checked = !tabHidden(name);
          box.onchange = () => {
            const next = hiddenTabs().filter((id) => id !== name);
            if (!box.checked) next.push(name);
            localStorage.setItem("valvet-hidden-tabs", JSON.stringify(next));
            if (tabHidden(tab)) setTab("live");
            else setTab(tab);
          };
          label.append(box, document.createTextNode(" " + (button.textContent || name)));
          host.append(label);
        }
      }
      paintHidePages();
      applyHiddenNav();
      const logBtn = document.getElementById("set-log");
      if (logBtn) logBtn.onclick = async () => {
        const box = document.getElementById("app-log");
        if (!box) return;
        if (!box.hidden) { box.hidden = true; return; }
        box.hidden = false;
        box.textContent = "Reading…";
        try {
          const data = await (await fetch("/api/log", { cache: "no-store" })).json();
          box.textContent = (data && data.text) || "Nothing logged yet.";
        } catch {
          box.textContent = "Couldn't read the log.";
        }
      };
      const AGENT_LIGHT = { Jett: "#9ad7ff", Phoenix: "#ff6a2a", Sage: "#3dce9a", Reyna: "#c94bff", Raze: "#ffb020", Omen: "#7a5cff", Killjoy: "#ffd23a", Cypher: "#d7c4a1", Sova: "#4aa3ff", Breach: "#ff7a3c", Brimstone: "#ff5a3c", Viper: "#3dce6a", Skye: "#7dffb2", Yoru: "#6a7cff", Astra: "#c9a6ff", "KAY/O": "#d7dde8", Chamber: "#e6c27a", Neon: "#3dfff2", Fade: "#c98bff", Harbor: "#3db7ff", Gekko: "#b6ff4a", Deadlock: "#e7e2dc", Iso: "#9aa4ff", Clove: "#e7a0ff", Vyse: "#c6ff4a", Tejo: "#ff5a3c", Waylay: "#8af0ff" };
      function queryMatches(text, rows) {
        const words = String(text || "").trim().toLowerCase().split(/\s+/).filter(Boolean);
        if (!words.length) return rows || [];
        return (rows || []).filter((match) => {
          const blob = [(match.map || ""), (match.mode || ""), (match.side || "")].concat((match.players || []).map((player) => (player.name || "") + " " + (player.tag || "") + " " + (player.agent || ""))).join(" ").toLowerCase();
          return words.every((word) => blob.includes(word));
        });
      }
      async function saveHeatPicture(map, spots, tone, name) {
        const canvas = document.createElement("canvas");
        canvas.width = 1024;
        canvas.height = 1024;
        const ctx = canvas.getContext("2d");
        ctx.fillStyle = "#140910";
        ctx.fillRect(0, 0, 1024, 1024);
        try {
          const img = new Image();
          img.crossOrigin = "anonymous";
          img.src = map.displayIcon;
          await img.decode();
          ctx.drawImage(img, 0, 0, 1024, 1024);
        } catch {}
        const heat = heatLayer(spots, tone);
        await heat.decode();
        ctx.drawImage(heat, 0, 0, 1024, 1024);
        let data = heat.src;
        try { data = canvas.toDataURL("image/png"); } catch {}
        const res = await fetch("/api/picture", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: "heat-" + String(name || "map").toLowerCase() + "-" + tone + ".png", data }) });
        const body = await res.json();
        if (!body.ok) throw new Error(body.error || "save failed");
        return body.path || "";
      }
      function lightEvent(name) {
        const defaults = { kill: "#28dc78", death: "#dc2846", clutch: "#ffc440", queue: "#ff5fa2" };
        const raw = String(localStorage.getItem("valvet-light-" + name) || defaults[name] || "#ff5fa2").trim();
        const hex = raw.replace("#", "");
        if (!/^[0-9a-fA-F]{6}$/.test(hex)) return;
        lightRoom(parseInt(hex.slice(0, 2), 16), parseInt(hex.slice(2, 4), 16), parseInt(hex.slice(4, 6), 16));
      }
      function lightRoom(r, g, b) {
        const kind = localStorage.getItem("valvet-lights") || "off";
        if (!kind || kind === "off") return;
        fetch("/api/lights", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ kind, open: localStorage.getItem("valvet-openrgb") || "127.0.0.1", wled: localStorage.getItem("valvet-wled") || "", r, g, b }) }).catch(() => {});
      }
      function notePop(data) {
        const start = Number(localStorage.getItem("valvet-queue-start") || 0);
        const wait = start ? Math.max(0, Date.now() - start) : 0;
        const you = ((data && data.players) || []).find((player) => player.you) || ((data && data.party) || []).find((player) => player.you);
        const rows = readJSON("valvet-pops") || [];
        rows.push({ t: Date.now(), hour: new Date().getHours(), wait, rank: (you && you.rank) || "" });
        writeJSON("valvet-pops", rows.slice(-180));
        localStorage.removeItem("valvet-queue-start");
        paintQueueClock();
      }
      function paintQueueClock() {
        const node = document.getElementById("queue-clock");
        if (!node) return;
        const rows = readJSON("valvet-pops") || [];
        if (!rows.length) { node.textContent = "Queue pops will land here after a few games."; return; }
        const byHour = {};
        for (const row of rows) {
          const hour = row.hour;
          byHour[hour] = byHour[hour] || [];
          if (row.wait) byHour[hour].push(row.wait);
        }
        let best = null;
        for (const [hour, waits] of Object.entries(byHour)) {
          if (!waits.length) continue;
          const avg = waits.reduce((sum, value) => sum + value, 0) / waits.length;
          if (!best || avg < best.avg) best = { hour: Number(hour), avg };
        }
        node.textContent = best ? "Fastest pops around " + String(best.hour).padStart(2, "0") + ":00, about " + Math.round(best.avg / 1000) + "s, from " + rows.length + " queues on this PC." : rows.length + " queues saved.";
      }
      function spawnMotes(anchor, up) {
        const box = anchor.getBoundingClientRect();
        const cx = box.left + box.width / 2;
        const cy = box.top + box.height / 2;
        for (let i = 0; i < 16; i++) {
          const mote = document.createElement("i");
          mote.className = "mote";
          const sx = up ? cx + (Math.random() - 0.5) * 180 : cx;
          const sy = up ? cy + 80 + Math.random() * 90 : cy;
          mote.style.left = sx + "px";
          mote.style.top = sy + "px";
          document.body.append(mote);
          const tx = up ? cx : cx + (Math.random() - 0.5) * 160;
          const ty = up ? cy : cy - 40 - Math.random() * 80;
          mote.animate([{ transform: "translate(0,0)", opacity: 1 }, { transform: "translate(" + (tx - sx) + "px," + (ty - sy) + "px)", opacity: 0.1 }], { duration: 700 + Math.random() * 400, easing: "cubic-bezier(.2,.7,.2,1)" }).finished.then(() => mote.remove()).catch(() => mote.remove());
        }
      }
      let lastMoteKey = "";
      function stageLive(data) {
        const phase = (data && data.phase) || "";
        if (phase === "In queue" && !localStorage.getItem("valvet-queue-start")) localStorage.setItem("valvet-queue-start", String(Date.now()));
        if (phase !== "In queue") localStorage.removeItem("valvet-queue-start");
        const you = ((data && data.players) || []).find((player) => player.you);
        const saved = readJSON("valvet-colours") || {};
        document.documentElement.style.setProperty("--accent", saved.accent || "#ff5fa2");
        const ranked = /comp/i.test((data.queue || "") + " " + (data.mode || ""));
        const moteKey = ranked && you ? (you.rank || "") + ":" + (you.rr || "") : "";
        if (moteKey && lastMoteKey && moteKey !== lastMoteKey) {
          const coin = document.querySelector("#board .coin");
          if (coin) spawnMotes(coin, Number(String(moteKey).split(":")[1] || 0) >= Number(String(lastMoteKey).split(":")[1] || 0));
        }
        if (moteKey) lastMoteKey = moteKey;
        paintQueueClock();
      }
      async function openDirector(match) {
        match = Object.assign({}, match || {});
        if (match.id && !(match.edges || []).length) {
          try {
            const data = await (await fetch("/api/replay?id=" + encodeURIComponent(match.id), { cache: "no-store" })).json();
            if (data && data.ok) {
              if ((data.edges || []).length) match.edges = data.edges;
              if ((data.rounds || []).length) match.rounds = data.rounds;
              if ((data.buys || []).length) match.buys = data.buys;
              if ((data.players || []).length) match.players = data.players;
            }
          } catch {}
        }
        const old = document.getElementById("director");
        if (old) old.remove();
        const box = el("div", "map-lightbox");
        box.id = "director";
        const stage = el("div", "director-stage");
        const play = el("div", "director-play");
        const mapWrap = el("div", "director-map");
        const board = el("div", "director-board");
        const canvas = document.createElement("canvas");
        canvas.width = 760;
        canvas.height = 760;
        const pinsLayer = el("div", "director-pins");
        board.append(canvas, pinsLayer);
        mapWrap.append(board);
        const side = el("aside", "director-side");
        const sideKicker = el("div", "eyebrow", "Map");
        const sideTitle = el("h2", null, mapName(match.map) || "Match");
        const sideCopy = el("p", "sub", "");
        const sideRound = el("p", "sub", "");
        const sideBuy = el("p", "director-buy", "");
        const events = el("div", "director-events");
        side.append(sideKicker, sideTitle, sideCopy, sideRound, sideBuy, events);
        play.append(mapWrap, side);
        const bar = el("div", "director-bar");
        const playBtn = el("button", "tiny solid", "Pause");
        const prev = el("button", "tiny", "Prev");
        const next = el("button", "tiny", "Next");
        const roundLabel = el("b", null, "Round");
        const speeds = el("div", "director-speeds");
        const full = el("button", "tiny", "Full screen");
        const saveVid = el("button", "tiny", "Save video");
        const lensAll = el("button", "tiny on", "All");
        const lensKills = el("button", "tiny", "Your kills");
        const lensDeaths = el("button", "tiny", "Your deaths");
        const lensClutch = el("button", "tiny", "Clutch");
        const mark = el("span", "director-mark", "");
        playBtn.type = prev.type = next.type = full.type = saveVid.type = lensAll.type = lensKills.type = lensDeaths.type = lensClutch.type = "button";
        for (const [label, value] of [["Slow", 0.35], ["Normal", 1], ["Fast", 2]]) {
          const button = el("button", "tiny" + (value === 1 ? " on" : ""), label);
          button.type = "button";
          button.dataset.rate = String(value);
          speeds.append(button);
        }
        bar.append(playBtn, prev, roundLabel, next, lensAll, lensKills, lensDeaths, lensClutch, speeds, saveVid, full, mark);
        const close = el("button", "tiny solid map-close", "Close");
        close.type = "button";
        stage.append(play, bar);
        box.append(close, stage);
        document.body.append(box);
        const sheet = document.getElementById("matchbox");
        const reopenSheet = !!(sheet && !sheet.hidden);
        if (reopenSheet) sheet.hidden = true;
        let roster = match.players || [];
        const youPlayer = (() => {
          const marked = roster.filter((player) => player.you && String(player.puuid || "").trim());
          const account = String((document.getElementById("you-name") || {}).textContent || "").trim().toLowerCase();
          if (account && account !== "waiting" && account !== "this pc") {
            const named = roster.find((player) => String(player.name || "").trim().toLowerCase() === account);
            if (named) return named;
          }
          return marked[0] || roster.find((player) => player.you) || null;
        })();
        const youId = String((youPlayer && youPlayer.puuid) || "").toLowerCase();
        const youName = String((youPlayer && youPlayer.name) || "").trim().toLowerCase();
        const you = youName;
        const isYou = (name) => {
          const text = String(name || "").trim().toLowerCase();
          if (!text) return false;
          if (text === "you" || text.startsWith("you ·")) return true;
          return !!youName && youName !== "player" && youName !== "hidden" && text === youName;
        };
        const weak = (name) => {
          const text = String(name || "").trim().toLowerCase();
          return !text || text === "player" || text === "hidden";
        };
        const GUNS = {
          "29a0cfab-485b-f5d5-779a-b59f85e204a8": "Classic",
          "42da8ccc-40d5-affc-beec-15aa47b42eda": "Shorty",
          "44d4e95c-4157-0037-81b2-17841bf2e8e3": "Frenzy",
          "1baa85b4-4c70-1284-64bb-6481dfc3bb4e": "Ghost",
          "410b2e0b-4ceb-1321-1727-20858f7f3477": "Bandit",
          "e336c6b8-418d-9340-d77f-7a9e4cfe0702": "Sheriff",
          "f7e1b454-4ad4-1063-ec0a-159e56b58941": "Stinger",
          "462080d1-4035-2937-7c09-27aa2a5c27a7": "Spectre",
          "910be174-449b-c412-ab22-d0873436b21b": "Bucky",
          "ec845bf4-4f79-ddda-a3da-0db3774b2794": "Judge",
          "ae3de142-4d85-2547-dd26-4e90bed35cf7": "Bulldog",
          "4ade7faa-4cf1-8376-95ef-39884480959b": "Guardian",
          "ee8e8d15-496b-07ac-e5f6-8fae5d4c7b1a": "Phantom",
          "9c82e19d-4575-0200-1a81-3eacf00cf872": "Vandal",
          "8db0a1bf-4a50-832a-4566-faaaa6d250ca": "Warden",
          "c4883e50-4494-202c-3ec3-6b8a9284f00b": "Marshal",
          "5f0aaf7a-4289-3998-d5ff-eb9a5cf7ef5c": "Outlaw",
          "a03b24d3-4319-996d-0f8c-94bbfba1dfc7": "Operator",
          "55d8a0f4-4274-ca67-fe2c-06ab45efdf58": "Ares",
          "63e6c2b6-4a8e-869c-3d4c-e38355226584": "Odin",
          "2f59173c-4bed-b6c3-2191-dea9b58be9c7": "Melee",
        };
        const ABILITY = { grenadeability: "Grenade", ability1: "Ability", ability2: "Signature", ultimate: "Ultimate" };
        const gunName = (raw) => {
          const text = String(raw || "").trim();
          if (!text) return "";
          const known = GUNS[text.toLowerCase()] || ABILITY[text.toLowerCase()];
          if (known) return known;
          if (/^[0-9a-f-]{36}$/i.test(text)) return "";
          return text;
        };
        const near = (spot, x, y, round) => spot && (!spot.round || spot.round === round) && Math.abs(Number(spot.x) - Number(x)) < 50 && Math.abs(Number(spot.y) - Number(y)) < 50;
        const playerFor = (id, name) => roster.find((player) => id && String(player.puuid || "").toLowerCase() === id) || roster.find((player) => name && !weak(name) && String(player.name || "").toLowerCase() === String(name).toLowerCase());
        const isMineEdge = (edge, side) => {
          const id = String((side === "k" ? edge.kid : edge.vid) || "").toLowerCase();
          const raw = String((side === "k" ? edge.killer : edge.victim) || "").trim().toLowerCase();
          const account = String((document.getElementById("you-name") || {}).textContent || "").trim().toLowerCase();
          const named = account && account !== "waiting" && account !== "this pc" ? roster.find((player) => String(player.name || "").trim().toLowerCase() === account) : null;
          const marked = roster.find((player) => player.you && String(player.puuid || "").trim());
          const mine = named || marked || roster.find((player) => player.you) || null;
          const mineId = String((mine && mine.puuid) || youId || "").toLowerCase();
          const mineName = String((mine && mine.name) || youName || "").trim().toLowerCase();
          if (mineId && id) return id === mineId;
          if (raw === "you" || raw.startsWith("you ·")) return true;
          if (mineName && mineName !== "player" && mineName !== "hidden" && raw === mineName) return true;
          return false;
        };
        const personLabel = (edge, side) => {
          const id = String((side === "k" ? edge.kid : edge.vid) || "").toLowerCase();
          const raw = side === "k" ? edge.killer : edge.victim;
          const player = playerFor(id, raw);
          const mine = isMineEdge(edge, side);
          const youPlayer = roster.find((person) => person.you);
          const agent = (player && player.agent && player.agent !== "Locked" && player.agent !== "Picking" ? player.agent : "") || (mine && youPlayer && youPlayer.agent && youPlayer.agent !== "Locked" ? youPlayer.agent : "");
          if (mine) return agent ? "You · " + agent : "You";
          if (!weak(raw)) return raw;
          return agent || "Player";
        };
        const mineSide = (edge, side) => isMineEdge(edge, side);
        const edgesIn = (round) => {
          let local = edges.filter((edge) => edge.round === round);
          if (lens === "kills") local = local.filter((edge) => isMineEdge(edge, "k"));
          if (lens === "deaths") local = local.filter((edge) => isMineEdge(edge, "v"));
          return local;
        };
        const ctx = canvas.getContext("2d");
        const book = await mapBook();
        if (!document.body.contains(box)) return;
        const map = book[mapName(match.map)] || {};
        const places = calloutPins(map);
        const placeScale = (n) => (18 + n / 100 * 724) / 760 * 100;
        const pinNodes = [];
        for (const pin of places) {
          const mark = el("span", "call", pin.label);
          mark.style.left = placeScale(pin.x) + "%";
          mark.style.top = placeScale(pin.y) + "%";
          pinsLayer.append(mark);
          pinNodes.push({ node: mark, x: pin.x, y: pin.y });
        }
        sideCopy.textContent = "Pink is you. Green is the killer. Red is the death.";
        const img = new Image();
        if (map.displayIcon) {
          img.crossOrigin = "anonymous";
          img.src = cachedArt(map.displayIcon);
        }
        let edges = (match.edges || []).slice().sort((a, b) => a.round - b.round || a.t - b.t);
        let lens = "all";
        let rounds = [];
        const roundHit = (n) => {
          if (lens === "clutch") {
            const beat = (match.rounds || []).find((item) => item.n === n);
            return !!(beat && beat.vs > 0);
          }
          const local = edges.filter((edge) => edge.round === n);
          if (lens === "kills") return local.some((edge) => mineSide(edge, "k"));
          if (lens === "deaths") return local.some((edge) => mineSide(edge, "v"));
          return true;
        };
        const syncRounds = () => {
          rounds = [];
          for (const edge of edges) if (!rounds.includes(edge.round) && roundHit(edge.round)) rounds.push(edge.round);
          if (lens === "clutch") {
            for (const beat of match.rounds || []) if (beat.vs > 0 && !rounds.includes(beat.n)) rounds.push(beat.n);
            rounds.sort((a, b) => a - b);
          }
        };
        syncRounds();
        let index = 0;
        let rate = 1;
        let elapsed = 0;
        let stamp = performance.now();
        let playing = true;
        let alive = true;
        let listed = -1;
        let shown = -2;
        let solo = -1;
        const onKey = (event) => {
          if (event.key !== "Escape" || document.fullscreenElement) return;
          stop();
        };
        const stop = () => {
          alive = false;
          document.removeEventListener("keydown", onKey);
          document.removeEventListener("fullscreenchange", syncFull);
          if (document.fullscreenElement === box) document.exitFullscreen().catch(() => {});
          box.remove();
          if (reopenSheet && sheet) sheet.hidden = false;
        };
        const syncFull = () => { full.textContent = document.fullscreenElement === box ? "Window" : "Full screen"; };
        document.addEventListener("keydown", onKey);
        document.addEventListener("fullscreenchange", syncFull);
        close.onclick = stop;
        box.addEventListener("click", (event) => { if (event.target === box) stop(); });
        const span = () => 4200 / rate;
        const localOf = () => edgesIn(rounds.length ? rounds[index] : 0);
        const jump = (delta) => {
          if (!rounds.length) return;
          index = Math.max(0, Math.min(rounds.length - 1, index + delta));
          elapsed = 0;
          listed = -1;
          solo = -1;
        };
        const showKill = (killAt) => {
          const local = localOf();
          const t = local.length ? 0.86 * (killAt + 1) / local.length : 1;
          elapsed = Math.min(0.98, t) * span();
          playing = false;
          playBtn.textContent = "Play";
          solo = killAt;
        };
        prev.onclick = () => jump(-1);
        next.onclick = () => jump(1);
        playBtn.onclick = () => {
          playing = !playing;
          if (playing) solo = -1;
          playBtn.textContent = playing ? "Pause" : "Play";
          if (playing && rounds.length && elapsed >= span() && index >= rounds.length - 1) elapsed = 0;
        };
        speeds.onclick = (event) => {
          const button = event.target.closest("button");
          if (!button) return;
          const nextRate = Number(button.dataset.rate) || 1;
          const t = elapsed / span();
          rate = nextRate;
          elapsed = t * span();
          for (const node of speeds.querySelectorAll("button")) node.classList.toggle("on", node === button);
        };
        const setLens = (next, button) => {
          lens = next;
          for (const node of [lensAll, lensKills, lensDeaths, lensClutch]) node.classList.toggle("on", node === button);
          syncRounds();
          index = 0;
          elapsed = 0;
          listed = -1;
          solo = -1;
          playing = true;
          playBtn.textContent = "Pause";
          if ((next === "kills" || next === "deaths") && rounds.length) showKill(0);
        };
        lensAll.onclick = () => setLens("all", lensAll);
        lensKills.onclick = () => setLens("kills", lensKills);
        lensDeaths.onclick = () => setLens("deaths", lensDeaths);
        lensClutch.onclick = () => setLens("clutch", lensClutch);
        saveVid.onclick = async () => {
          if (saveVid.dataset.busy === "1") return;
          const fail = (text) => {
            saveVid.dataset.busy = "";
            saveVid.textContent = text;
            setTimeout(() => { saveVid.textContent = "Save video"; }, 1800);
          };
          if (!canvas.captureStream || typeof MediaRecorder === "undefined") {
            fail("Not on this PC");
            return;
          }
          let mime = "video/webm";
          const mp4s = ["video/mp4;codecs=avc1.42E01E", "video/mp4;codecs=avc1", "video/mp4"];
          if (MediaRecorder.isTypeSupported) {
            for (const next of mp4s) {
              if (MediaRecorder.isTypeSupported(next)) { mime = next; break; }
            }
            if (mime === "video/webm" && MediaRecorder.isTypeSupported("video/webm;codecs=vp8")) mime = "video/webm;codecs=vp8";
          }
          let stream;
          try { stream = canvas.captureStream(30); }
          catch { fail("Not on this PC"); return; }
          let rec;
          try { rec = new MediaRecorder(stream, { mimeType: mime }); }
          catch {
            try { mime = "video/webm"; rec = new MediaRecorder(stream); }
            catch { fail("Not on this PC"); return; }
          }
          const chunks = [];
          const ext = mime.indexOf("mp4") >= 0 ? "mp4" : "webm";
          rec.ondataavailable = (event) => { if (event.data && event.data.size) chunks.push(event.data); };
          rec.onerror = () => fail("Couldn't save");
          rec.onstop = () => {
            const blob = new Blob(chunks, { type: ext === "mp4" ? "video/mp4" : "video/webm" });
            if (blob.size < 800) { fail("Nothing recorded"); return; }
            const reader = new FileReader();
            reader.onload = async () => {
              try {
                const result = await (await fetch("/api/picture", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: "valvet-round." + ext, data: reader.result, video: true }) })).json();
                if (result && result.ok) {
                  saveVid.dataset.busy = "";
                  saveVid.dataset.note = "Saved. Open it from Settings, Storage.";
                  saveVid.textContent = "Saved";
                  setTimeout(() => { saveVid.dataset.note = ""; }, 4000);
                } else fail((result && result.error) || "Couldn't save");
              } catch { fail("Couldn't save"); }
              setTimeout(() => { if (saveVid.textContent === "Saved") saveVid.textContent = "Save video"; }, 2200);
            };
            reader.onerror = () => fail("Couldn't save");
            reader.readAsDataURL(blob);
          };
          saveVid.dataset.busy = "1";
          saveVid.dataset.note = "Recording this round";
          saveVid.textContent = "Saving…";
          setLens("all", lensAll);
          index = 0;
          elapsed = 0;
          playing = true;
          solo = -1;
          playBtn.textContent = "Pause";
          try { rec.start(250); }
          catch { fail("Not on this PC"); return; }
          const started = performance.now();
          const watch = () => {
            if (!alive) { try { rec.stop(); } catch {} return; }
            const done = rounds.length ? index >= rounds.length - 1 && elapsed >= span() : performance.now() - started > 8000;
            if (done) { try { rec.stop(); } catch {} return; }
            setTimeout(watch, 300);
          };
          setTimeout(watch, 600);
        };
        full.onclick = () => {
          if (document.fullscreenElement === box) document.exitFullscreen().catch(() => {});
          else if (box.requestFullscreen) box.requestFullscreen().catch(() => {});
        };
        const placeOf = (x, y) => {
          const spot = plotSpot(map, { x, y });
          if (!spot || !places.length) return "";
          let best = places[0];
          let dist = Infinity;
          for (const pin of places) {
            const d = (pin.x - spot.x) ** 2 + (pin.y - spot.y) ** 2;
            if (d < dist) { dist = d; best = pin; }
          }
          return best.label || "";
        };
        const clockOf = (ms) => {
          const s = Math.max(0, Math.round(Number(ms) / 1000));
          return Math.floor(s / 60) + ":" + String(s % 60).padStart(2, "0");
        };
        const paintList = (local) => {
          if (listed === index) return;
          listed = index;
          shown = -2;
          events.replaceChildren();
          if (!local.length) {
            events.append(el("p", "sub", "This round has no kill lines."));
            return;
          }
          local.forEach((edge, at) => {
            const mineKill = mineSide(edge, "k");
            const mineDeath = mineSide(edge, "v");
            const row = el("button", "director-event" + (mineKill ? " you-kill" : mineDeath ? " you-death" : ""));
            row.type = "button";
            const title = document.createElement("div");
            const killer = el("b", mineKill ? "who" : "", personLabel(edge, "k"));
            const victim = el("b", mineDeath ? "who" : "", personLabel(edge, "v"));
            title.append(killer, document.createTextNode(" killed "), victim);
            const bits = [clockOf(edge.t), gunName(edge.gun), placeOf(edge.vx, edge.vy) || placeOf(edge.kx, edge.ky)].filter(Boolean);
            if (edge.first) bits.push("First blood");
            if (edge.trade) bits.push("Trade");
            if (edge.ult) bits.push("Ult");
            row.append(title, el("span", null, bits.join(" · ")));
            row.onclick = () => showKill(at);
            events.append(row);
          });
        };
        const paintMark = (local) => {
          if (saveVid.dataset.note) {
            mark.className = "director-mark";
            mark.textContent = saveVid.dataset.note;
            return;
          }
          const kill = local.some((edge) => mineSide(edge, "k"));
          const death = local.some((edge) => mineSide(edge, "v"));
          mark.className = "director-mark" + (kill && death ? " both" : kill ? " kill" : death ? " death" : "");
          mark.textContent = kill && death ? "Your kill and death" : kill ? "Your kill" : death ? "Your death" : "No kill or death";
        };
        const iconCache = new Map();
        const agentImage = (edge, side) => {
          const player = playerFor(side === "k" ? edge.kid : edge.vid, side === "k" ? edge.killer : edge.victim);
          const url = player && player.agentIcon;
          if (!url) return null;
          const src = cachedArt(url);
          let img = iconCache.get(src);
          if (!img) {
            img = new Image();
            img.src = src;
            iconCache.set(src, img);
          }
          return img.complete && img.naturalWidth ? img : null;
        };
        const draw = (now) => {
          if (!alive || !document.body.contains(box)) return;
          if (playing) elapsed += now - stamp;
          stamp = now;
          const length = span();
          let t = rounds.length ? Math.min(1, elapsed / length) : 1;
          if (playing && t >= 1 && rounds.length && index < rounds.length - 1 && elapsed >= length + 280) {
            index += 1;
            elapsed = 0;
            t = 0;
            listed = -1;
          }
          const round = rounds.length ? rounds[index] : 0;
          const local = edgesIn(round);
          const beat = (match.rounds || []).find((item) => item.n === round);
          paintMark(local);
          paintList(local);
          const buys = (match.buys || []).filter((row) => row.round === round);
          const youBuy = buys.find((row) => (youId && String(row.id || "").toLowerCase() === youId) || String(row.name || "").startsWith("You"));
          const fullN = buys.filter((row) => row.buy === "Full").length;
          const forceN = buys.filter((row) => row.buy === "Force").length;
          const ecoN = buys.filter((row) => row.buy === "Eco").length;
          const ults = buys.filter((row) => row.ult).map((row) => row.name || "Player");
          sideBuy.textContent = buys.length ? ["You " + ((youBuy && youBuy.buy) || "—") + (youBuy && youBuy.gun ? " · " + youBuy.gun : ""), fullN + " full", forceN + " force", ecoN + " eco", ults.length ? "Ult " + ults.slice(0, 3).join(", ") : ""].filter(Boolean).join(" · ") : "";
          roundLabel.textContent = rounds.length ? "Round " + round : "No lines";
          sideRound.textContent = [beat ? (beat.you ? "Won" : "Lost") : "", beat && beat.how, beat && beat.side, "Pink is you. Green kills. Red is the death."].filter(Boolean).join(" · ");
          prev.disabled = index <= 0;
          next.disabled = !rounds.length || index >= rounds.length - 1;
          const count = local.length ? Math.max(1, Math.ceil(local.length * Math.min(1, t / 0.86))) : 0;
          const focus = solo >= 0 ? solo : count - 1;
          if (focus !== shown) {
            shown = focus;
            events.querySelectorAll("button").forEach((node, at) => node.classList.toggle("on", at === focus));
            const on = events.querySelector(".on");
            if (on) on.scrollIntoView({ block: "nearest" });
          }
          ctx.fillStyle = "#10080d";
          ctx.fillRect(0, 0, 760, 760);
          if (img.complete && img.naturalWidth) ctx.drawImage(img, 18, 18, 724, 724);
          const px = (spot) => 18 + spot.x / 100 * 724;
          const py = (spot) => 18 + spot.y / 100 * 724;
          const paintSpike = (x, y, label, color) => {
            if (!x && !y) return;
            const spot = plotSpot(map, { x, y });
            if (!spot) return;
            ctx.fillStyle = color;
            ctx.fillRect(px(spot) - 7, py(spot) - 7, 14, 14);
            ctx.font = "700 12px Segoe UI";
            ctx.fillStyle = color;
            ctx.fillText(label, px(spot) + 12, py(spot) + 4);
          };
          if (beat) {
            paintSpike(beat.px, beat.py, ["Spike", beat.site, beat.planter].filter(Boolean).join(" "), "#e2b657");
            paintSpike(beat.dx, beat.dy, ["Defuse", beat.defuser].filter(Boolean).join(" "), "#7ec8ff");
          }
          const visible = solo >= 0 && solo < local.length ? [local[solo]] : local.slice(0, count);
          const hot = [];
          const pushSpot = (x, y) => {
            const spot = plotSpot(map, { x, y });
            if (spot) hot.push(spot);
          };
          for (const edge of visible) {
            pushSpot(edge.kx, edge.ky);
            pushSpot(edge.vx, edge.vy);
          }
          if (beat) {
            pushSpot(beat.px, beat.py);
            pushSpot(beat.dx, beat.dy);
          }
          for (const pin of pinNodes) {
            const covered = hot.some((spot) => (pin.x - spot.x) ** 2 + (pin.y - spot.y) ** 2 < 420);
            pin.node.classList.toggle("back", covered);
            pin.node.style.opacity = covered ? "0" : "";
          }
          visible.forEach((edge, at) => {
            const killer = plotSpot(map, { x: edge.kx, y: edge.ky });
            const victim = plotSpot(map, { x: edge.vx, y: edge.vy });
            const mineKill = mineSide(edge, "k");
            const mineDeath = mineSide(edge, "v");
            const fade = solo >= 0 || at === visible.length - 1 ? 1 : 0.4;
            const color = mineKill ? "#ff5fa2" : mineDeath ? "#ff4d6a" : "rgba(61,206,134," + (0.85 * fade) + ")";
            if (killer && victim) {
              ctx.strokeStyle = color;
              ctx.lineWidth = mineKill || mineDeath ? 4 : 2;
              ctx.beginPath();
              ctx.moveTo(px(killer), py(killer));
              ctx.lineTo(px(victim), py(victim));
              ctx.stroke();
              const ang = Math.atan2(py(victim) - py(killer), px(victim) - px(killer));
              ctx.fillStyle = color;
              ctx.beginPath();
              ctx.moveTo(px(victim), py(victim));
              ctx.lineTo(px(victim) - 12 * Math.cos(ang - 0.4), py(victim) - 12 * Math.sin(ang - 0.4));
              ctx.lineTo(px(victim) - 12 * Math.cos(ang + 0.4), py(victim) - 12 * Math.sin(ang + 0.4));
              ctx.closePath();
              ctx.fill();
            }
            const paintDot = (spot, name, killerDot) => {
              if (!spot) return;
              const mine = personLabel(edge, killerDot ? "k" : "v").startsWith("You");
              const icon = agentImage(edge, killerDot ? "k" : "v");
              ctx.globalAlpha = fade;
              if (icon) {
                ctx.save();
                ctx.beginPath();
                ctx.arc(px(spot), py(spot), mine ? 14 : 11, 0, Math.PI * 2);
                ctx.clip();
                ctx.drawImage(icon, px(spot) - (mine ? 14 : 11), py(spot) - (mine ? 14 : 11), mine ? 28 : 22, mine ? 28 : 22);
                ctx.restore();
              } else {
                ctx.beginPath();
                ctx.arc(px(spot), py(spot), mine ? 10 : 6, 0, Math.PI * 2);
                ctx.fillStyle = mine ? "#ff5fa2" : killerDot ? "#3dce86" : "#ff4d6a";
                ctx.fill();
              }
              ctx.globalAlpha = 1;
              if (mine) {
                ctx.lineWidth = 3;
                ctx.strokeStyle = "#fff7fa";
                ctx.beginPath();
                ctx.arc(px(spot), py(spot), icon ? 14 : 10, 0, Math.PI * 2);
                ctx.stroke();
              }
              const label = personLabel(edge, killerDot ? "k" : "v");
              ctx.font = mine ? "700 15px Segoe UI" : "600 13px Segoe UI";
              ctx.lineWidth = 4;
              ctx.strokeStyle = "#10080d";
              ctx.strokeText(label, px(spot) + 16, py(spot) - 10);
              ctx.fillStyle = mine ? "#ff5fa2" : "#fff7fa";
              ctx.fillText(label, px(spot) + 16, py(spot) - 10);
            };
            paintDot(killer, edge.killer, true);
            paintDot(victim, edge.victim, false);
            if (victim && at === visible.length - 1) {
              const pulse = (now / 700) % 1;
              ctx.beginPath();
              ctx.arc(px(victim), py(victim), 16 + pulse * 22, 0, Math.PI * 2);
              ctx.strokeStyle = "rgba(255,247,250," + ((1 - pulse) * 0.85).toFixed(3) + ")";
              ctx.lineWidth = 2;
              ctx.stroke();
            }
          });
          requestAnimationFrame(draw);
        };
        requestAnimationFrame(draw);
        const weakRows = edges.filter((edge) => weak(edge.killer) || weak(edge.victim));
        if (match.id && edges.length && weakRows.length * 2 >= edges.length) {
          fetch("/api/replay?id=" + encodeURIComponent(match.id), { cache: "no-store" }).then((res) => res.json()).then((data) => {
            if (!alive || !data || !data.ok || !(data.edges || []).length) return;
            edges = data.edges.slice().sort((a, b) => a.round - b.round || a.t - b.t);
            if ((data.players || []).length) roster = data.players;
            syncRounds();
            index = Math.max(0, Math.min(rounds.length - 1, index));
            listed = -1;
          }).catch(() => {});
        }
      }
      const LIGHTS = {
        pink: [1.0, 0.45, 0.62],
        gold: [1.0, 0.78, 0.28],
        violet: [0.72, 0.48, 1.0],
        ice: [0.55, 0.86, 1.0],
        ember: [1.0, 0.32, 0.12],
      };
      const LIGHT_POWER = { faint: 0.035, soft: 0.07, bright: 0.13 };
      function lightState() {
        const color = LIGHTS[localStorage.getItem("valvet-light") || "off"];
        if (!color) return null;
        return { color, strength: LIGHT_POWER[localStorage.getItem("valvet-light-power") || "soft"] || LIGHT_POWER.soft };
      }
      function startEngine() {
        const canvas = document.getElementById("engine");
        if (canvas) {
          canvas.style.display = "none";
          canvas.dataset.live = "";
        }
        document.body.classList.add("engine-off");
      }
      async function bootWebGPU(canvas, mouse) {
        const adapter = await navigator.gpu.requestAdapter();
        if (!adapter) return false;
        const device = await adapter.requestDevice();
        const context = canvas.getContext("webgpu");
        if (!context) return false;
        const format = navigator.gpu.getPreferredCanvasFormat();
        context.configure({ device, format, alphaMode: "premultiplied" });
        const module = device.createShaderModule({ code: "struct U { res: vec2f, mouse: vec2f, time: f32, strength: f32, pad: vec2f, color: vec3f }; @group(0) @binding(0) var<uniform> u: U; @vertex fn vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f { var p = array<vec2f, 4>(vec2f(-1.0,-1.0), vec2f(1.0,-1.0), vec2f(-1.0,1.0), vec2f(1.0,1.0)); return vec4f(p[i], 0.0, 1.0); } @fragment fn fs(@builtin(position) pos: vec4f) -> @location(0) vec4f { let uv = pos.xy / u.res; let q = uv - vec2f(u.mouse.x, u.mouse.y); let d = length(q); let light = smoothstep(0.16, 0.0, d) * u.strength; return vec4f(u.color * light, light); }" });
        const pipeline = device.createRenderPipeline({
          layout: "auto",
          vertex: { module, entryPoint: "vs" },
          fragment: { module, entryPoint: "fs", targets: [{ format, blend: { color: { srcFactor: "one", dstFactor: "one-minus-src-alpha", operation: "add" }, alpha: { srcFactor: "one", dstFactor: "one-minus-src-alpha", operation: "add" } } }] },
          primitive: { topology: "triangle-strip" },
        });
        const uni = device.createBuffer({ size: 48, usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST });
        const bind = device.createBindGroup({ layout: pipeline.getBindGroupLayout(0), entries: [{ binding: 0, resource: { buffer: uni } }] });
        const frame = () => {
          const state = lightState();
          if (!state) {
            canvas.style.display = "none";
            document.body.classList.add("engine-off");
            setTimeout(frame, 700);
            return;
          }
          canvas.style.display = "block";
          document.body.classList.remove("engine-off");
          const w = innerWidth;
          const h = innerHeight;
          if (canvas.width !== w || canvas.height !== h) {
            canvas.width = w;
            canvas.height = h;
          }
          const data = new Float32Array(12);
          data[0] = canvas.width; data[1] = canvas.height; data[2] = mouse.x; data[3] = mouse.y; data[4] = performance.now() / 1000; data[5] = state.strength;
          data[8] = state.color[0]; data[9] = state.color[1]; data[10] = state.color[2];
          device.queue.writeBuffer(uni, 0, data);
          const encoder = device.createCommandEncoder();
          const pass = encoder.beginRenderPass({ colorAttachments: [{ view: context.getCurrentTexture().createView(), clearValue: { r: 0, g: 0, b: 0, a: 0 }, loadOp: "clear", storeOp: "store" }] });
          pass.setPipeline(pipeline);
          pass.setBindGroup(0, bind);
          pass.draw(4);
          pass.end();
          device.queue.submit([encoder.finish()]);
          requestAnimationFrame(frame);
        };
        requestAnimationFrame(frame);
        return true;
      }
      function bootWebGL(canvas, mouse) {
        const gl = canvas.getContext("webgl", { alpha: true, premultipliedAlpha: true });
        if (!gl) return;
        const vert = "attribute vec2 p;void main(){gl_Position=vec4(p,0.0,1.0);}";
        const frag = "precision mediump float;uniform vec2 r;uniform vec2 m;uniform float s;uniform vec3 c;void main(){vec2 u=gl_FragCoord.xy/r;vec2 q=u-vec2(m.x,1.0-m.y);float d=length(q);float light=smoothstep(0.16,0.0,d)*s;gl_FragColor=vec4(c*light,light);}";
        const compile = (type, src) => { const s = gl.createShader(type); gl.shaderSource(s, src); gl.compileShader(s); return s; };
        const program = gl.createProgram();
        gl.attachShader(program, compile(gl.VERTEX_SHADER, vert));
        gl.attachShader(program, compile(gl.FRAGMENT_SHADER, frag));
        gl.linkProgram(program);
        const buf = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, buf);
        gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]), gl.STATIC_DRAW);
        const frame = () => {
          const state = lightState();
          if (!state) {
            canvas.style.display = "none";
            document.body.classList.add("engine-off");
            setTimeout(frame, 700);
            return;
          }
          canvas.style.display = "block";
          document.body.classList.remove("engine-off");
          const w = innerWidth;
          const h = innerHeight;
          if (canvas.width !== w || canvas.height !== h) {
            canvas.width = w;
            canvas.height = h;
          }
          gl.viewport(0, 0, canvas.width, canvas.height);
          gl.clearColor(0, 0, 0, 0);
          gl.clear(gl.COLOR_BUFFER_BIT);
          gl.useProgram(program);
          gl.bindBuffer(gl.ARRAY_BUFFER, buf);
          const loc = gl.getAttribLocation(program, "p");
          gl.enableVertexAttribArray(loc);
          gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);
          gl.uniform2f(gl.getUniformLocation(program, "r"), canvas.width, canvas.height);
          gl.uniform2f(gl.getUniformLocation(program, "m"), mouse.x, mouse.y);
          gl.uniform1f(gl.getUniformLocation(program, "s"), state.strength);
          gl.uniform3f(gl.getUniformLocation(program, "c"), state.color[0], state.color[1], state.color[2]);
          gl.enable(gl.BLEND);
          gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA);
          gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
          requestAnimationFrame(frame);
        };
        requestAnimationFrame(frame);
      }
      const queryBox = document.getElementById("match-query");
      if (queryBox) queryBox.oninput = () => { recentKey = ""; drawRecent(history); };
      const rangeBtn = document.getElementById("xh-range");
      function sensScale() {
        const saved = readJSON("valvet-cache-game");
        const row = ((saved && saved.rows) || []).find((item) => item && item.label === "Sensitivity");
        const sens = Number(String((row && row.value) || "").replace(",", "."));
        if (!Number.isFinite(sens) || sens <= 0) return 1;
        return Math.max(0.45, Math.min(2.4, sens / 0.28));
      }
      if (rangeBtn) {
        rangeBtn.title = "Hold. The gap opens from your sensitivity.";
        let rangeTimer = 0;
        let spread = 0;
        const tickRange = () => {
          spread = Math.min(42, spread + 0.55 * sensScale());
          paintCrosshair(document.getElementById("xh-preview"), (document.getElementById("xh-code") || {}).value || "", spread);
          rangeTimer = requestAnimationFrame(tickRange);
        };
        rangeBtn.onpointerdown = (event) => { event.preventDefault(); spread = 0; try { rangeBtn.setPointerCapture(event.pointerId); } catch {} tickRange(); };
        const stopRange = () => { cancelAnimationFrame(rangeTimer); spread = 0; paintCrosshair(document.getElementById("xh-preview"), (document.getElementById("xh-code") || {}).value || "", 0); };
        rangeBtn.onpointerup = stopRange;
        rangeBtn.onpointercancel = stopRange;
      }
      if (!readJSON("valvet-cache-game")) fetch("/api/game", { cache: "no-store" }).then((res) => res.json()).then((data) => { if (data && (data.rows || []).length) { writeJSON("valvet-cache-game", data); stampCache("valvet-cache-game"); } }).catch(() => {});
      const lightPick = document.getElementById("set-light");
      const lightPower = document.getElementById("set-light-power");
      if (lightPick) {
        const saved = localStorage.getItem("valvet-light") || "off";
        lightPick.value = [...lightPick.options].some((opt) => opt.value === saved) ? saved : "off";
        lightPick.onchange = () => { localStorage.setItem("valvet-light", lightPick.value); startEngine(); };
      }
      if (lightPower) {
        const saved = localStorage.getItem("valvet-light-power") || "soft";
        lightPower.value = [...lightPower.options].some((opt) => opt.value === saved) ? saved : "soft";
        lightPower.onchange = () => { localStorage.setItem("valvet-light-power", lightPower.value); startEngine(); };
      }
      const lights = document.getElementById("set-lights");
      const openHost = document.getElementById("set-openrgb");
      const wledHost = document.getElementById("set-wled");
      if (lights) {
        lights.value = localStorage.getItem("valvet-lights") || "off";
        lights.onchange = () => localStorage.setItem("valvet-lights", lights.value);
      }
      if (openHost) { openHost.value = localStorage.getItem("valvet-openrgb") || ""; openHost.onchange = () => localStorage.setItem("valvet-openrgb", openHost.value.trim()); }
      if (wledHost) { wledHost.value = localStorage.getItem("valvet-wled") || ""; wledHost.onchange = () => localStorage.setItem("valvet-wled", wledHost.value.trim()); }
      for (const [id, name, fallback] of [["light-kill", "kill", "#28dc78"], ["light-death", "death", "#dc2846"], ["light-clutch", "clutch", "#ffc440"], ["light-queue", "queue", "#ff5fa2"]]) {
        const input = document.getElementById(id);
        if (!input) continue;
        const saved = localStorage.getItem("valvet-light-" + name) || fallback;
        input.value = /^#[0-9a-fA-F]{6}$/.test(saved) ? saved : fallback;
        input.oninput = () => localStorage.setItem("valvet-light-" + name, input.value);
      }
      const lightTest = document.getElementById("set-light-test");
      if (lightTest) lightTest.onclick = () => lightRoom(255, 95, 162);
      const broadcast = document.getElementById("set-broadcast");
      if (broadcast) broadcast.onclick = () => window.open("/broadcast", "valvet-broadcast", "width=1100,height=260");
      startEngine();
      paintQueueClock();
      const thanks = document.getElementById("thanks");
      const notesCard = document.getElementById("whats-new");
      if (!localStorage.getItem("valvet-tour")) {
        if (thanks) thanks.hidden = true;
        if (notesCard) notesCard.hidden = true;
        openTour();
      } else {
        if (thanks) thanks.hidden = true;
        showWhatsNew();
      }
      paintDemoLobby();
      function paintDemoLobby() {
        if (!new URLSearchParams(location.search).has("demo")) return;
        document.body.classList.remove("touring");
        for (const id of ["thanks", "tour", "whats-new"]) {
          const node = document.getElementById(id);
          if (node) node.hidden = true;
        }
        const icon = (n) => "https://media.valorant-api.com/competitivetiers/03621f52-342b-cf4e-4f86-9350a49c6d04/" + n + "/largeicon.png";
        const agent = {
          Jett: "https://media.valorant-api.com/agents/add6443a-41bd-e414-f6ad-e58d267f4e95/displayicon.png",
          Fade: "https://media.valorant-api.com/agents/dade69b4-4f5a-8528-247b-219e5a1facd6/displayicon.png",
          Chamber: "https://media.valorant-api.com/agents/22697a3d-45bf-8dd7-4fec-84a9e28c69d7/displayicon.png",
          Clove: "https://media.valorant-api.com/agents/1dbf2edd-4729-0984-3115-daa5eed44993/displayicon.png",
          Deadlock: "https://media.valorant-api.com/agents/cc8b64c8-4b25-4ff9-6e7f-37b4da43d235/displayicon.png",
          Harbor: "https://media.valorant-api.com/agents/95b78ed7-4637-86d9-7e41-71ba8c293152/displayicon.png",
          Sage: "https://media.valorant-api.com/agents/569fdd95-4d10-43ab-ca70-79becc718b46/displayicon.png",
          Omen: "https://media.valorant-api.com/agents/8e253930-4c05-31dd-1b6c-968525494517/displayicon.png",
          Reyna: "https://media.valorant-api.com/agents/a3bfb853-43b2-7238-a4f1-ad90e9e46bcc/displayicon.png",
          Killjoy: "https://media.valorant-api.com/agents/1e58de9c-4950-5125-93e9-a0aee9f98746/displayicon.png",
        };
        const person = (extra) => Object.assign({ puuid: extra.name, kills: 0, deaths: 0, assists: 0, hs: "", adr: "", score: "", level: "", party: "", partyN: 0, rankPeak: false, peak: "", peakIcon: "", you: false, enemy: false }, extra);
        const players = [
          person({ you: true, team: "Attackers", name: "Mira", tag: "cat", agent: "Jett", agentIcon: agent.Jett, rank: "Diamond 2", rankIcon: icon(19), rr: 64, kills: 14, deaths: 6, assists: 3, hs: "28", adr: "162", score: "241", party: "A", partyN: 2, level: 214 }),
          person({ team: "Attackers", name: "Kelpi", tag: "EUNE", agent: "Fade", agentIcon: agent.Fade, rank: "Diamond 1", rankIcon: icon(18), rr: 40, kills: 11, deaths: 8, assists: 5, hs: "22", adr: "148", score: "210", party: "A", partyN: 2, level: 109 }),
          person({ team: "Attackers", name: "Aki", tag: "終わった", agent: "Chamber", agentIcon: agent.Chamber, rank: "Gold 2", rankIcon: icon(11), rr: 30, kills: 8, deaths: 9, assists: 2, hs: "19", adr: "131", score: "188", level: 88 }),
          person({ team: "Attackers", name: "TallAndHandsome", tag: "meow", agent: "Clove", agentIcon: agent.Clove, rank: "Platinum 2", rankIcon: icon(16), rr: 55, kills: 9, deaths: 7, assists: 6, hs: "24", adr: "154", score: "206", level: 301 }),
          person({ team: "Attackers", name: "pissprincessowo", tag: "owo", agent: "Deadlock", agentIcon: agent.Deadlock, rank: "Unranked", rankPeak: true, peak: "Ascendant 1", peakIcon: icon(21), kills: 6, deaths: 10, assists: 1, hs: "31", adr: "119", score: "164", level: 42 }),
          person({ enemy: true, team: "Defenders", name: "InkSomnia", tag: "420", agent: "Harbor", agentIcon: agent.Harbor, rank: "Immortal 1", rankIcon: icon(24), rr: 12, kills: 16, deaths: 4, assists: 2, hs: "27", adr: "188", score: "274", party: "B", partyN: 3, level: 412 }),
          person({ enemy: true, team: "Defenders", name: "Nova", tag: "EU", agent: "Sage", agentIcon: agent.Sage, rank: "Immortal 1", rankIcon: icon(24), rr: 4, kills: 7, deaths: 9, assists: 8, hs: "18", adr: "140", score: "196", party: "B", partyN: 3, level: 266 }),
          person({ enemy: true, team: "Defenders", name: "Quiet", tag: "line", agent: "Omen", agentIcon: agent.Omen, rank: "Ascendant 2", rankIcon: icon(22), rr: 71, kills: 10, deaths: 8, assists: 4, hs: "21", adr: "151", score: "203", party: "B", partyN: 3, level: 190 }),
          person({ enemy: true, team: "Defenders", name: "Solo", tag: "one", agent: "Reyna", agentIcon: agent.Reyna, rank: "Gold 2", rankIcon: icon(11), rr: 18, kills: 5, deaths: 11, assists: 1, hs: "16", adr: "102", score: "141", level: 77 }),
          person({ enemy: true, team: "Defenders", name: "Lurk", tag: "cat", agent: "Killjoy", agentIcon: agent.Killjoy, rank: "Unranked", rankPeak: true, peak: "Platinum 1", peakIcon: icon(15), kills: 4, deaths: 8, assists: 3, hs: "20", adr: "98", score: "133", level: 15 }),
        ];
        const pane = document.getElementById("pane-live");
        pane.classList.add("match");
        document.getElementById("mode").textContent = "Swiftplay";
        document.getElementById("map").textContent = "Breeze";
        document.getElementById("phase").textContent = "In game · Frankfurt";
        const score = document.getElementById("score");
        score.replaceChildren();
        const line = document.createElement("div");
        line.className = "line";
        line.innerHTML = "<b class='hi'>2</b><span class='lbl'>ATK</span><b>0</b><span class='lbl'>DEF</span>";
        score.append(line);
        document.getElementById("status").hidden = false;
        document.getElementById("map-wash").style.backgroundImage = "url('https://media.valorant-api.com/maps/2fb9a4fd-47b8-4e7d-a969-74b4046ebd53/splash.png')";
        document.getElementById("board").replaceChildren(roster(players));
        document.body.dataset.size = "wide";
      }
    
