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
        const mode = localStorage.getItem("valvet-heat") === "kills" ? "kills" : "deaths";
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
            localStorage.setItem("valvet-heat", id);
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
      function playClutch(row) {
        const phase = (latestState && latestState.phase) || "";
        if (phase === "In game" || phase === "Loading lobby" || phase === "Agent select") return;
        const match = (history || []).find((item) => row && row.id && item.id === row.id) || (history || []).find((item) => (item.rounds || []).some((round) => round.vs));
        if (match) openDirector(match);
      }
      function sessionLadder() {
        if (!rrPoints || rrPoints.length < 2) return null;
        const first = rrPoints[0];
        const last = rrPoints[rrPoints.length - 1];
        const box = el("div", "ladder");
        box.append(el("div", "eyebrow", "This act"));
        box.append(el("b", null, (first.rank || "Rank") + "  →  " + (last.rank || "Rank")));
        const track = el("div", "ladder-track");
        const marker = document.createElement("i");
        const start = (Number(first.tier) || 0) * 100 + (Number(first.rr) || 0);
        const end = (Number(last.tier) || 0) * 100 + (Number(last.rr) || 0);
        const span = Math.max(40, Math.abs(end - start));
        marker.style.left = Math.max(6, Math.min(94, 50 + ((end - start) / span) * 40)) + "%";
        track.append(marker);
        box.append(track);
        const delta = end - start;
        box.append(el("div", "sub", (delta >= 0 ? "+" : "") + delta + " RR across " + rrPoints.length + " games"));
        return box;
      }
      function paintClutches() {
        const root = document.getElementById("clutches");
        if (!root) return;
        const rows = clutchRows();
        const clutchSig = clutchPick + "|" + (clutchMore ? "1" : "0") + "|" + rows.map((row) => [row.map, row.n, row.fail ? 1 : 0, row.vs, row.agent].join(":")).join("|");
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
        for (let n = 1; n <= 3; n++) {
          const group = rows.filter((row) => Number(row.vs) === n);
          const wins = group.filter((row) => !row.fail).length;
          const pct = group.length ? Math.round((wins / group.length) * 100) : 0;
          const card = el("button", "clutch-pick" + (clutchPick === n ? " on" : ""));
          card.type = "button";
          card.append(el("div", "lab", "1v" + n));
          card.append(el("b", null, group.length ? pct + "%" : "—"));
          card.append(el("em", null, group.length ? wins + " held · " + (group.length - wins) + " died" : "none"));
          const meter = el("div", "clutch-meter");
          const fill = document.createElement("span");
          fill.style.width = (group.length ? pct : 0) + "%";
          meter.append(fill);
          card.append(meter);
          card.onclick = () => {
            clutchPick = clutchPick === n ? 0 : n;
            root.dataset.sig = "";
            paintClutches();
          };
          split.append(card);
        }
        board.append(rateCard, split);
        const playAll = el("button", "tiny solid", "Play clutches");
        playAll.type = "button";
        playAll.onclick = () => playClutch(rows[0]);
        board.append(playAll);
        root.append(board);
        const list = el("div", "clutch-list");
        const picked = clutchPick ? rows.filter((row) => Number(row.vs) === clutchPick) : rows;
        const shown = clutchMore ? picked : picked.slice(0, 8);
        shown.forEach((row) => {
          const card = el("article", "clutch-row " + (row.fail ? "died" : "held"));
          card.append(el("div", "vs", "1v" + row.vs));
          const mid = el("div");
          mid.append(el("b", null, [row.map || "Map", row.agent, "Round " + row.n].filter(Boolean).join(" · ")));
          const when = row.when ? new Date(row.when).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }) : "";
          mid.append(el("div", "sub", [row.side, row.how, row.mode, row.score, when].filter(Boolean).join(" · ")));
          card.append(mid, el("div", "end", row.fail ? "Died" : "Held"));
          card.onclick = () => playClutch(row);
          list.append(card);
        });
        root.append(list);
        if (picked.length > 8) {
          root.append(viewMore(clutchMore, picked.length - 8, () => {
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
        const saved = readJSON("valvet-team");
        return saved && typeof saved === "object" ? { name: saved.name || "", roster: saved.roster || [] } : { name: "", roster: [] };
      }
      function saveTeam(team) {
        writeJSON("valvet-team", team);
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
      async function loadPremier() {
        const root = document.getElementById("team");
        if (!root) return;
        paintTeam();
        let data = {};
        try { data = await (await fetch("/api/premier", { cache: "no-store" })).json(); }
        catch { data = { error: "No Premier team on this client" }; }
        let card = document.getElementById("premier-card");
        if (!card) {
          card = el("section", "set-card");
          card.id = "premier-card";
          root.prepend(card);
        }
        card.replaceChildren();
        card.append(el("div", "eyebrow", "Premier"));
        const roster = (data && data.roster) || [];
        if (!data || (!data.name && !roster.length)) {
          card.append(el("h2", null, "Not in a Premier team"));
          card.append(el("p", "sub", (data && data.error) || "The roster below is only for when this client has no Premier team."));
          return;
        }
        const typed = root.querySelector(".set-card:not(#premier-card)");
        if (typed) typed.hidden = true;
        const list = document.getElementById("team-list");
        if (list) list.hidden = true;
        card.append(el("h2", null, [data.name || "Premier", data.tag ? "#" + data.tag : ""].filter(Boolean).join(" ")));
        const bits = [data.division, data.points ? data.points + " pts" : "", data.when ? "When " + data.when : ""].filter(Boolean).join(" · ");
        if (bits) card.append(el("p", "sub", bits));
        if (data.maps) card.append(el("p", "sub", "Maps " + data.maps));
        if (!roster.length) card.append(el("p", "sub", "The client named the team and did not send the roster."));
        for (const mate of roster) card.append(el("div", "sub", [mate.name || "Player", mate.tag ? "#" + mate.tag : "", mate.role].filter(Boolean).join(" ")));
      }
      function guessMatchLength(match) {
        const mode = String((match && match.mode) || "").toLowerCase();
        if (/deathmatch|escalation|replication/.test(mode)) return 10 * 60000;
        if (/spike|swift|snowball|hurm/.test(mode)) return 14 * 60000;
        const rounds = ((match && match.rounds) || []).length;
        if (rounds) return rounds * 100000;
        return 35 * 60000;
      }
      function matchMs(match) {
        const n = Number(match && match.length) || 0;
        if (n >= 100000) return n;
        if (n > 0) return n * 1000;
        return guessMatchLength(match);
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
        for (const row of readJSON("valvet-sessions") || []) {
          if (!row || !row.end) continue;
          const key = dayStart(row.start || row.end);
          if (session[key] == null) continue;
          session[key] += Math.max(0, row.end - row.start);
        }
        const open = readJSON("valvet-session-open");
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
          matches[key] += matchMs(match);
        }
        return keys.map((key) => ({
          key,
          ms: (matches[key] || 0) || (session[key] || 0),
          now: key === today,
        })).map((bar) => bar.key === liveKey ? { ...bar, ms: bar.ms + liveMs } : bar);
      }
      let sessionDay = 0;
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
        plot.append(guide, tip);
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
          row.append(el("div", null, clockSpan(matchMs(match))));
          row.onclick = () => openMatchSheet(match);
          wrap.append(row);
        }
        return wrap;
      }
      function paintSession() {
        const root = document.getElementById("session");
        if (!root) return;
        const start = dayStart(Date.now());
        const rows = (readJSON("valvet-sessions") || []).filter((row) => row && row.end >= start);
        const open = readJSON("valvet-session-open");
        const games = dayMatches(start);
        let closed = 0;
        const items = [];
        for (const row of rows) closed += Math.max(0, row.end - row.start);
        let played = 0;
        for (const match of games) played += matchMs(match);
        let total = games.length ? played : closed;
        if (open && open.start) total += Math.max(0, Date.now() - open.start);
        if (games.length) {
          for (const match of games) {
            const when = Number(match.started) || Date.parse(match.when || "");
            items.push({ when: when ? new Date(when).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "—", queue: match.mode || "Game", map: match.map || "—", ms: matchMs(match), live: false });
          }
        } else {
          for (const row of rows) items.push({ when: new Date(row.start).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }), queue: row.queue || "Game", map: row.map || "—", ms: Math.max(0, row.end - row.start), live: false });
        }
        if (open && open.start) items.push({ when: "Now", queue: open.queue || "Game", map: open.map || "—", ms: Math.max(0, Date.now() - open.start), live: true });
        const picked = sessionDay ? dayMatches(sessionDay) : [];
        const sessionSig = [sessionDay, picked.map((match) => match.id).join(","), items.map((item) => [item.when, item.queue, item.map, Math.floor(item.ms / 60000), item.live ? 1 : 0].join(":")).join("|")].join("~");
        if (root.dataset.sig === sessionSig && root.querySelector(".session-hero, .empty")) return;
        root.dataset.sig = sessionSig;
        root.className = "";
        root.replaceChildren();
        root.append(paintMonth(monthBars()));
        const ladder = sessionLadder();
        if (ladder) root.append(ladder);
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
        return !!(person && person.mobile) || state === "mobile" || mode === "mobile" || mode === "riot mobile" || product === "bacon" || product.includes("mobile") || product.includes("bacon");
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
      const seenTouch = new Map();
      function stampSeen(people, fresh) {
        const book = readJSON("valvet-last-seen") || {};
        const now = Date.now();
        const here = new Set();
        let dirty = false;
        const put = (id, next) => {
          const prev = book[id];
          book[id] = next;
          if (!prev || prev.at !== next.at || prev.since !== next.since || prev.live !== next.live || prev.state !== next.state || prev.map !== next.map || prev.mode !== next.mode || prev.product !== next.product || prev.playSince !== next.playSince || prev.name !== next.name) dirty = true;
        };
        for (const person of people || []) {
          if (!person) continue;
          const id = String(person.puuid || "").toLowerCase();
          if (!id) continue;
          here.add(id);
          const prev = book[id] || {};
          if (isRiotMobile(person)) {
            const fake = !prev.real && prev.at && now - prev.at < 45 * 60000;
            if (!prev.real && !(prev.at > 0) && !fake) {
              if (book[id]) { delete book[id]; dirty = true; }
              continue;
            }
            put(id, {
              ...prev,
              name: person.name || prev.name || "Player",
              tag: person.tag || prev.tag || "",
              at: fake ? 0 : (prev.at || 0),
              live: false,
              since: 0,
              playSince: 0,
              mobileNow: true,
            });
            continue;
          }
          const online = onDesktop(person);
          const arrived = online && !prev.live;
          const state = online ? (person.state || "Online") : (prev.state || "Offline");
          const playing = /in game|agent select|loading/i.test(state);
          const wasPlaying = /in game|agent select|loading/i.test(String(prev.state || ""));
          if (online) seenTouch.set(id, now);
          let at = prev.at || 0;
          if (online) at = !prev.at || now - prev.at > 60000 ? now : prev.at;
          else if (prev.live && seenTouch.has(id)) at = seenTouch.get(id);
          put(id, {
            name: person.name || prev.name || "Player",
            tag: person.tag || prev.tag || "",
            at,
            since: online ? (arrived || !prev.since ? now : prev.since) : 0,
            playSince: playing ? (wasPlaying && prev.playSince ? prev.playSince : now) : 0,
            live: online,
            desk: online ? true : !!prev.desk,
            real: online ? true : !!prev.real,
            mobileNow: false,
            product: online ? (person.product || prev.product || "") : (prev.product || ""),
            state,
            mode: online ? (person.mode || "") : (prev.mode || ""),
            map: online ? (person.map || "") : (prev.map || ""),
          });
        }
        if (fresh) {
          for (const id of Object.keys(book)) {
            if (here.has(id) || !book[id] || !book[id].live) continue;
            const at = seenTouch.has(id) ? seenTouch.get(id) : book[id].at;
            put(id, { ...book[id], live: false, at, since: 0, playSince: 0 });
          }
        }
        if (dirty) writeJSON("valvet-last-seen", book);
      }
      function paintSeen() {
        const root = document.getElementById("seen");
        if (!root) return;
        const book = readJSON("valvet-last-seen") || {};
        const rows = Object.entries(book).map(([id, row]) => ({ ...row, id })).filter((row) => row && row.at && String(row.state || "").toLowerCase() !== "mobile" && String(row.product || "") !== "bacon").sort((a, b) => Number(!!b.live) - Number(!!a.live) || (b.at || 0) - (a.at || 0));
        const sig = rows.map((row) => [row.name, row.live ? 1 : 0, row.state, row.map, row.mode, row.product, row.at, row.playSince || 0].join(":")).join("|");
        if (root.dataset.sig === sig) return;
        root.dataset.sig = sig;
        const oldTip = document.getElementById("seen-tip");
        if (oldTip) oldTip.hidden = true;
        root.className = "";
        root.replaceChildren();
        if (!rows.length) {
          root.append(el("div", "empty", "Shows when a friend is online or AFK in the Riot Client or League. Riot Mobile is ignored."));
          return;
        }
        const place = (row) => {
          const state = String(row.state || "");
          const away = /afk|away|idle/i.test(state);
          const product = row.product || "";
          if (product === "league") {
            if (away) return "League · AFK";
            if (state && state !== "Online" && state !== "Offline") return "League · " + state;
            return "League";
          }
          if (product === "riot") return away ? "Riot Client · AFK" : "Riot Client";
          const map = row.map || "";
          if (map && product === "valorant") return map;
          if (product === "valorant") return away ? "Valorant · AFK" : (state && state !== "Online" && state !== "Offline" && state !== "In game" ? state : "Valorant");
          return away ? "AFK" : (state || "Riot Client");
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
          const show = () => showSeenTip(orb, row, where, clock, clockLabel(row.live ? (row.since || row.at) : row.at));
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
        const open = readJSON("valvet-session-open");
        const now = Date.now();
        if (playing) {
          if (!open) writeJSON("valvet-session-open", { start: now, queue: data.queue || data.mode || "Game", map: data.map || "" });
          else {
            const next = { ...open, map: data.map || open.map || "", queue: data.queue || data.mode || open.queue || "Game" };
            if (next.map !== open.map || next.queue !== open.queue) writeJSON("valvet-session-open", next);
          }
          return;
        }
        if (!open || !open.start) return;
        const rows = readJSON("valvet-sessions") || [];
        rows.push({ start: open.start, end: now, queue: open.queue || data.mode || "Game", map: open.map || "" });
        const cut = dayStart(now) - 29 * 86400000;
        writeJSON("valvet-sessions", rows.filter((row) => row && row.end >= cut).slice(-400));
        localStorage.removeItem("valvet-session-open");
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
        const key = "valvet-cache-news2";
        const saved = readJSON(key);
        const ok = Array.isArray(saved) && saved.length;
        const sig = ok ? saved.map((note) => (note.title || "") + (note.date || "")).join("|") : "";
        if (ok && root.dataset.sig === sig && root.querySelector(".patch-card")) {
          /* already on screen */
        } else if (ok) {
          paintNews(saved);
          root.dataset.sig = sig;
        }
        const force = live === "force";
        const need = force || !ok || !cacheFresh(key, 6 * 60 * 60 * 1000);
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
          const nextSig = notes.map((note) => (note.title || "") + (note.date || "")).join("|");
          if (force || root.dataset.sig !== nextSig) {
            paintNews(notes);
            root.dataset.sig = nextSig;
          }
        } catch {
          if (!ok) root.replaceChildren(el("div", "empty", "Couldn't load Valorant news."));
        }
      }
      function paintNews(notes) {
        const root = document.getElementById("news");
        if (!root) return;
        root.className = "";
        root.replaceChildren();
        savedBar(root, "valvet-cache-news2", () => loadNews("force"));
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
        const seen = localStorage.getItem("valvet-seen-" + name) || "";
        if (!seen) {
          localStorage.setItem("valvet-seen-" + name, sig);
          if (button) button.classList.remove("fresh");
          return;
        }
        const show = seen !== sig && tab !== name;
        if (tab === name) localStorage.setItem("valvet-seen-" + name, sig);
        if (button) button.classList.toggle("fresh", show);
      }
      function hiddenTabs() {
        try {
          const rows = JSON.parse(localStorage.getItem("valvet-hidden-tabs") || "[]");
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
        localStorage.setItem("valvet-tab", next);
        for (const name of TABS) {
          const pane = document.getElementById("pane-" + name);
          const button = document.getElementById("tab-" + name);
          if (!pane || !button) continue;
          const on = tab === name;
          pane.hidden = !on;
          button.hidden = tabHidden(name);
          button.className = on ? "on" : "";
          if (!on && freshSig[name] && localStorage.getItem("valvet-seen-" + name) !== freshSig[name]) button.classList.add("fresh");
          if (on && freshSig[name]) localStorage.setItem("valvet-seen-" + name, freshSig[name]);
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
        if (next === "store" && lastShop) drawStore(lastShop);
        if (next === "live") {
          const party = ((latestState && latestState.party) || []).filter((person) => person && !isLeague(person));
          paintToday(history);
          paintStreak(history);
          paintDuos("duos", party, history);
        }
        if (next === "loadout") loadLoadout(!!user);
        if (next === "bundle" && lastShop) { bundleKey = ""; drawBundle(lastShop); }
        if (next === "patches") loadPatches(!!user);
        if (next === "esports") loadEsports(!!user);
        if (next === "team") loadPremier();
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
      async function loadGame(live) {
        const root = document.getElementById("game");
        const key = "valvet-cache-game";
        const saved = readJSON(key);
        const ok = saved && (saved.rows || []).length;
        const sig = ok ? (saved.rows || []).map((row) => (row.label || row.name || "") + (row.value || "")).join("|") : "";
        if (ok && root.dataset.sig === sig && root.querySelector(".set-card")) {
          /* already on screen */
        } else if (ok) {
          paintGame(saved);
          root.dataset.sig = sig;
        }
        const need = !ok || (live === "force") || (live && !cacheFresh(key, 6 * 60 * 60 * 1000));
        if (!need) return;
        if (!ok) { root.className = "empty"; root.textContent = "Reading the local Valorant config…"; }
        try {
          const data = await (await fetch("/api/game", { cache: "no-store" })).json();
          if (data && (data.rows || []).length) {
            writeJSON(key, data);
            stampCache(key);
          }
          const nextSig = (data.rows || []).map((row) => (row.label || row.name || "") + (row.value || "")).join("|");
          if (!saved || root.dataset.sig !== nextSig) {
            paintGame(data);
            root.dataset.sig = nextSig;
          }
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
        input.value = localStorage.getItem("valvet-dpi") || "800";
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
        target.value = localStorage.getItem("valvet-edpi-target") || "";
        want.append(target);
        const friend = document.createElement("input");
        friend.type = "number";
        friend.min = "1";
        friend.max = "20000";
        friend.step = "1";
        friend.placeholder = "eDPI";
        friend.setAttribute("aria-label", "Friend eDPI");
        friend.value = localStorage.getItem("valvet-friend-edpi") || "";
        const out = el("div", "edpi-out");
        const versus = el("div", "edpi-vs");
        const meter = el("div", "edpi-meter");
        meter.hidden = true;
        const pretty = (n) => {
          if (!Number.isFinite(n)) return "—";
          const text = n >= 100 ? String(Math.round(n)) : String(Math.round(n * 100) / 100);
          return text;
        };
        const draw = () => {
          const dpi = Number(input.value);
          localStorage.setItem("valvet-dpi", input.value || "");
          localStorage.setItem("valvet-edpi-target", target.value || "");
          localStorage.setItem("valvet-friend-edpi", friend.value || "");
          out.replaceChildren();
          const chip = (k, v) => {
            const card = el("article");
            card.append(el("span", null, k), el("b", null, v));
            return card;
          };
          const ready = Number.isFinite(sens) && sens > 0 && Number.isFinite(dpi) && dpi > 0;
          const edpi = ready ? dpi * sens : 0;
          if (!ready) out.append(chip("eDPI", "—"));
          else {
            const cm = (2.54 * 360) / (dpi * sens * 0.07);
            out.append(chip("eDPI", pretty(edpi)));
            out.append(chip("cm/360", pretty(cm) + " cm"));
            out.append(chip("in/360", pretty(cm / 2.54) + " in"));
            if (scoped && scoped !== 1) out.append(chip("Scoped eDPI", pretty(edpi * scoped)));
            const goal = Number(target.value);
            if (Number.isFinite(goal) && goal > 0) out.append(chip("Set sens", pretty(goal / dpi)));
          }
          const theirs = Number(friend.value);
          const youCard = el("article");
          youCard.append(el("span", null, "You"));
          youCard.append(el("b", null, edpi ? pretty(edpi) : "—"));
          youCard.append(el("em", null, ready ? pretty(dpi) + " DPI" : "Your eDPI"));
          const themCard = el("article");
          themCard.append(el("span", null, "Friend"));
          themCard.append(friend);
          const gap = Number.isFinite(theirs) && theirs > 0 && edpi ? Math.round(edpi - theirs) : null;
          themCard.append(el("em", null, gap == null ? "Type their eDPI" : gap === 0 ? "Same as you" : gap > 0 ? "You are " + gap + " higher" : "They are " + Math.abs(gap) + " higher"));
          versus.replaceChildren(youCard, themCard);
          meter.hidden = gap == null;
          meter.replaceChildren();
          if (gap != null) {
            const top = Math.max(edpi, theirs, 1);
            const youBar = el("i", "you");
            const themBar = el("i", "them");
            youBar.style.width = Math.round((edpi / top) * 100) + "%";
            themBar.style.width = Math.round((theirs / top) * 100) + "%";
            meter.append(youBar, themBar);
          }
        };
        input.oninput = draw;
        target.oninput = draw;
        friend.oninput = draw;
        box.append(field, chips, want, versus, meter, out, el("p", "sub hint", "eDPI is DPI × sensitivity. cm/360 uses Valorant's 0.07 yaw."));
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
        if (unfriendReady && localStorage.getItem("valvet-discord-unfriend") !== "off") {
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
        localStorage.setItem("valvet-size", size);
        document.body.dataset.size = size;
        document.querySelectorAll("#sizes button").forEach((button) => button.classList.toggle("on", button.dataset.size === size));
        if (typeof valvetResize === "function") valvetResize(size).catch(() => {});
      }
      document.querySelectorAll("#sizes button").forEach((button) => {
        button.onclick = () => applySize(button.dataset.size);
      });
      applySize(localStorage.getItem("valvet-size") || "standard");
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
        goal.value = localStorage.getItem("valvet-rr-goal") || "";
        goal.onchange = () => {
          const n = Math.max(0, Math.min(500, Number(goal.value) || 0));
          if (n) localStorage.setItem("valvet-rr-goal", String(n));
          else localStorage.removeItem("valvet-rr-goal");
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
        const bag = { valvet: 27, watch: social.watch || [] };
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i);
          if (key && key.startsWith("valvet-")) bag[key] = localStorage.getItem(key);
        }
        const button = document.getElementById("set-export");
        const blob = new Blob([JSON.stringify(bag, null, 2)], { type: "application/json" });
        const link = document.createElement("a");
        link.href = URL.createObjectURL(blob);
        link.download = "valvet-backup.json";
        link.click();
        if (button) {
          button.textContent = "Exported";
          setTimeout(() => { button.textContent = "Export"; }, 1200);
        }
        setTimeout(() => URL.revokeObjectURL(link.href), 2000);
      };
      document.getElementById("set-import").onclick = () => document.getElementById("set-file").click();
      document.getElementById("set-file").onchange = async (event) => {
        const file = event.target.files && event.target.files[0];
        event.target.value = "";
        if (!file) return;
        let bag;
        try { bag = JSON.parse(await file.text()); }
        catch { return; }
        if (!bag || typeof bag !== "object") return;
        Object.keys(bag).forEach((key) => {
          if (typeof bag[key] !== "string") return;
          const next = key.startsWith("velvet-") ? "valvet-" + key.slice(7) : key;
          if (next.startsWith("valvet-")) localStorage.setItem(next, bag[key]);
        });
        const social = await (await fetch("/api/social", { cache: "no-store" })).json().catch(() => ({ watch: [] }));
        for (const item of social.watch || []) await fetch("/api/social?q=" + encodeURIComponent(item.name + "#" + item.tag), { method: "DELETE" });
        for (const item of bag.watch || []) await fetch("/api/social?q=" + encodeURIComponent(item.name + "#" + item.tag), { method: "POST" });
        applySize(localStorage.getItem("valvet-size") || "standard");
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
        if (ageEl) localStorage.setItem("valvet-age", ageEl.value.trim());
        if (noteEl) localStorage.setItem("valvet-note", noteEl.value.trim());
        if (genderEl) localStorage.setItem("valvet-gender", genderEl.value);
        if (queueEl) localStorage.setItem("valvet-queue", queueEl.value);
        const on = !lookingOn();
        if (on) {
          const age = Number(localStorage.getItem("valvet-age") || 0);
          if (age && age < 16) {
            const state = document.getElementById("look-state");
            if (state) state.textContent = "Leave age empty, or enter 16 or older.";
            return;
          }
        }
        localStorage.setItem("valvet-looking", on ? "on" : "off");
        localStorage.setItem("valvet-ready", "off");
        if (on) {
          const mins = Number((document.getElementById("look-for") || {}).value || 0);
          if (mins) localStorage.setItem("valvet-until", String(Date.now() + mins * 60000));
          else localStorage.removeItem("valvet-until");
        } else localStorage.removeItem("valvet-until");
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
      const showCode = document.getElementById("show-code");
      if (showCode) showCode.onclick = () => {
        const node = document.getElementById("my-code");
        if (!node) return;
        const open = node.dataset.open !== "1";
        node.dataset.open = open ? "1" : "";
        showCode.textContent = open ? "Hide" : "Show";
        paintRoom();
      };
      paintRoom();
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
      fillRanks("rank-low", "valvet-rank-low");
      fillRanks("rank-high", "valvet-rank-high");
      const serverPick = document.getElementById("stack-server");
      if (serverPick) {
        serverPick.value = localStorage.getItem("valvet-server") || "";
        serverPick.onchange = () => { localStorage.setItem("valvet-server", serverPick.value); pushRoom(lookingOn()); refreshRoom(); };
      }
      const lookFor = document.getElementById("look-for");
      if (lookFor) {
        lookFor.value = localStorage.getItem("valvet-look-for") || "0";
        lookFor.onchange = () => {
          localStorage.setItem("valvet-look-for", lookFor.value);
          if (localStorage.getItem("valvet-looking") === "on") {
            const mins = Number(lookFor.value || 0);
            if (mins) localStorage.setItem("valvet-until", String(Date.now() + mins * 60000));
            else localStorage.removeItem("valvet-until");
          }
          pushRoom(lookingOn());
        };
      }
      const AGENTS = ["Astra", "Breach", "Brimstone", "Chamber", "Clove", "Cypher", "Deadlock", "Fade", "Gekko", "Harbor", "Hush", "Iso", "Jett", "KAY/O", "Killjoy", "Neon", "Omen", "Phoenix", "Raze", "Reyna", "Sage", "Skye", "Sova", "Tejo", "Viper", "Vyse", "Waylay", "Yoru"];
      const agentPick = document.getElementById("want-agent");
      if (agentPick) {
        for (const name of AGENTS) agentPick.append(new Option(name, name));
        agentPick.value = localStorage.getItem("valvet-agent") || "";
        agentPick.onchange = () => { localStorage.setItem("valvet-agent", agentPick.value); pushRoom(lookingOn()); };
      }
      const hideRank = document.getElementById("hide-rank");
      if (hideRank) {
        hideRank.checked = localStorage.getItem("valvet-hide-rank") === "on";
        hideRank.onchange = () => { localStorage.setItem("valvet-hide-rank", hideRank.checked ? "on" : "off"); pushRoom(lookingOn()); };
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
        shareDiscord.checked = localStorage.getItem("valvet-share-discord") === "on";
        discordName.value = localStorage.getItem("valvet-discord-name") || "";
        shareDiscord.onchange = () => { localStorage.setItem("valvet-share-discord", shareDiscord.checked ? "on" : "off"); pushRoom(lookingOn()); };
        discordName.onchange = () => { localStorage.setItem("valvet-discord-name", discordName.value.trim()); pushRoom(lookingOn()); };
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
      bindBoard("board-note", "valvet-note");
      bindBoard("board-age", "valvet-age");
      bindBoard("board-gender", "valvet-gender");
      bindBoard("board-queue", "valvet-queue");
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
      bindFilter("filter-server", "valvet-filter-server");
      bindFilter("filter-queue", "valvet-filter-queue");
      bindFilter("filter-mode", "valvet-filter-mode");
      bindFilter("filter-gender", "valvet-filter-gender");
      bindFilter("filter-age-min", "valvet-filter-age-min");
      bindFilter("filter-age-max", "valvet-filter-age-max");
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
      const savedTab = localStorage.getItem("valvet-tab");
      if (localStorage.getItem("valvet-tour") === "done" && savedTab && savedTab !== "missions" && TABS.includes(savedTab) && savedTab !== "live") setTab(savedTab);
      document.getElementById("scrim").onclick = (event) => { if (event.target.id === "scrim") event.currentTarget.hidden = true; };

      const XH_KEY = "valvet-crosshairs";
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
      const PRO_HAIRS = [
        ["TenZ", "0;P;c;5;h;0;0l;4;0o;2;0a;1;1b;0"],
        ["ZmjjKK", "0;s;1;P;c;7;u;000000FF;h;0;0l;3;0v;3;0g;1;0o;2;0a;1;0f;0;1b;0;S;c;0;s;1.2;o;1"],
        ["aspas", "0;s;1;P;c;5;o;1;0l;4;0v;3;0g;1;0o;0;0a;1;0f;0;1b;0;S;c;0;s;1.2;o;1"],
        ["yay", "0;s;1;P;o;1;d;1;z;1;0b;0;1b;0;S;o;1"],
        ["Demon1", "0;s;1;P;o;1;d;1;m;1;0b;0;1b;0"],
        ["Leo", "0;P;h;0;0l;3;0o;0;0a;1;0f;0;1b;0"],
        ["Derke", "0;P;c;8;b;1;t;1;o;1;z;2;a;1;0t;2;0l;6;0v;6;0o;3;0a;0.8;0s;1;0e;1;1t;2;1l;2;1v;2;1o;10;1a;0.35;1s;1;1e;1;u;FFFFFF;d;1;h;1;0g;0;1g;0;0f;0;1f;1;0m;0;1m;1;0b;0;1b;0;m;0"],
        ["Boaster", "0;s;1;P;c;1;o;1;d;1;0l;0;0o;2;0a;1;0f;0;1b;0;S;c;1;o;1"],
        ["Chronicle", "0;P;c;7;o;1;f;0;0t;1;0l;3;0v;3;0g;1;0o;2;0a;1;0f;0;1b;0"],
        ["Less", "0;s;1;P;c;8;u;000000FF;h;0;b;1;f;0;0l;4;0o;0;0a;1;0f;0;1b;0"]
      ];
      const proBtn = document.getElementById("xh-pros");
      if (proBtn) proBtn.onclick = () => {
        const next = loadXH();
        const have = new Set(next.map((item) => item.code));
        for (const [name, code] of PRO_HAIRS) if (!have.has(code)) next.push({ name, code });
        localStorage.setItem(XH_KEY, JSON.stringify(next.slice(0, 40)));
        drawXH();
        proBtn.textContent = "Added";
        setTimeout(() => { proBtn.textContent = "Add 10 pro codes"; }, 900);
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
        const THEME_KEY = "valvet-colours";
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
          if (window.valvetChrome) valvetChrome(next.bg, next.accent);
        }
        applyTheme(theme);
        const motion = document.getElementById("set-motion");
        const corners = document.getElementById("set-corners");
        const cats = document.getElementById("set-cats");
        const applyShape = () => {
          const motionValue = (motion && motion.value) || localStorage.getItem("valvet-motion") || "full";
          const cornerValue = (corners && corners.value) || localStorage.getItem("valvet-corners") || "soft";
          document.body.dataset.motion = motionValue;
          document.body.dataset.corners = cornerValue;
          document.body.classList.toggle("no-cats", !!(cats && !cats.checked));
          markThemes();
        };
        function markThemes() {
          const motionValue = document.body.dataset.motion || localStorage.getItem("valvet-motion") || "full";
          const cornerValue = document.body.dataset.corners || localStorage.getItem("valvet-corners") || "soft";
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
          motion.value = localStorage.getItem("valvet-motion") || "full";
          motion.onchange = () => { localStorage.setItem("valvet-motion", motion.value); applyShape(); };
        }
        if (corners) {
          corners.value = localStorage.getItem("valvet-corners") || "soft";
          corners.onchange = () => { localStorage.setItem("valvet-corners", corners.value); applyShape(); };
        }
        if (cats) {
          cats.checked = localStorage.getItem("valvet-cats") !== "off";
          cats.onchange = () => { localStorage.setItem("valvet-cats", cats.checked ? "on" : "off"); applyShape(); };
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
        sound.checked = localStorage.getItem("valvet-sound") !== "off";
        chime.checked = localStorage.getItem("valvet-chime") === "on";
        join.checked = localStorage.getItem("valvet-join") === "on";
        pop.checked = localStorage.getItem("valvet-pop") === "on";
        night.checked = localStorage.getItem("valvet-night") === "on";
        const wishNote = document.getElementById("set-wish-note");
        if (wishNote) {
          wishNote.checked = localStorage.getItem("valvet-wish-note") === "on";
          wishNote.onchange = () => {
            localStorage.setItem("valvet-wish-note", wishNote.checked ? "on" : "off");
            if (wishNote.checked) localStorage.removeItem("valvet-wish-sig");
          };
        }
        compact.checked = localStorage.getItem("valvet-compact") === "on";
        volume.value = localStorage.getItem("valvet-volume") || "15";
        volumeN.textContent = volume.value + "%";
        if (resultVolumeInput && resultVolumeN) {
          resultVolumeInput.value = localStorage.getItem("valvet-result-volume") || "40";
          resultVolumeN.textContent = resultVolumeInput.value + "%";
        }
        const savedPack = localStorage.getItem("valvet-soundpack") || "soft";
        pack.value = SOUND_PACKS[savedPack] || savedPack === "custom" ? savedPack : "soft";
        discord.checked = localStorage.getItem("valvet-discord") === "on";
        discordURLInput.value = localStorage.getItem("valvet-discord-url") || "";
        discordApp.value = localStorage.getItem("valvet-discord-app") || "";
        discordStatus.checked = localStorage.getItem("valvet-discord-status") !== "off";
        discordRounds.checked = localStorage.getItem("valvet-discord-rounds") !== "off";
        discordUnfriend.checked = localStorage.getItem("valvet-discord-unfriend") !== "off";
        discordLive.checked = localStorage.getItem("valvet-discord-live") === "on";
        discordEnd.checked = localStorage.getItem("valvet-discord-end") === "on";
        discordWatch.checked = localStorage.getItem("valvet-discord-watch") !== "off";
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
          glass.checked = localStorage.getItem("valvet-glass") === "on";
          glass.onchange = () => {
            localStorage.setItem("valvet-glass", glass.checked ? "on" : "off");
            syncLook();
          };
        }
        if (fx) {
          fx.checked = localStorage.getItem("valvet-fx") === "on";
          fx.onchange = () => {
            localStorage.setItem("valvet-fx", fx.checked ? "on" : "off");
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
          minimal.checked = localStorage.getItem("valvet-minimal") === "on";
          minimal.onchange = () => {
            localStorage.setItem("valvet-minimal", minimal.checked ? "on" : "off");
            rememberPrefs();
            applyMinimal(minimal.checked);
          };
          if (minimal.checked) applyMinimal(true);
        }
        const pauseLookups = document.getElementById("set-pause-lookups");
        if (pauseLookups) {
          pauseLookups.checked = localStorage.getItem("valvet-pause-lookups") === "on";
          const pushPause = () => {
            localStorage.setItem("valvet-pause-lookups", pauseLookups.checked ? "on" : "off");
            rememberPrefs();
            fetch("/api/pause", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ on: pauseLookups.checked }) }).catch(() => {});
          };
          pauseLookups.onchange = pushPause;
          pushPause();
        }
        const gmail = document.getElementById("set-gmail");
        if (gmail) {
          gmail.value = localStorage.getItem("valvet-gmail") || "";
          gmail.addEventListener("change", () => {
            localStorage.setItem("valvet-gmail", gmail.value.trim());
            rememberPrefs();
          });
        }
        const gmailSend = document.getElementById("set-gmail-send");
        if (gmailSend) gmailSend.onclick = () => {
          const email = gmail ? gmail.value.trim() : "";
          if (email) localStorage.setItem("valvet-gmail", email);
          rememberPrefs();
          const bag = {};
          for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            if (key && (key.startsWith("valvet-") || key.startsWith("valvet-"))) bag[key] = localStorage.getItem(key) || "";
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
          nick.value = localStorage.getItem("valvet-nick") || "";
          nick.onchange = () => localStorage.setItem("valvet-nick", nick.value.trim());
        }
        if (looking) {
          looking.checked = localStorage.getItem("valvet-looking") === "on";
          looking.onchange = () => {
            if (looking.checked) {
              const mins = Number((document.getElementById("look-for") || {}).value || 0);
              if (mins) localStorage.setItem("valvet-until", String(Date.now() + mins * 60000));
              else localStorage.removeItem("valvet-until");
            } else localStorage.removeItem("valvet-until");
            localStorage.setItem("valvet-looking", looking.checked ? "on" : "off");
            rememberPrefs();
            pushRoom(lookingOn());
            paintLooking();
          };
        }
        sound.onchange = () => localStorage.setItem("valvet-sound", sound.checked ? "on" : "off");
        chime.onchange = () => localStorage.setItem("valvet-chime", chime.checked ? "on" : "off");
        join.onchange = () => localStorage.setItem("valvet-join", join.checked ? "on" : "off");
        pop.onchange = () => localStorage.setItem("valvet-pop", pop.checked ? "on" : "off");
        night.onchange = () => localStorage.setItem("valvet-night", night.checked ? "on" : "off");
        const quietFrom = document.getElementById("set-quiet-from");
        const quietTo = document.getElementById("set-quiet-to");
        if (quietFrom && quietTo) {
          const dressTime = (input) => {
            if (!input || input.dataset.dressed) return;
            input.dataset.dressed = "1";
            const wrap = document.createElement("span");
            wrap.className = "time-field";
            const text = document.createElement("input");
            text.type = "text";
            text.inputMode = "numeric";
            text.placeholder = "00:00";
            text.maxLength = 5;
            text.spellcheck = false;
            text.autocomplete = "off";
            text.value = input.value || "";
            const open = document.createElement("button");
            open.type = "button";
            open.className = "time-open";
            open.textContent = "▾";
            open.setAttribute("aria-label", "Choose a time");
            const commit = (value) => {
              const bits = String(value || "").trim().match(/^(\d{1,2}):(\d{2})$/);
              const next = bits ? String(Math.min(23, Number(bits[1]))).padStart(2, "0") + ":" + String(Math.min(59, Number(bits[2]))).padStart(2, "0") : "";
              text.value = next;
              input.value = next;
              input.dispatchEvent(new Event("change"));
            };
            text.addEventListener("keydown", (event) => {
              if (event.key === "Backspace" && text.selectionStart === text.selectionEnd) {
                const at = text.selectionStart || 0;
                if (at > 0 && text.value[at - 1] === ":") {
                  event.preventDefault();
                  const digits = (text.value.slice(0, at - 1) + text.value.slice(at)).replace(/\D/g, "").slice(0, -1);
                  text.value = digits.length > 2 ? digits.slice(0, 2) + ":" + digits.slice(2) : digits;
                  text.setSelectionRange(text.value.length, text.value.length);
                }
              }
              if (event.key === "Enter") { event.preventDefault(); commit(text.value); text.blur(); }
            });
            text.addEventListener("input", () => {
              const digits = text.value.replace(/\D/g, "").slice(0, 4);
              const next = digits.length >= 2 ? digits.slice(0, 2) + ":" + digits.slice(2) : digits;
              if (text.value !== next) {
                text.value = next;
                text.setSelectionRange(next.length, next.length);
              }
            });
            text.addEventListener("blur", () => commit(text.value));
            open.onclick = (event) => {
              event.preventDefault();
              document.querySelectorAll(".sound-menu").forEach((menu) => menu.remove());
              const menu = document.createElement("div");
              menu.className = "sound-menu";
              const rect = wrap.getBoundingClientRect();
              menu.style.left = Math.max(8, rect.left) + "px";
              menu.style.top = Math.min(window.innerHeight - 12, rect.bottom + 4) + "px";
              menu.style.width = Math.max(140, rect.width) + "px";
              menu.style.maxHeight = "240px";
              for (let hour = 0; hour < 24; hour++) {
                for (const minute of [0, 30]) {
                  const value = String(hour).padStart(2, "0") + ":" + String(minute).padStart(2, "0");
                  const row = document.createElement("button");
                  row.type = "button";
                  row.textContent = value;
                  if (value === text.value) row.classList.add("on");
                  row.addEventListener("mousedown", (click) => {
                    click.preventDefault();
                    commit(value);
                    menu.remove();
                  });
                  menu.append(row);
                }
              }
              document.body.append(menu);
              const close = (click) => {
                if (menu.contains(click.target) || wrap.contains(click.target)) return;
                menu.remove();
                document.removeEventListener("mousedown", close, true);
              };
              setTimeout(() => document.addEventListener("mousedown", close, true), 0);
            };
            input.hidden = true;
            input.parentNode.insertBefore(wrap, input);
            wrap.append(text, open, input);
          };
          quietFrom.value = localStorage.getItem("valvet-quiet-from") || "";
          quietTo.value = localStorage.getItem("valvet-quiet-to") || "";
          dressTime(quietFrom);
          dressTime(quietTo);
          const saveQuiet = () => {
            localStorage.setItem("valvet-quiet-from", quietFrom.value || "");
            localStorage.setItem("valvet-quiet-to", quietTo.value || "");
          };
          quietFrom.onchange = saveQuiet;
          quietTo.onchange = saveQuiet;
        }
        let volumeTimer = 0;
        volume.oninput = () => {
          localStorage.setItem("valvet-volume", volume.value);
          volumeN.textContent = volume.value + "%";
          clearTimeout(volumeTimer);
          volumeTimer = setTimeout(() => playPack(true), 140);
        };
        if (resultVolumeInput && resultVolumeN) {
          let resultTimer = 0;
          resultVolumeInput.oninput = () => {
            localStorage.setItem("valvet-result-volume", resultVolumeInput.value);
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
        hearSelect(pack, () => { localStorage.setItem("valvet-soundpack", pack.value); playPack(true); });
        const winPick = document.getElementById("set-win");
        const lossPick = document.getElementById("set-loss");
        if (winPick) {
          winPick.value = localStorage.getItem("valvet-win") || "gg";
          hearSelect(winPick, () => { localStorage.setItem("valvet-win", winPick.value); playResult("win", true); });
        }
        if (lossPick) {
          lossPick.value = localStorage.getItem("valvet-loss") || "sad";
          hearSelect(lossPick, () => { localStorage.setItem("valvet-loss", lossPick.value); playResult("loss", true); });
        }
        const bindFile = (id, kind, after) => {
          const input = document.getElementById(id);
          if (!input) return;
          if (!input.dataset.dressed) {
            input.dataset.dressed = "1";
            const wrap = document.createElement("span");
            wrap.className = "file-field";
            const btn = document.createElement("button");
            btn.type = "button";
            btn.className = "tiny solid";
            btn.textContent = "Choose";
            const name = document.createElement("b");
            name.textContent = "None";
            btn.onclick = (event) => { event.preventDefault(); input.click(); };
            input.addEventListener("change", () => {
              const file = input.files && input.files[0];
              name.textContent = file ? file.name : "None";
            });
            input.classList.add("file-hidden");
            input.parentNode.insertBefore(wrap, input);
            wrap.append(btn, name, input);
          }
          input.onchange = () => send(input.files && input.files[0]);
          const row = input.closest(".set-row") || input;
          row.addEventListener("dragover", (event) => { event.preventDefault(); row.classList.add("drop"); });
          row.addEventListener("dragleave", () => row.classList.remove("drop"));
          row.addEventListener("drop", (event) => {
            event.preventDefault();
            row.classList.remove("drop");
            const file = event.dataTransfer && event.dataTransfer.files && event.dataTransfer.files[0];
            send(file);
          });
          function send(file) {
            if (!file) return;
            fetch("/api/media?kind=" + kind, { method: "POST", body: file }).then((res) => { if (res.ok && after) after(); }).catch(() => {});
          }
        };
        bindFile("set-notify-file", "notify", () => { localStorage.setItem("valvet-soundpack", "custom"); pack.value = "custom"; playPack(true); });
        bindFile("set-win-file", "win", () => { localStorage.setItem("valvet-win", "custom"); if (winPick) winPick.value = "custom"; playResult("win", true); });
        bindFile("set-loss-file", "loss", () => { localStorage.setItem("valvet-loss", "custom"); if (lossPick) lossPick.value = "custom"; playResult("loss", true); });
        const bindSound = (id, kind) => {
          const node = document.getElementById(id);
          if (!node) return;
          const defaults = { friend: "pack", watch: "none", played: "none" };
          node.value = localStorage.getItem("valvet-" + kind + "-sound") || defaults[kind];
          hearSelect(node, () => { localStorage.setItem("valvet-" + kind + "-sound", node.value); playSaved(kind, true); });
        };
        bindSound("set-friend-sound", "friend");
        bindSound("set-watch-sound", "watch");
        bindSound("set-played-sound", "played");
        bindFile("set-friend-file", "friend", () => { localStorage.setItem("valvet-friend-sound", "custom"); const node = document.getElementById("set-friend-sound"); if (node) node.value = "custom"; playSaved("friend", true); });
        bindFile("set-watch-file", "watch", () => { localStorage.setItem("valvet-watch-sound", "custom"); const node = document.getElementById("set-watch-sound"); if (node) node.value = "custom"; playSaved("watch", true); });
        bindFile("set-played-file", "played", () => { localStorage.setItem("valvet-played-sound", "custom"); const node = document.getElementById("set-played-sound"); if (node) node.value = "custom"; playSaved("played", true); });
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
            localStorage.setItem("valvet-backdrop", "on");
            loadBackdrop();
          }).catch(() => {});
        };
        if (bgClear) bgClear.onclick = () => {
          localStorage.setItem("valvet-backdrop", "off");
          document.body.classList.remove("custom-bg");
          fetch("/api/media?kind=background", { method: "DELETE" }).catch(() => {});
          applyTheme(theme);
        };
        if (localStorage.getItem("valvet-backdrop") === "on") loadBackdrop();
        const saveDiscord = () => {
          localStorage.setItem("valvet-discord", discord.checked ? "on" : "off");
          localStorage.setItem("valvet-discord-url", discordURLInput.value.trim());
          localStorage.setItem("valvet-discord-app", discordApp.value.trim());
          localStorage.setItem("valvet-discord-status", discordStatus.checked ? "on" : "off");
          localStorage.setItem("valvet-discord-rounds", discordRounds.checked ? "on" : "off");
          localStorage.setItem("valvet-discord-unfriend", discordUnfriend.checked ? "on" : "off");
          localStorage.setItem("valvet-discord-live", discordLive.checked ? "on" : "off");
          localStorage.setItem("valvet-discord-end", discordEnd.checked ? "on" : "off");
          localStorage.setItem("valvet-discord-watch", discordWatch.checked ? "on" : "off");
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
          localStorage.setItem("valvet-discord-url", url);
          localStorage.setItem("valvet-discord-app", app);
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
          localStorage.setItem("valvet-compact", compact.checked ? "on" : "off");
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
              localStorage.setItem("valvet-corners", btn.dataset.corners);
              if (corners) corners.value = btn.dataset.corners;
            }
            if (btn.dataset.motion) {
              localStorage.setItem("valvet-motion", btn.dataset.motion);
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
          localStorage.setItem("valvet-corners", "soft");
          localStorage.setItem("valvet-motion", "full");
          localStorage.setItem("valvet-glass", "off");
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

