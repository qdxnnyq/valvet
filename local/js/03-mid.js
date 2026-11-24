      let latestState = null;
      let ownedNames = [];
      let roomPeople = [];
      let lfgPeople = [];
      let boardRates = {};
      let roomWatches = new Set();
      let roomPinged = new Set();
      let roomPing = "";
      function valvetID() {
        let id = localStorage.getItem("valvet-id");
        if (!id) {
          id = Math.random().toString(36).slice(2, 10);
          localStorage.setItem("valvet-id", id);
        }
        return id;
      }
      function valvetNick() {
        return (localStorage.getItem("valvet-nick") || (latestState && latestState.you) || "Player").trim() || "Player";
      }
      function ensureCode() {
        let code = localStorage.getItem("valvet-code") || "";
        if (!/^[A-Z0-9]{8}$/.test(code)) {
          const abc = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
          code = "";
          for (let i = 0; i < 8; i++) code += abc[Math.floor(Math.random() * abc.length)];
          localStorage.setItem("valvet-code", code);
        }
        return code;
      }
      function loadWith() {
        try { return JSON.parse(localStorage.getItem("valvet-with") || "[]"); } catch { return []; }
      }
      function saveWith(rows) {
        localStorage.setItem("valvet-with", JSON.stringify(rows.slice(0, 30)));
      }
      function riotID() {
        const you = String((latestState && latestState.you) || "").trim();
        if (!you || you === "This PC" || you === "Waiting") return "";
        return you;
      }
      const PLAY_MODES = ["Competitive", "Unrated", "Swiftplay", "Spike Rush", "Deathmatch", "Premier", "Escalation", "Team Deathmatch", "Replication", "Custom"];
      const PLAY_ROLES = ["Duelist", "Initiator", "Controller", "Sentinel", "Flex"];
      function loadList(key) {
        try {
          const rows = JSON.parse(localStorage.getItem(key) || "[]");
          return Array.isArray(rows) ? rows.filter((row) => typeof row === "string") : [];
        } catch { return []; }
      }
      function loadModes() { return loadList("valvet-modes"); }
      function loadRoles() { return loadList("valvet-roles"); }
      function lookingOn() {
        const until = Number(localStorage.getItem("valvet-until") || 0);
        if (localStorage.getItem("valvet-looking") === "on" && until && Date.now() > until) {
          localStorage.setItem("valvet-looking", "off");
          localStorage.setItem("valvet-ready", "off");
          localStorage.removeItem("valvet-until");
          paintLooking();
          return false;
        }
        return localStorage.getItem("valvet-looking") === "on";
      }
      function paintLooking() {
        const on = localStorage.getItem("valvet-looking") === "on";
        const button = document.getElementById("look-go");
        const state = document.getElementById("look-state");
        if (button) {
          button.textContent = on ? "Take post down" : "Post to the board";
          button.classList.toggle("on", on);
        }
        if (state) state.textContent = on ? "Your post is on the board. It stays for about 12 hours, or until you take it down." : "Not posted. Write what you want, then post it to the board.";
        const status = document.getElementById("duo-status");
        if (status && !on) status.textContent = "You have not posted yet.";
      }
      function myRankTier() {
        const you = ((latestState && latestState.players) || []).find((player) => player.you);
        return rankTier(you && you.rank);
      }
      function rankTier(text) {
        const raw = String(text || "");
        for (let i = RANK_NAMES.length - 1; i >= 3; i--) if (raw.startsWith(RANK_NAMES[i])) return i;
        return 0;
      }
      function inRankRange(tier, low, high) {
        const a = tierOf(low);
        const b = tierOf(high);
        if (!a && !b) return true;
        if (!tier) return true;
        const min = a && b ? Math.min(a, b) : (a || b);
        const max = a && b ? Math.max(a, b) : (a || b);
        return tier >= min && tier <= max;
      }
      function myRiot() {
        const live = riotID().trim();
        if (live) return live;
        const posted = (lfgPeople || []).find((row) => row.id === valvetID());
        return String((posted && posted.riot) || "").trim();
      }
      function isMine(person) {
        if (!person) return false;
        if (person.id && person.id === valvetID()) return true;
        const riot = String(person.riot || "").trim().toLowerCase();
        const mine = myRiot().trim().toLowerCase();
        return riot !== "" && mine !== "" && riot === mine;
      }
      function showInFind(person) {
        if (!person || isMine(person) || !person.looking) return false;
        if (person.id === "diag" || String(person.riot || "").toLowerCase() === "diag#0000") return false;
        if (loadBlock().includes(person.id)) return false;
        const server = localStorage.getItem("valvet-filter-server") || "";
        const queue = localStorage.getItem("valvet-filter-queue") || "";
        const mode = localStorage.getItem("valvet-filter-mode") || "";
        const gender = localStorage.getItem("valvet-filter-gender") || "";
        const ageMin = Number(localStorage.getItem("valvet-filter-age-min") || 0);
        const ageMax = Number(localStorage.getItem("valvet-filter-age-max") || 0);
        if (server && person.server !== server) return false;
        if (queue && person.queue !== queue && !(person.modes || []).includes(queue)) return false;
        if (mode && !(person.modes || []).includes(mode) && person.queue !== mode) return false;
        if (gender && person.gender !== gender) return false;
        if ((ageMin || ageMax) && !person.age) return false;
        if (ageMin && person.age < ageMin) return false;
        if (ageMax && person.age > ageMax) return false;
        return true;
      }
      function loadGiven() {
        try {
          const bag = JSON.parse(localStorage.getItem("valvet-given-rates") || "{}");
          return bag && typeof bag === "object" ? bag : {};
        } catch { return {}; }
      }
      function rateFor(person) {
        const key = String((person && person.riot) || "").trim().toLowerCase();
        if (!key) return { avg: 0, n: 0 };
        return boardRates[key] || { avg: 0, n: 0 };
      }
      function boardAge() {
        const n = Math.round(Number(localStorage.getItem("valvet-age") || 0));
        if (!n || n < 16 || n > 99) return 0;
        return n;
      }
      function savedGames() {
        const rec = latestState && latestState.records;
        const n = Number(rec && rec.games) || 0;
        if (n > 0) return n;
        return (history || []).length;
      }
      function boardFacts(person) {
        const rate = rateFor(person);
        const bits = [];
        if (person.age) bits.push(person.age + " yrs");
        if (person.gender) bits.push(person.gender);
        if (person.server) bits.push(person.server);
        if (person.queue) bits.push(person.queue);
        const modes = (person.modes || []).join(", ");
        if (modes) bits.push(modes);
        const roles = (person.roles || []).join(", ");
        if (roles) bits.push(roles);
        if (person.low || person.high) bits.push((person.low || "Any") + "–" + (person.high || "Any"));
        if (person.agent) bits.push(person.agent);
        if (person.games) bits.push(person.games + " games");
        if (rate.n) bits.push(Number(rate.avg).toFixed(1) + " from " + rate.n);
        if (person.phase && person.phase !== "In menu" && person.phase !== "In client") bits.push(person.phase + (person.map ? " · " + person.map : ""));
        return bits.join(" · ");
      }
      function partyRecord(person) {
        const cut = String(person.riot || "").split("#");
        if (cut.length < 2 || !cut[1]) return "";
        let games = 0;
        let wins = 0;
        let last = "";
        for (const match of history || []) {
          const hit = (match.players || []).find((player) => String(player.name || "").toLowerCase() === cut[0].toLowerCase() && String(player.tag || "").toLowerCase() === cut[1].toLowerCase());
          if (!hit) continue;
          games += 1;
          if (match.won === true) wins += 1;
          if (!last) last = match.map || "";
        }
        if (!games) return "No recent games together";
        return games + " together · " + wins + "–" + (games - wins) + (last ? " · last " + last : "");
      }
      function inviteText() {
        return [riotID(), loadRoles().join(", "), loadModes().join(", "), ensureCode()].filter(Boolean).join(" · ");
      }
      function paintChipList(rootId, names, key) {
        const root = document.getElementById(rootId);
        if (!root) return;
        const picked = new Set(loadList(key));
        root.replaceChildren();
        for (const name of names) {
          const button = el("button", picked.has(name) ? "on" : "", name);
          button.type = "button";
          button.onclick = () => {
            const next = new Set(loadList(key));
            if (next.has(name)) next.delete(name);
            else next.add(name);
            localStorage.setItem(key, JSON.stringify(names.filter((mode) => next.has(mode))));
            paintChipList(rootId, names, key);
            pushRoom(lookingOn());
          };
          root.append(button);
        }
      }
      function paintModes() {
        paintChipList("modes", PLAY_MODES, "valvet-modes");
        paintChipList("roles", PLAY_ROLES, "valvet-roles");
      }
      async function addByCode(raw) {
        const code = String(raw || "").toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 12);
        if (code.length < 4 || code === ensureCode()) return;
        let person = null;
        try {
          const rows = (await (await fetch("/api/room?code=" + encodeURIComponent(code), { cache: "no-store" })).json()).people || [];
          person = rows.find((row) => row.id && row.id !== valvetID() && row.room === code) || rows.find((row) => row.id && row.id !== valvetID());
        } catch {}
        addWith(person || { id: "code:" + code, nick: code, room: code, riot: "" });
      }
      function addWith(person) {
        if (!person || !person.id || !person.room || person.id === valvetID()) return;
        const rows = loadWith().filter((row) => row.id !== person.id);
        rows.push({ id: person.id, nick: person.riot || person.nick || "Player", room: person.room });
        saveWith(rows);
        refreshRoom();
        pushRoom(lookingOn());
      }
      function dropWith(id) {
        saveWith(loadWith().filter((row) => row.id !== id));
        refreshRoom();
      }
      function roomHit(player) {
        if (!player || player.you) return false;
        const full = ((player.name || "") + (player.tag ? "#" + player.tag : "")).toLowerCase();
        const name = String(player.name || "").toLowerCase();
        return roomWatches.has(full) || (name && roomWatches.has(name));
      }
      function syncRoomHits(data) {
        for (const player of (data && data.players) || []) {
          if (!roomHit(player)) continue;
          const key = ((player.name || "") + "#" + (player.tag || "")).toLowerCase();
          if (roomPinged.has(key)) continue;
          roomPinged.add(key);
          roomPing = (player.name || "Player") + (player.tag ? "#" + player.tag : "");
          if (!quietNow()) playPack();
        }
      }
      function loadBlock() {
        try {
          const rows = JSON.parse(localStorage.getItem("valvet-block") || "[]");
          return Array.isArray(rows) ? rows : [];
        } catch { return []; }
      }
      function blockPerson(id) {
        const rows = loadBlock().filter((row) => row !== id);
        rows.push(id);
        localStorage.setItem("valvet-block", JSON.stringify(rows.slice(-80)));
        refreshRoom();
      }
      function myParty() {
        const party = (latestState && latestState.party) || [];
        const me = party.find((person) => person.you) || {};
        const size = Number(me.partySize || party.length || 1) || 1;
        const max = Number(me.maxParty || 5) || 5;
        return { size, max, full: size >= max && max > 1 };
      }
      const duoSeen = new Map();
      function paintPeople(root, rows, empty, mode) {
        if (!root) return;
        root.className = rows.length ? "duo-list" : "duo-list empty";
        root.replaceChildren();
        if (!rows.length) {
          root.textContent = empty;
          return;
        }
        const added = new Set(loadWith().map((row) => row.id));
        for (const person of rows) {
          const board = mode === "board";
          const changed = duoSeen.has(person.id) && duoSeen.get(person.id) !== ((person.ready ? "1" : "0") + "|" + (person.score || ""));
          duoSeen.set(person.id, (person.ready ? "1" : "0") + "|" + (person.score || ""));
          const row = el("div", "duo-person" + (board ? " board" : "") + (person.ready ? " ready" : "") + (person.full ? " full" : "") + (changed ? " pulse" : ""));
          const left = el("div");
          const top = el("div", "duo-top");
          top.append(el("b", null, person.riot || person.nick || "Player"));
          if (person.ready) top.append(el("span", "pill", "Ready"));
          if (!board && person.full) top.append(el("span", "pill", "Full party"));
          else if (!board && person.party) top.append(el("span", "pill", person.party));
          if (person.rank) top.append(el("span", "pill", person.rank));
          left.append(top);
          if (board) {
            if (person.note) left.append(el("div", "sub", person.note));
            const facts = boardFacts(person);
            if (facts) left.append(el("div", "sub", facts));
            const together = partyRecord(person);
            if (together && together.indexOf("together") !== -1) left.append(el("div", "sub", together));
            if (String(person.riot || "").includes("#") && !isMine(person)) {
              const given = Number(loadGiven()[String(person.riot).toLowerCase()] || 0);
              const stars = el("div", "rate-row");
              for (let n = 1; n <= 5; n++) {
                const star = el("button", "tiny" + (given === n ? " on" : ""), String(n));
                star.type = "button";
                star.title = "Rate after you played with them";
                star.onclick = () => ratePerson(person, n);
                stars.append(star);
              }
              left.append(stars);
            }
          } else {
            const facts = [person.agent, (person.roles || []).join(" "), (person.modes || []).join(" "), person.server, person.phase, person.map, person.score].filter(Boolean).join(" · ");
            if (facts) left.append(el("div", "sub", facts));
          }
          if (person.discord) {
            const chip = el("button", "pill discord", person.discord);
            chip.type = "button";
            chip.onclick = () => navigator.clipboard.writeText(person.discord).catch(() => {});
            left.append(chip);
          }
          if (mode === "drop") {
            const record = partyRecord(person);
            if (record) left.append(el("div", "sub", record));
          }
          row.append(left);
          const actions = el("div");
          if ((mode === "add" || mode === "board") && !person.full && !added.has(person.id)) {
            const button = el("button", "tiny", "Add");
            button.type = "button";
            button.onclick = () => addWith(person);
            actions.append(button);
          }
          if (mode === "add" || mode === "board") {
            const block = el("button", "tiny", "Block");
            block.type = "button";
            block.onclick = () => blockPerson(person.id);
            actions.append(block);
          }
          if (mode === "drop") {
            const button = el("button", "tiny", "Remove");
            button.type = "button";
            button.onclick = () => dropWith(person.id);
            actions.append(button);
          }
          if (actions.childElementCount) row.append(actions);
          root.append(row);
        }
      }
      function paintRoom() {
        const code = ensureCode();
        const node = document.getElementById("my-code");
        if (node) node.textContent = node.dataset.open === "1" ? code : "········";
        const other = document.getElementById("my-code-2");
        if (other) other.textContent = code;
        const hero = document.getElementById("duo-you");
        if (hero) {
          const riot = riotID();
          const bits = [riot || "Open Valorant to show your Riot ID"];
          if (lookingOn()) bits.push("posted");
          if (localStorage.getItem("valvet-share-discord") === "on" && localStorage.getItem("valvet-discord-name")) bits.push("Discord on the post");
          hero.textContent = bits.join(" · ");
        }
        const mine = valvetID();
        const live = roomPeople.filter((person) => person.id !== mine);
        const others = loadWith().map((row) => live.find((person) => person.id === row.id || person.room === row.room) || { id: row.id, nick: row.nick, room: row.room, phase: "Offline" });
        paintPeople(document.getElementById("room-people"), others, "Nobody added yet.", "drop");
        const phase = (latestState && latestState.phase) || "";
        const duo = others.filter((person) => person.phase && person.phase === phase && phase !== "In menu" && phase !== "In client");
        const duoRoot = document.getElementById("room-duo");
        if (duoRoot) {
          duoRoot.className = duo.length ? "" : "empty";
          duoRoot.textContent = duo.length ? duo.map((person) => person.nick + " is also " + person.phase).join(" · ") : "Nobody you added is queueing with you yet.";
        }
        const mineWish = new Set(loadWish().map((name) => name.toLowerCase()));
        const owned = new Set(ownedNames.map((name) => name.toLowerCase()));
        const lines = [];
        for (const person of others) {
          for (const name of person.wish || []) {
            if (!mineWish.has(name.toLowerCase())) continue;
            const bits = [name, person.nick];
            if (owned.has(name.toLowerCase())) bits.push("you own it");
            if ((person.owned || []).some((item) => item.toLowerCase() === name.toLowerCase())) bits.push("they own it");
            lines.push(bits.join(" · "));
          }
        }
        const wishRoot = document.getElementById("room-wishes");
        if (wishRoot) {
          wishRoot.className = lines.length ? "duo-list" : "duo-list empty";
          wishRoot.replaceChildren();
          if (!lines.length) wishRoot.textContent = "Wishes you and someone you added both want show up here.";
          else for (const line of lines) wishRoot.append(el("div", "room-card", line));
        }
        const status = document.getElementById("duo-status");
        if (status) {
          const mine = lfgPeople.find((person) => person.id === valvetID());
          if (!lookingOn()) status.textContent = "You have not posted yet.";
          else if (roomPostOk === false) status.textContent = "The board rejected the last post. Leave this open. Valvet tries again.";
          else if (mine) status.textContent = "Your post is on the board.";
          else status.textContent = "Posted. Waiting for the board to show it.";
        }
        paintPeople(document.getElementById("room-lfg"), lfgPeople.filter(showInFind), "Nobody has posted yet.", "board");
        paintLeaders();
        const watches = new Set();
        for (const person of others) for (const name of person.watch || []) watches.add(String(name).toLowerCase());
        roomWatches = watches;
      }
      function paintLeaders() {
        const root = document.getElementById("room-leaders");
        if (!root) return;
        const rows = lfgPeople.filter((person) => person.looking).map((person) => {
          const rate = rateFor(person);
          return { person, avg: Number(rate.avg) || 0, n: Number(rate.n) || 0, games: Number(person.games) || 0 };
        }).filter((row) => row.n > 0 && row.avg >= 4).sort((a, b) => b.games - a.games || b.avg - a.avg || String(a.person.riot || a.person.nick || "").localeCompare(String(b.person.riot || b.person.nick || "")));
        root.className = rows.length ? "duo-list" : "duo-list empty";
        root.replaceChildren();
        if (!rows.length) {
          root.textContent = "No one with a 4 or 5 average is on the board yet.";
          return;
        }
        rows.forEach((row, index) => {
          const line = el("div", "duo-person board");
          const left = el("div");
          left.append(el("b", null, (index + 1) + ". " + (row.person.riot || row.person.nick || "Player")));
          left.append(el("div", "sub", row.games + " games · " + row.avg.toFixed(1) + " from " + row.n + (row.n === 1 ? " rating" : " ratings")));
          line.append(left);
          root.append(line);
        });
      }
      async function ratePerson(person, stars) {
        if (isMine(person)) return;
        const target = String(person.riot || "").trim();
        if (!target.includes("#")) return;
        const bag = loadGiven();
        bag[target.toLowerCase()] = stars;
        localStorage.setItem("valvet-given-rates", JSON.stringify(bag));
        paintRoom();
        try {
          await fetch("/api/rate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ from: valvetID(), riot: myRiot(), target, stars }) });
        } catch {}
        lastRatePush = Date.now();
        refreshRoom(true);
      }
      async function refreshRoom() {
        return;
      }
      let roomPostOk = null;
      let roomWasLive = false;
      let lastPrivateAt = 0;
      let lastPrivateSig = "";
      let lastBoardAt = 0;
      let lastBoardSig = "";
      let lastRatePush = 0;
      async function sendRoom(person) {
        try {
          const res = await fetch("/api/room", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(person) });
          if (!res.ok) return false;
          const data = await res.json();
          return !!data.ok;
        } catch {
          return false;
        }
      }
      function currentPerson(looking) {
        const state = latestState || {};
        const you = (state.players || []).find((player) => player.you);
        return {
          id: valvetID(),
          nick: valvetNick(),
          game: "valorant",
          phase: state.phase || "In menu",
          map: state.map || "",
          mode: state.mode || "",
          score: state.score1 != null && state.score2 != null ? state.score1 + "-" + state.score2 : "",
          rank: localStorage.getItem("valvet-hide-rank") === "on" ? "" : ((you && you.rank) || ""),
          riot: riotID(),
          agent: localStorage.getItem("valvet-agent") || "",
          party: myParty().size + "/" + myParty().max,
          full: myParty().full,
          ready: false,
          looking: !!looking,
          board: true,
          room: ensureCode(),
          modes: loadModes(),
          roles: loadRoles(),
          server: localStorage.getItem("valvet-server") || "",
          low: localStorage.getItem("valvet-rank-low") || "",
          high: localStorage.getItem("valvet-rank-high") || "",
          discord: localStorage.getItem("valvet-share-discord") === "on" ? (localStorage.getItem("valvet-discord-name") || "") : "",
          until: Number(localStorage.getItem("valvet-until") || 0),
          wish: loadWish().slice(0, 8),
          watch: (watch || []).map((item) => item.name + "#" + item.tag).slice(0, 8),
          owned: [],
          ping: "",
          age: boardAge(),
          gender: localStorage.getItem("valvet-gender") || "",
          note: localStorage.getItem("valvet-note") || "",
          queue: localStorage.getItem("valvet-queue") || "",
          games: savedGames(),
        };
      }
      async function pushRates(force) {
        const bag = loadGiven();
        const keys = Object.keys(bag);
        if (!keys.length) return;
        const now = Date.now();
        if (!force && now - lastRatePush < 20 * 60 * 1000) return;
        lastRatePush = now;
        const mine = myRiot().trim().toLowerCase();
        for (const target of keys) {
          const stars = Number(bag[target]);
          if (!stars) continue;
          if (mine && target.toLowerCase() === mine) continue;
          await fetch("/api/rate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ from: valvetID(), riot: myRiot(), target, stars }) }).catch(() => {});
        }
      }
      async function pushRoom(force) {
        const posted = lookingOn();
        const sharing = loadWith().length > 0;
        if (!posted && !sharing && !roomWasLive && !force) return;
        const person = currentPerson(posted);
        const now = Date.now();
        const privateSig = [person.phase, person.map, person.score, person.riot, person.party].join("|");
        let doPrivate = sharing || roomWasLive;
        if (doPrivate && !force && privateSig === lastPrivateSig && now - lastPrivateAt < 4 * 60 * 1000) doPrivate = false;
        if (doPrivate && !force && now - lastPrivateAt < 15000) doPrivate = false;
        if (!doPrivate) return;
        lastPrivateAt = now;
        lastPrivateSig = privateSig;
        const quiet = { ...person, looking: false, board: false, note: "", age: 0, gender: "", ping: "" };
        await sendRoom(quiet);
        for (const friend of loadWith()) {
          if (!friend.room || friend.room === person.room) continue;
          await sendRoom({ ...quiet, room: friend.room });
        }
        roomWasLive = posted || sharing;
        paintRoom();
      }
      async function loadPatches(live) {
        const root = document.getElementById("patches");
        const key = "valvet-cache-patches";
        const saved = readJSON(key);
        const ok = Array.isArray(saved) && saved.length;
        if (ok && !(root.dataset.ready === "1" && root.childElementCount)) {
          paintPatchList(saved);
          root.dataset.ready = "1";
        } else if (ok) {
          const label = root.querySelector(".saved-bar span");
          if (label) label.textContent = savedAgo("valvet-cache-patches") || "Not saved yet";
        } else if (!ok) {
          root.className = "";
          root.replaceChildren(el("div", "empty", "Loading patch notes…"));
        }
        const force = live === "force";
        const need = force || !ok || !cacheFresh(key, 24 * 60 * 60 * 1000);
        if (!need) return;
        try {
          const data = await (await fetch("/api/patches", { cache: "no-store" })).json();
          const notes = data.notes || [];
          if (!notes.length) {
            if (!ok) root.replaceChildren(el("div", "empty", data.error || "No patch notes yet."));
            return;
          }
          writeJSON(key, notes);
          stampCache(key);
          if (force || JSON.stringify(saved) !== JSON.stringify(notes)) {
            paintPatchList(notes);
            root.dataset.ready = "1";
          }
        } catch {
          if (!ok) root.replaceChildren(el("div", "empty", "Couldn't load patch notes."));
        }
      }
      let patchMine = false;
      function paintPatchList(notes) {
        const root = document.getElementById("patches");
        const allNotes = notes || [];
        const newest = allNotes[0] || {};
        markFresh("patches", [newest.title, newest.date, newest.id].filter(Boolean).join("|"));
        root.className = "";
        root.replaceChildren();
        savedBar(root, "valvet-cache-patches", () => loadPatches("force"));
        const bar = el("div", "modes");
        const all = el("button", patchMine ? "" : "on", "All");
        const mine = el("button", patchMine ? "on" : "", "My agents");
        all.type = "button";
        mine.type = "button";
        all.onclick = () => { patchMine = false; paintPatchList(allNotes); };
        mine.onclick = () => { patchMine = true; paintPatchList(allNotes); };
        bar.append(all, mine);
        root.append(bar);
        const agents = myAgentList();
        if (patchMine) {
          const pick = el("div", "agent-picks");
          const chosen = new Set(agents);
          for (const name of Object.keys(AGENT_LIGHT)) {
            const button = el("button", chosen.has(name) ? "on" : "", name);
            button.type = "button";
            button.onclick = () => {
              const next = new Set(myAgentList());
              if (next.has(name)) next.delete(name);
              else next.add(name);
              writeJSON("valvet-my-agents", [...next]);
              paintPatchList(allNotes);
            };
            pick.append(button);
          }
          root.append(pick);
        }
        const rows = patchMine ? allNotes.filter((note) => agents.some((name) => String((note.title || "") + " " + (note.blurb || "") + " " + (note.body || "")).toLowerCase().includes(name.toLowerCase()))) : allNotes;
        root.append(el("h2", null, (patchMine ? "Your agents · " : "Patch notes · ") + rows.length));
        if (patchMine && !agents.length) {
          root.append(el("div", "empty", "Pick the agents you play."));
          return;
        }
        if (!rows.length) {
          root.append(el("div", "empty", "No patch notes for those agents."));
          return;
        }
        const grid = el("div", "patch-grid");
        for (const note of rows) {
          const card = el("button", "patch-card");
          card.type = "button";
          if (note.image) {
            const img = document.createElement("img");
            img.alt = "";
            img.src = note.image;
            card.append(img);
          }
          const body = el("div");
          body.append(el("div", "eyebrow", note.date || "Patch"));
          body.append(el("b", null, note.title || "Patch notes"));
          if (note.blurb) body.append(el("div", "sub", note.blurb));
          card.append(body);
          card.onclick = () => openPatch(note);
          grid.append(card);
        }
        root.append(grid);
      }
      async function openPatch(note) {
        const root = document.getElementById("patches");
        root.className = "";
        root.replaceChildren(el("div", "empty", "Loading " + (note.title || "patch") + "…"));
        try {
          const data = await (await fetch("/api/patch?path=" + encodeURIComponent(note.path || ""), { cache: "no-store" })).json();
          root.replaceChildren();
          const top = el("div", "patch-top");
          const back = el("button", "tiny", "All patches");
          back.type = "button";
          back.onclick = () => { root.dataset.ready = ""; loadPatches(true); };
          const link = el("button", "tiny", "Open on playvalorant.com");
          link.type = "button";
          link.onclick = () => fetch("/api/open?url=" + encodeURIComponent("https://playvalorant.com" + note.path)).catch(() => {});
          top.append(back, link);
          root.append(top);
          const read = el("div", "patch-read");
          read.append(el("div", "eyebrow", note.date || ""));
          read.append(el("h2", null, data.title || note.title || "Patch notes"));
          const text = data.body || data.error || note.blurb || "";
          for (const line of String(text).split("\n")) {
            const clean = line.trim();
            if (!clean) continue;
            if (clean.startsWith("§")) read.append(el("h3", null, clean.replace(/^§\s*/, "")));
            else if (clean.startsWith("•")) read.append(el("p", "bullet", clean));
            else read.append(el("p", null, clean));
          }
          root.append(read);
        } catch {
          root.replaceChildren(el("div", "empty", "Couldn't open that patch."));
        }
      }
      function sportCard(item, live) {
        const card = el("article", "sport-card");
        if (item.icon) {
          const img = document.createElement("img");
          img.alt = "";
          img.loading = "lazy";
          img.decoding = "async";
          img.src = item.icon;
          card.append(img);
        }
        const body = el("div");
        body.append(el("b", null, item.name || "Bundle"));
        const price = el("div", "price");
        if (live) {
          if (item.was && item.was !== item.price) price.append(el("s", null, item.was + " VP"));
          price.append(document.createTextNode((item.price || "—") + (item.price ? " VP" : "")));
          if (item.left) price.append(document.createTextNode(" · " + item.left));
        } else price.textContent = "Not in the store";
        body.append(price);
        if (live && item.offer && item.price) {
          const buy = el("button", "tiny", "Buy");
          buy.type = "button";
          buy.onclick = async () => {
            if (buy.dataset.arm !== "1") { buy.dataset.arm = "1"; buy.textContent = "Confirm"; return; }
            buy.textContent = "Buying…";
            try {
              const data = await (await fetch("/api/buy", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ offer: item.offer, price: item.price, currency: "VP" }) })).json();
              buy.textContent = data.error || "Owned";
              if (!data.error) rememberOwned(item.name);
            } catch { buy.textContent = "Failed"; }
          };
          body.append(buy);
        }
        card.append(body);
        card.onclick = async (event) => {
          if (event.target.closest("button")) return;
          const open = card.querySelector(".items");
          if (open) { open.remove(); return; }
          const list = el("div", "items");
          list.append(el("div", "sub", "Loading what's inside…"));
          body.append(list);
          let pieces = item.items || [];
          if (pieces.length < 2) {
            try {
              const data = await (await fetch("/api/esports/items?name=" + encodeURIComponent(item.name || ""), { cache: "no-store" })).json();
              if ((data.items || []).length) pieces = data.items;
            } catch {}
          }
          list.replaceChildren();
          if (item.desc) list.append(el("p", "sub", item.desc));
          if (!pieces.length) list.append(el("div", "sub", "No item list for this one yet."));
          for (const piece of pieces) list.append(el("div", "sub", [piece.currency, piece.name || "Item"].filter(Boolean).join(" · ")));
        };
        return card;
      }
      let esportCache = null;
      function esportKey(data) {
        if (!data) return "";
        const offers = (data.offers || []).map((item) => (item.name || "") + ":" + (item.price || 0) + ":" + (item.offer || "")).join(",");
        const catalog = (data.catalog || []).map((item) => item.name || "").join(",");
        const events = (data.events || []).map((item) => (item.name || "") + ":" + (item.status || "")).join(",");
        return offers + "|" + catalog + "|" + events;
      }
      async function loadEsports(live) {
        const pane = document.getElementById("pane-esports");
        if (live !== "force" && (tabHidden("esports") || !pane || pane.hidden)) return;
        const root = document.getElementById("esports");
        const key = "valvet-cache-esports";
        if (!esportCache) {
          const saved = readJSON(key);
          if (saved && ((saved.events || []).length || (saved.offers || []).length || (saved.catalog || []).length)) esportCache = saved;
        }
        const ok = !!(esportCache && ((esportCache.events || []).length || (esportCache.offers || []).length || (esportCache.catalog || []).length));
        if (ok && root.dataset.sig !== esportKey(esportCache)) paintEsports(esportCache);
        else if (ok) {
          const label = root.querySelector(".saved-bar span");
          if (label) label.textContent = savedAgo("valvet-cache-esports") || "Not saved yet";
        } else if (!ok && !root.querySelector(".sport-card, .event-card")) {
          root.className = "";
          root.replaceChildren(el("div", "empty", "Loading the esports store…"));
        }
        const force = live === "force";
        const need = force || !ok || !cacheFresh(key, 6 * 60 * 60 * 1000);
        if (!need) return;
        if (root.dataset.fetching === "1") return;
        root.dataset.fetching = "1";
        try {
          const data = await (await fetch("/api/esports", { cache: "no-store" })).json();
          const next = esportKey(data);
          const prev = esportKey(esportCache);
          if ((data.catalog || []).length || (data.offers || []).length || (data.events || []).length) {
            esportCache = data;
            const slim = {
              offers: data.offers || [],
              events: data.events || [],
              catalog: (data.catalog || []).map((item) => ({ name: item.name, icon: item.icon })),
            };
            writeJSON(key, slim);
            stampCache(key);
          }
          if ((force || (next && next !== prev)) && ((data.catalog || []).length || (data.offers || []).length || (data.events || []).length)) paintEsports(data);
          else if (!esportCache) root.replaceChildren(el("div", "empty", data.error || "Couldn't load the esports store."));
        } catch {
          if (!esportCache) root.replaceChildren(el("div", "empty", "Couldn't load the esports store."));
        } finally {
          root.dataset.fetching = "";
        }
      }
      function paintEsports(data) {
        const root = document.getElementById("esports");
        root.className = "";
        const offers = data.offers || [];
        const catalog = data.catalog || [];
        root.replaceChildren();
        savedBar(root, "valvet-cache-esports", () => loadEsports("force"));
          const events = (data.events || []).filter((event) => {
            const s = String(event.status || "").toLowerCase();
            if (!s || /inactive|complet|finish|cancel|past/.test(s)) return false;
            return /ongo|live|upcom|soon|progress|running/.test(s) || s === "active" || s === "not started";
          });
          if (events.length) {
            root.append(el("h2", null, "Events"));
            const row = el("div", "event-row");
            const prev = el("button", "event-arrow", "‹");
            const next = el("button", "event-arrow", "›");
            prev.type = "button";
            next.type = "button";
            prev.setAttribute("aria-label", "Previous");
            next.setAttribute("aria-label", "Next");
            const frame = el("div", "event-window");
            const rail = el("div", "event-rail");
            const liveNow = (status) => {
              const s = String(status || "").toLowerCase();
              return /ongo|live|progress|running/.test(s) || s === "active";
            };
            for (const event of events) {
              const card = el("article", "event-card");
              if (event.image) {
                const img = document.createElement("img");
                img.alt = "";
                img.src = event.image;
                card.append(img);
              }
              const box = el("div");
              if (liveNow(event.status)) box.append(el("div", "live-pip", "Live"));
              else if (event.dates) box.append(el("div", "date-chip", event.dates));
              box.append(el("b", null, event.name));
              card.append(box);
              if (event.url) {
                card.onclick = () => fetch("/api/open?url=" + encodeURIComponent(event.url)).catch(() => {});
              }
              rail.append(card);
            }
            frame.append(rail);
            row.append(prev, frame, next);
            let index = 0;
            const step = () => {
              const width = frame.clientWidth || 1;
              const card = 272;
              const visible = Math.max(1, Math.floor((width + 12) / card));
              const max = Math.max(0, events.length - visible);
              index = Math.max(0, Math.min(index, max));
              rail.style.transform = "translateX(" + (-index * card) + "px)";
              prev.disabled = index <= 0;
              next.disabled = index >= max;
            };
            prev.onclick = () => { index -= 1; step(); };
            next.onclick = () => { index += 1; step(); };
            root.append(row);
            requestAnimationFrame(step);
          } else {
            root.append(el("h2", null, "Events"));
            root.append(el("div", "empty", "No active or upcoming events."));
          }
          root.append(el("h2", null, "In the store"));
          root.append(el("p", "sub", "Team capsules and esports bundles you can buy right now."));
          if (!offers.length) root.append(el("div", "empty", "Nothing is for sale in the esports store right now."));
          else {
            const grid = el("div", "sport-grid");
            for (const item of offers) grid.append(sportCard(item, true));
            root.append(grid);
          }
          root.append(el("h2", null, "All esports bundles"));
          const search = document.createElement("input");
          search.className = "wish-search";
          search.placeholder = "Search teams";
          const grid = el("div", "sport-grid");
          let shown = 24;
          const paint = () => {
            const q = search.value.trim().toLowerCase();
            grid.replaceChildren();
            const rows = catalog.filter((item) => !q || String(item.name || "").toLowerCase().includes(q));
            if (!rows.length) {
              grid.append(el("div", "empty", catalog.length ? "No bundle matches." : "The esports list hasn't loaded."));
              return;
            }
            const limit = Math.min(rows.length, q ? Math.max(shown, 48) : shown);
            for (const item of rows.slice(0, limit)) grid.append(sportCard(item, false));
            if (rows.length > limit) {
              const more = el("button", "tiny solid sport-more", "Show more");
              more.type = "button";
              more.onclick = (event) => { event.preventDefault(); event.stopPropagation(); shown += 24; paint(); };
              grid.append(more);
            }
          };
          search.oninput = () => { shown = 24; paint(); };
          root.append(search, grid);
          paint();
          root.dataset.sig = esportKey(data);
      }
      const BIG_NOTES = new Set(["1.7.13", "1.7.22", "1.7.28", "1.7.42", "1.8.0"]);
      function visibleNotes() {
        const seen = new Set();
        const out = [];
        for (const note of [...APP_NOTES.slice(0, 5), ...APP_NOTES.filter((row) => row.big || BIG_NOTES.has(row.ver))]) {
          if (!note || seen.has(note.ver)) continue;
          seen.add(note.ver);
          out.push(note);
        }
        return out;
      }
      const ROLE_COLOR = { Duelist: "#ff5a6a", Initiator: "#5ec8ff", Controller: "#c9a6ff", Sentinel: "#3dce86" };
      const NOTES_VER = "__VALVET__";
      const APP_NOTES = [
        { ver: "1.9.23", title: "Valvet", lines: ["Live ranks and every party load together, without waiting on each other."] },
        { ver: "1.9.16", title: "Valvet", lines: ["When a match is found, the live lobby shows again. Your side is on the left and theirs is on the right."] },
        { ver: "1.8.43", title: "Valvet", lines: ["History filters use the dark menu, and a name in the search finds that player's games.", "Each map shows attack and defense. Two players can sit side by side.", "A bundle says how many pieces you own. The other team's agents copy as one line.", "Drop an mp3 on a sound row. Clutches split into 1v1, 1v2, and 1v3.", "Backup is one file. Parties stay on when stat lookups are paused."] },
        { ver: "1.8.34", title: "Valvet", lines: ["The live board is the same as before. Names, agent, party, and rank. No KDA on the rows."] },
        { ver: "1.8.33", title: "Valvet", lines: ["The window title is VAL in white and VET in pink, same as the sidebar.", "Check for update is under Settings, Backup and feedback.", "Your name on the left opens your profile. Party marks stay on stacks of 2 to 5."] },
        { ver: "1.8.32", title: "Valvet", lines: ["The name is VALVET again. VAL is white, VET is pink, and the window title is pink.", "Play the round saves an mp4. The folder stays closed.", "Storage opens the round videos folder.", "A profile shows the rank and the agent picture without waiting on every game."] },
      ];
      function mapName(name) {
        const raw = String(name || "").trim();
        if (!raw) return "";
        const bit = raw.split(/[\\/]/).filter(Boolean).pop() || raw;
        const known = __MAPS__;
        return known[bit.toLowerCase()] || raw;
      }
      function actParts(iso) {
        const end = Date.parse(iso || "");
        if (!end) return null;
        let ms = end - Date.now();
        if (ms <= 0) return { d: 0, h: 0, m: 0, over: true };
        const d = Math.floor(ms / 86400000);
        ms -= d * 86400000;
        const h = Math.floor(ms / 3600000);
        ms -= h * 3600000;
        const m = Math.floor(ms / 60000);
        return { d, h, m };
      }
      function paintActClock(node, iso) {
        if (!node) return;
        const parts = actParts(iso);
        node.className = "pass-clock";
        node.replaceChildren();
        if (!parts) return;
        if (parts.over) {
          node.append(el("span", null, "Act ended"));
          return;
        }
        for (const [n, label] of [[parts.d, "days"], [parts.h, "hours"], [parts.m, "min"]]) {
          const chip = el("i");
          chip.append(el("b", null, String(n)));
          chip.append(el("span", null, label));
          node.append(chip);
        }
      }
      function actLeft(iso) {
        const end = Date.parse(iso || "");
        if (!end) return "";
        let ms = end - Date.now();
        if (ms <= 0) return "Act ended";
        const d = Math.floor(ms / 86400000);
        ms -= d * 86400000;
        const h = Math.floor(ms / 3600000);
        ms -= h * 3600000;
        const m = Math.floor(ms / 60000);
        return d + "d " + h + "h " + m + "m left";
      }
      function dayStart(ts) {
        const d = new Date(ts);
        if (d.getHours() < 6) d.setDate(d.getDate() - 1);
        d.setHours(6, 0, 0, 0);
        return d.getTime();
      }
      function clockSpan(ms) {
        const n = Math.max(0, Math.round(ms / 60000));
        const h = Math.floor(n / 60);
        const m = n % 60;
        return (h ? h + "h " : "") + m + "m";
      }
      function hsRing(pct) {
        const n = Math.max(0, Math.min(100, Number(pct) || 0));
        const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        svg.setAttribute("viewBox", "0 0 36 36");
        svg.setAttribute("class", "hs-ring");
        const track = document.createElementNS("http://www.w3.org/2000/svg", "circle");
        track.setAttribute("cx", "18");
        track.setAttribute("cy", "18");
        track.setAttribute("r", "14");
        track.setAttribute("fill", "none");
        track.setAttribute("stroke", "rgba(255,255,255,.16)");
        track.setAttribute("stroke-width", "3");
        const arc = document.createElementNS("http://www.w3.org/2000/svg", "circle");
        const c = 2 * Math.PI * 14;
        arc.setAttribute("cx", "18");
        arc.setAttribute("cy", "18");
        arc.setAttribute("r", "14");
        arc.setAttribute("fill", "none");
        arc.setAttribute("stroke", "#ff5fa2");
        arc.setAttribute("stroke-width", "3");
        arc.setAttribute("stroke-linecap", "butt");
        arc.setAttribute("stroke-dasharray", (c * n / 100).toFixed(2) + " " + (c * (1 - n / 100)).toFixed(2));
        arc.setAttribute("transform", "rotate(-90 18 18)");
        const label = document.createElementNS("http://www.w3.org/2000/svg", "text");
        label.setAttribute("x", "18");
        label.setAttribute("y", "19");
        label.setAttribute("text-anchor", "middle");
        label.setAttribute("dominant-baseline", "middle");
        label.setAttribute("fill", "#fff7fa");
        label.setAttribute("font-size", "8");
        label.setAttribute("font-weight", "700");
        label.textContent = n ? n + "" : "–";
        svg.append(track, arc, label);
        return svg;
      }
      function deskClock(ms) {
        const face = el("div", "session-hero");
        const copy = el("div");
        copy.append(el("div", "eyebrow", "Today"));
        copy.append(el("b", "time", clockSpan(ms)));
        face.append(copy);
        return face;
      }
      function myAgentList() {
        const saved = readJSON("valvet-my-agents");
        return Array.isArray(saved) ? saved.map((name) => String(name || "")).filter(Boolean) : [];
      }
      function agentRows() {
        const rec = latestState && latestState.records;
        if (rec && (rec.agents || []).length) return rec.agents;
        const book = {};
        for (const match of history || []) {
          const you = (match.players || []).find((player) => player.you);
          if (!you || !you.agent) continue;
          const row = book[you.agent] || { name: you.agent, wins: 0, losses: 0, games: 0, hsSum: 0, hsN: 0, acsSum: 0, acsN: 0, kills: 0, deaths: 0 };
          row.games += 1;
          if (match.won === true) row.wins += 1;
          if (match.won === false) row.losses += 1;
          row.kills += Number(you.kills || 0);
          row.deaths += Number(you.deaths || 0);
          const hs = parseInt(you.hs, 10);
          if (!Number.isNaN(hs)) { row.hsSum += hs; row.hsN += 1; }
          const acs = parseInt(you.acs, 10);
          if (!Number.isNaN(acs)) { row.acsSum += acs; row.acsN += 1; }
          book[you.agent] = row;
        }
        return Object.values(book).map((row) => ({
          name: row.name, wins: row.wins, losses: row.losses, games: row.games, kills: row.kills, deaths: row.deaths,
          hs: row.hsN ? Math.round(row.hsSum / row.hsN) : 0,
          acs: row.acsN ? Math.round(row.acsSum / row.acsN) : 0,
        })).sort((a, b) => b.games - a.games);
      }
      function mapRows() {
        const book = {};
        for (const match of history || []) {
          const name = mapName(match.map);
          if (!name || (match.won !== true && match.won !== false)) continue;
          const row = book[name] || { name, wins: 0, losses: 0, games: 0, atkW: 0, atkL: 0, defW: 0, defL: 0 };
          row.games += 1;
          if (match.won) row.wins += 1;
          else row.losses += 1;
          for (const round of match.rounds || []) {
            if (round.you !== true && round.you !== false) continue;
            const side = String(round.side || "");
            if (/attack/i.test(side)) {
              if (round.you) row.atkW += 1;
              else row.atkL += 1;
            } else if (/defen/i.test(side)) {
              if (round.you) row.defW += 1;
              else row.defL += 1;
            }
          }
          book[name] = row;
        }
        const rows = Object.values(book).sort((a, b) => b.games - a.games);
        if (rows.length) return rows;
        const rec = latestState && latestState.records;
        return (rec && rec.maps) || [];
      }
      function friendBook() {
        const ids = new Set();
        const names = new Set();
        for (const id of (latestState && latestState.friendIds) || []) {
          const key = String(id || "").toLowerCase();
          if (key) ids.add(key);
        }
        for (const person of (latestState && latestState.friends) || []) {
          const id = String(person.puuid || "").toLowerCase();
          if (id) ids.add(id);
          if (person.name) names.add(((person.name || "") + "#" + (person.tag || "")).toLowerCase());
        }
        return { ids, names };
      }
      function isFriendRow(row) {
        const book = friendBook();
        if (!book.ids.size && !book.names.size) return false;
        const id = String((row && row.puuid) || "").toLowerCase();
        if (id && book.ids.has(id)) return true;
        return book.names.has((((row && row.name) || "") + "#" + ((row && row.tag) || "")).toLowerCase());
      }
      function duoRows() {
        const rec = latestState && latestState.records;
        if (rec && (rec.duos || []).length) return rec.duos;
        const book = {};
        for (const match of history || []) {
          const you = (match.players || []).find((player) => player.you);
          if (!you) continue;
          for (const player of match.players || []) {
            if (!player || player.you || player.enemy) continue;
            if (you.team && player.team && player.team !== you.team) continue;
            if (!isFriendRow(player)) continue;
            const id = String(player.puuid || "").toLowerCase();
            if (!id) continue;
            const row = book[id] || { name: player.name || "Player", tag: player.tag || "", puuid: player.puuid || "", wins: 0, losses: 0, games: 0, ms: 0 };
            row.games += 1;
            row.ms += Number(match.length) > 0 ? Number(match.length) : guessMatchLength(match);
            if (match.won === true) row.wins += 1;
            if (match.won === false) row.losses += 1;
            if (player.name) row.name = player.name;
            book[id] = row;
          }
        }
        return Object.values(book).map((row) => ({ ...row, minutes: Math.round((row.ms || 0) / 60000) })).sort((a, b) => b.games - a.games);
      }
      function clutchRows() {
        const out = [];
        for (const match of history || []) {
          for (const round of match.rounds || []) {
            if (!round.vs) continue;
            out.push({
              id: match.id || "",
              map: match.map || "Map",
              when: match.when || "",
              mode: match.mode || "",
              score: match.score1 != null && match.score2 != null ? match.score1 + "–" + match.score2 : "",
              agent: ((match.players || []).find((player) => player.you) || {}).agent || "",
              n: round.n,
              vs: round.vs,
              fail: round.fail === true || round.you === false,
              how: round.how || "",
              side: round.side || "",
            });
          }
        }
        return out;
      }
      let clutchSeen = -1;
      let clutchKey = "";
      function noteClutchLight() {
        const newest = (history && history[0]) || null;
        if (!newest || !newest.id) return;
        let n = 0;
        for (const round of newest.rounds || []) if (round && round.vs) n++;
        if (clutchSeen < 0) {
          clutchSeen = n;
          clutchKey = newest.id;
          return;
        }
        if (clutchKey === newest.id && n > clutchSeen) lightEvent("clutch");
        clutchKey = newest.id;
        clutchSeen = n;
      }
      function deskTable(headers, rows) {
        const table = document.createElement("table");
        table.className = "desk-table";
        const head = document.createElement("tr");
        for (const name of headers) {
          const th = document.createElement("th");
          th.textContent = name;
          head.append(th);
        }
        table.append(head);
        for (const cells of rows) {
          const tr = document.createElement("tr");
          for (const value of cells) {
            const td = document.createElement("td");
            td.textContent = value == null ? "" : String(value);
            tr.append(td);
          }
          table.append(tr);
        }
        return table;
      }
      function formStrip(matches) {
        const rows = (matches || []).filter((match) => match.won === true || match.won === false).slice(0, 18);
        if (!rows.length) return null;
        const strip = el("div", "form-strip");
        bindHoverTip(strip, (tip) => {
          tip.append(el("div", "tip-kicker", "Form"));
          tip.append(el("b", null, "Last " + rows.length + " results"));
        });
        for (const match of rows) {
          const pip = el("i", match.won ? "w" : "l");
          bindHoverTip(pip, (tip) => {
            tip.append(el("div", "tip-kicker", match.won ? "Win" : "Loss"));
            tip.append(el("b", null, match.map || "Match"));
            if (match.mode) tip.append(el("div", "tip-line", match.mode));
          });
          strip.append(pip);
        }
        return strip;
      }
      function paintRecords() {
        const root = document.getElementById("records");
        if (!root) return;
        const rows = agentRows();
        root.className = "";
        const recSig = rows.map((row) => [row.name, row.games, row.wins, row.hs, row.acs].join(":")).join("|") + "|" + (localStorage.getItem("valvet-rec-sort") || "games") + "|" + ((history || []).length);
        if (root.dataset.sig === recSig && root.querySelector(".rec-hero, .empty")) return;
        root.dataset.sig = recSig;
        root.replaceChildren();
        if (!rows.length) {
          root.append(el("div", "empty", "No agent records yet. Games save on this PC while Valorant is open."));
          return;
        }
        const rec = (latestState && latestState.records) || {};
        const sum = (key) => rows.reduce((total, row) => total + (Number(row[key]) || 0), 0);
        const games = rec.games || sum("games");
        const wins = rec.wins || sum("wins");
        const losses = rec.losses || sum("losses");
        const kills = rec.kills || sum("kills");
        const deaths = rec.deaths || sum("deaths");
        const hsGames = rows.reduce((total, row) => total + ((row.hs && row.games) ? row.hs * row.games : 0), 0);
        const acsGames = rows.reduce((total, row) => total + ((row.acs && row.games) ? row.acs * row.games : 0), 0);
        const hs = rec.hs || (games ? Math.round(hsGames / games) + "%" : "—");
        const acs = rec.acs || (games ? Math.round(acsGames / games) : "—");
        const kd = rec.kd || (deaths ? (kills / deaths).toFixed(2) : kills ? kills + ".00" : "—");
        const hero = el("div", "rec-hero");
        for (const [value, label] of [[wins + "–" + losses, "Record"], [kd, "K/D"], [hs, "HS"], [acs, "ACS"]]) {
          const card = el("article", "rec-stat");
          card.append(el("b", null, String(value)));
          card.append(el("span", null, label));
          hero.append(card);
        }
        const podium = el("div", "rec-podium");
        const picks = [
          ["Best map", rec.bestMap ? rec.bestMap + (rec.bestMapRate ? " · " + rec.bestMapRate : "") : "—"],
          ["Best gun", rec.bestGun || "—"],
          ["Best game", rec.bestKills ? rec.bestKills + " kills" + (rec.bestKillsMap ? " · " + rec.bestKillsMap : "") : "—"],
        ];
        for (const [label, value] of picks) {
          const card = el("article", "rec-pick");
          card.append(el("span", null, label));
          card.append(el("b", null, value));
          podium.append(card);
        }
        const form = formStrip(history);
        const sortKey = localStorage.getItem("valvet-rec-sort") || "games";
        const sortBar = el("div", "rec-sort");
        const ordered = rows.slice().sort((a, b) => {
          const rate = (row) => row.games ? row.wins / row.games : 0;
          const ratio = (row) => row.deaths ? row.kills / row.deaths : row.kills || 0;
          if (sortKey === "win") return rate(b) - rate(a) || b.games - a.games;
          if (sortKey === "kd") return ratio(b) - ratio(a) || b.games - a.games;
          if (sortKey === "hs") return (b.hs || 0) - (a.hs || 0) || b.games - a.games;
          return b.games - a.games;
        });
        for (const [id, label] of [["games", "Games"], ["win", "Win rate"], ["kd", "K/D"], ["hs", "HS"]]) {
          const button = el("button", "tiny" + (sortKey === id ? " solid" : ""), label);
          button.type = "button";
          button.onclick = () => { localStorage.setItem("valvet-rec-sort", id); paintRecords(); };
          sortBar.append(button);
        }
        const list = el("div", "rec-list");
        const maxGames = Math.max(1, ...ordered.map((row) => row.games || 0));
        ordered.forEach((row, index) => {
          const card = el("article", "rec-row");
          card.append(el("div", "rankn", String(index + 1)));
          const mid = el("div");
          mid.append(el("b", null, row.name));
          mid.append(el("div", "sub", row.games + " games · " + row.wins + "–" + row.losses));
          const barwrap = el("div", "barwrap");
          const bar = el("div", "rec-bar");
          const fill = document.createElement("span");
          fill.style.width = Math.round(((row.games || 0) / maxGames) * 100) + "%";
          bar.append(fill);
          barwrap.append(bar);
          const side = el("div", "side");
          const rate = row.games ? Math.round((row.wins / row.games) * 100) : 0;
          const ratio = row.deaths ? (row.kills / row.deaths).toFixed(2) : row.kills ? row.kills + ".00" : "—";
          side.append(el("b", null, rate + "%"));
          side.append(el("div", "sub", ratio + " K/D" + (row.hs ? " · " + row.hs + "% HS" : "")));
          card.append(mid, barwrap, side);
          list.append(card);
        });
        const duel = el("section", "duel");
        duel.append(el("div", "eyebrow", "Head to head"));
        duel.append(el("h2", null, "Agent duel"));
        const picksRow = el("div", "duel-picks");
        const leftPick = document.createElement("select");
        const rightPick = document.createElement("select");
        for (const row of ordered) {
          leftPick.append(new Option(row.name, row.name));
          rightPick.append(new Option(row.name, row.name));
        }
        const savedDuel = readJSON("valvet-duel") || {};
        if (savedDuel.a && ordered.some((row) => row.name === savedDuel.a)) leftPick.value = savedDuel.a;
        if (savedDuel.b && ordered.some((row) => row.name === savedDuel.b)) rightPick.value = savedDuel.b;
        else if (ordered[1]) rightPick.value = ordered[1].name;
        const board = el("div", "duel-board");
        const drawDuel = () => {
          writeJSON("valvet-duel", { a: leftPick.value, b: rightPick.value });
          const left = ordered.find((row) => row.name === leftPick.value);
          const right = ordered.find((row) => row.name === rightPick.value);
          board.replaceChildren();
          if (!left || !right) return;
          const ratio = (row) => row.deaths ? row.kills / row.deaths : row.kills || 0;
          const metrics = [
            ["Games", left.games || 0, right.games || 0],
            ["Win %", left.games ? Math.round((left.wins / left.games) * 100) : 0, right.games ? Math.round((right.wins / right.games) * 100) : 0],
            ["K/D", Number(ratio(left).toFixed(2)), Number(ratio(right).toFixed(2))],
            ["HS", left.hs || 0, right.hs || 0],
            ["ACS", left.acs || 0, right.acs || 0],
          ];
          for (const [label, l, r] of metrics) {
            const line = el("div", "duel-metric");
            const max = Math.max(Number(l), Number(r), 0.01);
            const leftNum = el("div", "num" + (l > r ? " lead" : ""));
            leftNum.textContent = String(l);
            const rightNum = el("div", "num" + (r > l ? " lead" : ""));
            rightNum.textContent = String(r);
            const leftBar = el("div", "duel-track flip");
            const leftFill = document.createElement("span");
            leftFill.style.width = Math.round((Number(l) / max) * 100) + "%";
            leftBar.append(leftFill);
            const rightBar = el("div", "duel-track");
            const rightFill = document.createElement("span");
            rightFill.style.width = Math.round((Number(r) / max) * 100) + "%";
            rightBar.append(rightFill);
            line.append(leftNum, leftBar, el("div", "lab", label), rightBar, rightNum);
            board.append(line);
          }
        };
        leftPick.onchange = drawDuel;
        rightPick.onchange = drawDuel;
        picksRow.append(leftPick, rightPick);
        duel.append(picksRow, board);
        drawDuel();
        root.append(hero, podium);
        if (form) root.append(form);
        root.append(sortBar, list, duel);
      }
      function openMapStage(stage) {
        const box = el("div", "map-lightbox");
        const close = el("button", "tiny solid map-close", "Close");
        close.type = "button";
        const shut = () => {
          box.remove();
          document.removeEventListener("keydown", onKey);
        };
        const onKey = (event) => { if (event.key === "Escape") shut(); };
        close.onclick = (event) => { event.stopPropagation(); shut(); };
        const clone = stage.cloneNode(true);
        const bigger = clone.querySelector(".map-bigger");
        if (bigger) bigger.remove();
        box.append(close, clone);
        box.onclick = (event) => { if (event.target === box) shut(); };
        document.addEventListener("keydown", onKey);
        document.body.append(box);
      }
      function plotSpot(map, spot) {
        const xm = Number(map && map.xMultiplier);
        const ym = Number(map && map.yMultiplier);
        if (!map || !xm || !ym) return null;
        const x = Number(spot.y) * xm + Number(map.xScalarToAdd || 0);
        const y = Number(spot.x) * ym + Number(map.yScalarToAdd || 0);
        if (!Number.isFinite(x) || !Number.isFinite(y)) return null;
        if (x < -0.04 || x > 1.04 || y < -0.04 || y > 1.04) return null;
        return { x: Math.max(0, Math.min(100, x * 100)), y: Math.max(0, Math.min(100, y * 100)) };
      }
