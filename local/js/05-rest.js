      function heatSpots(match, mode) {
        if (mode !== "kills") return match.deaths || [];
        const you = ((match.players || []).find((player) => player.you) || {}).name || "";
        if (!you) return match.kills || [];
        const landed = (match.edges || []).filter((edge) => edge && (!you || edge.killer === you) && (Number(edge.vx) || Number(edge.vy))).map((edge) => ({ x: edge.vx, y: edge.vy, round: edge.round }));
        if (landed.length) return landed;
        return match.kills || [];
      }
      async function paintDeaths() {
        const root = document.getElementById("deaths");
        if (!root) return;
        const mode = localStorage.getItem("velvet-heat") === "kills" ? "kills" : "deaths";
        const groups = {};
        for (const match of history || []) {
          const spots = heatSpots(match, mode);
          const name = mapName(match.map);
          if (!spots.length || !name) continue;
          groups[name] = (groups[name] || []).concat(spots);
        }
        const names = Object.keys(groups);
        const sig = mode + "|" + names.map((name) => name + ":" + groups[name].length).join("|");
        if (root.dataset.sig === sig && root.querySelector(".map-stage, .empty, .heat-switch")) return;
        root.dataset.sig = sig;
        root.className = "";
        root.replaceChildren();
        const bar = el("div", "heat-switch");
        for (const [id, label] of [["deaths", "Deaths"], ["kills", "Kills"]]) {
          const button = el("button", "tiny" + (mode === id ? " solid" : ""), label);
          button.type = "button";
          button.onclick = () => {
            localStorage.setItem("velvet-heat", id);
            root.dataset.sig = "";
            paintDeaths();
          };
          bar.append(button);
        }
        root.append(bar);
        if (!names.length) {
          root.append(el("div", "empty", mode === "kills" ? "Kill spots show up after the next games Valvet reads." : "Death spots show up after the next games Valvet reads. Older history does not have the coordinates."));
          return;
        }
        const book = await mapBook();
        if (tab !== "deaths" || root.dataset.sig !== sig) return;
        for (const name of names) {
          const map = book[name] || {};
          const plotted = groups[name].map((spot) => plotSpot(map, spot)).filter(Boolean);
          const card = el("article", "desk-card");
          card.append(el("div", "eyebrow", mode === "kills" ? "Kills" : "Deaths"));
          card.append(el("h2", null, name + " · " + plotted.length));
          card.append(el("p", "sub", mode === "kills" ? plotted.length + " kills. Gold is where they were standing. The dot is the exact spot." : plotted.length + " deaths. Red pools where you died. The dot is the exact spot."));
          if (!map.displayIcon) {
            card.append(el("p", "sub", "No minimap for " + name + "."));
          } else {
            warmMap(map.displayIcon);
            card.append(mapFrame(map, [], plotted, mode));
            const save = el("button", "tiny", "Save picture");
            save.type = "button";
            save.onclick = async () => {
              if (save.dataset.busy === "1") return;
              save.dataset.busy = "1";
              save.textContent = "Saving…";
              try {
                await saveHeatPicture(map, plotted, mode, name);
                save.textContent = "Saved to Pictures";
              } catch {
                save.textContent = "Couldn't save";
              }
              setTimeout(() => { save.dataset.busy = ""; save.textContent = "Save picture"; }, 1800);
            };
            card.append(save);
            if (!plotted.length) card.append(el("p", "sub", groups[name].length + " spots saved. They sit outside this minimap, so they are not drawn on the edge."));
          }
          root.append(card);
        }
      }
      function paintClutches() {
        const root = document.getElementById("clutches");
        if (!root) return;
        const rows = clutchRows();
        const clutchSig = (clutchMore ? "1" : "0") + "|" + rows.map((row) => [row.map, row.n, row.fail ? 1 : 0, row.vs, row.agent].join(":")).join("|");
        if (root.dataset.sig === clutchSig && (root.querySelector(".clutch-row") || root.querySelector(".empty"))) return;
        root.dataset.sig = clutchSig;
        root.className = "";
        root.replaceChildren();
        if (!rows.length) {
          root.append(el("div", "empty", "1vX rounds show up after the next games. Valvet marks a clutch when you were the last one alive."));
          return;
        }
        const held = rows.filter((row) => !row.fail).length;
        const rate = rows.length ? Math.round((held / rows.length) * 100) : 0;
        const board = el("div", "clutch-board");
        const rateCard = el("div", "clutch-rate");
        rateCard.append(el("b", null, rate + "%"));
        rateCard.append(el("span", null, held + " held · " + (rows.length - held) + " died"));
        const split = el("div", "clutch-split");
        for (let n = 1; n <= 5; n++) {
          const group = rows.filter((row) => Number(row.vs) === n);
          const wins = group.filter((row) => !row.fail).length;
          const pct = group.length ? Math.round((wins / group.length) * 100) : 0;
          const card = el("article");
          card.append(el("div", "lab", "1v" + n));
          card.append(el("b", null, group.length ? pct + "%" : "—"));
          card.append(el("em", null, group.length ? wins + " / " + group.length : "none"));
          const meter = el("div", "clutch-meter");
          const fill = document.createElement("span");
          fill.style.width = (group.length ? pct : 0) + "%";
          meter.append(fill);
          card.append(meter);
          split.append(card);
        }
        board.append(rateCard, split);
        root.append(board);
        const list = el("div", "clutch-list");
        const shown = clutchMore ? rows : rows.slice(0, 8);
        shown.forEach((row) => {
          const card = el("article", "clutch-row " + (row.fail ? "died" : "held"));
          card.append(el("div", "vs", "1v" + row.vs));
          const mid = el("div");
          mid.append(el("b", null, [row.map || "Map", row.agent, "Round " + row.n].filter(Boolean).join(" · ")));
          const when = row.when ? new Date(row.when).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }) : "";
          mid.append(el("div", "sub", [row.side, row.how, row.mode, row.score, when].filter(Boolean).join(" · ")));
          card.append(mid, el("div", "end", row.fail ? "Died" : "Held"));
          list.append(card);
        });
        root.append(list);
        if (rows.length > 8) {
          root.append(viewMore(clutchMore, rows.length - 8, () => {
            clutchMore = !clutchMore;
            root.dataset.sig = "";
            paintClutches();
          }));
        }
      }
      function paintQueues() {
        const root = document.getElementById("queues");
        if (!root) return;
        let rows = duoRows();
        const duoSig = rows.map((row) => [row.name, row.tag, row.games, row.wins, row.minutes].join(":")).join("|");
        if (root.dataset.sig === duoSig && (root.querySelector(".duo-card") || root.querySelector(".empty"))) return;
        root.dataset.sig = duoSig;
        root.className = "";
        root.replaceChildren();
        if (!rows.length) {
          root.append(el("div", "empty", "No Riot friends in the saved games yet."));
          return;
        }
        const strip = el("div", "duo-list-modern");
        rows.forEach((row, index) => {
          const games = row.games || 0;
          const pct = games ? Math.round((row.wins / games) * 100) : 0;
          const card = el("article", "duo-card");
          card.style.animationDelay = Math.min(index, 14) * 40 + "ms";
          card.append(el("div", "duo-mark", (row.name || "?").slice(0, 1).toUpperCase()));
          const body = el("div");
          body.append(el("b", null, (row.name || "Player") + (row.tag ? " #" + row.tag : "")));
          body.append(el("div", "sub", games + " games · " + clockSpan((row.minutes || 0) * 60000) + " together"));
          const bar = el("div", "duo-bar");
          const fill = document.createElement("span");
          fill.style.width = pct + "%";
          bar.append(fill);
          body.append(bar);
          const stat = el("div", "duo-stat");
          stat.append(el("b", null, pct + "%"));
          stat.append(el("span", null, row.wins + "–" + row.losses));
          card.append(body, stat);
          strip.append(card);
        });
        root.append(strip);
      }
      function loadTeam() {
        const saved = readJSON("velvet-team");
        return saved && typeof saved === "object" ? { name: saved.name || "", roster: saved.roster || [] } : { name: "", roster: [] };
      }
      function saveTeam(team) {
        writeJSON("velvet-team", team);
      }
      function rankFor(riot) {
        const needle = String(riot || "").toLowerCase();
        const lists = [...((latestState && latestState.friends) || []), ...((latestState && latestState.party) || [])];
        for (const person of lists) {
          const id = (person.name || "") + "#" + (person.tag || "");
          if (id.toLowerCase() === needle && person.rank) return person.rank;
        }
        for (const match of history || []) {
          for (const player of match.players || []) {
            const id = (player.name || "") + "#" + (player.tag || "");
            if (id.toLowerCase() === needle && player.rank) return player.rank;
          }
        }
        return "";
      }
      function paintTeamList() {
        const list = document.getElementById("team-list");
        if (!list) return;
        const team = loadTeam();
        list.replaceChildren();
        if (!team.roster.length) list.append(el("div", "empty", "Nobody on the roster yet."));
        for (const riot of team.roster) {
          const row = el("div", "alert-row");
          const rank = rankFor(riot);
          row.append(el("b", null, riot + (rank ? " · " + rank : "")));
          const remove = el("button", "tiny", "Remove");
          remove.type = "button";
          remove.onclick = () => {
            const next = loadTeam();
            next.roster = next.roster.filter((name) => name !== riot);
            saveTeam(next);
            paintTeamList();
          };
          row.append(remove);
          list.append(row);
        }
      }
      function paintTeam() {
        const root = document.getElementById("team");
        if (!root) return;
        if (root.dataset.ready === "1") { paintTeamList(); return; }
        root.dataset.ready = "1";
        root.className = "";
        root.replaceChildren();
        const card = el("section", "set-card");
        card.append(el("div", "eyebrow", "Custom"));
        card.append(el("h2", null, "Roster"));
        const grid = el("div", "field-grid");
        const name = el("label", "field wide");
        name.append(el("span", null, "Team name"));
        const nameInput = document.createElement("input");
        nameInput.id = "team-name";
        nameInput.maxLength = 32;
        nameInput.value = loadTeam().name;
        nameInput.placeholder = "Premier or a stack";
        name.append(nameInput);
        const add = el("label", "field wide");
        add.append(el("span", null, "Add Name#Tag"));
        const addInput = document.createElement("input");
        addInput.id = "team-add";
        addInput.placeholder = "Name#Tag";
        addInput.maxLength = 32;
        add.append(addInput);
        grid.append(name, add);
        const go = el("button", "look-go", "Add to the roster");
        go.type = "button";
        go.onclick = () => {
          const team = loadTeam();
          team.name = nameInput.value.trim();
          const riot = addInput.value.trim();
          if (riot.includes("#") && !team.roster.some((item) => item.toLowerCase() === riot.toLowerCase())) team.roster.push(riot);
          saveTeam(team);
          addInput.value = "";
          paintTeamList();
        };
        nameInput.onchange = () => {
          const team = loadTeam();
          team.name = nameInput.value.trim();
          saveTeam(team);
        };
        card.append(grid, go);
        root.append(card);
        const list = el("div");
        list.id = "team-list";
        root.append(list);
        paintTeamList();
      }
      function guessMatchLength(match) {
        const mode = String((match && match.mode) || "").toLowerCase();
        if (/deathmatch|escalation|replication/.test(mode)) return 10 * 60000;
        if (/spike|swift|snowball|hurm/.test(mode)) return 14 * 60000;
        const rounds = ((match && match.rounds) || []).length;
        if (rounds) return rounds * 100000;
        return 35 * 60000;
      }
      function monthBars() {
        const today = dayStart(Date.now());
        const keys = [];
        for (let i = 29; i >= 0; i--) keys.push(today - i * 86400000);
        const session = {};
        const matches = {};
        for (const key of keys) {
          session[key] = 0;
          matches[key] = 0;
        }
        for (const row of readJSON("velvet-sessions") || []) {
          if (!row || !row.end) continue;
          const key = dayStart(row.start || row.end);
          if (session[key] == null) continue;
          session[key] += Math.max(0, row.end - row.start);
        }
        const open = readJSON("velvet-session-open");
        let liveKey = 0;
        let liveMs = 0;
        if (open && open.start) {
          liveKey = dayStart(open.start);
          liveMs = Math.max(0, Date.now() - open.start);
        }
        for (const match of history || []) {
          const start = Number(match.started) || Date.parse(match.when || "");
          if (!start) continue;
          const key = dayStart(start);
          if (matches[key] == null) continue;
          matches[key] += Number(match.length) > 0 ? Number(match.length) : guessMatchLength(match);
        }
        return keys.map((key) => ({
          key,
          ms: Math.max(session[key] || 0, matches[key] || 0) + (key === liveKey ? liveMs : 0),
          now: key === today,
        }));
      }
      let sessionDay = 0;
      let sessionLined = false;
      function dayMatches(key) {
        return (history || []).filter((match) => {
          const start = Number(match.started) || Date.parse(match.when || "");
          return start && dayStart(start) === key;
        }).sort((a, b) => (Number(b.started) || Date.parse(b.when || "") || 0) - (Number(a.started) || Date.parse(a.when || "") || 0));
      }
      function paintMonth(bars) {
        const card = el("section", "month");
        const total = bars.reduce((sum, bar) => sum + bar.ms, 0);
        const top = el("div", "month-top");
        top.append(el("b", null, "Last 30 days"));
        top.append(el("span", null, clockSpan(total) + " played"));
        const plot = el("div", "month-plot");
        const max = Math.max(1, ...bars.map((bar) => bar.ms));
        const guide = el("i", "month-guide");
        const tip = el("b", "month-tip", "");
        guide.hidden = true;
        tip.hidden = true;
        bars.forEach((bar, index) => {
          const col = el("button");
          col.type = "button";
          const when = new Date(bar.key);
          const pct = Math.max(bar.ms ? 6 : 0, Math.round((bar.ms / max) * 100));
          col.className = (bar.now ? "now" : "") + (sessionDay === bar.key ? " on" : "");
          const fill = document.createElement("span");
          fill.style.height = pct + "%";
          col.append(fill);
          bindHoverTip(col, (tip) => {
            tip.append(el("div", "tip-kicker", when.toLocaleDateString([], { month: "short", day: "numeric" })));
            tip.append(el("b", null, bar.ms ? clockSpan(bar.ms) : "0m"));
            tip.append(el("div", "tip-line", "Click to see that day's matches"));
          });
          if (bar.ms && (bar.now || bar.ms >= max * 0.72)) {
            const label = el("em", null, clockSpan(bar.ms));
            label.style.bottom = "calc(" + pct + "% + 3px)";
            col.append(label);
          }
          col.onclick = () => {
            sessionDay = sessionDay === bar.key ? 0 : bar.key;
            const root = document.getElementById("session");
            if (root) root.dataset.sig = "";
            paintSession();
          };
          plot.append(col);
        });
        const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        svg.setAttribute("class", "month-line");
        svg.setAttribute("viewBox", "0 0 300 100");
        svg.setAttribute("preserveAspectRatio", "none");
        const points = bars.map((bar, index) => {
          const x = bars.length === 1 ? 150 : (index / (bars.length - 1)) * 300;
          const y = 100 - Math.max(bar.ms ? 8 : 2, Math.round((bar.ms / max) * 92));
          return x.toFixed(1) + "," + y.toFixed(1);
        }).join(" ");
        const line = document.createElementNS("http://www.w3.org/2000/svg", "polyline");
        line.setAttribute("points", points);
        if (!sessionLined) line.classList.add("draw");
        svg.append(line);
        sessionLined = true;
        plot.append(svg, guide, tip);
        plot.onmousemove = (event) => {
          const box = plot.getBoundingClientRect();
          if (!box.width || !bars.length) return;
          const index = Math.max(0, Math.min(bars.length - 1, Math.round(((event.clientX - box.left) / box.width) * (bars.length - 1))));
          const bar = bars[index];
          const left = ((index + 0.5) / bars.length) * 100;
          guide.hidden = false;
          tip.hidden = false;
          guide.style.left = left + "%";
          tip.style.left = left + "%";
          const when = new Date(bar.key);
          tip.textContent = when.toLocaleDateString([], { month: "short", day: "numeric" }) + " · " + (bar.ms ? clockSpan(bar.ms) : "0m");
        };
        plot.onmouseleave = () => {
          guide.hidden = true;
          tip.hidden = true;
        };
        const axis = el("div", "month-axis");
        const first = new Date(bars[0].key);
        axis.append(el("span", null, first.toLocaleDateString([], { month: "short", day: "numeric" })));
        axis.append(el("span", null, "Today"));
        card.append(top, plot, axis);
        return card;
      }
      function paintDayMatches(key) {
        const wrap = el("div", "day-matches");
        const games = dayMatches(key);
        wrap.append(el("h2", null, new Date(key).toLocaleDateString([], { weekday: "short", month: "short", day: "numeric" })));
        if (!games.length) {
          wrap.append(el("div", "empty", "No saved match that day."));
          return wrap;
        }
        for (const match of games) {
          const row = el("button", "session-row");
          row.type = "button";
          const start = Number(match.started) || Date.parse(match.when || "");
          row.append(el("div", "when", start ? new Date(start).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "—"));
          const mid = el("div");
          mid.append(el("b", null, match.map || "Match"));
          const bits = [match.mode, match.won === true ? "Win" : match.won === false ? "Loss" : "", match.score1 != null && match.score2 != null ? match.score1 + "–" + match.score2 : ""].filter(Boolean);
          mid.append(el("div", "sub", bits.join(" · ")));
          row.append(mid);
          row.onclick = () => openMatchSheet(match);
          wrap.append(row);
        }
        return wrap;
      }
      function paintSession() {
        const root = document.getElementById("session");
        if (!root) return;
        const start = dayStart(Date.now());
        const rows = (readJSON("velvet-sessions") || []).filter((row) => row && row.end >= start);
        const open = readJSON("velvet-session-open");
        let total = 0;
        const items = [];
        for (const row of rows) {
          const ms = Math.max(0, row.end - row.start);
          total += ms;
          items.push({ when: new Date(row.start).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }), queue: row.queue || "Game", map: row.map || "—", ms, live: false });
        }
        if (open && open.start) {
          const ms = Math.max(0, Date.now() - open.start);
          total += ms;
          items.push({ when: "Now", queue: open.queue || "Game", map: open.map || "—", ms, live: true });
        }
        const picked = sessionDay ? dayMatches(sessionDay) : [];
        const sessionSig = [sessionDay, picked.map((match) => match.id).join(","), items.map((item) => [item.when, item.queue, item.map, Math.floor(item.ms / 60000), item.live ? 1 : 0].join(":")).join("|")].join("~");
        if (root.dataset.sig === sessionSig && root.querySelector(".session-hero, .empty")) return;
        root.dataset.sig = sessionSig;
        root.className = "";
        root.replaceChildren();
        root.append(paintMonth(monthBars()));
        if (sessionDay) root.append(paintDayMatches(sessionDay));
        root.append(deskClock(total));
        if (!items.length) root.append(el("div", "empty", "No session yet today. It starts when a game is found."));
        else {
          const list = el("div", "session-list");
          items.forEach((item, index) => {
            const row = el("article", "session-row" + (item.live ? " live" : ""));
            row.style.animationDelay = Math.min(index, 10) * 40 + "ms";
            row.append(el("div", "when", item.when));
            const mid = el("div");
            mid.append(el("b", null, item.map));
            mid.append(el("div", "sub", item.queue));
            row.append(mid, el("div", null, clockSpan(item.ms)));
            list.append(row);
          });
          root.append(list);
        }
      }
      function isRiotMobile(person) {
        const state = String((person && person.state) || "").trim().toLowerCase();
        const mode = String((person && person.mode) || "").trim().toLowerCase();
        const product = String((person && person.product) || "").trim().toLowerCase();
        return state === "mobile" || mode === "mobile" || mode === "riot mobile" || product.includes("mobile");
      }
      function onDesktop(person) {
        if (!person || isRiotMobile(person)) return false;
        const state = String(person.state || "").trim().toLowerCase();
        if (!state || state === "offline") return false;
        return true;
      }
      function isAway(person) {
        const state = String((person && person.state) || "").trim().toLowerCase();
        return state === "afk" || state === "away" || state === "idle";
      }
      function friendNotes() {
        const book = readJSON("valvet-friend-notes");
        return book && typeof book === "object" ? book : {};
      }
      function friendNote(id) {
        const key = String(id || "").toLowerCase();
        if (!key) return "";
        return String(friendNotes()[key] || "");
      }
      function saveFriendNote(id, text) {
        const key = String(id || "").toLowerCase();
        if (!key) return;
        const book = friendNotes();
        if (text) book[key] = text;
        else delete book[key];
        writeJSON("valvet-friend-notes", book);
      }
      const queueWas = new Map();
      function pingCareQueue(people) {
        const notes = friendNotes();
        for (const person of people || []) {
          const id = String(person.puuid || "").toLowerCase();
          if (!id) continue;
          const state = String(person.state || "");
          const prev = queueWas.get(id) || "";
          queueWas.set(id, state);
          if (!notes[id] || !prev || prev === "In queue" || state !== "In queue") continue;
          const name = person.name || "A friend";
          fetch("/api/notify", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ title: "Valvet", text: name + " queued" + (notes[id] ? " · " + notes[id] : "") }) }).catch(() => {});
          if (typeof playQueue === "function") playQueue();
        }
      }
      function stampSeen(people, fresh) {
        const book = readJSON("velvet-last-seen") || {};
        const now = Date.now();
        const here = new Set();
        for (const person of people || []) {
          const id = String(person.puuid || "").toLowerCase();
          if (!id) continue;
          here.add(id);
          const prev = book[id] || {};
          if (isRiotMobile(person)) {
            if (!prev.desk) {
              delete book[id];
              continue;
            }
            book[id] = { ...prev, live: false };
            continue;
          }
          const online = onDesktop(person);
          const arrived = online && !prev.live;
          const state = online ? (person.state || "Online") : (prev.state || "Offline");
          const playing = /in game|agent select|loading/i.test(state);
          const wasPlaying = /in game|agent select|loading/i.test(String(prev.state || ""));
          book[id] = {
            name: person.name || prev.name || "Player",
            tag: person.tag || prev.tag || "",
            at: arrived || !prev.at ? (online ? now : (prev.at || 0)) : prev.at,
            playSince: playing ? (wasPlaying && prev.playSince ? prev.playSince : now) : 0,
            live: online,
            desk: online ? true : !!prev.desk,
            product: online ? (person.product || prev.product || "") : (prev.product || ""),
            state,
            mode: online ? (person.mode || "") : (prev.mode || ""),
            map: online ? (person.map || "") : (prev.map || ""),
          };
        }
        if (fresh) {
          for (const id of Object.keys(book)) {
            if (!here.has(id) && book[id] && book[id].live) book[id] = { ...book[id], live: false };
          }
        }
        writeJSON("velvet-last-seen", book);
      }
      function paintSeen() {
        const root = document.getElementById("seen");
        if (!root) return;
        const book = readJSON("velvet-last-seen") || {};
        const rows = Object.entries(book).map(([id, row]) => ({ ...row, id })).filter((row) => row && row.at && String(row.state || "").toLowerCase() !== "mobile").sort((a, b) => Number(!!b.live) - Number(!!a.live) || (b.at || 0) - (a.at || 0));
        const sig = rows.map((row) => [row.name, row.live ? 1 : 0, row.state, row.map, row.mode, row.product, row.at, row.playSince || 0].join(":")).join("|");
        if (root.dataset.sig === sig) return;
        root.dataset.sig = sig;
        const oldTip = document.getElementById("seen-tip");
        if (oldTip) oldTip.hidden = true;
        root.className = "";
        root.replaceChildren();
        if (!rows.length) {
          root.append(el("div", "empty", "Nobody has opened Riot or a game yet while Valvet was open. Only Riot Mobile is ignored."));
          return;
        }
        const place = (row) => {
          const game = row.product === "league" ? "League" : row.product === "valorant" ? "Valorant" : row.product === "riot" ? "Riot" : "";
          const where = [row.map, row.mode].filter(Boolean).join(" · ");
          const state = String(row.state || "");
          if (where) return where;
          if (state && state !== "Online" && state !== "Offline") return [game, state].filter(Boolean).join(" · ");
          return game || "Riot";
        };
        const clockLabel = (at) => {
          const clock = new Date(at);
          const time = clock.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
          const today = new Date();
          const yest = new Date(today);
          yest.setDate(today.getDate() - 1);
          if (clock.toDateString() === today.toDateString()) return time;
          if (clock.toDateString() === yest.toDateString()) return "Yesterday " + time;
          return clock.toLocaleDateString([], { month: "short", day: "numeric" }) + " " + time;
        };
        const orbFor = (row) => {
          const age = row.at ? (Date.now() - row.at) / 3600000 : 48;
          const away = isAway(row);
          const tone = row.live && away ? "afk" : row.live ? "live" : age < 6 ? "recent" : "old";
          const playing = row.live && !away && /in game|agent select|loading/i.test(String(row.state || ""));
          const spent = playing && row.playSince ? Date.now() - row.playSince : 0;
          const clock = spent >= 60000 ? Math.floor(spent / 60000) + "m" : spent > 0 ? "<1m" : "";
          const orb = el("article", "seen-orb " + tone + (playing ? " ingame" : ""));
          const where = place(row);
          const halo = el("div", "halo");
          halo.append(el("div", "face", (row.name || "?").slice(0, 1).toUpperCase()));
          orb.append(halo);
          orb.append(el("b", null, row.name || "Friend"));
          if (row.live) {
            orb.append(el("span", "live-tag", away ? "AFK" : "Now"));
            orb.append(el("span", "where", where));
          } else {
            orb.append(el("time", null, clockLabel(row.at)));
            orb.append(el("span", "where", "Was " + where));
          }
          const show = () => showSeenTip(orb, row, where, clock, clockLabel(row.at));
          orb.addEventListener("mouseenter", show);
          orb.addEventListener("mouseleave", hideSeenTip);
          return orb;
        };
        const split = el("div", "seen-split");
        const live = rows.filter((row) => row.live && !isAway(row));
        const away = rows.filter((row) => row.live && isAway(row));
        const past = rows.filter((row) => !row.live);
        if (live.length) {
          const block = el("section", "now");
          block.append(el("h2", null, "Right now"));
          const grid = el("div", "seen-orbs");
          live.forEach((row) => grid.append(orbFor(row)));
          block.append(grid);
          split.append(block);
        }
        if (away.length) {
          const block = el("section", "away");
          block.append(el("h2", null, "AFK"));
          const grid = el("div", "seen-orbs");
          away.forEach((row) => grid.append(orbFor(row)));
          block.append(grid);
          split.append(block);
        }
        if (past.length) {
          const block = el("section");
          block.append(el("h2", null, "Last seen"));
          const grid = el("div", "seen-orbs");
          past.forEach((row) => grid.append(orbFor(row)));
          block.append(grid);
          split.append(block);
        }
        root.append(split);
      }
      function seenTipEl() {
        let tip = document.getElementById("seen-tip");
        if (tip) return tip;
        tip = el("div", "seen-tip");
        tip.id = "seen-tip";
        tip.hidden = true;
        document.body.append(tip);
        return tip;
      }
      function hideSeenTip() {
        const tip = document.getElementById("seen-tip");
        if (tip) tip.hidden = true;
      }
      function showSeenTip(anchor, row, where, clock, whenLabel) {
        const tip = seenTipEl();
        tip.replaceChildren();
        const away = isAway(row);
        tip.append(el("div", "tip-kicker", row.live ? (away ? "AFK" : "Online now") : "Last seen"));
        const name = el("b");
        name.append(document.createTextNode(row.name || "Friend"));
        if (row.tag) name.append(el("span", null, " #" + row.tag));
        tip.append(name);
        if (where) tip.append(el("div", "tip-line", where));
        const bits = [];
        if (row.live && clock) bits.push("In it " + clock);
        if (whenLabel) bits.push(row.live ? "Since " + whenLabel : whenLabel);
        if (bits.length) tip.append(el("div", "tip-line", bits.join(" · ")));
        const person = findSeenPerson(row.id);
        const rank = (person && person.rank) || rankFromHistory(row.id);
        const party = person && typeof company === "function" ? company(person) : "";
        if (rank) tip.append(el("div", "tip-line", rank));
        if (party) tip.append(el("div", "tip-line", "Party " + party));
        const last = lastTogether(row.id, history);
        if (last) tip.append(el("div", "tip-line", last));
        const note = friendNote(row.id);
        if (note) tip.append(el("div", "tip-note", note));
        tip.hidden = false;
        const box = anchor.getBoundingClientRect();
        const tipBox = tip.getBoundingClientRect();
        let left = box.left + (box.width - tipBox.width) / 2;
        let top = box.bottom + 10;
        if (top + tipBox.height > window.innerHeight - 8) top = box.top - tipBox.height - 10;
        left = Math.max(8, Math.min(left, window.innerWidth - tipBox.width - 8));
        if (top < 8) top = 8;
        tip.style.left = Math.round(left) + "px";
        tip.style.top = Math.round(top) + "px";
      }
      function findSeenPerson(id) {
        const key = String(id || "").toLowerCase();
        if (!key || typeof lastPeople === "undefined" || !lastPeople) return null;
        const lists = [].concat(lastPeople.party || [], lastPeople.friends || []);
        return lists.find((person) => String(person.puuid || "").toLowerCase() === key) || null;
      }
      function rankFromHistory(id) {
        const key = String(id || "").toLowerCase();
        if (!key) return "";
        for (const match of history || []) {
          const player = (match.players || []).find((row) => String(row.puuid || "").toLowerCase() === key && row.rank);
          if (player) return player.rank;
        }
        return "";
      }
      function placeSeenTip(anchor) {
        const tip = document.getElementById("seen-tip");
        if (!tip || tip.hidden || !anchor) return;
        const box = anchor.getBoundingClientRect();
        const tipBox = tip.getBoundingClientRect();
        let left = box.left + (box.width - tipBox.width) / 2;
        let top = box.bottom + 10;
        if (top + tipBox.height > window.innerHeight - 8) top = box.top - tipBox.height - 10;
        left = Math.max(8, Math.min(left, window.innerWidth - tipBox.width - 8));
        if (top < 8) top = 8;
        tip.style.left = Math.round(left) + "px";
        tip.style.top = Math.round(top) + "px";
      }
      function bindHoverTip(node, fill) {
        if (!node || node.dataset.tip === "1" || typeof fill !== "function") return;
        node.dataset.tip = "1";
        node.addEventListener("mouseenter", () => {
          const tip = seenTipEl();
          tip.replaceChildren();
          fill(tip);
          tip.hidden = false;
          placeSeenTip(node);
        });
        node.addEventListener("mouseleave", hideSeenTip);
      }
      function noteSession(data) {
        const phase = (data && data.phase) || "";
        const playing = phase === "In game" || phase === "Agent select" || phase === "Loading lobby";
        const open = readJSON("velvet-session-open");
        const now = Date.now();
        if (playing) {
          if (!open) writeJSON("velvet-session-open", { start: now, queue: data.queue || data.mode || "Game", map: data.map || "" });
          else {
            const next = { ...open, map: data.map || open.map || "", queue: data.queue || data.mode || open.queue || "Game" };
            if (next.map !== open.map || next.queue !== open.queue) writeJSON("velvet-session-open", next);
          }
          return;
        }
        if (!open || !open.start) return;
        const rows = readJSON("velvet-sessions") || [];
        rows.push({ start: open.start, end: now, queue: open.queue || data.mode || "Game", map: open.map || "" });
        const cut = dayStart(now) - 29 * 86400000;
        writeJSON("velvet-sessions", rows.filter((row) => row && row.end >= cut).slice(-400));
        localStorage.removeItem("velvet-session-open");
      }
      function paintDesk() {
        if (tab === "records") paintRecords();
        if (tab === "maps") paintMaps();
        if (tab === "clutches") paintClutches();
        if (tab === "queues") paintQueues();
        if (tab === "deaths") paintDeaths();
        if (tab === "session") paintSession();
        if (tab === "seen") paintSeen();
        if (tab === "team") paintTeam();
      }
      async function loadNews(live) {
        const root = document.getElementById("news");
        if (!root) return;
        const key = "velvet-cache-news2";
        const saved = readJSON(key);
        const ok = Array.isArray(saved) && saved.length;
        if (ok) paintNews(saved);
        const force = live === "force";
        const need = force || !ok || (live && !cacheFresh(key, 60 * 60 * 1000));
        if (!need) return;
        if (!ok) root.replaceChildren(el("div", "empty", "Loading Valorant news…"));
        try {
          const data = await (await fetch("/api/news", { cache: "no-store" })).json();
          const notes = data.notes || [];
          if (!notes.length) {
            if (!ok) root.replaceChildren(el("div", "empty", data.error || "No announcements yet."));
            return;
          }
          writeJSON(key, notes);
          stampCache(key);
          if (force || JSON.stringify(saved) !== JSON.stringify(notes)) paintNews(notes);
        } catch {
          if (!ok) root.replaceChildren(el("div", "empty", "Couldn't load Valorant news."));
        }
      }
      function paintNews(notes) {
        const root = document.getElementById("news");
        if (!root) return;
        root.className = "";
        root.replaceChildren();
        savedBar(root, "velvet-cache-news2", () => loadNews("force"));
        const bundles = notes.filter((note) => note.bundle);
        const rest = notes.filter((note) => !note.bundle);
        const gridFor = (title, rows) => {
          if (!rows.length) return;
          root.append(el("h2", null, title));
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
            body.append(el("div", "eyebrow", [note.kind || "News", note.date].filter(Boolean).join(" · ")));
            body.append(el("b", null, note.title || "Announcement"));
            if (note.blurb) body.append(el("div", "sub", note.blurb));
            card.append(body);
            card.onclick = () => {
              if (note.path) fetch("/api/open?url=" + encodeURIComponent("https://playvalorant.com" + note.path)).catch(() => {});
            };
            grid.append(card);
          }
          root.append(grid);
        };
        gridFor("Bundles", bundles);
        gridFor("News", rest);
      }
      const freshSig = {};
      function markFresh(name, sig) {
        if (!sig) return;
        freshSig[name] = sig;
        const button = document.getElementById("tab-" + name);
        const seen = localStorage.getItem("velvet-seen-" + name) || "";
        if (!seen) {
          localStorage.setItem("velvet-seen-" + name, sig);
          if (button) button.classList.remove("fresh");
          return;
        }
        const show = seen !== sig && tab !== name;
        if (tab === name) localStorage.setItem("velvet-seen-" + name, sig);
        if (button) button.classList.toggle("fresh", show);
      }
      function hiddenTabs() {
        try {
          const rows = JSON.parse(localStorage.getItem("velvet-hidden-tabs") || "[]");
          return Array.isArray(rows) ? rows : [];
        } catch { return []; }
      }
      function tabHidden(name) {
        if (name === "live" || name === "settings") return false;
        return hiddenTabs().indexOf(name) >= 0;
      }
      function applyHiddenNav() {
        const nav = document.querySelector("nav.nav");
        if (!nav) return;
        const kids = [...nav.children];
        for (let i = 0; i < kids.length; i++) {
          if (!kids[i].classList || !kids[i].classList.contains("nav-label")) continue;
          let any = false;
          for (let j = i + 1; j < kids.length; j++) {
            if (kids[j].classList && kids[j].classList.contains("nav-label")) break;
            if (kids[j].tagName === "BUTTON" && !kids[j].hidden) any = true;
          }
          kids[i].hidden = !any;
        }
        const shown = [...nav.querySelectorAll("button")].filter((button) => !button.hidden).length;
        nav.dataset.fill = shown <= 8 ? "few" : shown <= 16 ? "some" : "many";
      }
      const TABS = ["live", "friends", "history", "session", "seen", "store", "bundle", "patches", "esports", "loadout", "agents", "profile", "pass", "records", "maps", "deaths", "clutches", "queues", "team", "news", "alerts", "crosshairs", "game", "settings"];
      let quietNav = false;
      function setTab(next, user) {
        hideSeenTip();
        if (next === "together") next = "live";
        if (tabHidden(next)) next = "live";
        if (document.body.classList.contains("minimal") && next !== "live" && next !== "settings") next = "live";
        tab = next;
        localStorage.setItem("velvet-tab", next);
        for (const name of TABS) {
          const pane = document.getElementById("pane-" + name);
          const button = document.getElementById("tab-" + name);
          if (!pane || !button) continue;
          const on = tab === name;
          pane.hidden = !on;
          button.hidden = tabHidden(name);
          button.className = on ? "on" : "";
          if (!on && freshSig[name] && localStorage.getItem("velvet-seen-" + name) !== freshSig[name]) button.classList.add("fresh");
          if (on && freshSig[name]) localStorage.setItem("velvet-seen-" + name, freshSig[name]);
          if (on) {
            pane.classList.remove("enter");
            void pane.offsetWidth;
            pane.classList.add("enter");
          }
        }
        const traceBtn = document.getElementById("tab-alerts");
        if (next === "alerts") {
          localStorage.setItem("valvet-trace-seen", String(Date.now()));
          if (traceBtn) traceBtn.classList.remove("trace-ping");
          document.querySelectorAll("#watch-list .fresh").forEach((row) => row.classList.remove("fresh"));
        } else if (traceBtn && traceBtn.classList.contains("trace-ping") === false) {
          const seenAt = Number(localStorage.getItem("valvet-trace-seen") || 0);
          const book = readJSON("valvet-trace-hits") || {};
          if (Object.values(book).some((row) => row && row.at > seenAt)) traceBtn.classList.add("trace-ping");
        }
        if (next === "history" || next === "deaths" || next === "clutches" || next === "queues" || next === "profile" || next === "records" || next === "session") loadAct(false);
        if (next === "history") drawRecent(modeQueue ? modeMatches || [] : history);
        if (next === "history" && document.body.dataset.hist === "replays") loadReplays(false);
        if (next === "store" && lastShop) drawStore(lastShop);
        if (next === "live") {
          const party = ((latestState && latestState.party) || []).filter((person) => !isLeague(person));
          paintToday(history);
          paintChallenge();
          paintStreak(history);
          paintDuos("duos", party, history);
        }
        if (next === "loadout") loadLoadout(!!user);
        if (next === "bundle" && lastShop) { bundleKey = ""; drawBundle(lastShop); }
        if (next === "patches") loadPatches(!!user);
        if (next === "esports") loadEsports(!!user);
        if (next === "agents") loadAgents(!!user);
        if (next === "profile") {
          loadProfile(!!user);
          if (user) { tickMastery(); tickRR(); }
        }
        if (next === "pass") loadPass(!!user);
        if (next === "records" || next === "maps" || next === "deaths" || next === "clutches" || next === "queues" || next === "session" || next === "seen" || next === "team") paintDesk();
        if (next === "news") loadNews(!!user);
        if (next === "friends" || next === "alerts") loadSocial();
        if (next === "alerts") paintWallet();
        if (next === "store" || next === "loadout") refreshOwned();
        if (next === "crosshairs") drawXH();
        if (next === "together") refreshRoom(true);
        if (next === "game") loadGame(!!user);
        if (next === "settings") loadStorage();
        applyHiddenNav();
      }
      function showHistoryView(which) {
        const replays = which === "replays";
        document.body.dataset.hist = replays ? "replays" : "matches";
        const card = document.getElementById("replay-card");
        const recent = document.getElementById("recent");
        const bar = document.querySelector("#pane-history .hist-bar");
        if (card) card.hidden = !replays;
        if (recent) recent.hidden = replays;
        if (bar) bar.hidden = replays;
        const blurb = document.querySelector("#pane-history .page-hero .sub");
        if (blurb) blurb.textContent = replays ? "Replay files saved on this PC. Matches are the other switch." : "This act's matches, saved on this PC.";
        const matches = document.getElementById("hist-matches");
        const button = document.getElementById("hist-replays");
        if (matches) matches.classList.toggle("solid", !replays);
        if (button) button.classList.toggle("solid", replays);
        if (replays) loadReplays(false);
      }
      const histMatches = document.getElementById("hist-matches");
      const histReplays = document.getElementById("hist-replays");
      if (histMatches && histReplays && !histMatches.dataset.bound) {
        histMatches.dataset.bound = "1";
        histMatches.onclick = () => showHistoryView("matches");
        histReplays.onclick = () => showHistoryView("replays");
      }
      showHistoryView("matches");
      async function loadReplays(force) {
        const root = document.getElementById("replay-list");
        if (!root) return;
        const scan = document.getElementById("replay-scan");
        if (scan && !scan.dataset.bound) {
          scan.dataset.bound = "1";
          scan.onclick = () => loadReplays(true);
        }
        try {
          const data = await (await fetch("/api/replays", force ? { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" } : { cache: "no-store" })).json();
          const rows = (data && data.replays) || [];
          root.replaceChildren();
          if (!rows.length) {
            root.append(el("p", "sub", "No replay files saved yet. They copy over after Valorant downloads one."));
            return;
          }
          for (const row of rows) {
            const line = el("div", "status-line");
            const label = el("b", null, row.name || "Replay");
            const meta = el("span", "subline", [row.patch || "Patch unknown", row.saved ? "saved" : ""].filter(Boolean).join(" · "));
            const open = el("button", "tiny", "Open in Valorant");
            open.type = "button";
            open.onclick = async () => {
              open.textContent = "Opening…";
              try {
                const result = await (await fetch("/api/replays", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: row.name }) })).json();
                open.textContent = result && result.ok ? "Opened" : "Same patch only";
              } catch { open.textContent = "Open in Valorant"; }
              setTimeout(() => { open.textContent = "Open in Valorant"; }, 1600);
            };
            line.append(label, meta, open);
            root.append(line);
          }
        } catch {
          root.replaceChildren(el("p", "sub", "Replay files could not be read."));
        }
      }
      async function loadGame(live) {
        const root = document.getElementById("game");
        const key = "velvet-cache-game";
        const saved = readJSON(key);
        const ok = saved && (saved.rows || []).length;
        if (ok) paintGame(saved);
        const need = !ok || (live && !cacheFresh(key, 5 * 60 * 1000));
        if (!need) return;
        if (!ok) { root.className = "empty"; root.textContent = "Reading the local Valorant config…"; }
        try {
          const data = await (await fetch("/api/game", { cache: "no-store" })).json();
          if (data && (data.rows || []).length) {
            writeJSON(key, data);
            stampCache(key);
          }
          if (!saved || JSON.stringify(saved) !== JSON.stringify(data)) paintGame(data);
        } catch {
          if (!ok) root.textContent = "Couldn't read the game settings.";
        }
      }
      function paintGame(data) {
        const root = document.getElementById("game");
        const copy = document.getElementById("game-copy");
        if (copy) copy.onclick = async () => {
          const text = (data && data.text) || "";
          try { await navigator.clipboard.writeText(text); } catch {}
          copy.textContent = text ? "Copied" : "Nothing to copy";
          setTimeout(() => { copy.textContent = "Copy"; }, 1200);
        };
        const rows = (data && data.rows) || [];
        if (!rows.length) {
          root.className = "empty";
          root.textContent = (data && data.error) || "Open Valorant once on this PC.";
          return;
        }
        root.className = "set-grid";
        root.replaceChildren();
        let card = null;
        let group = "";
        let sens = "";
        let scoped = "1";
        for (const row of rows) {
          if (row.group !== group) {
            group = row.group;
            card = el("section", "set-card");
            card.append(el("div", "eyebrow", group || "Game"));
            card.append(el("h2", null, group || "Game"));
            root.append(card);
          }
          if (row.label === "Sensitivity") sens = row.value;
          if (row.label === "Scoped sensitivity") scoped = row.value;
          const line = el("label", "set-row" + (row.label === "Sensitivity" ? " sens-hero" : row.group === "Mouse" ? " mouse-quiet" : ""));
          line.append(el("span", null, row.label || ""));
          line.append(el("b", null, row.value || ""));
          card.append(line);
        }
        const mouse = root.querySelector(".set-card");
        if (mouse && sens) mouse.append(edpiBox(sens, scoped));
      }
      function edpiBox(sensText, scopedText) {
        const sens = Number(String(sensText || "").replace(",", "."));
        const scoped = Number(String(scopedText || "1").replace(",", ".")) || 1;
        const box = el("div", "edpi");
        box.append(el("div", "eyebrow", "eDPI"));
        const field = el("label", "edpi-in");
        field.append(el("span", null, "Mouse DPI"));
        const input = document.createElement("input");
        input.type = "number";
        input.min = "50";
        input.max = "36000";
        input.step = "50";
        input.value = localStorage.getItem("velvet-dpi") || "800";
        field.append(input);
        const chips = el("div", "edpi-chips");
        for (const dpi of [400, 800, 1600, 3200]) {
          const button = el("button", "tiny", String(dpi));
          button.type = "button";
          button.onclick = () => { input.value = String(dpi); draw(); };
          chips.append(button);
        }
        const want = el("label", "edpi-in");
        want.append(el("span", null, "Want eDPI"));
        const target = document.createElement("input");
        target.type = "number";
        target.min = "1";
        target.max = "20000";
        target.step = "1";
        target.placeholder = "280";
        target.value = localStorage.getItem("velvet-edpi-target") || "";
        want.append(target);
        const out = el("div", "edpi-out");
        const pretty = (n) => {
          if (!Number.isFinite(n)) return "—";
          const text = n >= 100 ? String(Math.round(n)) : String(Math.round(n * 100) / 100);
          return text;
        };
        const draw = () => {
          const dpi = Number(input.value);
          localStorage.setItem("velvet-dpi", input.value || "");
          localStorage.setItem("velvet-edpi-target", target.value || "");
          out.replaceChildren();
          const chip = (k, v) => {
            const card = el("article");
            card.append(el("span", null, k), el("b", null, v));
            return card;
          };
          if (!Number.isFinite(sens) || sens <= 0 || !Number.isFinite(dpi) || dpi <= 0) {
            out.append(chip("eDPI", "—"));
            return;
          }
          const edpi = dpi * sens;
          const cm = (2.54 * 360) / (dpi * sens * 0.07);
          out.append(chip("eDPI", pretty(edpi)));
          out.append(chip("cm/360", pretty(cm) + " cm"));
          out.append(chip("in/360", pretty(cm / 2.54) + " in"));
          if (scoped && scoped !== 1) out.append(chip("Scoped eDPI", pretty(edpi * scoped)));
          const goal = Number(target.value);
          if (Number.isFinite(goal) && goal > 0) out.append(chip("Set sens", pretty(goal / dpi)));
        };
        input.oninput = draw;
        target.oninput = draw;
        box.append(field, chips, want, out, el("p", "sub hint", "eDPI is DPI × sensitivity. cm/360 uses Valorant's 0.07 yaw."));
        draw();
        return box;
      }
      for (const name of TABS) {
        document.getElementById("tab-" + name).onclick = () => setTab(name, true);
      }
      document.getElementById("zoom-close").onclick = () => { document.getElementById("zoom").hidden = true; };
      document.getElementById("close").onclick = () => { document.getElementById("scrim").hidden = true; };
      document.getElementById("zoom").onclick = (event) => { if (event.target.id === "zoom") event.currentTarget.hidden = true; };
      const agentSheet = document.getElementById("agent-sheet");
      if (agentSheet) agentSheet.onclick = (event) => { if (event.target.id === "agent-sheet") closeAgent(); };
      function lastSeenMap(item) {
        if (!item) return "";
        const id = String(item.puuid || "").toLowerCase();
        const name = String(item.name || "").toLowerCase();
        const tag = String(item.tag || "").toLowerCase();
        const live = ((latestState && latestState.players) || []).some((player) => {
          if (player.you) return false;
          if (id && String(player.puuid || "").toLowerCase() === id) return true;
          return name && String(player.name || "").toLowerCase() === name && String(player.tag || "").toLowerCase() === tag;
        });
        if (live) return "";
        for (const match of history || []) {
          const hit = (match.players || []).find((player) => {
            if (id && String(player.puuid || "").toLowerCase() === id) return true;
            return name && String(player.name || "").toLowerCase() === name && String(player.tag || "").toLowerCase() === tag;
          });
          if (hit && match.map) return "Last map " + match.map;
        }
        return "";
      }
      function paintSocialBook(data) {
        watch = data.watch || [];
        const nextGone = data.unfriends || [];
        if (unfriendReady && localStorage.getItem("velvet-discord-unfriend") !== "off") {
          const prev = new Set(unfriends.map((item) => (item.puuid || trackKey(item.name, item.tag)).toLowerCase()));
          for (const item of nextGone) {
            const id = (item.puuid || trackKey(item.name, item.tag)).toLowerCase();
            if (!prev.has(id)) postDiscord("Unfriended you", (item.name || "Player") + (item.tag ? "#" + item.tag : ""));
          }
        }
        unfriends = nextGone;
        unfriendReady = true;
        window.__valvetSocial = data;
        const list = document.getElementById("watch-list");
        const gone = document.getElementById("unfriends");
        if (!list || !gone) return;
        list.replaceChildren();
        if (!watch.length) list.append(el("div", "empty", "Nobody on the watch list yet."));
        for (const item of watch) {
          const row = el("div", "alert-row");
          const copy = el("div");
          copy.append(el("b", null, item.name + "#" + item.tag));
          const key = String(item.name + "#" + item.tag).toLowerCase();
          const hit = (readJSON("valvet-trace-hits") || {})[key] || {};
          const seenAt = Number(localStorage.getItem("valvet-trace-seen") || 0);
          if (hit.at && hit.at > seenAt) row.classList.add("fresh");
          const last = hit.map ? "Last map " + hit.map : lastSeenMap(item);
          const bits = [];
          if (last) bits.push(last);
          if (hit.finished) {
            const when = new Date(hit.finished);
            if (!Number.isNaN(when.getTime())) bits.push("Finished " + when.toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }));
          }
          if (bits.length) copy.append(el("div", "sub", bits.join(" · ")));
          row.append(copy);
          const remove = el("button", "tiny", "Remove");
          remove.type = "button";
          remove.onclick = () => changeWatch("DELETE", item.name + "#" + item.tag);
          row.append(remove);
          list.append(row);
        }
        gone.className = "";
        gone.replaceChildren();
        const rows = data.unfriends || [];
        if (!rows.length) gone.append(el("div", "empty", "No unfriends yet. Leave Valvet open with the Riot Client."));
        for (const item of rows) {
          const row = el("div", "alert-row");
          const copy = el("div");
          copy.append(el("b", null, (item.name || "Player") + (item.tag ? "#" + item.tag : "")));
          const when = item.when ? new Date(item.when) : null;
          copy.append(el("div", "sub", when && !Number.isNaN(when.getTime()) ? when.toLocaleString() : "Removed"));
          row.append(copy);
          if (item.puuid) {
            const open = el("button", "tiny", "Profile");
            open.type = "button";
            open.onclick = () => openPlayer({ puuid: item.puuid, name: item.name, tag: item.tag });
            row.append(open);
          }
          gone.append(row);
        }
        const names = document.getElementById("renames");
        if (!names) return;
        names.className = "";
        names.replaceChildren();
        const changed = data.renames || [];
        if (!changed.length) names.append(el("div", "empty", "No name changes yet. Leave Valvet open with the Riot Client."));
        for (const item of changed) {
          const row = el("div", "alert-row");
          const copy = el("div");
          const oldName = item.from + (item.fromTag ? "#" + item.fromTag : "");
          const newName = item.to + (item.toTag ? "#" + item.toTag : "");
          copy.append(el("b", null, oldName + "  →  " + newName));
          const when = item.when ? new Date(item.when) : null;
          copy.append(el("div", "sub", when && !Number.isNaN(when.getTime()) ? when.toLocaleString() : "Changed"));
          row.append(copy);
          names.append(row);
        }
      }
      async function loadSocial() {
        if (document.body.classList.contains("minimal")) return;
        try { paintSocialBook(await (await fetch("/api/social", { cache: "no-store" })).json()); } catch {}
      }
      async function changeWatch(method, q) {
        const res = await fetch("/api/social?q=" + encodeURIComponent(q), { method, cache: "no-store" });
        paintSocialBook(await res.json());
      }
      paintWishList(lastShop);
      const wishSearch = document.getElementById("wish-q");
      if (wishSearch) wishSearch.oninput = () => {
        const root = document.getElementById("wish-list");
        if (root) root.dataset.key = "";
        paintWishList(lastShop);
      };
      async function loadKC() {
        try {
          const data = await (await fetch("/api/kc", { cache: "no-store" })).json();
          kcItems = data.items || [];
          const root = document.getElementById("wish-list");
          if (root) root.dataset.key = "";
          paintWishList(lastShop);
        } catch {}
      }
      loadKC();
      function applySize(name) {
        const allowed = { compact: 1, standard: 1, wide: 1, large: 1 };
        const size = allowed[name] ? name : "standard";
        localStorage.setItem("velvet-size", size);
        document.body.dataset.size = size;
        document.querySelectorAll("#sizes button").forEach((button) => button.classList.toggle("on", button.dataset.size === size));
        if (typeof velvetResize === "function") velvetResize(size).catch(() => {});
      }
      document.querySelectorAll("#sizes button").forEach((button) => {
        button.onclick = () => applySize(button.dataset.size);
      });
      applySize(localStorage.getItem("velvet-size") || "standard");
      async function paintLaunch() {
        let mode = "off";
        try { mode = (await (await fetch("/api/launch", { cache: "no-store" })).json()).mode || "off"; } catch {}
        document.querySelectorAll("#launch button").forEach((button) => button.classList.toggle("on", button.dataset.launch === mode));
      }
      document.querySelectorAll("#launch button").forEach((button) => {
        button.onclick = async () => {
          await fetch("/api/launch", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ mode: button.dataset.launch }) });
          paintLaunch();
        };
      });
      paintLaunch();
      const goal = document.getElementById("set-goal");
      if (goal) {
        goal.value = localStorage.getItem("velvet-rr-goal") || "";
        goal.onchange = () => {
          const n = Math.max(0, Math.min(500, Number(goal.value) || 0));
          if (n) localStorage.setItem("velvet-rr-goal", String(n));
          else localStorage.removeItem("velvet-rr-goal");
          const slot = document.getElementById("rr-slot");
          if (slot) paintRR(slot);
        };
      }
      const openFolder = document.getElementById("set-folder");
      if (openFolder) openFolder.onclick = () => fetch("/api/folder", { method: "POST" }).catch(() => {});
      const openRounds = document.getElementById("set-rounds");
      if (openRounds) openRounds.onclick = () => fetch("/api/folder?which=rounds", { method: "POST" }).catch(() => {});
      const clearStorage = document.getElementById("set-clear");
      async function loadStorage() {
        const node = document.getElementById("storage-use");
        const detail = document.getElementById("storage-detail");
        if (!node) return;
        try {
          const data = await (await fetch("/api/storage", { cache: "no-store" })).json();
          node.textContent = (data && data.label) || "Couldn't read the folder.";
          if (detail) detail.textContent = (data && data.detail) || "";
          if (openRounds) openRounds.textContent = data && data.videos ? "Round videos · " + data.videos : "Round videos";
        } catch {
          node.textContent = "Couldn't read the folder.";
        }
      }
      if (clearStorage) clearStorage.onclick = async () => {
        clearStorage.disabled = true;
        try {
          const data = await (await fetch("/api/storage", { method: "POST" })).json();
          const node = document.getElementById("storage-use");
          const detail = document.getElementById("storage-detail");
          if (node) node.textContent = (data && data.cleared) || (data && data.label) || "Cleared.";
          if (detail && data && data.detail) detail.textContent = data.detail;
          setTimeout(loadStorage, 400);
        } catch {
          const node = document.getElementById("storage-use");
          if (node) node.textContent = "Couldn't clear that.";
        }
        clearStorage.disabled = false;
      };
      loadStorage();
      document.getElementById("set-export").onclick = async () => {
        const social = await (await fetch("/api/social", { cache: "no-store" })).json().catch(() => ({}));
        const bag = { velvet: 27, watch: social.watch || [] };
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i);
          if (key && (key.startsWith("valvet-") || key.startsWith("velvet-"))) bag[key] = localStorage.getItem(key);
        }
        const blob = new Blob([JSON.stringify(bag, null, 2)], { type: "application/json" });
        const link = document.createElement("a");
        link.href = URL.createObjectURL(blob);
        link.download = "valvet-backup.json";
        link.click();
        setTimeout(() => URL.revokeObjectURL(link.href), 2000);
      };
      document.getElementById("set-import").onclick = () => document.getElementById("set-file").click();
      document.getElementById("set-file").onchange = async (event) => {
        const file = event.target.files && event.target.files[0];
        event.target.value = "";
        if (!file) return;
        const bag = JSON.parse(await file.text());
        Object.keys(bag).forEach((key) => {
          if ((key.startsWith("valvet-") || key.startsWith("velvet-")) && typeof bag[key] === "string") localStorage.setItem(key, bag[key]);
        });
        const social = await (await fetch("/api/social", { cache: "no-store" })).json().catch(() => ({ watch: [] }));
        for (const item of social.watch || []) await fetch("/api/social?q=" + encodeURIComponent(item.name + "#" + item.tag), { method: "DELETE" });
        for (const item of bag.watch || []) await fetch("/api/social?q=" + encodeURIComponent(item.name + "#" + item.tag), { method: "POST" });
        applySize(localStorage.getItem("velvet-size") || "standard");
        loadSocial();
        location.reload();
      };
      document.getElementById("room-ready") && (document.getElementById("room-ready").onchange = () => {});
      const lookGo = document.getElementById("look-go");
      if (lookGo) lookGo.onclick = () => {
        const ageEl = document.getElementById("board-age");
        const noteEl = document.getElementById("board-note");
        const genderEl = document.getElementById("board-gender");
        const queueEl = document.getElementById("board-queue");
        if (ageEl) localStorage.setItem("velvet-age", ageEl.value.trim());
        if (noteEl) localStorage.setItem("velvet-note", noteEl.value.trim());
        if (genderEl) localStorage.setItem("velvet-gender", genderEl.value);
        if (queueEl) localStorage.setItem("velvet-queue", queueEl.value);
        const on = !lookingOn();
        if (on) {
          const age = Number(localStorage.getItem("velvet-age") || 0);
          if (age && age < 16) {
            const state = document.getElementById("look-state");
            if (state) state.textContent = "Leave age empty, or enter 16 or older.";
            return;
          }
        }
        localStorage.setItem("velvet-looking", on ? "on" : "off");
        localStorage.setItem("velvet-ready", "off");
        if (on) {
          const mins = Number((document.getElementById("look-for") || {}).value || 0);
          if (mins) localStorage.setItem("velvet-until", String(Date.now() + mins * 60000));
          else localStorage.removeItem("velvet-until");
        } else localStorage.removeItem("velvet-until");
        paintLooking();
        rememberPrefs();
        pushRoom(true);
      };
      paintLooking();
      async function postCardBlob() {
        if (!postCard) return null;
        const canvas = document.createElement("canvas");
        canvas.width = 960;
        canvas.height = 420;
        const pen = canvas.getContext("2d");
        pen.fillStyle = "#12080d";
        pen.fillRect(0, 0, 960, 420);
        pen.fillStyle = "#ff5fa2";
        pen.fillRect(0, 0, 12, 420);
        pen.font = "700 22px Segoe UI";
        pen.fillText("VALVET", 48, 64);
        pen.fillStyle = "#fff7fa";
        pen.font = "700 48px Segoe UI";
        pen.fillText(String(postCard.title || "Match over").slice(0, 28), 48, 170);
        pen.fillStyle = "#b89aaa";
        pen.font = "28px Segoe UI";
        pen.fillText(String(postCard.score || "").slice(0, 12), 48, 230);
        pen.fillStyle = "#b89aaa";
        pen.font = "28px Segoe UI";
        pen.fillText(String(postCard.line || "").slice(0, 52), 48, 290);
        const rounds = (postCard.rounds || []).slice(0, 30);
        rounds.forEach((round, index) => {
          const x = 48 + (index % 15) * 28;
          const y = 330 + Math.floor(index / 15) * 28;
          pen.fillStyle = round.you ? "#ff5fa2" : "#3a2a32";
          pen.fillRect(x, y, 22, 18);
        });
        return await new Promise((resolve) => canvas.toBlob(resolve, "image/png"));
      }
      const postCopy = document.getElementById("post-copy");
      if (postCopy) postCopy.onclick = async () => {
        const blob = await postCardBlob();
        if (!blob) return;
        try {
          await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
          postCopy.textContent = "Copied";
        } catch {
          const link = document.createElement("a");
          link.href = URL.createObjectURL(blob);
          link.download = "valvet-match.png";
          link.click();
          postCopy.textContent = "Saved";
        }
        setTimeout(() => { postCopy.textContent = "Copy card"; }, 1200);
      };
      const postSave = document.getElementById("post-save");
      if (postSave) postSave.onclick = async () => {
        const blob = await postCardBlob();
        if (!blob) return;
        const link = document.createElement("a");
        link.href = URL.createObjectURL(blob);
        link.download = "valvet-match.png";
        link.click();
        setTimeout(() => URL.revokeObjectURL(link.href), 1500);
        postSave.textContent = "Saved";
        setTimeout(() => { postSave.textContent = "Save image"; }, 1200);
      };
      ensureCode();
      paintModes();
      function copyId(button) {
        const code = ensureCode();
        const label = button.textContent;
        navigator.clipboard.writeText(code).catch(() => {});
        button.textContent = "Copied";
        setTimeout(() => { button.textContent = label; }, 1200);
      }
      const copyCode = document.getElementById("copy-code");
      if (copyCode) copyCode.onclick = () => copyId(copyCode);
      function bindAdd(formId, inputId) {
        const form = document.getElementById(formId);
        if (!form) return;
        form.onsubmit = (event) => {
          event.preventDefault();
          const input = document.getElementById(inputId);
          addByCode(input.value);
          input.value = "";
        };
      }
      bindAdd("add-code", "add-code-q");
      document.getElementById("copy-invite").onclick = async () => {
        const button = document.getElementById("copy-invite");
        try { await navigator.clipboard.writeText(inviteText()); } catch {}
        button.textContent = "Copied";
        setTimeout(() => { button.textContent = "Copy invite"; }, 1200);
      };
      function fillRanks(id, key) {
        const select = document.getElementById(id);
        if (!select) return;
        select.append(new Option("Any", ""));
        for (const name of RANK_NAMES) if (name) select.append(new Option(name, name));
        select.value = localStorage.getItem(key) || "";
        select.onchange = () => { localStorage.setItem(key, select.value); pushRoom(lookingOn()); refreshRoom(); };
      }
      fillRanks("rank-low", "velvet-rank-low");
      fillRanks("rank-high", "velvet-rank-high");
      const serverPick = document.getElementById("stack-server");
      if (serverPick) {
        serverPick.value = localStorage.getItem("velvet-server") || "";
        serverPick.onchange = () => { localStorage.setItem("velvet-server", serverPick.value); pushRoom(lookingOn()); refreshRoom(); };
      }
      const lookFor = document.getElementById("look-for");
      if (lookFor) {
        lookFor.value = localStorage.getItem("velvet-look-for") || "0";
        lookFor.onchange = () => {
          localStorage.setItem("velvet-look-for", lookFor.value);
          if (localStorage.getItem("velvet-looking") === "on") {
            const mins = Number(lookFor.value || 0);
            if (mins) localStorage.setItem("velvet-until", String(Date.now() + mins * 60000));
            else localStorage.removeItem("velvet-until");
          }
          pushRoom(lookingOn());
        };
      }
      const AGENTS = ["Astra", "Breach", "Brimstone", "Chamber", "Clove", "Cypher", "Deadlock", "Fade", "Gekko", "Harbor", "Hush", "Iso", "Jett", "KAY/O", "Killjoy", "Neon", "Omen", "Phoenix", "Raze", "Reyna", "Sage", "Skye", "Sova", "Tejo", "Viper", "Vyse", "Waylay", "Yoru"];
      const agentPick = document.getElementById("want-agent");
      if (agentPick) {
        for (const name of AGENTS) agentPick.append(new Option(name, name));
        agentPick.value = localStorage.getItem("velvet-agent") || "";
        agentPick.onchange = () => { localStorage.setItem("velvet-agent", agentPick.value); pushRoom(lookingOn()); };
      }
      const hideRank = document.getElementById("hide-rank");
      if (hideRank) {
        hideRank.checked = localStorage.getItem("velvet-hide-rank") === "on";
        hideRank.onchange = () => { localStorage.setItem("velvet-hide-rank", hideRank.checked ? "on" : "off"); pushRoom(lookingOn()); };
      }
      document.addEventListener("pointermove", (event) => {
        const card = event.target.closest && event.target.closest(".duo-person");
        document.querySelectorAll(".duo-person").forEach((node) => { if (node !== card) { node.style.setProperty("--rx", "0deg"); node.style.setProperty("--ry", "0deg"); } });
        if (!card) return;
        const box = card.getBoundingClientRect();
        const x = (event.clientX - box.left) / box.width - 0.5;
        const y = (event.clientY - box.top) / box.height - 0.5;
        card.style.setProperty("--ry", (x * 8).toFixed(2) + "deg");
        card.style.setProperty("--rx", (-y * 6).toFixed(2) + "deg");
      });
      const discordName = document.getElementById("discord-name");
      const shareDiscord = document.getElementById("share-discord");
      if (shareDiscord && discordName) {
        shareDiscord.checked = localStorage.getItem("velvet-share-discord") === "on";
        discordName.value = localStorage.getItem("velvet-discord-name") || "";
        shareDiscord.onchange = () => { localStorage.setItem("velvet-share-discord", shareDiscord.checked ? "on" : "off"); pushRoom(lookingOn()); };
        discordName.onchange = () => { localStorage.setItem("velvet-discord-name", discordName.value.trim()); pushRoom(lookingOn()); };
      }
      function bindBoard(id, key) {
        const node = document.getElementById(id);
        if (!node) return;
        node.value = localStorage.getItem(key) || "";
        node.onchange = () => {
          localStorage.setItem(key, String(node.value || "").trim());
          if (lookingOn()) pushRoom(true);
        };
      }
      bindBoard("board-note", "velvet-note");
      bindBoard("board-age", "velvet-age");
      bindBoard("board-gender", "velvet-gender");
      bindBoard("board-queue", "velvet-queue");
      function bindFilter(id, key) {
        const node = document.getElementById(id);
        if (!node) return;
        node.value = localStorage.getItem(key) || "";
        node.onchange = () => {
          localStorage.setItem(key, String(node.value || "").trim());
          paintRoom();
        };
      }
      const filterMode = document.getElementById("filter-mode");
      if (filterMode) for (const name of PLAY_MODES) filterMode.append(new Option(name, name));
      bindFilter("filter-server", "velvet-filter-server");
      bindFilter("filter-queue", "velvet-filter-queue");
      bindFilter("filter-mode", "velvet-filter-mode");
      bindFilter("filter-gender", "velvet-filter-gender");
      bindFilter("filter-age-min", "velvet-filter-age-min");
      bindFilter("filter-age-max", "velvet-filter-age-max");
      setInterval(() => {
        if (tab !== "together" && !lookingOn()) return;
        refreshRoom();
        if (lookingOn()) pushRoom(false);
      }, 20000);
      let ownedAt = 0;
      async function refreshOwned() {
        if (ownedNames.length && Date.now() - ownedAt < 10 * 60 * 1000) return;
        try {
          ownedNames = (await (await fetch("/api/owned", { cache: "no-store" })).json()).names || [];
          ownedAt = Date.now();
          storeKey = "";
          const wishRoot = document.getElementById("wish-list");
          if (wishRoot) wishRoot.dataset.key = "";
          if (lastShop && tab === "store") drawStore(lastShop);
        } catch {}
      }
      document.getElementById("watch-form").onsubmit = async (event) => {
        event.preventDefault();
        const q = document.getElementById("watch-q").value.trim();
        if (!q.includes("#")) return;
        document.getElementById("watch-q").value = "";
        await changeWatch("POST", q);
        tone(660, 0.06);
      };
      loadSocial();
      const savedTab = localStorage.getItem("velvet-tab");
      if (localStorage.getItem("velvet-tour") === "done" && savedTab && savedTab !== "missions" && TABS.includes(savedTab) && savedTab !== "live") setTab(savedTab);
      document.getElementById("scrim").onclick = (event) => { if (event.target.id === "scrim") event.currentTarget.hidden = true; };

      const XH_KEY = "velvet-crosshairs";
      function loadXH() { try { return JSON.parse(localStorage.getItem(XH_KEY) || "[]"); } catch { return []; } }
      function readCrosshair(code) {
        const parts = String(code || "").split(";").filter((part) => part !== "");
        const map = {};
        let i = parts[0] === "0" ? 1 : 0;
        let section = "P";
        while (i < parts.length) {
          const key = parts[i++];
          if (key === "P" || key === "A" || key === "S") { section = key; continue; }
          const value = parts[i++];
          if (section === "P") map[key] = value;
        }
        return map;
      }
      function paintCrosshair(canvas, code, spread) {
        const ctx = canvas.getContext("2d");
        const w = canvas.width;
        const h = canvas.height;
        ctx.clearRect(0, 0, w, h);
        ctx.fillStyle = "#141018";
        ctx.fillRect(0, 0, w, h);
        const map = readCrosshair(code);
        const colors = ["#f2f2f2", "#00ff00", "#7cff00", "#d6ff00", "#ffe600", "#2ee6ff", "#ff5fa2", "#ff3b3b", "#ffffff"];
        let color = colors[Number(map.c)] || "#f2f2f2";
        if (String(map.c) === "8" && map.u) {
          const hex = String(map.u).replace(/[^0-9a-fA-F]/g, "");
          if (hex.length >= 6) color = "#" + hex.slice(-6);
        }
        const unit = Math.min(w, h) / 52;
        const cx = Math.round(w / 2);
        const cy = Math.round(h / 2);
        const fade = (value, fallback) => {
          if (value == null || value === "") return fallback;
          const n = Number(value);
          return Number.isFinite(n) ? Math.max(0, Math.min(1, n)) : fallback;
        };
        const outlineOn = map.h !== "0";
        const outThick = Math.max(0, Number(map.t) || 1);
        const outOp = fade(map.o, 0.5);
        const rect = (x, y, rw, rh, fill, alpha) => {
          ctx.globalAlpha = alpha;
          ctx.fillStyle = fill;
          ctx.fillRect(Math.round(x), Math.round(y), Math.max(1, Math.round(rw)), Math.max(1, Math.round(rh)));
        };
        const bar = (x, y, rw, rh, alpha) => {
          if (outlineOn && outThick > 0) {
            const t = outThick * unit * 0.28;
            rect(x - t, y - t, rw + t * 2, rh + t * 2, "#000", alpha * outOp);
          }
          rect(x, y, rw, rh, color, alpha);
        };
        const arms = (prefix, enabled) => {
          if (!enabled) return;
          const len = map[prefix + "l"] == null ? (prefix === "0" ? 6 : 2) : Number(map[prefix + "l"]);
          const thick = map[prefix + "t"] == null ? 2 : Number(map[prefix + "t"]);
          const gap = map[prefix + "o"] == null ? (prefix === "0" ? 2 : 8) : Number(map[prefix + "o"]);
          const alpha = fade(map[prefix + "a"], 1);
          if (len <= 0 || thick <= 0 || alpha <= 0) return;
          const L = len * unit;
          const T = Math.max(1, thick * unit * 0.42);
          const G = gap * unit + (Number(spread) || 0);
          bar(cx - G - L, cy - T / 2, L, T, alpha);
          bar(cx + G, cy - T / 2, L, T, alpha);
          bar(cx - T / 2, cy - G - L, T, L, alpha);
          bar(cx - T / 2, cy + G, T, L, alpha);
        };
        arms("0", map["0b"] !== "0");
        arms("1", map["1b"] === "1");
        if (map.d === "1") {
          const size = Math.max(1, (Number(map.z) || 2) * unit * 0.42);
          bar(cx - size / 2, cy - size / 2, size, size, fade(map.a, 1));
        }
        if (spread > 0) {
          ctx.globalAlpha = 0.45;
          ctx.strokeStyle = color;
          ctx.lineWidth = 2;
          ctx.beginPath();
          ctx.arc(cx, cy, 8 + spread * 1.7, 0, Math.PI * 2);
          ctx.stroke();
        }
        ctx.globalAlpha = 1;
      }
      function flairCard(item) {
        const card = el("article", "collect-card");
        if (item.anim || item.icon) {
          const img = document.createElement("img");
          img.alt = "";
          img.src = item.anim || item.icon;
          card.append(img);
        }
        card.append(el("div", "meta", item.slot || item.kind || ""));
        card.append(el("b", null, item.name || item.kind || "Equipped"));
        return card;
      }
      async function equipPiece(body) {
        const data = await (await fetch("/api/loadout", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) })).json();
        if (data.error && !(data.guns || []).length) return data.error;
        await loadLoadout("force");
        return "";
      }
      function confirmEquip(button, run) {
        button.addEventListener("click", async (event) => {
          event.preventDefault();
          event.stopPropagation();
          if (button.dataset.busy === "1") return;
          const label = button.querySelector("b");
          if (button.dataset.arm !== "1") {
            button.dataset.arm = "1";
            if (label) {
              button.dataset.prev = label.textContent;
              label.textContent = "Confirm";
            }
            return;
          }
          button.dataset.arm = "";
          button.dataset.busy = "1";
          try { await run(); }
          finally {
            button.dataset.busy = "";
            if (label && button.isConnected && button.dataset.prev) label.textContent = button.dataset.prev;
          }
        });
      }
      async function openWardrobe(gun) {
        const panel = document.getElementById("armory");
        if (!panel) return;
        document.querySelectorAll(".weapon").forEach((node) => node.classList.toggle("on", node.dataset.id === gun.id));
        panel.replaceChildren(el("h2", null, gun.name || "Weapon"), el("div", "empty", "Loading skins you own…"));
        try {
          const data = await (await fetch("/api/loadout?weapon=" + encodeURIComponent(gun.id || ""), { cache: "no-store" })).json();
          panel.replaceChildren(el("h2", null, (gun.name || "Weapon") + " · skins you own"));
          if (data.error) panel.append(el("div", "empty", data.error));
          const skins = el("div", "rail");
          for (const skin of data.skins || []) {
            const pick = el("button", "skin weapon");
            pick.type = "button";
            if (skin.icon) {
              const img = document.createElement("img");
              img.className = "gun";
              img.alt = "";
              img.src = skin.icon;
              pick.append(img);
            }
            pick.append(el("b", null, skin.name || "Skin"));
            confirmEquip(pick, async () => {
              panel.append(el("div", "sub", "Saving…"));
              const err = await equipPiece({ weapon: gun.id, skin: skin.id, level: skin.level });
              if (err) panel.append(el("div", "empty", err));
            });
            skins.append(pick);
          }
          panel.append(scroller(skins));
          if ((data.buddies || []).length) {
            panel.append(el("h2", null, "Buddies"));
            const row = el("div", "rail");
            for (const buddy of data.buddies) {
              const button = el("button", "buddy-pick weapon");
              button.type = "button";
              if (buddy.icon) {
                const img = document.createElement("img");
                img.className = "gun";
                img.alt = "";
                img.src = buddy.icon;
                button.append(img);
              }
              button.append(el("b", null, buddy.name || "Buddy"));
              confirmEquip(button, async () => { await equipPiece({ weapon: gun.id, buddy: buddy.id, instance: buddy.instance }); });
              row.append(button);
            }
            panel.append(scroller(row));
          }
          filterCollection();
        } catch {
          panel.replaceChildren(el("div", "empty", "Couldn't open this weapon."));
        }
      }
      function scroller(row) {
        const wrap = el("div", "scroller");
        const prev = el("button", "nudge", "‹");
        const next = el("button", "nudge", "›");
        prev.type = "button";
        next.type = "button";
        prev.onclick = (event) => { event.preventDefault(); event.stopPropagation(); row.scrollBy({ left: -280, behavior: "smooth" }); };
        next.onclick = (event) => { event.preventDefault(); event.stopPropagation(); row.scrollBy({ left: 280, behavior: "smooth" }); };
        wrap.append(prev, row, next);
        return wrap;
      }
      function trackShadeLight(event) {
        if (!document.body.classList.contains("fx")) return;
        document.body.style.setProperty("--cx", event.clientX + "px");
        document.body.style.setProperty("--cy", event.clientY + "px");
      }
      function startShade() {
        const canvas = document.getElementById("shade");
        if (canvas) canvas.dataset.live = "";
      }
      function attachRim(stage, img) {
        const draw = () => {
          if (!document.body.classList.contains("fx") || stage.querySelector("canvas.rim") || !img.naturalWidth) return;
          const w = 320;
          const h = 180;
          const src = document.createElement("canvas");
          src.width = w;
          src.height = h;
          const sx = src.getContext("2d", { willReadFrequently: true });
          if (!sx) return;
          sx.drawImage(img, 0, 0, w, h);
          let pixels;
          try { pixels = sx.getImageData(0, 0, w, h); } catch { return; }
          const out = sx.createImageData(w, h);
          const d = pixels.data;
          const o = out.data;
          for (let y = 1; y < h - 1; y++) {
            for (let x = 1; x < w - 1; x++) {
              const i = (y * w + x) * 4;
              const a = d[i + 3];
              const edge = Math.abs(a - d[i - 4 + 3]) + Math.abs(a - d[i + 4 + 3]) + Math.abs(a - d[((y - 1) * w + x) * 4 + 3]) + Math.abs(a - d[((y + 1) * w + x) * 4 + 3]);
              if (edge > 40 && a > 12) {
                o[i] = 255; o[i + 1] = 214; o[i + 2] = 232; o[i + 3] = Math.min(210, edge);
              }
            }
          }
          const rim = document.createElement("canvas");
          rim.className = "rim";
          rim.width = w;
          rim.height = h;
          rim.getContext("2d").putImageData(out, 0, 0);
          stage.append(rim);
        };
        if (img.complete) draw();
        else img.addEventListener("load", draw, { once: true });
      }
      function gunMedia(gun) {
        const stage = el("div", "turntable");
        if (gun.anim) {
          const vid = document.createElement("video");
          vid.className = "gun";
          vid.src = gun.anim;
          vid.muted = true;
          vid.loop = true;
          vid.autoplay = true;
          vid.playsInline = true;
          vid.onerror = () => {
            const img = document.createElement("img");
            img.className = "gun";
            img.alt = "";
            img.src = gun.icon || "";
            vid.replaceWith(img);
            attachRim(stage, img);
          };
          stage.append(vid);
          return stage;
        }
        const img = document.createElement("img");
        img.className = "gun";
        img.alt = "";
        img.src = gun.icon || "";
        stage.append(img);
        attachRim(stage, img);
        return stage;
      }
      let expressionSlot = 0;
      async function fillExpressions() {
        const root = document.getElementById("expressions");
        if (!root) return;
        root.replaceChildren(el("h2", null, "Sprays and flex"), el("div", "empty", "Loading what you own…"));
        try {
          const data = await (await fetch("/api/loadout?weapon=flair", { cache: "no-store" })).json();
          root.replaceChildren(el("h2", null, "Sprays and flex"));
          if (data.error) root.append(el("div", "empty", data.error));
          const note = el("div", "sub", "Pick a slot, then a spray or flex.");
          root.append(note);
          const slots = el("div", "rail");
          (data.slots || []).forEach((item, index) => {
            const button = el("button", "flair-pick");
            button.type = "button";
            if (item.icon) {
              const img = document.createElement("img");
              img.alt = "";
              img.src = item.icon;
              button.append(img);
            }
            button.append(el("b", null, "Slot " + (index + 1)));
            if (index === expressionSlot) button.style.outline = "2px solid var(--accent)";
            button.onclick = () => { expressionSlot = index; fillExpressions(); };
            slots.append(button);
          });
          if (!slots.childElementCount) slots.append(el("div", "empty", "No expression slots on this loadout."));
          root.append(scroller(slots));
          const paint = (title, rows, kind) => {
            root.append(el("h2", null, title));
            const rail = el("div", "rail");
            if (!rows.length) rail.append(el("div", "empty", "None owned."));
            for (const item of rows) {
              const button = el("button", "flair-pick weapon");
              button.type = "button";
              if (item.icon) {
                const img = document.createElement("img");
                img.className = "gun";
                img.alt = "";
                img.src = item.icon;
                button.append(img);
              }
              button.append(el("b", null, item.name || kind));
              confirmEquip(button, async () => {
                note.textContent = "Saving " + (item.name || kind) + "…";
                const err = await equipPiece({ kind, skin: item.id, slot: expressionSlot, type: item.type || "" });
                note.textContent = err || "Saved to slot " + (expressionSlot + 1) + ".";
              });
              rail.append(button);
            }
            root.append(scroller(rail));
          };
          paint("Your sprays", data.sprays || [], "spray");
          paint("Your flex", data.flex || [], "flex");
          filterCollection();
        } catch {
          root.replaceChildren(el("div", "empty", "Couldn't load sprays and flex."));
        }
      }
      function crosshairCard(item, onDelete) {
        const card = el("article", "xh-card" + (item.current ? " current" : ""));
        const canvas = document.createElement("canvas");
        canvas.width = 320;
        canvas.height = 320;
        paintCrosshair(canvas, item.code);
        card.append(canvas);
        const label = item.current ? "Using · " + (item.name || "Current") : (item.name || "Crosshair");
        card.append(el("b", null, label));
        card.append(el("code", null, item.code));
        const buttons = el("div", "add");
        const copyBtn = el("button", "tiny", "Copy");
        copyBtn.type = "button";
        copyBtn.onclick = async () => {
          try { await navigator.clipboard.writeText(item.code); copyBtn.textContent = "Copied"; tone(700, 0.05); }
          catch { copyBtn.textContent = "Copy failed"; }
        };
        buttons.append(copyBtn);
        if (onDelete) {
          const remove = el("button", "tiny", "Delete");
          remove.type = "button";
          remove.onclick = onDelete;
          buttons.append(remove);
        }
        card.append(buttons);
        return card;
      }
      function drawXH() {
        const list = loadXH();
        const root = document.getElementById("crosshairs");
        if (!root) return;
        root.replaceChildren();
        if (!list.length) root.append(el("div", "empty", "Nothing saved yet. Paste a code and import it."));
        list.forEach((item, index) => {
          root.append(crosshairCard(item, () => {
            const next = loadXH();
            next.splice(index, 1);
            localStorage.setItem(XH_KEY, JSON.stringify(next));
            drawXH();
          }));
        });
      }
      document.getElementById("xh-form").onsubmit = (event) => {
        event.preventDefault();
        const name = document.getElementById("xh-name").value.trim() || "Crosshair";
        const code = document.getElementById("xh-code").value.trim();
        if (!code) return;
        const next = loadXH();
        next.push({ name, code });
        localStorage.setItem(XH_KEY, JSON.stringify(next));
        document.getElementById("xh-code").value = "";
        drawXH();
        tone(640, 0.06);
      };
      const preview = document.getElementById("xh-preview");
      const previewCode = document.getElementById("xh-code");
      const paintPreview = () => paintCrosshair(preview, previewCode.value.trim() || "0;P;c;5;h;0;0l;4;0o;2;0a;1;0f;0;1b;0");
      previewCode.oninput = paintPreview;
      paintPreview();
      document.getElementById("xh-export").onclick = async () => {
        const text = loadXH().map((item) => (item.name || "Crosshair") + "\n" + item.code).join("\n\n");
        const button = document.getElementById("xh-export");
        if (!text) { button.textContent = "Nothing saved"; return; }
        try { await navigator.clipboard.writeText(text); button.textContent = "Copied"; }
        catch { button.textContent = "Copy failed"; }
      };
      drawXH();

      try {
        const THEMES = {
          Valvet: { bg: "#10080d", panel: "#1b1218", accent: "#ff5fa2", you: "#c9a6ff", enemy: "#ff5fa2" },
          Ice: { bg: "#0c1118", panel: "#141b26", accent: "#7ec8ff", you: "#7ec8ff", enemy: "#ff7a9c" },
          Gold: { bg: "#12100c", panel: "#1c1914", accent: "#e2b657", you: "#e2b657", enemy: "#e07a5f" },
          Night: { bg: "#101012", panel: "#18181c", accent: "#f4f4f6", you: "#9ad0ff", enemy: "#ff8aa0" },
          Mint: { bg: "#071410", panel: "#10201a", accent: "#3dce9a", you: "#9dffc3", enemy: "#ff7a9c" },
          Blood: { bg: "#14080c", panel: "#241016", accent: "#ff3355", you: "#ffb3c0", enemy: "#ff5fa2" },
          Violet: { bg: "#0e0a16", panel: "#1a1428", accent: "#b388ff", you: "#d4c2ff", enemy: "#ff7ad9" },
          Ember: { bg: "#140e0a", panel: "#24180f", accent: "#ff8a3d", you: "#ffd0a8", enemy: "#ff5f7a" },
        };
        const THEME_KEY = "velvet-colours";
        let theme = { ...THEMES.Valvet };
        try { const saved = JSON.parse(localStorage.getItem(THEME_KEY) || "null"); if (saved && saved.bg) theme = saved; } catch {}
        function applyTheme(next) {
          const root = document.documentElement.style;
          root.setProperty("--bg", next.bg);
          root.setProperty("--panel", next.panel);
          root.setProperty("--accent", next.accent);
          root.setProperty("--you", next.you);
          root.setProperty("--enemy", next.enemy);
          document.body.style.background = document.body.classList.contains("custom-bg") ? document.body.style.background : next.bg;
          if (window.velvetChrome) velvetChrome(next.bg, next.accent);
        }
        applyTheme(theme);
        const motion = document.getElementById("set-motion");
        const corners = document.getElementById("set-corners");
        const cats = document.getElementById("set-cats");
        const applyShape = () => {
          const motionValue = (motion && motion.value) || localStorage.getItem("velvet-motion") || "full";
          const cornerValue = (corners && corners.value) || localStorage.getItem("velvet-corners") || "soft";
          document.body.dataset.motion = motionValue;
          document.body.dataset.corners = cornerValue;
          document.body.classList.toggle("no-cats", !!(cats && !cats.checked));
          markThemes();
        };
        function markThemes() {
          const motionValue = document.body.dataset.motion || localStorage.getItem("velvet-motion") || "full";
          const cornerValue = document.body.dataset.corners || localStorage.getItem("velvet-corners") || "soft";
          document.querySelectorAll("#sample-shape [data-corners]").forEach((btn) => btn.classList.toggle("on", btn.dataset.corners === cornerValue));
          document.querySelectorAll("#sample-shape [data-motion]").forEach((btn) => btn.classList.toggle("on", btn.dataset.motion === motionValue));
          document.querySelectorAll("#swatches .theme-chip, #sample-swatches .theme-chip").forEach((chip) => {
            const label = ((chip.querySelector("span") || {}).textContent) || "";
            const preset = THEMES[label];
            chip.classList.toggle("on", !!(preset && preset.bg === theme.bg && preset.accent === theme.accent));
          });
          const sampleGlass = document.getElementById("sample-glass");
          const glassBox = document.getElementById("set-glass");
          if (sampleGlass && glassBox) sampleGlass.checked = !!glassBox.checked;
        }
        if (motion) {
          motion.value = localStorage.getItem("velvet-motion") || "full";
          motion.onchange = () => { localStorage.setItem("velvet-motion", motion.value); applyShape(); };
        }
        if (corners) {
          corners.value = localStorage.getItem("velvet-corners") || "soft";
          corners.onchange = () => { localStorage.setItem("velvet-corners", corners.value); applyShape(); };
        }
        if (cats) {
          cats.checked = localStorage.getItem("velvet-cats") !== "off";
          cats.onchange = () => { localStorage.setItem("velvet-cats", cats.checked ? "on" : "off"); applyShape(); };
        }
        applyShape();
        const sound = document.getElementById("set-sound");
        const chime = document.getElementById("set-chime");
        const join = document.getElementById("set-join");
        const pop = document.getElementById("set-pop");
        const night = document.getElementById("set-night");
        const compact = document.getElementById("set-compact");
        const volume = document.getElementById("set-volume");
        const volumeN = document.getElementById("set-volume-n");
        const resultVolumeInput = document.getElementById("set-result-volume");
        const resultVolumeN = document.getElementById("set-result-volume-n");
        const pack = document.getElementById("set-pack");
        const discord = document.getElementById("set-discord");
        const discordURLInput = document.getElementById("set-discord-url");
        const discordLive = document.getElementById("set-discord-live");
        const discordEnd = document.getElementById("set-discord-end");
        const discordWatch = document.getElementById("set-discord-watch");
        const discordStatus = document.getElementById("set-discord-status");
        const discordRounds = document.getElementById("set-discord-rounds");
        const discordUnfriend = document.getElementById("set-discord-unfriend");
        const discordApp = document.getElementById("set-discord-app");
        sound.checked = localStorage.getItem("velvet-sound") !== "off";
        chime.checked = localStorage.getItem("velvet-chime") === "on";
        join.checked = localStorage.getItem("velvet-join") === "on";
        pop.checked = localStorage.getItem("velvet-pop") === "on";
        night.checked = localStorage.getItem("velvet-night") === "on";
        const wishNote = document.getElementById("set-wish-note");
        if (wishNote) {
          wishNote.checked = localStorage.getItem("velvet-wish-note") === "on";
          wishNote.onchange = () => {
            localStorage.setItem("velvet-wish-note", wishNote.checked ? "on" : "off");
            if (wishNote.checked) localStorage.removeItem("velvet-wish-sig");
          };
        }
        compact.checked = localStorage.getItem("velvet-compact") === "on";
        volume.value = localStorage.getItem("velvet-volume") || "15";
        volumeN.textContent = volume.value + "%";
        if (resultVolumeInput && resultVolumeN) {
          resultVolumeInput.value = localStorage.getItem("velvet-result-volume") || "40";
          resultVolumeN.textContent = resultVolumeInput.value + "%";
        }
        const savedPack = localStorage.getItem("velvet-soundpack") || "soft";
        pack.value = SOUND_PACKS[savedPack] || savedPack === "custom" ? savedPack : "soft";
        discord.checked = localStorage.getItem("velvet-discord") === "on";
        discordURLInput.value = localStorage.getItem("velvet-discord-url") || "";
        discordApp.value = localStorage.getItem("velvet-discord-app") || "";
        discordStatus.checked = localStorage.getItem("velvet-discord-status") !== "off";
        discordRounds.checked = localStorage.getItem("velvet-discord-rounds") !== "off";
        discordUnfriend.checked = localStorage.getItem("velvet-discord-unfriend") !== "off";
        discordLive.checked = localStorage.getItem("velvet-discord-live") === "on";
        discordEnd.checked = localStorage.getItem("velvet-discord-end") === "on";
        discordWatch.checked = localStorage.getItem("velvet-discord-watch") !== "off";
        document.body.classList.toggle("compact", compact.checked);
        const glass = document.getElementById("set-glass");
        const fx = document.getElementById("set-fx");
        const syncLook = () => {
          document.body.classList.toggle("glass", !!(glass && glass.checked));
          document.body.classList.toggle("fx", !!(fx && fx.checked));
          startShade();
          if (document.body.classList.contains("fx")) {
            document.addEventListener("pointermove", trackShadeLight);
          }
          markThemes();
        };
        if (glass) {
          glass.checked = localStorage.getItem("velvet-glass") === "on";
          glass.onchange = () => {
            localStorage.setItem("velvet-glass", glass.checked ? "on" : "off");
            syncLook();
          };
        }
        if (fx) {
          fx.checked = localStorage.getItem("velvet-fx") === "on";
          fx.onchange = () => {
            localStorage.setItem("velvet-fx", fx.checked ? "on" : "off");
            syncLook();
          };
        }
        syncLook();
        const minimal = document.getElementById("set-minimal");
        const applyMinimal = (on) => {
          document.body.classList.toggle("minimal", !!on);
          if (on && tab !== "live" && tab !== "settings") setTab("live");
        };
        if (minimal) {
          minimal.checked = localStorage.getItem("velvet-minimal") === "on";
          minimal.onchange = () => {
            localStorage.setItem("velvet-minimal", minimal.checked ? "on" : "off");
            rememberPrefs();
            applyMinimal(minimal.checked);
          };
          if (minimal.checked) applyMinimal(true);
        }
        const pauseLookups = document.getElementById("set-pause-lookups");
        if (pauseLookups) {
          pauseLookups.checked = localStorage.getItem("velvet-pause-lookups") === "on";
          const pushPause = () => {
            localStorage.setItem("velvet-pause-lookups", pauseLookups.checked ? "on" : "off");
            rememberPrefs();
            fetch("/api/pause", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ on: pauseLookups.checked }) }).catch(() => {});
          };
          pauseLookups.onchange = pushPause;
          pushPause();
        }
        const gmail = document.getElementById("set-gmail");
        if (gmail) {
          gmail.value = localStorage.getItem("velvet-gmail") || "";
          gmail.addEventListener("change", () => {
            localStorage.setItem("velvet-gmail", gmail.value.trim());
            rememberPrefs();
          });
        }
        const gmailSend = document.getElementById("set-gmail-send");
        if (gmailSend) gmailSend.onclick = () => {
          const email = gmail ? gmail.value.trim() : "";
          if (email) localStorage.setItem("velvet-gmail", email);
          rememberPrefs();
          const bag = {};
          for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            if (key && (key.startsWith("valvet-") || key.startsWith("velvet-"))) bag[key] = localStorage.getItem(key) || "";
          }
          const text = JSON.stringify(bag);
          if (text.length > 1500) {
            const link = document.createElement("a");
            link.href = URL.createObjectURL(new Blob([text], { type: "application/json" }));
            link.download = "valvet-backup.json";
            link.click();
          }
          const note = text.length > 1500 ? "Attach valvet-backup.json from your downloads. That file is the Valvet backup." : text;
          const url = "https://mail.google.com/mail/?view=cm&fs=1&to=" + encodeURIComponent(email) + "&su=" + encodeURIComponent("Valvet backup") + "&body=" + encodeURIComponent(note);
          fetch("/api/open?url=" + encodeURIComponent(url)).catch(() => {});
        };
        const notesButton = document.getElementById("app-notes");
        if (notesButton) notesButton.onclick = () => {
          const box = document.getElementById("matchbox");
          document.getElementById("sheet-kicker").textContent = "Versions";
          document.getElementById("sheet-title").textContent = "App notes";
          const body = document.getElementById("sheet-body");
          body.replaceChildren();
          for (const note of visibleNotes()) {
            const card = el("article", "note-card");
            const head = el("header");
            head.append(el("h3", null, note.title || "Valvet"));
            head.append(el("span", "note-ver", note.ver));
            card.append(head);
            const list = el("ul");
            for (const line of note.lines) list.append(el("li", null, line));
            card.append(list);
            body.append(card);
          }
          if (box) box.classList.add("notes");
          box.hidden = false;
        };
        document.addEventListener("pointermove", (event) => {
          const card = event.target.closest && event.target.closest(".skin, .duo-person");
          if (!card) return;
          const box = card.getBoundingClientRect();
          if (!box.width || !box.height) return;
          card.style.setProperty("--mx", ((event.clientX - box.left) / box.width * 100) + "%");
          card.style.setProperty("--my", ((event.clientY - box.top) / box.height * 100) + "%");
        });
        const nick = document.getElementById("set-nick");
        const looking = document.getElementById("set-looking");
        if (nick) {
          nick.value = localStorage.getItem("velvet-nick") || "";
          nick.onchange = () => localStorage.setItem("velvet-nick", nick.value.trim());
        }
        if (looking) {
          looking.checked = localStorage.getItem("velvet-looking") === "on";
          looking.onchange = () => {
            if (looking.checked) {
              const mins = Number((document.getElementById("look-for") || {}).value || 0);
              if (mins) localStorage.setItem("velvet-until", String(Date.now() + mins * 60000));
              else localStorage.removeItem("velvet-until");
            } else localStorage.removeItem("velvet-until");
            localStorage.setItem("velvet-looking", looking.checked ? "on" : "off");
            rememberPrefs();
            pushRoom(lookingOn());
            paintLooking();
          };
        }
        sound.onchange = () => localStorage.setItem("velvet-sound", sound.checked ? "on" : "off");
        chime.onchange = () => localStorage.setItem("velvet-chime", chime.checked ? "on" : "off");
        join.onchange = () => localStorage.setItem("velvet-join", join.checked ? "on" : "off");
        pop.onchange = () => localStorage.setItem("velvet-pop", pop.checked ? "on" : "off");
        night.onchange = () => localStorage.setItem("velvet-night", night.checked ? "on" : "off");
        const quietFrom = document.getElementById("set-quiet-from");
        const quietTo = document.getElementById("set-quiet-to");
        if (quietFrom && quietTo) {
          quietFrom.value = localStorage.getItem("velvet-quiet-from") || "";
          quietTo.value = localStorage.getItem("velvet-quiet-to") || "";
          const saveQuiet = () => {
            localStorage.setItem("velvet-quiet-from", quietFrom.value || "");
            localStorage.setItem("velvet-quiet-to", quietTo.value || "");
          };
          quietFrom.onchange = saveQuiet;
          quietTo.onchange = saveQuiet;
        }
        let volumeTimer = 0;
        volume.oninput = () => {
          localStorage.setItem("velvet-volume", volume.value);
          volumeN.textContent = volume.value + "%";
          clearTimeout(volumeTimer);
          volumeTimer = setTimeout(() => playPack(true), 140);
        };
        if (resultVolumeInput && resultVolumeN) {
          let resultTimer = 0;
          resultVolumeInput.oninput = () => {
            localStorage.setItem("velvet-result-volume", resultVolumeInput.value);
            resultVolumeN.textContent = resultVolumeInput.value + "%";
            clearTimeout(resultTimer);
            resultTimer = setTimeout(() => playResult("win", true), 140);
          };
        }
        function hearSelect(node, play) {
          if (!node || node.dataset.hear === "1") return;
          node.dataset.hear = "1";
          let heard = 0;
          const once = () => {
            const now = Date.now();
            if (now - heard < 280) return;
            heard = now;
            play();
          };
          node.addEventListener("change", () => once());
          node.addEventListener("mousedown", (event) => {
            if (event.button !== 0) return;
            event.preventDefault();
            node.focus();
            document.querySelectorAll(".sound-menu").forEach((menu) => menu.remove());
            const menu = document.createElement("div");
            menu.className = "sound-menu";
            const rect = node.getBoundingClientRect();
            menu.style.left = Math.max(8, rect.left) + "px";
            menu.style.top = Math.min(window.innerHeight - 12, rect.bottom + 4) + "px";
            menu.style.width = Math.max(180, rect.width) + "px";
            menu.style.maxHeight = Math.max(140, window.innerHeight - rect.bottom - 16) + "px";
            for (const option of [...node.options]) {
              if (!option.value && option.hidden) continue;
              const row = document.createElement("button");
              row.type = "button";
              row.textContent = option.textContent;
              if (option.value === node.value) row.classList.add("on");
              row.addEventListener("mousedown", (click) => {
                click.preventDefault();
                click.stopPropagation();
                node.value = option.value;
                node.dispatchEvent(new Event("change"));
                menu.remove();
              });
              menu.append(row);
            }
            document.body.append(menu);
            const close = (click) => {
              if (menu.contains(click.target)) return;
              menu.remove();
              document.removeEventListener("mousedown", close, true);
            };
            setTimeout(() => document.addEventListener("mousedown", close, true), 0);
          });
        }
        hearSelect(pack, () => { localStorage.setItem("velvet-soundpack", pack.value); playPack(true); });
        const winPick = document.getElementById("set-win");
        const lossPick = document.getElementById("set-loss");
        if (winPick) {
          winPick.value = localStorage.getItem("velvet-win") || "gg";
          hearSelect(winPick, () => { localStorage.setItem("velvet-win", winPick.value); playResult("win", true); });
        }
        if (lossPick) {
          lossPick.value = localStorage.getItem("velvet-loss") || "sad";
          hearSelect(lossPick, () => { localStorage.setItem("velvet-loss", lossPick.value); playResult("loss", true); });
        }
        const bindFile = (id, kind, after) => {
          const input = document.getElementById(id);
          if (!input) return;
          input.onchange = () => {
            const file = input.files && input.files[0];
            if (!file) return;
            fetch("/api/media?kind=" + kind, { method: "POST", body: file }).then((res) => { if (res.ok && after) after(); }).catch(() => {});
          };
        };
        bindFile("set-notify-file", "notify", () => { localStorage.setItem("velvet-soundpack", "custom"); pack.value = "custom"; playPack(true); });
        bindFile("set-win-file", "win", () => { localStorage.setItem("velvet-win", "custom"); if (winPick) winPick.value = "custom"; playResult("win", true); });
        bindFile("set-loss-file", "loss", () => { localStorage.setItem("velvet-loss", "custom"); if (lossPick) lossPick.value = "custom"; playResult("loss", true); });
        const bindSound = (id, kind) => {
          const node = document.getElementById(id);
          if (!node) return;
          const defaults = { friend: "none", watch: "none", played: "none" };
          node.value = localStorage.getItem("velvet-" + kind + "-sound") || defaults[kind];
          hearSelect(node, () => { localStorage.setItem("velvet-" + kind + "-sound", node.value); playSaved(kind, true); });
        };
        bindSound("set-friend-sound", "friend");
        bindSound("set-watch-sound", "watch");
        bindSound("set-played-sound", "played");
        bindFile("set-friend-file", "friend", () => { localStorage.setItem("velvet-friend-sound", "custom"); const node = document.getElementById("set-friend-sound"); if (node) node.value = "custom"; playSaved("friend", true); });
        bindFile("set-watch-file", "watch", () => { localStorage.setItem("velvet-watch-sound", "custom"); const node = document.getElementById("set-watch-sound"); if (node) node.value = "custom"; playSaved("watch", true); });
        bindFile("set-played-file", "played", () => { localStorage.setItem("velvet-played-sound", "custom"); const node = document.getElementById("set-played-sound"); if (node) node.value = "custom"; playSaved("played", true); });
        function shadeHex(hex, amount) {
          const n = parseInt(String(hex || "").slice(1), 16);
          if (!Number.isFinite(n)) return "#10080d";
          const mix = (channel) => Math.max(0, Math.min(255, Math.round(channel * amount)));
          return "#" + [mix((n >> 16) & 255), mix((n >> 8) & 255), mix(n & 255)].map((value) => value.toString(16).padStart(2, "0")).join("");
        }
        function samplePalette(img) {
          const canvas = document.createElement("canvas");
          canvas.width = 32;
          canvas.height = 32;
          const ctx = canvas.getContext("2d", { willReadFrequently: true });
          ctx.drawImage(img, 0, 0, 32, 32);
          const data = ctx.getImageData(0, 0, 32, 32).data;
          let accent = { sat: 0, color: "#ff5fa2" };
          let dark = { light: 1, color: "#10080d" };
          for (let i = 0; i < data.length; i += 16) {
            const red = data[i], green = data[i + 1], blue = data[i + 2];
            const max = Math.max(red, green, blue) / 255;
            const min = Math.min(red, green, blue) / 255;
            const light = (max + min) / 2;
            const sat = max === min ? 0 : (max - min) / (1 - Math.abs(2 * light - 1));
            const color = "#" + [red, green, blue].map((value) => value.toString(16).padStart(2, "0")).join("");
            if (light < dark.light) dark = { light, color };
            if (sat > accent.sat && light > 0.22 && light < 0.82) accent = { sat, color };
          }
          return { bg: shadeHex(dark.color, 0.42), panel: shadeHex(dark.color, 0.72), accent: accent.color, you: accent.color, enemy: accent.color };
        }
        function applyPalette(colors) {
          const root = document.documentElement.style;
          root.setProperty("--bg", colors.bg);
          root.setProperty("--panel", colors.panel);
          root.setProperty("--accent", colors.accent);
          root.setProperty("--you", colors.you);
          root.setProperty("--enemy", colors.enemy);
          document.body.classList.add("custom-bg");
          document.body.style.background = "linear-gradient(180deg, rgba(6,3,6,.4), rgba(6,3,6,.7)), url(/api/media?kind=background)";
        }
        function loadBackdrop() {
          const img = new Image();
          img.onload = () => applyPalette(samplePalette(img));
          img.src = "/api/media?kind=background&t=" + Date.now();
        }
        const bg = document.getElementById("set-bg");
        const bgFile = document.getElementById("set-bg-file");
        const bgClear = document.getElementById("set-bg-clear");
        if (bg && bgFile) bg.onclick = () => bgFile.click();
        if (bgFile) bgFile.onchange = () => {
          const file = bgFile.files && bgFile.files[0];
          if (!file) return;
          fetch("/api/media?kind=background", { method: "POST", body: file }).then((res) => {
            if (!res.ok) return;
            localStorage.setItem("velvet-backdrop", "on");
            loadBackdrop();
          }).catch(() => {});
        };
        if (bgClear) bgClear.onclick = () => {
          localStorage.setItem("velvet-backdrop", "off");
          document.body.classList.remove("custom-bg");
          fetch("/api/media?kind=background", { method: "DELETE" }).catch(() => {});
          applyTheme(theme);
        };
        if (localStorage.getItem("velvet-backdrop") === "on") loadBackdrop();
        const saveDiscord = () => {
          localStorage.setItem("velvet-discord", discord.checked ? "on" : "off");
          localStorage.setItem("velvet-discord-url", discordURLInput.value.trim());
          localStorage.setItem("velvet-discord-app", discordApp.value.trim());
          localStorage.setItem("velvet-discord-status", discordStatus.checked ? "on" : "off");
          localStorage.setItem("velvet-discord-rounds", discordRounds.checked ? "on" : "off");
          localStorage.setItem("velvet-discord-unfriend", discordUnfriend.checked ? "on" : "off");
          localStorage.setItem("velvet-discord-live", discordLive.checked ? "on" : "off");
          localStorage.setItem("velvet-discord-end", discordEnd.checked ? "on" : "off");
          localStorage.setItem("velvet-discord-watch", discordWatch.checked ? "on" : "off");
          rememberPrefs();
        };
        discord.onchange = saveDiscord;
        discordURLInput.oninput = saveDiscord;
        discordApp.oninput = saveDiscord;
        discordStatus.onchange = () => { saveDiscord(); statusKey = ""; };
        discordRounds.onchange = saveDiscord;
        discordUnfriend.onchange = saveDiscord;
        discordLive.onchange = saveDiscord;
        discordEnd.onchange = saveDiscord;
        discordWatch.onchange = saveDiscord;
        document.getElementById("set-discord-test").onclick = async () => {
          const note = document.getElementById("discord-note");
          const url = discordURLInput.value.trim();
          const app = discordApp.value.trim();
          localStorage.setItem("velvet-discord-url", url);
          localStorage.setItem("velvet-discord-app", app);
          if (!url && !app) { note.textContent = "Paste a webhook or an application id."; return; }
          note.textContent = "Sending…";
          let channel = "";
          let profile = "";
          if (url) {
            try {
              const result = await (await fetch("/api/discord", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ url, title: "Valvet is connected", text: "Match alerts from this PC will land in this channel." }),
              })).json();
              channel = result.ok ? "Channel alert sent." : (result.error || "Discord rejected the webhook.");
            } catch {
              channel = "Couldn't reach Discord.";
            }
          }
          if (app) {
            try {
              const result = await (await fetch("/api/presence", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ clientId: app, details: "Valvet", state: "Connected" }),
              })).json();
              profile = result.ok ? "Profile status is live." : (result.error || "Profile status failed.");
            } catch {
              profile = "Couldn't reach Discord.";
            }
          }
          note.textContent = [channel, profile].filter(Boolean).join(" ");
        };
        compact.onchange = () => {
          localStorage.setItem("velvet-compact", compact.checked ? "on" : "off");
          document.body.classList.toggle("compact", compact.checked);
        };
        const fields = [["bg", "Background"], ["panel", "Cards"], ["accent", "Accent"], ["you", "Your side"], ["enemy", "Enemy"]];
        const pickers = document.getElementById("pickers");
        function paintPickers() {
          pickers.replaceChildren();
          for (const [key, label] of fields) {
            const row = el("label", "set-row");
            row.append(el("span", null, label));
            const input = document.createElement("input");
            input.type = "color";
            input.value = theme[key];
            input.oninput = () => { theme = { ...theme, [key]: input.value }; localStorage.setItem(THEME_KEY, JSON.stringify(theme)); applyTheme(theme); markThemes(); };
            row.append(input);
            pickers.append(row);
          }
        }
        function themeChip(name, preset) {
          const dot = el("button", "theme-chip");
          dot.type = "button";
          const swatch = document.createElement("i");
          swatch.style.background = preset.accent;
          dot.append(swatch, el("span", null, name));
          dot.onclick = () => { theme = { ...preset }; localStorage.setItem(THEME_KEY, JSON.stringify(theme)); applyTheme(theme); paintPickers(); markThemes(); };
          return dot;
        }
        const swatchHost = document.getElementById("swatches");
        const sampleHost = document.getElementById("sample-swatches");
        for (const [name, preset] of Object.entries(THEMES)) {
          if (swatchHost) swatchHost.append(themeChip(name, preset));
          if (sampleHost) sampleHost.append(themeChip(name, preset));
        }
        const sampleShape = document.getElementById("sample-shape");
        if (sampleShape && !sampleShape.dataset.bound) {
          sampleShape.dataset.bound = "1";
          sampleShape.addEventListener("click", (event) => {
            const btn = event.target.closest("button");
            if (!btn) return;
            if (btn.dataset.corners) {
              localStorage.setItem("velvet-corners", btn.dataset.corners);
              if (corners) corners.value = btn.dataset.corners;
            }
            if (btn.dataset.motion) {
              localStorage.setItem("velvet-motion", btn.dataset.motion);
              if (motion) motion.value = btn.dataset.motion;
            }
            applyShape();
          });
        }
        const sampleGlass = document.getElementById("sample-glass");
        const glassBox = document.getElementById("set-glass");
        if (sampleGlass && glassBox && !sampleGlass.dataset.bound) {
          sampleGlass.dataset.bound = "1";
          sampleGlass.onchange = () => {
            glassBox.checked = sampleGlass.checked;
            glassBox.dispatchEvent(new Event("change"));
          };
        }
        paintPickers();
        markThemes();
        const resetLook = document.getElementById("reset-look");
        if (resetLook) resetLook.onclick = () => {
          theme = { ...THEMES.Valvet };
          localStorage.setItem(THEME_KEY, JSON.stringify(theme));
          applyTheme(theme);
          localStorage.setItem("velvet-corners", "soft");
          localStorage.setItem("velvet-motion", "full");
          localStorage.setItem("velvet-glass", "off");
          if (corners) corners.value = "soft";
          if (motion) motion.value = "full";
          const box = document.getElementById("set-glass");
          if (box) {
            box.checked = false;
            box.dispatchEvent(new Event("change"));
          }
          applyShape();
          paintPickers();
          markThemes();
        };
      } catch {}

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
          grabJSON("https://raw.githubusercontent.com/qdxnnyq/valvet/main/version.json?t=" + stamp, 7000),
          grabJSON("https://api.github.com/repos/qdxnnyq/valvet/releases/latest", 7000),
          grabJSON("/api/update?fresh=1&t=" + stamp, 7000),
        ]);
        if (ticket !== updateTicket) return;
        const remoteOk = !!(file && (file.name || file.version)) || !!releaseName(rel);
        if (!remoteOk && (!local || local.error)) {
          paintUpdate({ label: (local && local.label) || NOTES_VER, update: false });
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
        if (localStorage.getItem("velvet-sound") === "off") return;
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
        { tab: "live", mark: "challenge", kicker: "Live", title: "The match", body: "Your side is on the left and the other side is on the right. Each player shows name, agent, party, and rank. The header is the average rank of that side." },
        { tab: "live", mark: "post", card: "sample", kicker: "After the match", title: "Result", body: "When the match ends, this card shows the result. Copy card saves a picture." },
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
      function renderTour() {
        const step = TOUR[tourAt];
        document.getElementById("tour-kicker").textContent = (tourAt + 1) + " / " + TOUR.length + " · " + step.kicker;
        document.getElementById("tour-title").textContent = step.title;
        document.getElementById("tour-body").textContent = step.body;
        document.getElementById("tour-next").textContent = tourAt === TOUR.length - 1 ? "Done" : "Next";
        const dots = document.getElementById("tour-dots");
        dots.replaceChildren();
        TOUR.forEach((_, index) => {
          const dot = document.createElement("i");
          if (index === tourAt) dot.className = "on";
          dots.append(dot);
        });
        document.querySelectorAll(".tour-mark").forEach((node) => node.classList.remove("tour-mark"));
        quietNav = true;
        setTab(step.tab);
        quietNav = false;
        if (step.card === "sample") showSampleCard();
        else hideSampleCard();
        const challenge = document.getElementById("challenge");
        if (challenge && challenge.dataset.sample === "1" && step.mark !== "challenge") {
          challenge.dataset.sample = "";
          paintChallenge();
        }
        const mark = document.getElementById(step.mark);
        if (mark) {
          if (step.mark === "challenge" && mark.hidden) {
            mark.hidden = false;
            mark.dataset.sample = "1";
            mark.replaceChildren(el("b", null, "Daily"), document.createTextNode(" Win a game on Ascent."));
          }
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
        const v = localStorage.getItem("velvet-tour");
        return v === "43" || v === "done" || v === "seen";
      }
      function markNotes() {
        localStorage.setItem("velvet-notes", NOTES_VER);
      }
      function showWhatsNew() {
        if (localStorage.getItem("velvet-notes") === NOTES_VER) return;
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
        localStorage.setItem("velvet-tour", "seen");
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
        if (tourAt >= TOUR.length - 1) closeTour();
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
        const pick = allowed.indexOf(name) >= 0 ? name : (localStorage.getItem("velvet-set-tab") || "look");
        const use = allowed.indexOf(pick) >= 0 ? pick : "look";
        localStorage.setItem("velvet-set-tab", use);
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
      applySetTab(localStorage.getItem("velvet-set-tab") || "look");
      const frame = document.getElementById("app-frame");
      if (frame) {
        const frameMin = document.getElementById("frame-min");
        const frameMax = document.getElementById("frame-max");
        const frameClose = document.getElementById("frame-close");
        if (frameMin && window.velvetWin) frameMin.onclick = () => velvetWin("min");
        if (frameMax && window.velvetWin) frameMax.onclick = () => velvetWin("max");
        if (frameClose && window.velvetWin) frameClose.onclick = () => velvetWin("close");
        frame.addEventListener("pointerdown", (event) => {
          if (event.button !== 0 || event.target.closest("button")) return;
          if (window.velvetDrag) velvetDrag();
        });
        frame.addEventListener("dblclick", (event) => {
          if (event.target.closest("button") || !window.velvetWin) return;
          velvetWin("max");
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
            localStorage.setItem("velvet-hidden-tabs", JSON.stringify(next));
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
        const q = String(text || "").trim().toLowerCase();
        if (!q) return rows || [];
        const on = q.match(/\bon\s+([a-z0-9' ]+?)(?=\s+after|\s+as|$)/);
        const asAgent = q.match(/\bas\s+([a-z0-9/'' ]+)$/);
        const after = q.match(/after round\s+(\d+)/);
        const map = on ? on[1].trim() : "";
        const agent = asAgent ? asAgent[1].trim() : "";
        const roundN = after ? Number(after[1]) : 0;
        const wantDeaths = /\bdeaths?\b/.test(q);
        const wantKills = /\bkills?\b/.test(q);
        const wantWins = /\bwins?\b/.test(q);
        return (rows || []).filter((match) => {
          if (map && !String(match.map || "").toLowerCase().includes(map)) return false;
          const you = (match.players || []).find((player) => player.you);
          if (agent && !(you && String(you.agent || "").toLowerCase().includes(agent))) return false;
          if (wantWins && match.won !== true) return false;
          if (wantDeaths && roundN && !(match.deaths || []).some((spot) => (spot.round || 0) > roundN) && !(match.rounds || []).some((round) => round.n > roundN && !round.you)) return false;
          if (wantKills && roundN && !(match.edges || []).some((edge) => edge.round > roundN) && !(match.kills || []).some((spot) => (spot.round || 0) > roundN)) return false;
          if ((wantDeaths || wantKills) && !roundN && wantDeaths && !(match.deaths || []).length && !(match.rounds || []).some((round) => !round.you)) return false;
          return true;
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
        const raw = String(localStorage.getItem("velvet-light-" + name) || defaults[name] || "#ff5fa2").trim();
        const hex = raw.replace("#", "");
        if (!/^[0-9a-fA-F]{6}$/.test(hex)) return;
        lightRoom(parseInt(hex.slice(0, 2), 16), parseInt(hex.slice(2, 4), 16), parseInt(hex.slice(4, 6), 16));
      }
      function lightRoom(r, g, b) {
        const kind = localStorage.getItem("velvet-lights") || "off";
        if (!kind || kind === "off") return;
        fetch("/api/lights", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ kind, open: localStorage.getItem("velvet-openrgb") || "127.0.0.1", wled: localStorage.getItem("velvet-wled") || "", r, g, b }) }).catch(() => {});
      }
      function notePop(data) {
        const start = Number(localStorage.getItem("velvet-queue-start") || 0);
        const wait = start ? Math.max(0, Date.now() - start) : 0;
        const you = ((data && data.players) || []).find((player) => player.you) || ((data && data.party) || []).find((player) => player.you);
        const rows = readJSON("velvet-pops") || [];
        rows.push({ t: Date.now(), hour: new Date().getHours(), wait, rank: (you && you.rank) || "" });
        writeJSON("velvet-pops", rows.slice(-180));
        localStorage.removeItem("velvet-queue-start");
        paintQueueClock();
      }
      function paintQueueClock() {
        const node = document.getElementById("queue-clock");
        if (!node) return;
        const rows = readJSON("velvet-pops") || [];
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
        if (phase === "In queue" && !localStorage.getItem("velvet-queue-start")) localStorage.setItem("velvet-queue-start", String(Date.now()));
        if (phase !== "In queue") localStorage.removeItem("velvet-queue-start");
        const you = ((data && data.players) || []).find((player) => player.you);
        const saved = readJSON("velvet-colours") || {};
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
        const color = LIGHTS[localStorage.getItem("velvet-light") || "off"];
        if (!color) return null;
        return { color, strength: LIGHT_POWER[localStorage.getItem("velvet-light-power") || "soft"] || LIGHT_POWER.soft };
      }
      function startEngine() {
        const canvas = document.getElementById("engine");
        const on = !!lightState();
        document.body.classList.toggle("engine-off", !on);
        if (!canvas) return;
        canvas.style.display = on ? "block" : "none";
        if (!on || canvas.dataset.live === "1") return;
        canvas.dataset.live = "1";
        const mouse = { x: 0.5, y: 0.5 };
        addEventListener("pointermove", (event) => { mouse.x = event.clientX / innerWidth; mouse.y = event.clientY / innerHeight; });
        const boot = async () => {
          if (navigator.gpu) {
            try { if (await bootWebGPU(canvas, mouse)) return; } catch {}
          }
          bootWebGL(canvas, mouse);
        };
        boot();
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
        const saved = readJSON("velvet-cache-game");
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
      if (!readJSON("velvet-cache-game")) fetch("/api/game", { cache: "no-store" }).then((res) => res.json()).then((data) => { if (data && (data.rows || []).length) { writeJSON("velvet-cache-game", data); stampCache("velvet-cache-game"); } }).catch(() => {});
      const lightPick = document.getElementById("set-light");
      const lightPower = document.getElementById("set-light-power");
      if (lightPick) {
        const saved = localStorage.getItem("velvet-light") || "off";
        lightPick.value = [...lightPick.options].some((opt) => opt.value === saved) ? saved : "off";
        lightPick.onchange = () => { localStorage.setItem("velvet-light", lightPick.value); startEngine(); };
      }
      if (lightPower) {
        const saved = localStorage.getItem("velvet-light-power") || "soft";
        lightPower.value = [...lightPower.options].some((opt) => opt.value === saved) ? saved : "soft";
        lightPower.onchange = () => { localStorage.setItem("velvet-light-power", lightPower.value); startEngine(); };
      }
      const lights = document.getElementById("set-lights");
      const openHost = document.getElementById("set-openrgb");
      const wledHost = document.getElementById("set-wled");
      if (lights) {
        lights.value = localStorage.getItem("velvet-lights") || "off";
        lights.onchange = () => localStorage.setItem("velvet-lights", lights.value);
      }
      if (openHost) { openHost.value = localStorage.getItem("velvet-openrgb") || ""; openHost.onchange = () => localStorage.setItem("velvet-openrgb", openHost.value.trim()); }
      if (wledHost) { wledHost.value = localStorage.getItem("velvet-wled") || ""; wledHost.onchange = () => localStorage.setItem("velvet-wled", wledHost.value.trim()); }
      for (const [id, name, fallback] of [["light-kill", "kill", "#28dc78"], ["light-death", "death", "#dc2846"], ["light-clutch", "clutch", "#ffc440"], ["light-queue", "queue", "#ff5fa2"]]) {
        const input = document.getElementById(id);
        if (!input) continue;
        const saved = localStorage.getItem("velvet-light-" + name) || fallback;
        input.value = /^#[0-9a-fA-F]{6}$/.test(saved) ? saved : fallback;
        input.oninput = () => localStorage.setItem("velvet-light-" + name, input.value);
      }
      const lightTest = document.getElementById("set-light-test");
      if (lightTest) lightTest.onclick = () => lightRoom(255, 95, 162);
      const broadcast = document.getElementById("set-broadcast");
      if (broadcast) broadcast.onclick = () => window.open("/broadcast", "velvet-broadcast", "width=1100,height=260");
      startEngine();
      paintQueueClock();
      const thanks = document.getElementById("thanks");
      const notesCard = document.getElementById("whats-new");
      if (!localStorage.getItem("velvet-tour")) {
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
    
