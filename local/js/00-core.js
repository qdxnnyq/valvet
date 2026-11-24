      (function migrateValvetKeys() {
        const legacy = "velvet-";
        const found = [];
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i);
          if (key && key.startsWith(legacy)) found.push(key);
        }
        for (const key of found) {
          const next = "valvet-" + key.slice(legacy.length);
          if (localStorage.getItem(next) == null) localStorage.setItem(next, localStorage.getItem(key) || "");
          localStorage.removeItem(key);
        }
        const read = Storage.prototype.getItem;
        const write = Storage.prototype.setItem;
        const drop = Storage.prototype.removeItem;
        const mapKey = (key) => (typeof key === "string" && key.startsWith(legacy) ? "valvet-" + key.slice(legacy.length) : key);
        Storage.prototype.getItem = function (key) {
          const next = mapKey(key);
          const value = read.call(this, next);
          if (value != null || next === key) return value;
          return read.call(this, key);
        };
        Storage.prototype.setItem = function (key, value) {
          const next = mapKey(key);
          write.call(this, next, String(value));
          if (next !== key) drop.call(this, key);
          if (typeof next === "string" && next.startsWith("valvet-")) markPrefsDirty();
        };
        Storage.prototype.removeItem = function (key) {
          const next = mapKey(key);
          drop.call(this, next);
          if (next !== key) drop.call(this, key);
          markPrefsDirty();
        };
      })();
      let prefsTimer = 0;
      let prefsSent = "";
      function markPrefsDirty() {
        clearTimeout(prefsTimer);
        prefsTimer = setTimeout(rememberPrefs, 400);
      }
      function rememberPrefs() {
        const bag = {};
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i);
          if (key && key.startsWith("valvet-")) bag[key] = localStorage.getItem(key) || "";
        }
        if (!Object.keys(bag).length) return;
        const text = JSON.stringify(bag);
        if (text === prefsSent) return;
        prefsSent = text;
        try {
          const req = new XMLHttpRequest();
          req.open("POST", "/api/prefs", false);
          req.setRequestHeader("Content-Type", "application/json");
          req.send(text);
        } catch {}
      }
      function restorePrefs() {
        try {
          const req = new XMLHttpRequest();
          req.open("GET", "/api/prefs", false);
          req.send();
          const disk = JSON.parse(req.responseText || "{}");
          if (!disk || typeof disk !== "object") return;
          Object.keys(disk).forEach((key) => {
            if (!key.startsWith("valvet-") || typeof disk[key] !== "string") return;
            const cur = localStorage.getItem(key);
            if (cur == null || cur === "") localStorage.setItem(key, disk[key]);
          });
        } catch {}
        rememberPrefs();
      }
      restorePrefs();
      window.addEventListener("beforeunload", rememberPrefs);
      let watch = [];
      let unfriends = [];
      let unfriendReady = false;
      const alerted = new Set();
      let trackedNow = new Set();
      function trackKey(name, tag) {
        return String(name || "").trim().toLowerCase() + "#" + String(tag || "").trim().toLowerCase();
      }
      const watchHeard = new Set();
      function watchToken(player) {
        const id = String((player && player.puuid) || "").toLowerCase();
        if (id) return id;
        const key = trackKey(player && player.name, player && player.tag);
        return key !== "#" ? key : "";
      }
      function onWatchList(player) {
        if (!player) return false;
        const key = trackKey(player.name, player.tag);
        const id = String(player.puuid || "").toLowerCase();
        const watched = key !== "#" && watch.some((item) => trackKey(item.name, item.tag) === key);
        const unfriendHit = unfriends.some((item) => (item.puuid && id && String(item.puuid).toLowerCase() === id) || trackKey(item.name, item.tag) === key);
        return watched || unfriendHit;
      }
      function noteWatch(data) {
        const present = new Set();
        let arrived = false;
        const take = (player) => {
          if (!player || player.you || !onWatchList(player)) return;
          const token = watchToken(player);
          if (!token) return;
          present.add(token);
          if (!watchHeard.has(token)) arrived = true;
        };
        for (const player of (data && data.players) || []) take(player);
        for (const player of (data && data.party) || []) take(player);
        for (const token of [...watchHeard]) if (!present.has(token)) watchHeard.delete(token);
        if (!arrived) return;
        for (const token of present) watchHeard.add(token);
        playWatch();
      }
      function refreshTracked(data) {
        trackedNow = new Set();
        const party = new Set((data.party || []).filter((person) => person && person.puuid && !person.you).map((person) => person.puuid.toLowerCase()));
        const friends = new Set((data.friendIds || []).map((id) => String(id).toLowerCase()));
        for (const person of [...(data.friends || []), ...(data.party || [])]) {
          if (person && person.puuid && !person.you) friends.add(person.puuid.toLowerCase());
        }
        const you = (data.players || []).find((player) => player.you);
        const yourParty = you && you.party;
        const live = data.phase === "Agent select" || data.phase === "In game" || data.phase === "Loading lobby";
        if (!live) alerted.clear();
        let fresh = false;
        let freshWatch = false;
        const names = [];
        for (const player of data.players || []) {
          if (!player || player.you) continue;
          const id = String(player.puuid || "").toLowerCase();
          const key = trackKey(player.name, player.tag);
          const watched = key !== "#" && watch.some((item) => trackKey(item.name, item.tag) === key);
          const unfriendHit = unfriends.some((item) => (item.puuid && id && String(item.puuid).toLowerCase() === id) || trackKey(item.name, item.tag) === key);
          const inParty = (id && party.has(id)) || (yourParty && player.party && player.party === yourParty);
          const randomFriend = !!(id && friends.has(id) && !inParty);
          if (!watched && !randomFriend && !unfriendHit) continue;
          trackedNow.add(id || key);
          if (live && !alerted.has(id || key)) {
            alerted.add(id || key);
            fresh = true;
            if (watched || unfriendHit) freshWatch = true;
            const label = (player.name || "Player") + (player.tag ? "#" + player.tag : "");
            names.push(unfriendHit ? label + " (unfriended you)" : label);
          }
        }
        if (fresh && !freshWatch) alertTone();
        noteWatch(data);
        paintWatchLive(data);
        paintAvoidLive(data);
        if (names.length && discordBooted && localStorage.getItem("valvet-discord") === "on" && localStorage.getItem("valvet-discord-watch") !== "off") {
          postDiscord("In your game", names.join("\n"));
        }
      }
      function quietNow() {
        const from = localStorage.getItem("valvet-quiet-from") || "";
        const to = localStorage.getItem("valvet-quiet-to") || "";
        if (!from || !to || from === to) return false;
        const mins = (value) => { const [h, m] = value.split(":").map(Number); return h * 60 + (m || 0); };
        const now = new Date();
        const cur = now.getHours() * 60 + now.getMinutes();
        const start = mins(from);
        const end = mins(to);
        return start < end ? cur >= start && cur < end : cur >= start || cur < end;
      }
      function alertTone() {
        if (quietNow()) return;
        playSaved("friend", false);
      }
      function playSting() {
        playNotes([740, 988, 740, 1480, 1976], "square", 0.08, true, { step: 0.055, decay: 11 });
      }
      function playSaved(kind, force) {
        const defaults = { friend: "pack", watch: "none", played: "none" };
        const pick = localStorage.getItem("valvet-" + kind + "-sound") || defaults[kind] || "soft";
        if (pick === "none") return;
        if (!force && (quietNow() || localStorage.getItem("valvet-sound") === "off")) return;
        if (pick === "pack") { playPack(!!force); return; }
        if (pick === "sting") { playSting(); return; }
        if (pick === "watch") { playSaved("watch", !!force); return; }
        if (pick === "custom") { playFile("/api/media?kind=" + kind, !!force); return; }
        if (pick === "meow") { playFile("/sounds/meow", !!force, 1, true); return; }
        const pack = SOUND_PACKS[pick];
        if (pack && pack.file) { playFile(pack.file, !!force, 1, true); return; }
        if (pack) playNotes(pack.notes, pack.wave, pack.dur || 0.18, !!force, pack);
      }
      function playWatch(force) {
        playSaved("watch", !!force);
      }
      function playPlayed(force) {
        playSaved("played", !!force);
      }
      function playWish(force) {
        if (!force && (quietNow() || localStorage.getItem("valvet-sound") === "off")) return;
        playNotes([1046, 1318, 1568, 2093, 1568], "bell", 0.22, true, { step: 0.085, decay: 1.55 });
      }
      function paintWatchLive(data) {
        const node = document.getElementById("watch-live");
        if (!node) return;
        const names = [];
        const seen = new Set();
        const take = (player) => {
          if (!player || player.you || !onWatchList(player)) return;
          const name = shownName(player);
          if (!name || seen.has(name)) return;
          seen.add(name);
          names.push(name);
        };
        for (const player of (data && data.players) || []) take(player);
        for (const player of (data && data.party) || []) take(player);
        node.hidden = !names.length;
        const phase = (data && data.phase) || "";
        const where = phase === "In game" || phase === "Agent select" || phase === "Loading lobby" ? "in this game" : "in this lobby";
        node.textContent = names.length ? "Watch list " + where + " · " + names.join(", ") : "";
      }
      function avoidList() {
        const rows = readJSON("valvet-avoid");
        return Array.isArray(rows) ? rows : [];
      }
      function isAvoided(player) {
        if (!player) return false;
        const id = String(player.puuid || "").toLowerCase();
        const name = String(player.name || "").toLowerCase();
        const tag = String(player.tag || "").toLowerCase();
        return avoidList().some((row) => (id && String(row.puuid || "").toLowerCase() === id) || (name && String(row.name || "").toLowerCase() === name && String(row.tag || "").toLowerCase() === tag));
      }
      function paintAvoidLive(data) {
        const node = document.getElementById("avoid-live");
        if (!node) return;
        const names = [];
        const seen = new Set();
        const take = (player) => {
          if (!player || player.you || !isAvoided(player)) return;
          const name = shownName(player);
          if (!name || seen.has(name)) return;
          seen.add(name);
          names.push(name);
        };
        for (const player of (data && data.players) || []) take(player);
        for (const player of (data && data.party) || []) take(player);
        node.hidden = !names.length;
        node.textContent = names.length ? "Avoid list in this lobby · " + names.join(", ") : "";
      }
      function playerTracked(player) {
        const id = String(player.puuid || "").toLowerCase();
        return trackedNow.has(id) || trackedNow.has(trackKey(player.name, player.tag));
      }
      let tab = "live";
      let mode = "All";
      let history = [];
      let actLoaded = 0;
      function mergeHistory(rows) {
        if (!rows || !rows.length) return;
        const by = new Map();
        const weight = (match) => ((match.rounds || []).length * 3) + ((match.deaths || []).length) + ((match.kills || []).length) + ((match.edges || []).length) + ((match.players || []).length * 2) + (match.won === true || match.won === false ? 6 : 0);
        for (const row of history) if (row && row.id) by.set(String(row.id).toLowerCase(), row);
        const named = (match) => ((match && match.players) || []).filter((player) => {
          const name = String((player && player.name) || "").trim();
          return name && name !== "Player" && name !== "Hidden";
        }).length;
        const rankScore = (player) => (player && player.rank && player.rank !== "Unranked" ? 2 : 0) + (player && player.peak ? 2 : 0) + (player && player.rankIcon ? 1 : 0);
        const takeRank = (dst, src) => {
          if (!dst || !src || rankScore(src) <= rankScore(dst)) return false;
          if (src.rank) dst.rank = src.rank;
          if (src.rankIcon) dst.rankIcon = src.rankIcon;
          if (src.peak) dst.peak = src.peak;
          if (src.peakIcon) dst.peakIcon = src.peakIcon;
          if (src.rr) dst.rr = src.rr;
          dst.rankPeak = !!src.rankPeak;
          dst.tier = src.tier || dst.tier;
          return true;
        };
        const mergeRanks = (dst, src) => {
          const by = new Map();
          for (const player of (src && src.players) || []) {
            if (!player) continue;
            by.set(String(player.puuid || "").toLowerCase(), player);
          }
          let changed = false;
          for (const player of (dst && dst.players) || []) {
            if (!player) continue;
            const next = by.get(String(player.puuid || "").toLowerCase());
            if (takeRank(player, next)) changed = true;
          }
          return changed;
        };
        for (const row of keepNames(rows)) {
          if (!row || !row.id) continue;
          if (row.map) row.map = mapName(row.map);
          const id = String(row.id).toLowerCase();
          const prev = by.get(id);
          if (prev) mergeRanks(prev, row);
          if (prev) {
            const nextW = weight(row);
            const prevW = weight(prev);
            if (nextW < prevW || named(row) + 1 < named(prev)) continue;
            if (nextW === prevW && named(row) <= named(prev)) continue;
            mergeRanks(row, prev);
          }
          by.set(id, row);
        }
        history = [...by.values()].sort((a, b) => (Number(b.started) || 0) - (Number(a.started) || 0) || String(b.when || "").localeCompare(String(a.when || "")));
        for (const match of history) {
          if (match.won == null) {
            const won = resultOf(match);
            if (won === true || won === false) match.won = won;
          }
        }
      }
      async function loadAct(force) {
        if (!force && Date.now() - actLoaded < 45000) return;
        actLoaded = Date.now();
        try {
          const data = await (await fetch("/api/history", { cache: "no-store" })).json();
          mergeHistory(data.matches || []);
          if (tab === "history" && !modeQueue) drawRecent(history);
          if (tab === "deaths") { const root = document.getElementById("deaths"); if (root) root.dataset.sig = ""; paintDeaths(); }
          if (tab === "clutches") { const root = document.getElementById("clutches"); if (root) root.dataset.sig = ""; paintClutches(); }
          if (tab === "queues") { const root = document.getElementById("queues"); if (root) root.dataset.sig = ""; paintQueues(); }
          if (tab === "records" || tab === "session" || tab === "profile") paintDesk();
        } catch {
          if (tab === "history" && !history.length) {
            const recent = document.getElementById("recent");
            if (recent) recent.replaceChildren(el("div", "empty", "Couldn't load matches."));
          }
        }
      }
      let samplePost = false;
      let announced = false;
      let discordBooted = false;
      let discordPhase = "";
      let lastLive = null;
      let lastPhase = "";
      let clipAudio = null;
      const SOUND_PACKS = {
        soft: { wave: "sine", step: 0.1, dur: 0.22, decay: 6, notes: [880, 1175, 1568] },
        ping: { wave: "triangle", step: 0.08, dur: 0.14, decay: 8, notes: [740, 988, 1480] },
        chime: { wave: "bell", step: 0.16, dur: 0.42, decay: 2.4, notes: [523, 784, 1046] },
        arcade: { wave: "square", step: 0.055, dur: 0.08, decay: 10, notes: [440, 554, 659, 880, 1174] },
        radar: { wave: "sine", step: 0.18, dur: 0.07, decay: 14, notes: [1320, 1320, 1760] },
        drop: { wave: "saw", step: 0.07, dur: 0.28, decay: 3.2, notes: [294, 196, 131, 98] },
        spark: { wave: "square", step: 0.035, dur: 0.05, decay: 16, notes: [1568, 2093, 2637, 3136] },
        bell: { wave: "bell", step: 0, dur: 0.7, decay: 1.8, notes: [880] },
        neon: { wave: "saw", step: 0.075, dur: 0.16, decay: 5, notes: [392, 494, 587, 784, 988] },
        siren: { wave: "sine", step: 0.13, dur: 0.12, decay: 4, notes: [740, 1175, 740, 1175] },
        honk: { wave: "square", step: 0.02, dur: 0.1, decay: 4, notes: [220, 180, 140] },
        boing: { wave: "sine", step: 0.05, dur: 0.18, decay: 3, notes: [180, 320, 540, 240] },
        coin: { wave: "square", step: 0.06, dur: 0.08, decay: 8, notes: [988, 1318] },
        drama: { wave: "saw", step: 0.12, dur: 0.2, decay: 2.2, notes: [196, 185, 174, 146] },
        trombone: { wave: "saw", step: 0.08, dur: 0.16, decay: 3, notes: [233, 196, 174, 130] },
        airhorn: { wave: "square", step: 0.01, dur: 0.26, decay: 1.5, notes: [440, 466, 440, 494] },
        cricket: { wave: "triangle", step: 0.03, dur: 0.04, decay: 12, notes: [3200, 1800, 3200, 1800] },
        typewriter: { wave: "square", step: 0.07, dur: 0.03, decay: 18, notes: [180, 220, 140, 260, 190] },
        slot: { wave: "square", step: 0.06, dur: 0.05, decay: 9, notes: [660, 880, 660, 880, 1320] },
        probe: { wave: "alien", step: 0.09, dur: 0.12, decay: 6, notes: [1480, 1760, 2100, 1320, 2400] },
        mothership: { wave: "alien", step: 0.16, dur: 0.34, decay: 2.2, notes: [92, 110, 78, 140, 64] },
        abduct: { wave: "alien", step: 0.08, dur: 0.2, decay: 3.4, notes: [220, 330, 494, 740, 1174, 1760] },
        signal: { wave: "alien", step: 0.045, dur: 0.07, decay: 9, notes: [2100, 880, 2460, 640, 2800, 1200] },
        ufo: { wave: "alien", step: 0.11, dur: 0.18, decay: 4, notes: [523, 784, 392, 988, 311, 1568] },
        knock: { file: "/sounds/knock" },
        bloom: { file: "/sounds/bloom" },
        pulse: { file: "/sounds/pulse" },
        clutch: { file: "/sounds/clutch" },
      };
      function soundPack() {
        return SOUND_PACKS[localStorage.getItem("valvet-soundpack") || "soft"] || SOUND_PACKS.soft;
      }
      function volumeGain() {
        const raw = Number(localStorage.getItem("valvet-volume"));
        const pct = Number.isFinite(raw) ? raw : 15;
        const t = Math.max(0, Math.min(1, pct / 100));
        return Math.pow(t, 2.15);
      }
      function resultVolume() {
        const raw = localStorage.getItem("valvet-result-volume");
        const pct = raw == null || raw === "" ? 40 : Number(raw);
        return Math.max(0, Math.min(100, Number.isFinite(pct) ? pct : 40)) / 100;
      }
      function waveSample(wave, phase, t, freq) {
        if (wave === "square") return Math.sin(phase) >= 0 ? 0.62 : -0.62;
        if (wave === "triangle") return (2 / Math.PI) * Math.asin(Math.max(-1, Math.min(1, Math.sin(phase))));
        if (wave === "saw") return (2 * ((freq * t) % 1) - 1) * 0.42;
        if (wave === "bell") {
          return Math.sin(phase) * 0.72 + Math.sin(phase * 2) * 0.22 + Math.sin(phase * 3.02) * 0.1 + Math.sin(phase * 4.2) * 0.05;
        }
        if (wave === "alien") {
          const wobble = 1 + Math.sin(t * 34) * 0.22 + Math.sin(t * 7) * 0.08;
          const chirp = Math.sin(2 * Math.PI * freq * wobble * t);
          const ring = Math.sin(2 * Math.PI * freq * 1.51 * t + Math.sin(t * 19) * 3) * 0.28;
          return (chirp * 0.72 + ring) * 0.85;
        }
        return Math.sin(phase);
      }
      function wavBlob(samples, rate) {
        const n = samples.length;
        const buf = new ArrayBuffer(44 + n * 2);
        const view = new DataView(buf);
        const write = (offset, text) => { for (let i = 0; i < text.length; i++) view.setUint8(offset + i, text.charCodeAt(i)); };
        write(0, "RIFF");
        view.setUint32(4, 36 + n * 2, true);
        write(8, "WAVE");
        write(12, "fmt ");
        view.setUint32(16, 16, true);
        view.setUint16(20, 1, true);
        view.setUint16(22, 1, true);
        view.setUint32(24, rate, true);
        view.setUint32(28, rate * 2, true);
        view.setUint16(32, 2, true);
        view.setUint16(34, 16, true);
        write(36, "data");
        view.setUint32(40, n * 2, true);
        for (let i = 0; i < n; i++) {
          const x = Math.max(-1, Math.min(1, samples[i]));
          view.setInt16(44 + i * 2, x < 0 ? x * 0x8000 : x * 0x7fff, true);
        }
        return new Blob([buf], { type: "audio/wav" });
      }
      function playNotes(notes, wave, seconds, force, pack) {
        if (!force && localStorage.getItem("valvet-sound") === "off") return Promise.resolve(false);
        const base = pack && pack.volume != null ? pack.volume : volumeGain();
        const vol = base * (pack && pack.gain != null ? pack.gain : 1);
        if (vol <= 0.001) return Promise.resolve(false);
        const rate = 22050;
        const step = pack && pack.step != null ? pack.step : 0.09;
        const decay = pack && pack.decay ? pack.decay : 7;
        const dur = seconds || 0.16;
        const total = Math.max(dur + 0.05, notes.length * step + dur);
        const count = Math.max(1, Math.floor(rate * total));
        const samples = new Float32Array(count);
        notes.forEach((freq, index) => {
          const start = Math.floor(rate * index * step);
          const len = Math.floor(rate * dur);
          for (let s = 0; s < len && start + s < count; s++) {
            const t = s / rate;
            const env = Math.min(1, s / 70) * Math.exp(-t * decay);
            const phase = 2 * Math.PI * freq * t;
            samples[start + s] += waveSample(wave, phase, t, freq) * env * vol;
          }
        });
        let peak = 0.001;
        for (let i = 0; i < count; i++) peak = Math.max(peak, Math.abs(samples[i]));
        const cap = 0.92;
        if (peak > cap) {
          const scale = cap / peak;
          for (let i = 0; i < count; i++) samples[i] *= scale;
        }
        const url = URL.createObjectURL(wavBlob(samples, rate));
        const audio = document.createElement("audio");
        audio.src = url;
        audio.volume = 1;
        document.body.append(audio);
        if (clipAudio && clipAudio !== audio) { try { clipAudio.pause(); clipAudio.remove(); } catch {} }
        clipAudio = audio;
        const cleanup = () => { URL.revokeObjectURL(url); if (audio.parentNode) audio.remove(); };
        audio.onended = cleanup;
        const played = audio.play();
        if (played && typeof played.then === "function") {
          return played.then(() => true).catch(() => { cleanup(); return false; });
        }
        return Promise.resolve(true);
      }
      function tone(freq, dur) {
        playNotes([freq], "sine", Math.max(0.06, dur || 0.08), false, { step: 0, dur: dur || 0.08, decay: 8 });
      }
      let soundCtx = null;
      function soundContext() {
        const Ctx = window.AudioContext || window.webkitAudioContext;
        if (!Ctx) return null;
        if (!soundCtx) soundCtx = new Ctx();
        if (soundCtx.state === "suspended") soundCtx.resume();
        return soundCtx;
      }
      function playFile(url, force, gain, absolute) {
        if (!force && localStorage.getItem("valvet-sound") === "off") return Promise.resolve(false);
        const pct = Number(localStorage.getItem("valvet-volume"));
        const scale = gain == null ? 1 : gain;
        const base = absolute ? 1 : ((Number.isFinite(pct) ? pct : 15) / 100);
        const level = Math.max(0, Math.min(1, base * scale));
        if (level <= 0) return Promise.resolve(false);
        return fetch(url, { cache: "no-store" }).then((res) => {
          if (!res.ok) throw new Error("sound");
          return res.arrayBuffer();
        }).then((buf) => {
          const ctx = soundContext();
          if (!ctx) throw new Error("sound");
          const ready = ctx.state === "suspended" ? ctx.resume() : Promise.resolve();
          return ready.then(() => ctx.decodeAudioData(buf.slice(0))).then((decoded) => {
            const src = ctx.createBufferSource();
            const amp = ctx.createGain();
            amp.gain.value = level;
            src.buffer = decoded;
            src.connect(amp);
            amp.connect(ctx.destination);
            src.start(0);
            return true;
          });
        }).catch(() => false);
      }
      function playPack(force) {
        if ((localStorage.getItem("valvet-soundpack") || "") === "none") return Promise.resolve(false);
        if ((localStorage.getItem("valvet-soundpack") || "") === "custom") {
          playFile("/api/media?kind=notify", !!force);
          return Promise.resolve(true);
        }
        if ((localStorage.getItem("valvet-soundpack") || "") === "meow") {
          playFile("/sounds/meow", !!force, 1, true);
          return Promise.resolve(true);
        }
        const pack = soundPack();
        if (pack.file) {
          playFile(pack.file, !!force, 1, true);
          return Promise.resolve(true);
        }
        return playNotes(pack.notes, pack.wave, pack.dur || 0.18, !!force, pack);
      }
      const RESULT_SOUNDS = {
        gg: { file: "/sounds/gg" },
        cheer: { notes: [523, 659, 784, 1046], wave: "triangle", step: 0.09, dur: 0.18, decay: 4 },
        fanfare: { notes: [392, 523, 659, 784, 1046], wave: "bell", step: 0.11, dur: 0.26, decay: 2.2 },
        spark: { notes: [880, 1174, 1568, 2093], wave: "square", step: 0.05, dur: 0.08, decay: 12 },
        sad: { file: "/sounds/sad" },
        low: { notes: [330, 262, 196, 147], wave: "sine", step: 0.12, dur: 0.28, decay: 3 },
        drop: { notes: [494, 370, 247, 165], wave: "saw", step: 0.08, dur: 0.22, decay: 3.4 },
        hush: { notes: [440, 349, 294], wave: "triangle", step: 0.14, dur: 0.3, decay: 2.6 },
        meow: { file: "/sounds/meow" },
        coin: { notes: [988, 1318, 1568], wave: "square", step: 0.06, dur: 0.08, decay: 8 },
        trombone: { notes: [233, 196, 174, 130], wave: "saw", step: 0.08, dur: 0.16, decay: 3 },
      };
      function playResult(kind, force) {
        const pick = localStorage.getItem(kind === "loss" ? "valvet-loss" : "valvet-win") || (kind === "loss" ? "sad" : "gg");
        if (pick === "none") return;
        if (!force && quietNow()) return;
        const vol = resultVolume();
        if (vol <= 0) return;
        if (pick === "custom") {
          playFile("/api/media?kind=" + (kind === "loss" ? "loss" : "win"), true, vol, true);
          return;
        }
        const sound = RESULT_SOUNDS[pick] || RESULT_SOUNDS[kind === "loss" ? "sad" : "gg"];
        if (sound.file) {
          const ran = playFile(sound.file, true, vol, true);
          if (ran && typeof ran.then === "function") {
            ran.then((ok) => {
              if (ok) return;
              const tone = kind === "loss" ? RESULT_SOUNDS.low : RESULT_SOUNDS.cheer;
              playNotes(tone.notes, tone.wave, tone.dur || 0.18, true, { ...tone, gain: 1, volume: vol });
            });
          }
          return;
        }
        playNotes(sound.notes, sound.wave, sound.dur || 0.18, !!force, { ...sound, gain: 1, volume: vol });
      }
      function playJoin() {
        return playNotes([494, 659, 880], "triangle", 0.12, false, { step: 0.07, decay: 5 });
      }
      function playQueue() {
        return playNotes([784, 1046, 1318], "square", 0.08, false, { step: 0.05, decay: 8 });
      }
      const MAP_ART = {
        Ascent: "/pack/map/Ascent.png",
        Split: "/pack/map/Split.png",
        Fracture: "/pack/map/Fracture.png",
        Bind: "/pack/map/Bind.png",
        Breeze: "/pack/map/Breeze.png",
        District: "/pack/map/District.png",
        Kasbah: "/pack/map/Kasbah.png",
        Drift: "/pack/map/Drift.png",
        Glitch: "/pack/map/Glitch.png",
        Piazza: "/pack/map/Piazza.png",
        Abyss: "/pack/map/Abyss.png",
        Lotus: "/pack/map/Lotus.png",
        Sunset: "/pack/map/Sunset.png",
        Pearl: "/pack/map/Pearl.png",
        Icebox: "/pack/map/Icebox.png",
        Haven: "/pack/map/Haven.png",
        Corrode: "/pack/map/Corrode.png",
        Summit: "/pack/map/Summit.png",
      };
      function paintMapArt(name) {
        const wash = document.getElementById("map-wash");
        if (!wash) return;
        const art = MAP_ART[name || ""] || "";
        wash.style.backgroundImage = art ? "url(" + art + ")" : "";
      }
      function edgeFlash(kind) {
        const node = document.getElementById("flash");
        if (!node) return;
        node.hidden = false;
        node.className = "flash " + kind;
        clearTimeout(edgeFlash.timer);
        edgeFlash.timer = setTimeout(() => { node.hidden = true; node.className = "flash"; }, 720);
      }
      function paintStreak(matches) {
        const root = document.getElementById("you-streak");
        if (!root) return;
        const five = (matches || []).filter((match) => match.won === true || match.won === false).slice().sort((a, b) => Date.parse(b.when || 0) - Date.parse(a.when || 0)).slice(0, 5).reverse();
        const key = five.map((match) => (match.won ? "w" : "l") + (match.id || "")).join("");
        if (root.dataset.key === key) return;
        root.dataset.key = key;
        root.replaceChildren();
        for (const match of five) root.append(el("i", match.won ? "w" : "l"));
      }
      let partySeen = new Set();
      let partyReady = false;
      function syncPartyJoin(party, ok) {
        if (!ok) return;
        const ids = new Set((party || []).filter((person) => !person.you && person.puuid).map((person) => String(person.puuid).toLowerCase()));
        if (!partyReady) {
          partySeen = ids;
          partyReady = true;
          return;
        }
        let joined = false;
        for (const id of ids) if (!partySeen.has(id)) joined = true;
        partySeen = ids;
        if (joined && localStorage.getItem("valvet-join") === "on") playJoin();
      }
      let resultWait = 0;
      let resultPlayed = "";
      let beforeMatch = "";
      let sawLiveGame = false;
      let postCard = null;
      let postUntil = 0;
      let wasLive = false;
      let heldLive = null;
      function hsLabel(you) {
        if (!you || you.hs == null || you.hs === "") return "";
        const raw = String(you.hs).trim();
        if (!raw || raw === "—" || raw === "-") return "";
        return raw.includes("%") ? raw : raw + "%";
      }
      function finishedYou(before) {
        const match = (history || [])[0];
        if (!match || !match.id || match.id === before) return null;
        return match;
      }
      function syncPost(data) {
        const phase = data.phase || "";
        const playing = phase === "In game";
        const live = !!(data.ok && (phase === "Agent select" || phase === "In game" || phase === "Loading lobby"));
        if (playing) {
          sawLiveGame = true;
          if (!beforeMatch) beforeMatch = ((history || [])[0] && history[0].id) || "none";
        }
        if (live) {
          heldLive = { map: data.map, mode: data.mode, you: (data.players || []).find((player) => player.you) };
          wasLive = true;
          return;
        }
        if (wasLive && sawLiveGame) {
          const match = finishedYou(beforeMatch) || (history || [])[0];
          postCard = buildPost(heldLive, match && match.id !== beforeMatch ? match : null);
          postUntil = Date.now() + 45000;
          fillPostRR();
          resultWait = Date.now() + 90000;
          sawLiveGame = false;
        }
        wasLive = false;
      }
      function buildPost(held, match) {
        const you = (match && (match.players || []).find((player) => player.you)) || (held && held.you) || null;
        const hs = hsLabel(you);
        const kda = you && (you.kills || you.deaths || you.assists) ? (you.kills || "0") + " / " + (you.deaths || "0") + " / " + (you.assists || "0") : "";
        const score = match && match.score1 != null && match.score2 != null ? match.score1 + "–" + match.score2 : "";
        return {
          title: (match && match.map) || (held && held.map) || "Match over",
          mode: (match && match.mode) || (held && held.mode) || "",
          kda,
          hs,
          acs: you && (you.acs || you.score) ? String(you.acs || you.score) : "",
          adr: you && you.adr ? String(you.adr) : "",
          score,
          won: match && (match.won === true || match.won === false) ? match.won : null,
          line: [kda, hs ? hs + " HS" : "", you && you.adr ? you.adr + " ADR" : "", you && (you.acs || you.score) ? (you.acs || you.score) + " ACS" : ""].filter(Boolean).join(" · "),
          rounds: match && match.rounds,
        };
      }
      function catFace(won) {
        const box = document.createElement("div");
        box.className = "mood-box " + (won ? "win" : "loss");
        const img = document.createElement("img");
        img.className = "mood";
        img.alt = "";
        img.src = won ? "/yippee.gif" : "/cry.gif";
        box.append(img);
        return box;
      }
      function paintPostStats(card) {
        const host = document.getElementById("post-stats");
        const score = document.getElementById("post-score");
        const node = document.getElementById("post");
        if (node) node.classList.toggle("win", card && card.won === true);
        if (node) node.classList.toggle("loss", card && card.won === false);
        if (score) {
          const next = (card && card.score) || "";
          if (score.dataset.score !== next) {
            score.dataset.score = next;
            score.replaceChildren();
            if (next) {
              const text = document.createTextNode(next);
              score.append(text);
              const tag = document.createElement("span");
              tag.textContent = card.won === true ? "Win" : card.won === false ? "Loss" : (card.mode || "Score");
              score.append(tag);
              const node = document.getElementById("post");
              if (node) {
                node.classList.remove("reveal");
                void node.offsetWidth;
                node.classList.add("reveal");
              }
              const parts = String(next).match(/(\d+)\s*[–-]\s*(\d+)/);
              if (parts) {
                const a = Number(parts[1]);
                const b = Number(parts[2]);
                let step = 0;
                clearInterval(score.timer);
                score.timer = setInterval(() => {
                  step += 1;
                  text.textContent = Math.round(a * step / 10) + "–" + Math.round(b * step / 10);
                  if (step >= 10) clearInterval(score.timer);
                }, 45);
              }
            }
          }
        }
        const cat = document.getElementById("post-cat");
        if (cat) {
          const showCat = !!(card && (card.won === true || card.won === false));
          cat.hidden = !showCat;
          cat.className = "mood-box post-cat " + (card && card.won ? "win" : "loss");
          if (showCat) {
            let img = cat.querySelector("img");
            if (!img) {
              img = document.createElement("img");
              img.alt = "";
              cat.append(img);
            }
            const src = card.won ? "/yippee.gif" : "/cry.gif";
            if (img.getAttribute("src") !== src) img.src = src;
          }
        }
        if (!host) return;
        host.replaceChildren();
        const bits = [["KDA", card && card.kda], ["HS", card && card.hs], ["ADR", card && card.adr], ["ACS", card && card.acs]];
        for (const [label, value] of bits) {
          if (!value) continue;
          const cell = document.createElement("span");
          const strong = document.createElement("b");
          strong.textContent = value;
          const name = document.createElement("i");
          name.textContent = label;
          cell.append(strong, name);
          host.append(cell);
        }
      }
      async function fillPostRR() {
        try {
          const data = await (await fetch("/api/rr", { cache: "no-store" })).json();
          const last = (data.points || []).slice(-1)[0];
          if (!postCard || !last || last.delta == null || String(postCard.line).includes("RR")) return;
          postCard.line = [postCard.line, (Number(last.delta) >= 0 ? "+" : "") + last.delta + " RR"].filter(Boolean).join(" · ");
          paintPost();
        } catch {}
      }
      function paintPost() {
        const node = document.getElementById("post");
        if (!node) return;
        const pane = document.getElementById("pane-live");
        const show = !!(postCard && Date.now() < postUntil && pane && !pane.classList.contains("match"));
        if (!show && samplePost) {
          node.hidden = false;
          return;
        }
        if (show) samplePost = false;
        node.hidden = !show;
        if (!show) return;
        document.getElementById("post-title").textContent = postCard.title;
        document.getElementById("post-line").textContent = postCard.mode || "";
        paintPostStats(postCard);
        const paintRounds = () => {
          const host = document.getElementById("post-rounds");
          if (!host) return;
          host.replaceChildren();
          const strip = roundStrip(postCard.rounds);
          if (strip) host.append(strip);
        };
        paintRounds();
        const arrived = (history || [])[0];
        if (arrived && arrived.id && arrived.id !== beforeMatch && arrived.id !== resultPlayed) {
          const next = buildPost(heldLive, arrived);
          if (postCard.won === true || postCard.won === false) next.won = postCard.won;
          postCard = next;
          document.getElementById("post-title").textContent = postCard.title;
          document.getElementById("post-line").textContent = postCard.mode || "";
          paintPostStats(postCard);
          paintRounds();
          attachDeathBeat(arrived);
        }
        if (!resultWait || Date.now() > resultWait) return;
        const ended = (history || [])[0];
        if (!ended || !ended.id || ended.id === beforeMatch || ended.id === resultPlayed) return;
        if (ended.won !== true && ended.won !== false) return;
        const when = Date.parse(ended.when || "");
        if (Number.isFinite(when) && Date.now() - when > 20 * 60 * 1000) return;
        resultPlayed = ended.id;
        beforeMatch = ended.id;
        resultWait = 0;
        const next = buildPost(heldLive, ended);
        next.line = [next.line, ended.won ? "Win" : "Loss"].filter(Boolean).join(" · ");
        postCard = next;
        document.getElementById("post-title").textContent = postCard.title;
        document.getElementById("post-line").textContent = postCard.mode || "";
        paintPostStats(postCard);
        paintRounds();
        playResult(ended.won ? "win" : "loss");
      }
      let nightReady = false;
      let nightWas = false;
      function syncNight(shop) {
        const open = ((shop && shop.night) || []).length > 0;
        if (nightReady && open && !nightWas && localStorage.getItem("valvet-night") === "on") {
          playNotes([784, 988, 1174, 1568], "triangle", 0.16, false, { step: 0.08, decay: 4 });
        }
        nightWas = open;
        nightReady = true;
      }
      function discordURL() {
        return (localStorage.getItem("valvet-discord-url") || "").trim();
      }
      function postDiscord(title, text) {
        const url = discordURL();
        if (!url || localStorage.getItem("valvet-discord") !== "on") return;
        fetch("/api/discord", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ url, title, text }),
        }).catch(() => {});
      }
      function syncDiscord(data) {
        const phase = data.phase || "";
        const live = phase === "Agent select" || phase === "In game" || phase === "Loading lobby";
        const key = live ? [phase, data.map || "", data.mode || ""].join("|") : "";
        if (live) {
          lastLive = {
            map: data.map || "",
            mode: data.mode || "",
            server: data.server || "",
            score1: data.score1,
            score2: data.score2,
          };
        }
        if (!discordBooted) {
          discordBooted = true;
          discordPhase = key;
          return;
        }
        if (localStorage.getItem("valvet-discord") !== "on" || !discordURL()) {
          discordPhase = key;
          if (!live) lastLive = null;
          return;
        }
        if (live && key !== discordPhase && localStorage.getItem("valvet-discord-live") === "on") {
          const bits = [data.mode, data.server].filter(Boolean);
          if (data.score1 != null && data.score2 != null) bits.push(data.score1 + " – " + data.score2);
          postDiscord(phase + (data.map ? " · " + data.map : ""), bits.join(" · ") || "Match found");
        }
        if (!live && discordPhase && localStorage.getItem("valvet-discord-end") === "on" && lastLive && lastLive.score1 != null && lastLive.score2 != null) {
          postDiscord("Match over" + (lastLive.map ? " · " + lastLive.map : ""), lastLive.score1 + " – " + lastLive.score2 + (lastLive.mode ? " · " + lastLive.mode : ""));
          lastLive = null;
        }
        discordPhase = key;
      }
      let roundMark = null;
      let statusKey = "";
      let nightEnds = 0;
      let nightOpen = false;
      function yourRound(data) {
        const you = (data.players || []).find((player) => player.you);
        const side = you && you.team;
        const left = Number(data.score1);
        const right = Number(data.score2);
        if (!Number.isFinite(left) || !Number.isFinite(right)) return null;
        if (side === "Defenders") return { us: right, them: left };
        return { us: left, them: right };
      }
      function syncRounds(data) {
        const phase = data.phase || "";
        if (phase !== "In game") {
          roundMark = null;
          return;
        }
        const now = yourRound(data);
        if (!now) return;
        if (!roundMark) {
          roundMark = now;
          return;
        }
        if (now.us === roundMark.us && now.them === roundMark.them) return;
        const prev = roundMark;
        roundMark = now;
        const line = now.us + " – " + now.them + (data.map ? " · " + data.map : "");
        if (now.us > prev.us && now.them === prev.them) {
          lightEvent("kill");
          if (localStorage.getItem("valvet-discord-rounds") !== "off") postDiscord("Round won", line);
        } else if (now.them > prev.them && now.us === prev.us) {
          lightEvent("death");
          if (localStorage.getItem("valvet-discord-rounds") !== "off") postDiscord("Round lost", line);
        }
      }
      function discordStatusText(data) {
        const phase = data.phase || "In menu";
        const title = data.map ? phase + " · " + data.map : phase;
        const bits = [data.mode, data.server].filter(Boolean);
        const party = (data.party || []).length;
        if (party > 1) bits.push(party + " in party");
        if (data.score1 != null && data.score2 != null && (phase === "In game" || phase === "Loading lobby")) bits.push(data.score1 + " – " + data.score2);
        return { title, text: bits.join(" · ") || "Waiting for a game" };
      }
      async function pushStatus(data) {
        const status = discordStatusText(data);
        const key = status.title + "|" + status.text;
        const app = (localStorage.getItem("valvet-discord-app") || "").trim();
        if (app && localStorage.getItem("valvet-discord") === "on" && key !== statusKey) {
          fetch("/api/presence", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ clientId: app, details: status.title, state: status.text }),
          }).catch(() => {});
        }
        if (localStorage.getItem("valvet-discord") !== "on" || localStorage.getItem("valvet-discord-status") === "off" || !discordURL()) {
          statusKey = key;
          return;
        }
        if (key === statusKey) return;
        statusKey = key;
        try {
          const result = await (await fetch("/api/discord", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              url: discordURL(),
              title: status.title,
              text: status.text,
              status: true,
              message: localStorage.getItem("valvet-discord-msg") || "",
            }),
          })).json();
          if (result && result.message) localStorage.setItem("valvet-discord-msg", result.message);
        } catch {}
      }
      function duoLines(party, matches) {
        return (party || []).filter((person) => person && !person.you && person.puuid).map((mate) => {
          const id = String(mate.puuid).toLowerCase();
          let games = 0;
          let wins = 0;
          let rounds = 0;
          for (const match of matches || []) {
            const players = match.players || [];
            const them = players.find((player) => String(player.puuid || "").toLowerCase() === id);
            const me = players.find((player) => player.you);
            if (!them || !me) continue;
            if (them.enemy || (me.team && them.team && me.team !== them.team)) continue;
            games++;
            const s1 = Number(match.score1);
            const s2 = Number(match.score2);
            if (Number.isFinite(s1) && Number.isFinite(s2)) rounds += s1 + s2;
            if (match.won === true) wins++;
            else if (match.won == null && me.team === "Attackers" && s1 > s2) wins++;
            else if (match.won == null && me.team === "Defenders" && s2 > s1) wins++;
          }
          return {
            name: mate.name || "Player",
            tag: mate.tag || "",
            games,
            text: games ? wins + "–" + (games - wins) + " · " + Math.round((wins / games) * 100) + "% · " + (rounds / games).toFixed(1) + " rounds" : "No games together in recent history",
          };
        });
      }
      function paintDuos(id, party, matches) {
        const root = document.getElementById(id);
        if (!root) return;
        const lines = duoLines(party, matches);
        const key = JSON.stringify(lines);
        if (root.dataset.key === key) return;
        root.dataset.key = key;
        root.replaceChildren();
        for (const line of lines) {
          const row = el("div", "duo");
          row.append(el("b", null, line.name + (line.tag ? "#" + line.tag : "")));
          row.append(el("span", null, line.text));
          root.append(row);
        }
      }
      function paintNightClock() {
        const leftNode = document.getElementById("pass-left");
        if (leftNode && leftNode.dataset.ends) paintActClock(leftNode, leftNode.dataset.ends);
        const node = document.getElementById("night-clock");
        if (!node) return;
        if (!nightEnds) {
          node.textContent = nightOpen ? "" : "Not up right now";
          return;
        }
        const left = nightEnds - Math.floor(Date.now() / 1000);
        if (left <= 0) {
          node.textContent = nightOpen ? "Ended" : "Not up right now";
          return;
        }
        const days = Math.floor(left / 86400);
        const hours = Math.floor((left % 86400) / 3600);
        const mins = Math.floor((left % 3600) / 60);
        const secs = left % 60;
        const pad = (n) => String(n).padStart(2, "0");
        const clock = (days ? days + "d " : "") + pad(hours) + ":" + pad(mins) + ":" + pad(secs);
        node.textContent = (nightOpen ? "Closes in " : "Up in ") + clock;
      }
      const openMatches = new Set();
      let recentKey = "";
      let partyKey = "";
      let lastPeople = null;
      let showOfflineFriends = localStorage.getItem("valvet-friends-offline") === "on";
      let storeKey = "";
      let boardKey = "";
      let stageKey = "";
      function paintIdentity(name, level) {
        const title = document.getElementById("you-name");
        const sub = document.getElementById("you-level");
        const clean = name || "This PC";
        const line = level ? "Level " + level : "";
        if (title && title.textContent !== clean) title.textContent = clean;
        if (sub && sub.textContent !== line) sub.textContent = line;
      }

      function el(tag, cls, text) {
        const node = document.createElement(tag);
        if (cls) node.className = cls;
        if (text != null) node.textContent = text;
        return node;
      }
      function score(n) { return n == null || Number.isNaN(Number(n)) ? "–" : String(n); }
      function sideLabel(name) {
        if (name === "Attackers") return "ATK";
        if (name === "Defenders") return "DEF";
        return name || "";
      }
      function cachedArt(url) {
        const text = String(url || "");
        const tier = text.match(/competitivetiers\/[^/]+\/(\d+)\//i);
        if (tier) return "/pack/rank/" + tier[1] + ".png";
        if (!/^https:\/\/([a-z0-9-]+\.)?valorant-api\.com\//i.test(text)) return text;
        return "/api/art?u=" + encodeURIComponent(text);
      }
      function agentArt(name) {
        const icons = {
          Jett: "add6443a-41bd-e414-f6ad-e58d267f4e95",
          Fade: "dade69b4-4f5a-8528-247b-219e5a1facd6",
          Chamber: "22697a3d-45bf-8dd7-4fec-84a9e28c69d7",
          Clove: "1dbf2edd-4729-0984-3115-daa5eed44993",
          Deadlock: "cc8b64c8-4b25-4ff9-6e7f-37b4da43d235",
          Harbor: "95b78ed7-4637-86d9-7e41-71ba8c293152",
          Sage: "569fdd95-4d10-43ab-ca70-79becc718b46",
          Omen: "8e253930-4c05-31dd-1b6c-968525494517",
          Reyna: "a3bfb853-43b2-7238-a4f1-ad90e9e46bcc",
          Killjoy: "1e58de9c-4950-5125-93e9-a0aee9f98746",
          Phoenix: "eb93336a-449b-9c1b-0a54-a891f7921d69",
          Sova: "320b2a48-4d9b-a075-30f1-1f93a9b638fa",
          Viper: "707eab51-4836-f488-046a-cda6bf494859",
          Cypher: "117ed9e3-49f3-6512-3ccf-0cada7e3823b",
          Raze: "f94c3b30-42be-e959-889c-5aa313dba261",
          Breach: "5f8d3a7f-467b-97f3-062c-13acf203c006",
          Skye: "6f2a04ca-43e0-be17-7f36-b3908627744d",
          Yoru: "7f94d92c-4234-0a36-9646-3a87eb8b5c89",
          Astra: "41fb69c1-4189-7b37-f117-bcaf1e96f1bf",
          "KAY/O": "601dbbe7-43ce-be57-2a40-4abd24953621",
          Neon: "bb2a4828-46eb-8cd1-e765-15848195d751",
          Gekko: "e370fa57-4757-3604-3648-499e1f642d3f",
          Iso: "0e38b510-41a8-5780-5e8f-568b2a4f2d6c",
          Vyse: "efba5359-4016-a1e5-7626-b1ae76895940",
          Tejo: "b444168c-4e35-8076-db47-ef9bf368f384"
        };
        const id = icons[String(name || "").trim()];
        return id ? "https://media.valorant-api.com/agents/" + id + "/displayicon.png" : "";
      }
      function face(player) {
        const img = document.createElement("img");
        img.alt = "";
        const icon = (player && player.agentIcon) || agentArt(player && player.agent);
        if (icon) img.src = cachedArt(icon);
        return img;
      }
      function nameButton(player) {
        const button = el("button", "who");
        button.type = "button";
        button.append(document.createTextNode(player.name || "Player"));
        if (player.tag) button.append(el("span", "meta", " #" + player.tag));
        button.onclick = (event) => { event.preventDefault(); event.stopPropagation(); openPlayer(player); };
        return button;
      }
      function groups(players) {
        const order = [];
        const map = new Map();
        for (const player of players || []) {
          if (!player) continue;
          const team = player.team || "Lobby";
          if (!map.has(team)) { map.set(team, []); order.push(team); }
          map.get(team).push(player);
        }
        const rank = (team) => (team === "Attackers" ? 0 : team === "Defenders" ? 1 : 2);
        order.sort((a, b) => rank(a) - rank(b));
        return order.map((team) => [team, map.get(team)]);
      }
      function statLine(player) {
        const bits = [];
        if (player.agent) bits.push(player.agent);
        const kills = player.kills;
        const deaths = player.deaths;
        const assists = player.assists;
        if (kills != null && kills !== "" || deaths != null && deaths !== "" || assists != null && assists !== "") {
          bits.push((kills || "0") + "/" + (deaths || "0") + "/" + (assists || "0"));
        }
        if (player.score) bits.push(player.score + " ACS");
        if (player.hs) bits.push(player.hs + " HS");
        if (player.adr) bits.push(player.adr + " ADR");
        if (player.was) bits.push("was " + player.was);
        if (player.rank) bits.push(player.rank + (player.rr ? " " + player.rr : ""));
        return bits.join(" · ");
      }
      function biggestStack(list) {
        const counts = {};
        for (const player of list || []) {
          if (!player.party) continue;
          const size = Number(player.partyN) || 0;
          counts[player.party] = Math.max(counts[player.party] || 0, size);
        }
        let max = 0;
        for (const size of Object.values(counts)) max = Math.max(max, size);
        if (max >= 2) return max;
        const seen = {};
        for (const player of list || []) {
          if (!player.party) continue;
          seen[player.party] = (seen[player.party] || 0) + 1;
        }
        for (const size of Object.values(seen)) max = Math.max(max, size);
        return max;
      }
      function ago(ms) {
        const mins = Math.max(1, Math.round((Date.now() - ms) / 60000));
        if (mins < 60) return mins + "m ago";
        const hours = Math.round(mins / 60);
        if (hours < 36) return hours + "h ago";
        return Math.round(hours / 24) + "d ago";
      }
      function lastTogether(puuid, matches) {
        const id = String(puuid || "").toLowerCase();
        if (!id) return "";
        let best = 0;
        let map = "";
        for (const match of matches || []) {
          const when = Date.parse(match.when || "");
          if (!Number.isFinite(when) || when < best) continue;
          const them = (match.players || []).find((player) => String(player.puuid || "").toLowerCase() === id);
          const me = (match.players || []).find((player) => player.you);
          if (!them || !me) continue;
          if (them.enemy || (me.team && them.team && me.team !== them.team)) continue;
          best = when;
          map = match.map || "";
        }
        if (!best) return "";
        return "Last together " + ago(best) + (map ? " · " + map : "");
      }
      function hsPercent(value) {
        const n = parseFloat(String(value == null ? "" : value).replace(/[^\d.]/g, ""));
        return Number.isFinite(n) ? n : NaN;
      }
      function matchStart(match) {
        const started = Number(match && match.started);
        if (Number.isFinite(started) && started > 0) return started < 1e12 ? started * 1000 : started;
        const when = Date.parse((match && match.when) || "");
        return Number.isFinite(when) ? when : 0;
      }
      function todayStats(matches) {
        const start = dayStart(Date.now());
        let wins = 0;
        let losses = 0;
        let hsSum = 0;
        let hsN = 0;
        let kills = 0;
        let deaths = 0;
        const list = [];
        for (const match of matches || []) {
          const when = matchStart(match);
          if (!when || when < start) continue;
          if (match.won === true) wins += 1;
          else if (match.won === false) losses += 1;
          const me = (match.players || []).find((player) => player.you);
          const hs = hsPercent(me && me.hs);
          if (Number.isFinite(hs)) { hsSum += hs; hsN += 1; }
          kills += Number((me && me.kills) || 0);
          deaths += Number((me && me.deaths) || 0);
          list.push(match);
        }
        return { wins, losses, games: wins + losses, hs: hsN ? Math.round(hsSum / hsN) : null, kills, deaths, list };
      }
      function rrGraph(points, height) {
        const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        svg.setAttribute("viewBox", "0 0 320 90");
        svg.setAttribute("class", "rr-graph");
        if (height) svg.style.height = height + "px";
        const vals = (points || []).map((point) => Number(point.tier) * 100 + Number(point.rr));
        if (vals.length < 2) return svg;
        const min = Math.min(...vals);
        const max = Math.max(...vals);
        const span = Math.max(8, max - min);
        const coords = vals.map((value, index) => {
          const x = 8 + (index / (vals.length - 1)) * 304;
          const y = 78 - ((value - min) / span) * 64;
          return [x, y];
        });
        const line = coords.map((pair, index) => (index ? "L" : "M") + pair[0].toFixed(1) + " " + pair[1].toFixed(1)).join(" ");
        const fill = document.createElementNS("http://www.w3.org/2000/svg", "path");
        fill.setAttribute("class", "fill");
        fill.setAttribute("d", line + " L" + coords[coords.length - 1][0].toFixed(1) + " 88 L" + coords[0][0].toFixed(1) + " 88 Z");
        const stroke = document.createElementNS("http://www.w3.org/2000/svg", "path");
        stroke.setAttribute("class", "line");
        stroke.setAttribute("d", line);
        svg.append(fill, stroke);
        return svg;
      }
      let rrPoints = [];
      let weekRR = 0;
      const RANK_NAMES = ["", "", "", "Iron 1", "Iron 2", "Iron 3", "Bronze 1", "Bronze 2", "Bronze 3", "Silver 1", "Silver 2", "Silver 3", "Gold 1", "Gold 2", "Gold 3", "Platinum 1", "Platinum 2", "Platinum 3", "Diamond 1", "Diamond 2", "Diamond 3", "Ascendant 1", "Ascendant 2", "Ascendant 3", "Immortal 1", "Immortal 2", "Immortal 3", "Radiant"];
      function tierOf(rank) {
        const index = RANK_NAMES.indexOf(rank || "");
        return index > 2 ? index : 0;
      }
      function averageRank(list) {
        const tiers = [];
        for (const person of list || []) {
          let tier = Number(person.tier) || 0;
          if (tier < 3) tier = tierOf(person.rank);
          if (tier < 3) tier = tierOf(person.peak);
          if (tier >= 3) tiers.push(tier);
        }
        if (!tiers.length) return "";
        const avg = Math.round(tiers.reduce((sum, tier) => sum + tier, 0) / tiers.length);
        return RANK_NAMES[avg] || "";
      }
      function mapRecord(matches, map) {
        const rows = (matches || []).filter((match) => match && match.map === map && (match.won === true || match.won === false));
        if (!rows.length || !map) return "";
        const wins = rows.filter((match) => match.won).length;
        return wins + "–" + (rows.length - wins) + " on this map";
      }
      function seenLately(matches) {
        const ids = new Set();
        const cutoff = Date.now() - 24 * 60 * 60 * 1000;
        for (const match of matches || []) {
          if (!match) continue;
          const at = Date.parse(match.when || "");
          if (Number.isNaN(at) || at < cutoff) continue;
          for (const player of match.players || []) {
            if (player && !player.you && player.puuid) ids.add(String(player.puuid).toLowerCase());
          }
        }
        return ids;
      }
      function agentRecord(name) {
        let games = 0;
        let wins = 0;
        let losses = 0;
        let kills = 0;
        let deaths = 0;
        let assists = 0;
        let hs = 0;
        let hsN = 0;
        for (const match of history || []) {
          const you = (match.players || []).find((player) => player.you && player.agent === name);
          if (!you) continue;
          games += 1;
          if (match.won === true) wins += 1;
          if (match.won === false) losses += 1;
          kills += Number(you.kills || 0);
          deaths += Number(you.deaths || 0);
          assists += Number(you.assists || 0);
          const rate = parseInt(you.hs, 10);
          if (!Number.isNaN(rate)) { hs += rate; hsN += 1; }
        }
        if (!games) return "No recent games on this agent.";
        const kda = (kills / games).toFixed(1) + "/" + (deaths / games).toFixed(1) + "/" + (assists / games).toFixed(1);
        return wins + "–" + losses + " in the last games · " + kda + (hsN ? " · " + Math.round(hs / hsN) + "% HS" : "");
      }
      function rrDelta(match) {
        const id = String((match && match.id) || "").toLowerCase().replace(/-/g, "");
        if (!id) return null;
        const mode = String((match && match.mode) || "").toLowerCase();
        if (mode && !/competitive|premier|ranked/.test(mode)) return null;
        const hit = (rrPoints || []).find((point) => String(point.id || "").toLowerCase().replace(/-/g, "") === id);
        if (!hit || hit.earned == null || hit.earned === "") return null;
        const n = Number(hit.earned);
        return Number.isFinite(n) ? n : null;
      }
      function weekRecap(matches) {
        const start = Date.now() - 7 * 86400000;
        const maps = {};
        const agents = {};
        let games = 0;
        let wins = 0;
        for (const match of matches || []) {
          const when = matchStart(match);
          if (!when || when < start || (match.won !== true && match.won !== false)) continue;
          games += 1;
          if (match.won === true) wins += 1;
          const you = (match.players || []).find((player) => player.you);
          const take = (book, key) => {
            if (!key) return;
            const row = book[key] || { w: 0, n: 0 };
            row.n += 1;
            if (match.won === true) row.w += 1;
            book[key] = row;
          };
          take(maps, match.map || "");
          take(agents, you && you.agent && you.agent !== "Picking" ? you.agent : "");
        }
        const best = (book) => Object.entries(book).sort((a, b) => (b[1].n - a[1].n) || ((b[1].w / b[1].n) - (a[1].w / a[1].n)))[0];
        return { games, wins, map: best(maps), agent: best(agents), rr: Number(weekRR) || 0 };
      }
      function paintToday(matches) {
        const root = document.getElementById("today");
        if (!root) return;
        const stats = todayStats(matches);
        const week = weekRecap(matches);
        const last = rrPoints.length ? rrPoints[rrPoints.length - 1] : null;
        const kd = !stats.games ? "—" : stats.deaths ? (stats.kills / stats.deaths).toFixed(2) : stats.kills.toFixed(2);
        const ids = (stats.list || []).slice(0, 8).map((match) => (match.id || "") + ":" + match.score1 + "-" + match.score2 + ":" + match.won);
        const key = JSON.stringify([stats.wins, stats.losses, stats.games, stats.hs, stats.kills, stats.deaths, last && last.rank, last && last.rr, ids, week.games, week.wins, week.rr, week.map && week.map[0], week.agent && week.agent[0]]);
        if (root.dataset.key === key) return;
        root.dataset.key = key;
        root.replaceChildren();
        const bar = el("div", "today-bar");
        const cell = (kicker, value) => {
          const box = el("div", "today-cell");
          box.append(el("span", null, kicker), el("b", null, value));
          return box;
        };
        bar.append(
          cell("Today", stats.games ? stats.wins + "–" + stats.losses : "No games"),
          cell("HS", stats.hs == null ? "—" : stats.hs + "%"),
          cell("K/D", kd),
          cell("Rank", last ? (last.rank || "RR") + " " + last.rr : "—")
        );
        root.append(bar);
        const row = el("div", "today-games");
        for (const match of (stats.list || []).slice(0, 8)) {
          const you = (match.players || []).find((player) => player.you);
          const agent = you && you.agent && you.agent !== "Picking" ? you.agent : "";
          const score = match.score1 != null && match.score2 != null ? match.score1 + "–" + match.score2 : "";
          const pill = el("button", "today-pill" + (match.won === true ? " w" : match.won === false ? " l" : ""), [match.map || match.mode || "Game", agent, score].filter(Boolean).join(" · "));
          pill.type = "button";
          pill.onclick = () => openMatchSheet(match);
          row.append(pill);
        }
        if (row.childElementCount) root.append(row);
        if (week.games || week.rr) {
          const mapBit = week.map && week.map[1].n ? week.map[0] + " " + Math.round((week.map[1].w / week.map[1].n) * 100) + "%" : "";
          const agentBit = week.agent ? week.agent[0] : "";
          const rrBit = (week.rr > 0 ? "+" : "") + week.rr + " RR";
          const line = el("div", "week-bar");
          line.append(el("span", null, "This week"));
          line.append(el("b", null, [week.games ? week.wins + "–" + (week.games - week.wins) : "No games", rrBit, mapBit, agentBit].filter(Boolean).join(" · ")));
          root.append(line);
        }
      }
      function paintRR(slot) {
        if (!slot) return;
        slot.replaceChildren();
        slot.append(el("h2", null, "Rank this act"));
        if (rrPoints.length < 2) {
          slot.append(el("div", "empty", "Ranked games show up here after a competitive match."));
          return;
        }
        const last = rrPoints[rrPoints.length - 1];
        const first = rrPoints[0];
        const delta = (last.tier * 100 + last.rr) - (first.tier * 100 + first.rr);
        slot.append(el("p", "sub", last.rank + " " + last.rr + " RR · " + (delta >= 0 ? "+" : "") + delta + " across " + rrPoints.length + " games"));
        slot.append(rrGraph(rrPoints));
        const goal = Number(localStorage.getItem("valvet-rr-goal") || 0);
        if (goal > 0) {
          const week = Number(weekRR) || 0;
          slot.append(el("p", "sub goal", "This week " + (week >= 0 ? "+" : "") + week + " RR · goal " + goal));
          const meter = el("div", "meter");
          const bar = document.createElement("span");
          bar.style.width = Math.max(0, Math.min(100, (week / goal) * 100)) + "%";
          meter.append(bar);
          slot.append(meter);
        }
      }
      async function openTracker(player) {
        if (!player || !player.puuid) return;
        await fetch("/api/rank?id=" + encodeURIComponent(player.puuid));
      }
