      const MODES = [
        ["All", ""],
        ["Competitive", "competitive"],
        ["Unrated", "unrated"],
        ["Swiftplay", "swiftplay"],
        ["Spike Rush", "spikerush"],
        ["Deathmatch", "deathmatch"],
        ["Escalation", "ggteam"],
        ["Team Deathmatch", "hurm"],
        ["Replication", "onefa"],
        ["Premier", "premier"],
        ["Skirmish 2v2", "skirmish"],
        ["Gauntlet", "abilitydraftarena"],
        ["One Site", "aros"],
        ["Knockout", "dodgeball"],
        ["Retake", "fortcollins"],
        ["Custom", "custom"],
      ];
      let modeQueue = "";
      let modeMatches = null;
      let histMap = localStorage.getItem("valvet-hist-map") || "";
      let histAgent = localStorage.getItem("valvet-hist-agent") || "";
      let histParty = localStorage.getItem("valvet-hist-party") || "";
      let histWith = "";
      let hideHistStats = localStorage.getItem("valvet-hist-stats") === "off";
      const histStats = document.getElementById("hist-stats");
      function paintHistStatsBtn() {
        if (!histStats) return;
        histStats.textContent = hideHistStats ? "Show stats" : "Hide stats";
        histStats.classList.toggle("solid", !hideHistStats);
      }
      if (histStats) {
        paintHistStatsBtn();
        histStats.onclick = () => {
          hideHistStats = !hideHistStats;
          localStorage.setItem("valvet-hist-stats", hideHistStats ? "off" : "on");
          paintHistStatsBtn();
          recentKey = "";
          drawRecent(modeQueue ? modeMatches || [] : history);
        };
      }
      function youOf(match) {
        return ((match && match.players) || []).find((player) => player.you) || null;
      }
      function withFriend(match) {
        const you = youOf(match);
        if (!you) return false;
        const friends = new Set(((latestState && latestState.friendIds) || []).map((id) => String(id).toLowerCase()));
        return (match.players || []).some((player) => !player.you && player.team && player.team === you.team && friends.has(String(player.puuid || "").toLowerCase()));
      }
      function passesHist(match) {
        const you = youOf(match);
        if (histMap && String(match.map || "") !== histMap) return false;
        if (histAgent && String((you && you.agent) || "") !== histAgent) return false;
        const stacked = match.knownParty ? !!(you && you.party) : withFriend(match);
        if (histParty === "stack" && !stacked) return false;
        if (histParty === "solo" && stacked) return false;
        if (histWith && !((match.players || []).some((player) => String(player.puuid || "").toLowerCase() === histWith))) return false;
        return true;
      }
      function paintHistPicks(rows) {
        const host = document.getElementById("hist-picks");
        if (!host) return;
        const maps = [];
        const agents = [];
        const seenM = new Set();
        const seenA = new Set();
        for (const match of rows || []) {
          if (match.map && !seenM.has(match.map)) { seenM.add(match.map); maps.push(match.map); }
          const agent = (youOf(match) || {}).agent || "";
          if (agent && agent !== "Picking" && !seenA.has(agent)) { seenA.add(agent); agents.push(agent); }
        }
        maps.sort();
        agents.sort();
        const sig = [maps.join(","), agents.join(","), histMap, histAgent, histParty, histWith ? "1" : "", mode, modeQueue].join("|");
        if (host.dataset.sig === sig) return;
        host.dataset.sig = sig;
        host.replaceChildren();
        const queue = document.createElement("select");
        queue.className = "hist-pick";
        for (const [label, id] of MODES) {
          const opt = document.createElement("option");
          opt.value = id;
          opt.textContent = label;
          queue.append(opt);
        }
        queue.value = modeQueue || "";
        queue.onchange = () => {
          const picked = MODES.find((row) => row[1] === queue.value) || MODES[0];
          pickMode(picked[0], picked[1]);
        };
        host.append(queue);
        const pick = (label, current, list, apply) => {
          const select = document.createElement("select");
          select.className = "hist-pick";
          const any = document.createElement("option");
          any.value = "";
          any.textContent = label;
          select.append(any);
          for (const item of list) {
            const opt = document.createElement("option");
            opt.value = item;
            opt.textContent = item;
            select.append(opt);
          }
          select.value = list.includes(current) ? current : "";
          select.onchange = () => apply(select.value);
          host.append(select);
        };
        const redraw = () => { recentKey = ""; drawRecent(modeQueue ? modeMatches || [] : history); };
        pick("Map", histMap, maps, (value) => { histMap = value; localStorage.setItem("valvet-hist-map", value); redraw(); });
        pick("Agent", histAgent, agents, (value) => { histAgent = value; localStorage.setItem("valvet-hist-agent", value); redraw(); });
        const party = document.createElement("select");
        party.className = "hist-pick";
        for (const [value, label] of [["", "Party"], ["stack", "Stacked"], ["solo", "Solo"]]) {
          const opt = document.createElement("option");
          opt.value = value;
          opt.textContent = label;
          party.append(opt);
        }
        party.value = histParty === "stack" || histParty === "solo" ? histParty : "";
        party.onchange = () => { histParty = party.value; localStorage.setItem("valvet-hist-party", histParty); redraw(); };
        host.append(party);
        if (histWith) {
          const clear = el("button", "tiny", "Clear player");
          clear.type = "button";
          clear.onclick = () => { histWith = ""; redraw(); };
          host.append(clear);
        }
      }
      function performanceCard(matches) {
        const map = {};
        const agent = {};
        const site = {};
        const guns = {};
        for (const match of matches || []) {
          const you = youOf(match);
          if (!you) continue;
          const win = match.won === true ? 1 : match.won === false ? 0 : null;
          const take = (book, key) => {
            if (!key || win == null) return;
            const row = book[key] || { w: 0, n: 0 };
            row.n += 1;
            row.w += win;
            book[key] = row;
          };
          take(map, match.map || "");
          take(agent, you.agent && you.agent !== "Picking" ? you.agent : "");
          for (const round of match.rounds || []) {
            if (!(round.plant || round.hold) || !round.site) continue;
            const row = site[round.site] || { w: 0, n: 0 };
            row.n += 1;
            if (round.you) row.w += 1;
            site[round.site] = row;
          }
          for (const part of String(you.gunHs || "").split(",")) {
            const bits = part.split(":");
            const ratio = String(bits[1] || "").split("/");
            if (!bits[0] || !Number(ratio[1])) continue;
            const row = guns[bits[0]] || { h: 0, s: 0 };
            row.h += Number(ratio[0]) || 0;
            row.s += Number(ratio[1]) || 0;
            guns[bits[0]] = row;
          }
        }
        const best = (book) => Object.entries(book).sort((a, b) => (b[1].n - a[1].n) || (b[1].w / b[1].n) - (a[1].w / a[1].n))[0];
        const rate = (row) => (row && row.n ? Math.round((row.w / row.n) * 100) + "%" : "");
        const tiles = [];
        const add = (value, label) => {
          const text = String(value || "").trim();
          if (!text || text === "—" || text === "0" || text === "0%") return;
          tiles.push([text, label]);
        };
        const topMap = best(map);
        const topAgent = best(agent);
        const topSite = best(site);
        const gun = Object.entries(guns).sort((a, b) => b[1].s - a[1].s)[0];
        add(topMap ? rate(topMap[1]) : "", topMap ? topMap[0] : "Map");
        add(topAgent ? rate(topAgent[1]) : "", topAgent ? topAgent[0] : "Agent");
        add(topSite ? rate(topSite[1]) : "", topSite ? "Site " + topSite[0] : "Site");
        add(gun && gun[1].s ? Math.round((gun[1].h / gun[1].s) * 100) + "%" : "", gun ? gun[0] + " HS" : "Gun HS");
        let hsN = 0;
        let hsSum = 0;
        let multis = 0;
        const seenMulti = new Set();
        for (const match of matches || []) {
          const you = (match.players || []).find((player) => player.you);
          if (!you) continue;
          const hs = parseInt(String(you.hs || you.aim || "").replace("%", ""), 10);
          if (!Number.isNaN(hs)) { hsN += 1; hsSum += hs; }
          const id = String(you.puuid || "").toLowerCase();
          const byRound = {};
          for (const edge of match.edges || []) {
            const mine = (id && String(edge.kid || "").toLowerCase() === id) || /^you\b/i.test(String(edge.killer || ""));
            if (!mine) continue;
            const key = match.id + ":" + edge.round;
            byRound[key] = (byRound[key] || 0) + 1;
          }
          for (const [key, count] of Object.entries(byRound)) {
            if (count >= 2 && !seenMulti.has(key)) { seenMulti.add(key); multis += 1; }
          }
        }
        add(hsN ? Math.round(hsSum / hsN) + "%" : "", "First bullet");
        add(multis ? String(multis) : "", "Multikills");
        const fight = fightFacts(matches);
        if (fight.n) add(Math.round(fight.head / fight.n) + " / " + Math.round(fight.body / fight.n) + " / " + Math.round(fight.leg / fight.n), "Head / body / legs");
        if (fight.firstK || fight.firstD) add(fight.firstK + " / " + fight.firstD, "First blood");
        if (fight.plants) add(String(fight.plants), "Plants");
        if (fight.defuses) add(String(fight.defuses), "Defuses");
        if (fight.post) add(String(fight.post), "After plant");
        if (fight.trades) add(String(fight.trades), "Trades");
        const pair = (name) => fight.kind[name][1] ? Math.round(fight.kind[name][0] / fight.kind[name][1] * 100) + "%" : "";
        add(pair("pistol"), "Pistol");
        add(pair("bonus"), "Bonus");
        add(pair("gun"), "Gun rounds");
        if (!tiles.length) return null;
        const card = el("div", "perf");
        for (const [value, label] of tiles) {
          const box = el("article");
          box.append(el("b", null, value));
          box.append(el("span", null, label));
          card.append(box);
        }
        return card;
      }
      function fightFacts(matches) {
        const kind = { pistol: [0, 0], bonus: [0, 0], gun: [0, 0] };
        const out = { n: 0, head: 0, body: 0, leg: 0, firstK: 0, firstD: 0, trades: 0, plants: 0, defuses: 0, post: 0, kind };
        for (const match of matches || []) {
          const you = (match.players || []).find((player) => player.you);
          if (!you) continue;
          const id = String(you.puuid || "").toLowerCase();
          const hs = parseInt(you.hs, 10);
          const body = parseInt(you.body, 10);
          const legs = parseInt(you.legs, 10);
          if (!Number.isNaN(hs) && !Number.isNaN(body) && !Number.isNaN(legs)) {
            out.head += hs; out.body += body; out.leg += legs; out.n += 1;
          }
          out.plants += Number(you.plants) || 0;
          out.defuses += Number(you.defuses) || 0;
          const rounds = match.rounds || [];
          const zero = rounds.some((round) => Number(round.n) === 0);
          for (const round of rounds) {
            const num = Number(round.n);
            const slot = zero ? num : num - 1;
            const bucket = slot === 0 || slot === 12 ? "pistol" : slot === 1 || slot === 13 ? "bonus" : "gun";
            if (round.you === true) kind[bucket][0] += 1;
            if (round.you === true || round.you === false) kind[bucket][1] += 1;
            if ((round.how === "Detonate" || round.how === "Defuse") && round.you) out.post += 1;
          }
          for (const edge of match.edges || []) {
            const kid = String(edge.kid || "").toLowerCase();
            const vid = String(edge.vid || "").toLowerCase();
            if (!id) continue;
            if (edge.first && kid === id) out.firstK += 1;
            if (edge.first && vid === id) out.firstD += 1;
            if (edge.trade && kid === id) out.trades += 1;
          }
        }
        return out;
      }
      const actBtn = document.getElementById("hist-act");
      if (actBtn && !actBtn.dataset.bound) {
        actBtn.dataset.bound = "1";
        actBtn.onclick = async () => {
          showAct = !showAct;
          actBtn.classList.toggle("solid", showAct);
          if (showAct && !actArchive) {
            try {
              const data = await (await fetch("/api/acts", { cache: "no-store" })).json();
              actArchive = (data && data.matches) || [];
            } catch { actArchive = []; }
          }
          recentKey = "";
          drawRecent(history);
        };
      }
      function showShared(player) {
        const id = String((player && player.puuid) || "").toLowerCase();
        if (!id) return;
        histWith = id;
        histMap = "";
        histAgent = "";
        histParty = "";
        const scrim = document.getElementById("scrim");
        if (scrim) scrim.hidden = true;
        const sheet = document.getElementById("matchbox");
        if (sheet) sheet.hidden = true;
        recentKey = "";
        setTab("history");
      }
      let actArchive = null;
      let showAct = false;
      function drawRecent(matches) {
        const typed = ((document.getElementById("match-query") || {}).value || "").trim();
        const source = showAct ? (actArchive || []) : (modeQueue ? modeMatches || [] : matches || []);
        paintHistPicks(history && history.length ? history : source);
        const showing = queryMatches(typed, (source || []).filter(passesHist));
        for (const match of showing) markAwards(match);
        let key = mode + "|" + modeQueue + "|" + typed.toLowerCase() + "|" + histMap + "|" + histAgent + "|" + histParty + "|" + histWith + "|" + (showAct ? "a" : "") + "|" + (hideHistStats ? "0" : "1");
        for (const match of showing) {
          const won = resultOf(match);
          key += (match.id || "") + ":" + match.score1 + "-" + match.score2 + ":" + (won === true ? "w" : won === false ? "l" : "") + ":" + (rrDelta(match) == null ? "" : rrDelta(match)) + ":";
          for (const player of byKDA(match.players)) key += whoKey(player) + "|" + shownName(player) + "|" + (player.kills || 0) + "|" + (player.deaths || 0) + "|" + (player.assists || 0) + "|" + (player.rank || "") + "|" + (player.rankPeak ? "p" : "") + ";";
        }
        if (key === recentKey) return;
        recentKey = key;
        const recent = document.getElementById("recent");
        const pane = document.getElementById("pane-history");
        const scroll = pane ? pane.scrollTop : 0;
        const ids = showing.map((match) => match.id || "").join(",");
        recent.dataset.ids = ids;
        recent.classList.add("settled");
        recent.replaceChildren();
        if (modeQueue && modeMatches == null) {
          recent.append(el("div", "empty", "Loading " + mode + "…"));
          if (pane) pane.scrollTop = scroll;
          return;
        }
        if (!showing.length) {
          recent.append(el("div", "empty", showAct && !(actArchive || []).length ? "Last act is not saved on this PC yet. It is kept the next time an act ends." : typed || histMap || histAgent || histParty || histWith ? "Nothing matches that filter." : mode === "All" ? "No games loaded yet." : "No " + mode + " games in the recent list."));
          if (pane) pane.scrollTop = scroll;
          return;
        }
        const perf = hideHistStats ? null : performanceCard(showing);
        if (perf) recent.append(perf);
        const form = hideHistStats ? null : formStrip(showing);
        if (form) recent.append(form);
        showing.forEach((match, index) => {
          const won = resultOf(match);
          const card = el("article", "game" + (won == null ? "" : won ? " win" : " loss"));
          card.style.animationDelay = Math.min(index, 10) * 36 + "ms";
          const button = el("button", "match");
          button.type = "button";
          const grow = el("div");
          grow.append(el("div", "mapname", match.map || "Match"));
          const you = youOf(match);
          const agent = you && you.agent && you.agent !== "Picking" ? you.agent : "";
          const delta = rrDelta(match);
          const sub = el("div", "sub");
          const lead = [match.mode, agent].filter(Boolean).join(" · ");
          if (lead) sub.append(document.createTextNode(lead));
          if (delta != null) {
            if (lead) sub.append(document.createTextNode(" · "));
            sub.append(el("span", "rr-delta " + (delta > 0 ? "up" : delta < 0 ? "down" : ""), (delta > 0 ? "+" : "") + delta + " RR"));
          }
          if (sub.childNodes.length) grow.append(sub);
          const yours = match.side === "Attack" ? "Attackers" : match.side === "Defense" ? "Defenders" : "";
          const result = el("div", "result");
          result.append(el("div", "wl" + (won == null ? "" : won ? " w" : " l"), won == null ? "—" : won ? "Win" : "Loss"));
          if (won === true || won === false) result.append(catFace(won === true));
          button.append(result, grow, tally(match.score1, match.score2, match.team1, match.team2, yours));
          const sides = el("div", "sides");
          const colors = partyColors(match.players);
          for (const [team, list] of groups(match.players)) {
            const mine = list.some((player) => player.you) || (you && you.team === team);
            const tone = mine ? "ally" : "enemy";
            const block = sideList(list, team, tone, colors);
            if (block) sides.append(block);
          }
          const box = el("div");
          if (openMatches.has(match.id)) {
            const scrub = roundScrub(match);
            if (scrub) box.append(scrub);
            box.append(statTable(match.players));
          }
          button.onclick = (event) => {
            event.preventDefault();
            if (openMatches.has(match.id)) openMatches.delete(match.id);
            else openMatches.add(match.id);
            box.replaceChildren();
            if (openMatches.has(match.id)) {
              const scrub = roundScrub(match);
              if (scrub) box.append(scrub);
              box.append(statTable(match.players));
            }
          };
          const strip = roundStrip(match.rounds, match);
          card.append(button);
          if (strip) card.append(strip);
          card.append(sides, box);
          wireHover(card);
          recent.append(card);
        });
        if (pane) pane.scrollTop = scroll;
      }
      function whoKey(player) {
        const id = String((player && player.puuid) || "").toLowerCase();
        if (id) return id;
        return shownName(player).toLowerCase();
      }
      function wireHover(root) {
        if (!root || root.dataset.hover === "1") return;
        root.dataset.hover = "1";
        root.addEventListener("mouseover", (event) => {
          const node = event.target.closest && event.target.closest("[data-who]");
          const id = node ? node.dataset.who : "";
          root.querySelectorAll("[data-who]").forEach((other) => other.classList.toggle("linked", !!id && other.dataset.who === id));
        });
        root.addEventListener("mouseleave", () => {
          root.querySelectorAll(".linked").forEach((other) => other.classList.remove("linked"));
        });
      }
      function shownName(player) {
        const name = String((player && player.name) || "").trim();
        if (name && name !== "Player" && name !== "Hidden") return name + (player.tag ? " #" + player.tag : "");
        if (player && player.agent) return player.agent;
        return "Player";
      }
      function statTable(players) {
        const board = el("div", "boardstat");
        const head = () => {
          const row = el("div", "boardline head");
          for (const label of ["", "Player", "KDA", "Dmg", "H B L", "ADR", "ACS"]) row.append(el("span", null, label));
          return row;
        };
        const addRow = (player) => {
          const row = el("div", "boardline" + (player.you ? " me" : ""));
          row.dataset.who = whoKey(player);
          const agent = el("span");
          if (player.agentIcon) agent.append(face(player));
          const who = el("button", "who");
          who.type = "button";
          who.append(document.createTextNode(shownName(player)));
          const badge = rankMark(player);
          if (badge) who.append(badge);
          if (player.mvp) who.append(el("span", "badge mvp", "MVP"));
          else if (player.svp) who.append(el("span", "badge svp", "SVP"));
          who.onclick = (event) => { event.preventDefault(); event.stopPropagation(); openPlayer(player); };
          row.append(
            agent,
            who,
            el("span", null, (player.kills || "0") + "/" + (player.deaths || "0") + "/" + (player.assists || "0")),
            el("span", null, player.damage || "—"),
            el("span", null, [player.hs, player.body, player.legs].filter(Boolean).join(" ") || "—"),
            el("span", null, player.adr || "—"),
            el("span", null, player.acs || "—")
          );
          board.append(row);
        };
        const grouped = groups(players);
        grouped.sort((a, b) => Number(b[1].some((player) => player.you)) - Number(a[1].some((player) => player.you)));
        if (grouped.length < 2) {
          board.append(head());
          for (const player of byKDA(players)) addRow(player);
          return board;
        }
        for (const [team, list] of grouped) {
          const yours = list.some((player) => player.you);
          board.append(el("div", "boardteam", (yours ? "Your team" : "Enemy") + (team ? " · " + team : "")));
          board.append(head());
          for (const player of byKDA(list)) addRow(player);
        }
        return board;
      }
      let lastShop = null;
      let knownLevel = 0;
      let bundleKey = "";
      function walletLog() {
        try { return JSON.parse(localStorage.getItem("valvet-wallet") || "[]"); } catch { return []; }
      }
      let walletFilter = "all";
      const walletOpen = {};
      function paintWallet(now) {
        const slot = document.getElementById("wallet-now");
        const log = document.getElementById("wallet-log");
        if (!slot || !log) return;
        slot.className = "wallet-now";
        slot.replaceChildren();
        if (now) {
          slot.append(el("span", null, (now.vp || 0) + " VP"), el("span", null, (now.kc || 0) + " KC"), el("span", null, (now.rad || 0) + " Radianite"));
        } else slot.textContent = "Waiting for the store balance.";
        const rows = walletLog();
        log.replaceChildren();
        const filters = el("div", "wallet-filters");
        for (const name of ["All", "VP", "KC", "Radianite"]) {
          const key = name === "All" ? "all" : name;
          const btn = el("button", "wallet-filter" + (walletFilter === key ? " on" : ""), name);
          btn.type = "button";
          btn.onclick = () => { walletFilter = key; paintWallet(now); };
          filters.append(btn);
        }
        log.append(filters);
        if (!rows.length) {
          log.append(el("div", "sub", "No changes yet. Leave Valvet open while you buy something."));
          return;
        }
        const kinds = walletFilter === "all" ? ["VP", "KC", "Radianite"] : [walletFilter];
        for (const kind of kinds) {
          const list = rows.filter((row) => row.kind === kind);
          const net = list.reduce((sum, row) => sum + (Number(row.delta) || 0), 0);
          const box = el("div", "wallet-group");
          const head = el("button", "wallet-head");
          head.type = "button";
          head.append(el("b", null, (walletOpen[kind] ? "▾ " : "▸ ") + kind));
          head.append(el("span", null, list.length + " · " + (net > 0 ? "+" : "") + net));
          head.onclick = () => { walletOpen[kind] = !walletOpen[kind]; paintWallet(now); };
          box.append(head);
          if (walletOpen[kind]) {
            const body = el("div", "wallet-body");
            if (!list.length) body.append(el("div", "sub", "No " + kind + " changes yet."));
            for (const row of list) {
              const line = el("div", "wallet-row");
              line.append(el("span", null, new Date(row.t).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })));
              line.append(el("b", row.delta < 0 ? "down" : "up", (row.delta > 0 ? "+" : "") + row.delta + " " + row.kind));
              body.append(line);
            }
            box.append(body);
          }
          log.append(box);
        }
      }
      function trackWallet(store) {
        if (!store) return;
        const now = { vp: Number(store.vp) || 0, kc: Number(store.kc) || 0, rad: Number(store.radianite) || 0, t: Date.now() };
        if (!(now.vp || now.kc || now.rad)) return;
        let prev = null;
        try { prev = JSON.parse(localStorage.getItem("valvet-wallet-last") || "null"); } catch {}
        if (prev && (prev.vp || prev.kc || prev.rad)) {
          const rows = walletLog();
          let changed = false;
          for (const [key, label] of [["vp", "VP"], ["kc", "KC"], ["rad", "Radianite"]]) {
            const delta = now[key] - (Number(prev[key]) || 0);
            if (!delta || Math.abs(delta) > 200000) continue;
            rows.unshift({ t: now.t, kind: label, delta });
            changed = true;
          }
          if (changed) localStorage.setItem("valvet-wallet", JSON.stringify(rows.slice(0, 80)));
        }
        localStorage.setItem("valvet-wallet-last", JSON.stringify(now));
        paintWallet(now);
      }
      function draw(data) {
        if (new URLSearchParams(location.search).has("demo")) return;
        latestState = data;
        if (data && data.map) data.map = mapName(data.map);
        const pauseBox = document.getElementById("set-pause-lookups");
        if (pauseBox && document.activeElement !== pauseBox && pauseBox.checked !== !!data.pauseLookups) {
          pauseBox.checked = !!data.pauseLookups;
          localStorage.setItem("valvet-pause-lookups", pauseBox.checked ? "on" : "off");
        }
        if (data && data.tray === "settings") setTab("settings");
        syncRoomHits(data);
        if (data.recent && data.recent.length) mergeHistory(data.recent);
        noteClutchLight();
        if (data.level) knownLevel = data.level;
        const level = data.level || knownLevel;
        paintIdentity(data.you || "This PC", level);
        if (Array.isArray(data.party) || Array.isArray(data.friends)) drawPeople(data);
        refreshTracked(data);
        notePings(data);
        const pen = document.getElementById("penalty-live");
        if (pen) pen.hidden = true;
        maybeInsights();
        if (data.store) {
          lastShop = data.store;
          trackWallet(data.store);
          wishlistHits(data.store);
          if (tab === "store") drawStore(data.store);
          if (tab === "bundle") drawBundle(data.store);
        }
        const note = document.getElementById("note");
        const wait = document.getElementById("wait");
        const waitSub = document.getElementById("wait-sub");
        const status = document.getElementById("status");
        const board = document.getElementById("board");
        const stage = document.getElementById("stage");
        const pane = document.getElementById("pane-live");
        const inMatch = data.phase === "Agent select" || data.phase === "In game" || data.phase === "Loading lobby";
        const party = (data.party || []).filter((person) => !isLeague(person));
        const stageWant = party.length >= 2 && !inMatch ? party.map((person) => [person.puuid, person.name, person.card, person.rank, person.state, person.partySize].join(":")).join("|") : "";
        if (stageWant !== stageKey) {
          stageKey = stageWant;
          stage.replaceChildren();
          stage.hidden = !stageWant;
          if (stageWant) for (const person of party) stage.append(heroCard(person));
        }
        if (pane) pane.classList.toggle("match", !!inMatch);
        if (pane) pane.classList.toggle("party-live", party.length >= 2 && !inMatch);
        if (wait) {
          wait.hidden = !!inMatch;
          wait.classList.toggle("riot", !data.ok);
          const riotAsk = document.getElementById("riot-ask");
          const waitScene = wait.querySelector(".wait-scene");
          if (riotAsk) riotAsk.hidden = !!data.ok;
          if (waitScene) waitScene.hidden = !data.ok;
          if (wait.dataset.phase && wait.dataset.phase !== (data.phase || "") && !inMatch) {
            wait.classList.remove("phase");
            void wait.offsetWidth;
            wait.classList.add("phase");
          }
          wait.dataset.phase = data.phase || "";
        }
        if (waitSub) {
          const partyRank = averageRank(party);
          const next = !data.ok ? (data.error || "Start the Riot Client, then leave this window open.") : data.phase === "In queue" ? [data.mode, "In queue", data.penalty].filter(Boolean).join(" · ") : inMatch ? "" : [partyRank ? "Party " + partyRank : "", data.penalty, "Stays here until a game is found."].filter(Boolean).join(" · ");
          if (waitSub.textContent !== next) waitSub.textContent = next;
        }
        if (!data.ok) {
          status.hidden = true;
          note.hidden = true;
          paintMapArt("");
          applyLiveLayout("");
          if (boardKey) { boardKey = ""; board.replaceChildren(); }
        } else if (inMatch) {
          status.hidden = false;
          note.hidden = true;
          document.getElementById("mode").textContent = data.mode || "Live";
          document.getElementById("map").textContent = data.map || "Lobby";
          document.getElementById("phase").textContent = [data.phase, data.server, mapRecord(history, data.map), data.penalty].filter(Boolean).join(" · ");
          paintMapArt(data.map);
          const yours = (data.players || []).find((player) => player.you);
          const lineups = document.getElementById("lineups");
          const youAgent = yours && yours.agent && yours.agent !== "Picking" ? yours.agent : "";
          if (lineups) {
            lineups.hidden = !(data.map && youAgent);
            lineups.onclick = (event) => {
              event.preventDefault();
              const url = "https://lineupsvalorant.com/?map=" + encodeURIComponent(data.map || "") + "&agent=" + encodeURIComponent(youAgent);
              fetch("/api/open?url=" + encodeURIComponent(url)).catch(() => {});
            };
          }
          const seenKey = [...seenLately(history)].sort().join(",");
          const roomKey = [...roomWatches].sort().join("|");
          const liveKey = JSON.stringify([data.phase, data.map, data.mode, data.server, data.score1, data.score2, data.team1, data.team2, seenKey, roomKey, (data.players || []).map((player) => [player.puuid, player.name, player.tag, player.agent, player.rank, player.peak, player.rankPeak, player.rr, player.party, player.partyN, player.hs, player.adr, player.acs, player.team, player.enemy, player.hidden, player.tier, player.level])]);
          if (liveKey !== boardKey) {
            boardKey = liveKey;
            document.getElementById("score").replaceChildren(tally(data.score1, data.score2, data.team1, data.team2, yours ? yours.team : ""));
            const scoreEl = document.getElementById("score");
            const stamp = [data.score1, data.score2].join("-");
            if (scoreEl.dataset.n && scoreEl.dataset.n !== stamp) {
              scoreEl.classList.remove("tick");
              void scoreEl.offsetWidth;
              scoreEl.classList.add("tick");
            }
            scoreEl.dataset.n = stamp;
            board.replaceChildren(roster(data.players, data.mode || data.queue));
            board.classList.toggle("select-stage", data.phase === "Agent select");
            paintSameLobby(data);
            const markBtn = document.getElementById("mark-round");
            if (markBtn) markBtn.hidden = data.phase !== "In game";
          }
        } else {
          status.hidden = true;
          note.hidden = true;
          paintMapArt("");
          applyLiveLayout("");
          const markBtn = document.getElementById("mark-round");
          if (markBtn) markBtn.hidden = true;
          const sameLobby = document.getElementById("same-lobby");
          if (sameLobby) sameLobby.hidden = true;
          if (boardKey) { boardKey = ""; board.replaceChildren(); }
        }
        if (tab === "history" && !modeQueue) drawRecent(history);
        if (tab === "live") {
          paintDuos("duos", party, history);
          paintToday(history);
          paintStreak(history);
        }
        if (tab === "profile") {
          const profile = document.getElementById("profile");
          if (profile && (profile.querySelector(".calling") || profile.querySelector("#act-compare"))) paintActCompare(profile);
        }
        syncPartyJoin(party, data.ok);
        syncPost(data);
        paintPost();
        syncNight(data.store);
        if (data.ok) {
          syncRounds(data);
          pushStatus(data);
        }
        if (!announced) { announced = true; lastPhase = data.phase || ""; stageLive(data); syncDiscord(data); return; }
        const phase = data.phase || "";
        const popped = lastPhase === "In queue" && (phase === "Agent select" || phase === "Loading lobby");
        if (popped) {
          notePop(data);
          lightEvent("queue");
        }
        if (popped && localStorage.getItem("valvet-pop") === "on") playQueue();
        if (!popped && localStorage.getItem("valvet-chime") === "on" && phase !== lastPhase && (phase === "In game" || phase === "Loading lobby" || phase === "Agent select")) playPack();
        lastPhase = phase;
        noteSession(data);
        if (tab === "session" || tab === "seen") paintDesk();
        stageLive(data);
        syncDiscord(data);
      }
      async function pickMode(label, queue) {
        mode = label;
        modeQueue = queue;
        modeMatches = queue ? null : [];
        recentKey = "";
        if (!queue) {
          drawRecent(history);
          return;
        }
        drawRecent([]);
        try {
          const res = await fetch("/api/history?queue=" + encodeURIComponent(queue), { cache: "no-store" });
          const data = await res.json();
          if (modeQueue !== queue) return;
          modeMatches = data.matches || [];
        } catch {
          if (modeQueue !== queue) return;
          modeMatches = [];
        }
        recentKey = "";
        drawRecent([]);
      }
      function heroCard(person) {
        const card = el("button", "hero-card");
        card.type = "button";
        if (person.card && String(person.card).startsWith("http")) {
          const img = document.createElement("img");
          img.className = "wide";
          img.alt = "";
          img.src = person.card;
          card.append(img);
        }
        card.append(el("div", "shade"));
        const copy = el("div", "copy");
        copy.append(el("div", "eyebrow", person.you ? "You" : (person.state || "Lobby")));
        copy.append(el("b", null, person.name || "Player"));
        const bits = [person.tag ? "#" + person.tag : "", person.rank, company(person), person.mode].filter(Boolean);
        copy.append(el("div", "sub", bits.join(" · ")));
        card.append(copy);
        card.onclick = () => openPlayer(person);
        return card;
      }
      let collectQuery = "";
      function commas(n) {
        return String(n || 0).replace(/\B(?=(\d{3})+(?!\d))/g, ",");
      }
      function filterCollection() {
        const q = collectQuery.trim().toLowerCase();
        const nodes = document.querySelectorAll("#loadout .weapon, #loadout .flair-pick, #loadout .buddy-pick, #loadout .collect-card");
        let shown = 0;
        nodes.forEach((node) => {
          const hide = !!q && !node.textContent.toLowerCase().includes(q);
          node.hidden = hide;
          if (!hide) shown += 1;
        });
        const count = document.getElementById("collect-count");
        if (count) count.textContent = q ? shown + (shown === 1 ? " match" : " matches") : "";
      }
      function masteryView(row) {
        const box = el("div");
        box.append(el("h2", null, "Mastery"));
        if (!hasMastery(row)) {
          box.append(el("div", "empty", "Valorant didn't return an Act or Lifetime level for this agent."));
          return box;
        }
        box.append(masteryStats(row));
        box.append(el("p", "sub", "Act level resets each Act. Accents at 4, 7 and 10, then you can overlevel. Lifetime level never resets. Rewards open on the lifetime track and cost Kingdom Credits."));
        const meter = el("div", "pass-meter");
        const fill = document.createElement("span");
        const act = Math.max(0, Math.min(10, Number(row.act || 0)));
        fill.style.width = (act * 10 || 4) + "%";
        meter.append(fill);
        box.append(meter);
        box.append(el("h2", null, "Rewards"));
        const rewards = row.rewards || [];
        if (!rewards.length) {
          box.append(el("div", "empty", "No reward track came back from Valorant."));
          return box;
        }
        const rail = el("div", "rail");
        for (const level of rewards) {
          const card = el("article", "skin weapon" + (level.open || level.got ? "" : " locked"));
          const src = level.anim && !String(level.anim).includes(".mp4") ? level.anim : level.icon;
          if (src) {
            const img = document.createElement("img");
            img.className = "gun";
            img.alt = "";
            img.src = src;
            card.append(img);
          }
          card.append(el("b", null, level.name || "Reward"));
          const bits = ["Level " + level.level, level.kind, level.got ? "owned" : level.open ? "unlocked" : "locked", level.cost ? commas(level.cost) + " KC" : ""].filter(Boolean);
          card.append(el("div", "sub", bits.join(" · ")));
          if (level.open && !level.got && level.offer && level.cost) {
            const buy = el("button", "tiny", "Buy");
            buy.type = "button";
            buy.onclick = async (event) => {
              event.stopPropagation();
              if (buy.dataset.arm !== "1") { buy.dataset.arm = "1"; buy.textContent = "Confirm"; return; }
              buy.textContent = "Buying…";
              const result = await (await fetch("/api/buy", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ offer: level.offer, price: level.cost }) })).json();
              buy.textContent = result.error || "Bought";
            };
            card.append(buy);
          }
          if (level.icon || level.anim) card.onclick = () => openZoom({ icon: level.icon, anim: level.anim, name: level.name });
          rail.append(card);
        }
        box.append(scroller(rail));
        return box;
      }
      function hasMastery(row) {
        return Number(row && row.act || 0) > 0 || Number(row && row.lifetime || 0) > 0 || Number(row && row.points || 0) > 0;
      }
      function masteryStats(row) {
        const stats = el("div", "mastery-stats");
        const bit = (value, label) => {
          const box = el("div");
          box.append(el("b", null, String(value)));
          box.append(el("span", null, label));
          return box;
        };
        if (!hasMastery(row)) return stats;
        stats.append(bit(row.act || 0, "Act"));
        stats.append(bit(row.lifetime || 0, "Lifetime"));
        if (row.points) stats.append(bit(commas(row.points), "MP"));
        return stats;
      }
      function paintLoadout(data) {
        const root = document.getElementById("loadout");
        if (!root || !data) return;
        root.className = "";
        root.replaceChildren();
        const find = el("div", "collect-find");
        const search = document.createElement("input");
        search.placeholder = "Weapon, skin, or buddy";
        search.value = collectQuery;
        search.oninput = () => { collectQuery = search.value; filterCollection(); };
        const count = el("span");
        count.id = "collect-count";
        find.append(search, count);
        root.append(find);
        if (data.error && !(data.guns || []).length) {
          root.append(el("div", "empty", data.error));
          return;
        }
        const banner = el("div", "identity");
        if (data.card) banner.style.backgroundImage = "url(" + data.card + ")";
        banner.append(el("div", "veil"));
        const copy = el("div", "copy");
        copy.append(el("div", "eyebrow", "Collection"));
        copy.append(el("h1", null, data.title || "Loadout"));
        if (data.value) {
          const worth = el("div", "locker-value");
          worth.append(el("b", null, commas(data.value)));
          worth.append(el("span", null, "VP in skins"));
          copy.append(worth);
        } else if (data.level) copy.append(el("div", "sub", "Level " + data.level));
        banner.append(copy);
        root.append(banner);
        const flair = el("div", "flair");
        for (const item of [...(data.sprays || []), ...(data.expressions || [])]) flair.append(flairCard(item));
        if (flair.childElementCount) root.append(el("h2", null, "Equipped"), flair);
        const expressions = el("div");
        expressions.id = "expressions";
        root.append(expressions);
        root.append(el("h2", null, "Weapons"));
        const rail = el("div", "rail");
        for (const gun of data.guns || []) {
          const card = el("article", "skin weapon");
          card.dataset.id = gun.id || "";
          card.append(gunMedia(gun));
          const body = el("div");
          body.append(el("div", "meta", gun.name));
          body.append(el("b", null, gun.skin));
          if (gun.buddy || gun.buddyIcon) {
            const buddy = el("div", "buddy");
            if (gun.buddyIcon) {
              const img = document.createElement("img");
              img.alt = "";
              img.src = gun.buddyIcon;
              buddy.append(img);
            }
            buddy.append(document.createTextNode(gun.buddy || "Buddy"));
            body.append(buddy);
          }
          card.append(body);
          card.onclick = () => openWardrobe(gun);
          rail.append(card);
        }
        root.append(scroller(rail));
        const armory = el("div");
        armory.id = "armory";
        root.append(armory);
        if (!(data.guns || []).length) root.append(el("div", "empty", "No weapons on the loadout."));
        fillExpressions();
        filterCollection();
      }
      async function loadLoadout(live) {
        const root = document.getElementById("loadout");
        const key = "valvet-cache-loadout";
        const saved = readJSON(key);
        const ok = saved && ((saved.guns || []).length || saved.title || saved.card);
        const force = live === "force";
        const sig = ok ? JSON.stringify([(saved.guns || []).map((gun) => gun.name), saved.title, saved.card]) : "";
        if (ok && root.dataset.sig === sig && root.querySelector(".identity, .collect-find, h1")) {
          /* already on screen */
        } else if (ok) {
          paintLoadout(saved);
          root.dataset.sig = sig;
        }
        const need = force || !ok || !cacheFresh(key, 6 * 60 * 60 * 1000);
        if (!need) return;
        if (!ok) {
          root.className = "";
          root.replaceChildren(el("div", "empty", "Loading loadout…"));
        }
        try {
          const data = await (await fetch("/api/loadout", { cache: "no-store" })).json();
          if ((data.guns || []).length || data.title || data.card) {
            writeJSON(key, data);
            stampCache(key);
          }
          const nextSig = JSON.stringify([((data.guns) || []).map((gun) => gun.name), data.title, data.card]);
          if (force || root.dataset.sig !== nextSig) {
            paintLoadout(data);
            root.dataset.sig = nextSig;
          }
        } catch {
          if (!ok) root.replaceChildren(el("div", "empty", "Couldn't load the loadout."));
        }
      }
      let openBundle = "";
      function drawBundle(shop) {
        const root = document.getElementById("bundle");
        if (!root) return;
        const bundles = (shop && shop.bundles) || [];
        const key = JSON.stringify(bundles.map((bundle) => [bundle.name, bundle.icon, bundle.price, bundle.was, bundle.left, (bundle.items || []).length])) + "|" + ((shop && shop.error) || "") + "|" + openBundle;
        if (key === bundleKey && root.querySelector(".bundle-poster, .empty")) return;
        bundleKey = key;
        markFresh("bundle", bundles.map((bundle) => bundle.name || bundle.id).filter(Boolean).join("|"));
        root.className = "";
        root.replaceChildren();
        if (!bundles.length) {
          root.className = "empty";
          root.textContent = (shop && shop.error) || "No featured bundle right now. Open Valorant.";
          return;
        }
        bundles.forEach((bundle) => {
          const id = bundle.name || "Bundle";
          const open = openBundle === id;
          const poster = el("button", "bundle-poster");
          poster.type = "button";
          if (bundle.icon) {
            poster.style.backgroundImage = "linear-gradient(100deg, rgba(10,4,8,.2), rgba(10,4,8,.55)), url(" + JSON.stringify(bundle.icon) + ")";
            const art = document.createElement("img");
            art.className = "bundle-art";
            art.alt = "";
            art.src = bundle.icon;
            poster.append(art);
          }
          poster.append(el("div", "veil"));
          const copy = el("div", "copy");
          copy.append(el("div", "eyebrow", open ? "Open · click to close" : "Featured bundle"));
          copy.append(el("h1", null, bundle.name || "Bundle"));
          const items = bundle.items || [];
          const owned = items.filter((item) => owns(item.name)).length;
          const bits = [items.length ? owned + " of " + items.length + " owned" : "", bundle.left ? bundle.left + " left" : ""].filter(Boolean);
          if (bits.length) copy.append(el("p", "sub", bits.join(" · ")));
          if (bundle.price) {
            const pill = el("div", "price-pill");
            if (bundle.was && bundle.was !== bundle.price) pill.append(el("s", null, bundle.was + " VP"));
            pill.append(document.createTextNode(bundle.price + " VP"));
            copy.append(pill);
          }
          poster.append(copy);
          poster.onclick = () => {
            openBundle = open ? "" : id;
            bundleKey = "";
            drawBundle(shop);
          };
          root.append(poster);
          if (!open) return;
          const grid = el("div", "shop bundle-open");
          if (!items.length) grid.append(el("div", "empty", "Valorant didn't list the items in this bundle."));
          for (const item of items) grid.append(skinCard(item));
          root.append(grid);
        });
      }
      let agentCache = null;
      function cleanText(text) {
        return String(text || "").replace(/<\/?[^<>]*>/g, "").replace(/&nbsp;/g, " ").replace(/\s{2,}/g, " ").trim();
      }
      function closeAgent() {
        const sheet = document.getElementById("agent-sheet");
        if (sheet) sheet.hidden = true;
        document.querySelectorAll(".agent-pick.on").forEach((node) => node.classList.remove("on"));
      }
      function drawAgents(list) {
        const root = document.getElementById("agents");
        const sheet = document.getElementById("agent-sheet");
        const body = document.getElementById("agent-sheet-body");
        root.className = "";
        root.replaceChildren();
        const grid = el("div", "agents");
        async function show(agent, card) {
          if (!sheet || !body) return;
          document.querySelectorAll(".agent-pick.on").forEach((node) => node.classList.remove("on"));
          if (card) card.classList.add("on");
          sheet.hidden = false;
          const role = agent.role || "Agent";
          sheet.style.setProperty("--role", ROLE_COLOR[role] || "#ff5fa2");
          body.replaceChildren();
          const top = el("div", "sheet-top");
          const titles = el("div");
          titles.append(el("div", "eyebrow role-ink", role));
          titles.append(el("h1", null, agent.name));
          top.append(titles);
          const close = el("button", "tiny", "Close");
          close.type = "button";
          close.onclick = (event) => { event.stopPropagation(); closeAgent(); };
          top.append(close);
          body.append(top);
          if (agent.icon) {
            const portrait = el("div", "portrait");
            const img = document.createElement("img");
            img.alt = "";
            img.src = agent.icon;
            portrait.append(img);
            body.append(portrait);
          }
          if (agent.desc) body.append(el("p", "sub", cleanText(agent.desc)));
          body.append(el("p", "sub", agentRecord(agent.name)));
          for (const ability of agent.abilities || []) {
            const row = el("div", "ability");
            if (ability.icon) {
              const img = document.createElement("img");
              img.alt = "";
              img.src = ability.icon;
              row.append(img);
            } else row.append(el("div"));
            const text = el("div");
            text.append(el("b", null, (ability.slot ? ability.slot + " · " : "") + (ability.name || "Ability")));
            if (ability.desc) text.append(el("div", "sub", cleanText(ability.desc)));
            row.append(text);
            body.append(row);
          }
          const gear = el("div");
          gear.append(el("h2", null, "Mastery"));
          gear.append(el("div", "empty", "Loading act and lifetime levels…"));
          body.append(gear);
          try {
            const data = await (await fetch("/api/gear?id=" + encodeURIComponent(agent.id || ""), { cache: "no-store" })).json();
            const row = ((data.agents || [])[0]) || null;
            gear.replaceChildren();
            if (data.error && !row) {
              gear.append(el("h2", null, "Mastery"));
              gear.append(el("div", "empty", data.error));
            } else if (!row || !hasMastery(row)) {
              gear.append(el("h2", null, "Mastery"));
              gear.append(el("div", "empty", (data && data.error) || "Valorant didn't return Act or Lifetime levels for this agent. Open their mastery screen in game once, then try again."));
            } else {
              gear.append(masteryView(row));
            }
          } catch {
            gear.replaceChildren(el("div", "empty", "Couldn't load Agent Mastery."));
          }
        }
        for (const agent of list || []) {
          const card = el("button", "agent-pick");
          card.type = "button";
          if (agent.icon) {
            const img = document.createElement("img");
            img.alt = "";
            img.src = agent.icon;
            card.append(img);
          }
          card.append(el("b", null, agent.name));
          card.append(el("div", "sub", agent.role || ""));
          card.onclick = () => show(agent, card);
          grid.append(card);
        }
        if (!(list || []).length) root.append(el("div", "empty", "Couldn't load agents."));
        else root.append(grid);
      }
      function studyHistory(matches, friends) {
        const rivals = new Map();
        const pals = new Map();
        const guns = new Map();
        const maps = new Map();
        let bestKills = null;
        let bestACS = null;
        let kills = 0;
        let deaths = 0;
        let games = 0;
        let wins = 0;
        for (const match of matches || []) {
          const you = (match.players || []).find((player) => player.you);
          if (!you) continue;
          games += 1;
          if (match.won === true) wins += 1;
          const k = Number(you.kills) || 0;
          const d = Number(you.deaths) || 0;
          kills += k;
          deaths += d;
          const acs = Number(String(you.acs || "").replace(/[^\d.]/g, "")) || 0;
          if (!bestKills || k > bestKills.kills) bestKills = { kills: k, map: match.map || "a map" };
          if (acs && (!bestACS || acs > bestACS.acs)) bestACS = { acs, map: match.map || "a map" };
          for (const part of String(you.guns || "").split(",")) {
            const [name, count] = part.split(":");
            const n = Number(count) || 0;
            if (name && n) guns.set(name, (guns.get(name) || 0) + n);
          }
          if (match.map && (match.won === true || match.won === false)) {
            const row = maps.get(match.map) || { name: match.map, wins: 0, losses: 0 };
            if (match.won === true) row.wins += 1;
            else row.losses += 1;
            maps.set(match.map, row);
          }
          for (const player of match.players || []) {
            if (player.you) continue;
            const id = String(player.puuid || "").toLowerCase();
            if (!id) continue;
            const same = !!(player.team && you.team && player.team === you.team);
            const friend = friends && friends.has(id);
            if (same && !friend) continue;
            if (!same && friend) continue;
            if (match.won !== true && match.won !== false) continue;
            const book = same ? pals : rivals;
            const row = book.get(id) || { name: shownName(player), wins: 0, losses: 0, games: 0 };
            if (player.name && player.name !== "Player" && player.name !== "Hidden") row.name = shownName(player);
            row.games += 1;
            if (same ? match.won === true : match.won === false) row.wins += 1;
            else row.losses += 1;
            book.set(id, row);
          }
        }
        const keep = (row) => row.games >= 2;
        const rival = [...rivals.values()].filter(keep).sort((a, b) => b.games - a.games)[0] || null;
        const chemistry = [...pals.values()].filter(keep).sort((a, b) => b.games - a.games).slice(0, 6);
        const gunRows = [...guns.entries()].map(([name, count]) => ({ name, kills: count })).sort((a, b) => b.kills - a.kills).slice(0, 5);
        const mapRows = [...maps.values()].map((row) => ({ ...row, games: row.wins + row.losses })).sort((a, b) => b.games - a.games).slice(0, 6);
        return { rival, chemistry, bestKills, bestACS, guns: gunRows, maps: mapRows, games, wins, losses: games - wins, kills, deaths };
      }
      function resultOf(match) {
        if (!match) return null;
        if (match.won === true || match.won === false) return match.won;
        const you = (match.players || []).find((player) => player.you);
        const s1 = Number(match.score1);
        const s2 = Number(match.score2);
        if (Number.isFinite(s1) && Number.isFinite(s2) && s1 !== s2) {
          if (you && you.team && you.team === match.team1) return s1 > s2;
          if (you && you.team && you.team === match.team2) return s2 > s1;
        }
        const rounds = match.rounds || [];
        if (rounds.length) {
          const last = rounds[rounds.length - 1];
          const y = Number(last.scoreY) || 0;
          const n = Number(last.scoreN) || 0;
          if (y + n > 0 && y !== n) return y > n;
        }
        return null;
      }
      let insightSig = "";
      function maybeInsights() {
        if (tab !== "profile") return;
        const slot = document.getElementById("insight-slot");
        if (!slot) return;
        const rec = latestState && latestState.records;
        const sig = rec ? [rec.games, rec.wins, rec.bestGun, (rec.friends || []).length].join(":") : "0";
        if (sig === insightSig) return;
        insightSig = sig;
        paintInsights(slot);
      }
      const heardPings = new Set();
      try { JSON.parse(localStorage.getItem("valvet-pings") || "[]").forEach((id) => heardPings.add(id)); } catch {}
      function traceBook() {
        return readJSON("valvet-trace-hits") || {};
      }
      function saveTrace(book) {
        localStorage.setItem("valvet-trace-hits", JSON.stringify(book));
      }
      function notePings(data) {
        let fresh = false;
        const book = traceBook();
        for (const row of (data && data.pings) || []) {
          const parts = String(row).split("|");
          const id = parts[0] || "";
          if (!id || heardPings.has(id)) continue;
          heardPings.add(id);
          fresh = true;
          const name = parts[1] || parts.slice(1).join("|");
          const map = parts.length >= 3 ? parts[2] : "";
          const finished = Number(parts[3]) || 0;
          const key = String(name || id).toLowerCase();
          book[key] = { name, map, finished, at: Date.now() };
        }
        if (!fresh) return;
        saveTrace(book);
        localStorage.setItem("valvet-pings", JSON.stringify([...heardPings].slice(-40)));
        const tabBtn = document.getElementById("tab-alerts");
        if (tabBtn && tab !== "alerts") tabBtn.classList.add("trace-ping");
        if (typeof paintSocialBook === "function" && window.__valvetSocial) paintSocialBook(window.__valvetSocial);
        playPlayed(false);
      }
      function showSampleCard() {
        samplePost = true;
        const post = document.getElementById("post");
        if (!post) return;
        post.hidden = false;
        document.getElementById("post-title").textContent = "Ascent";
        document.getElementById("post-line").textContent = "Competitive";
        paintPostStats({ score: "13–8", won: true, mode: "Win", kda: "24 / 14 / 5", hs: "28%", adr: "164", acs: "248" });
      }
      function hideSampleCard() {
        if (!samplePost) return;
        samplePost = false;
        const post = document.getElementById("post");
        if (post && !(postCard && Date.now() < postUntil)) post.hidden = true;
      }
      let trendMore = false;
      let clutchMore = false;
      let clutchPick = 0;
      function viewMore(open, hidden, toggle) {
        const button = el("button", "tiny more-toggle", open ? "Show less" : "View more" + (hidden ? " · " + hidden : ""));
        button.type = "button";
        button.onclick = toggle;
        return button;
      }
      function paintTrend(days) {
        const rows = (days || []).filter((day) => day && day.games);
        if (!rows.length) return null;
        const wrap = el("div", "trend");
        wrap.append(el("h2", null, "Daily"));
        const last = rows[rows.length - 1];
        const nums = el("div", "trend-nums");
        const chip = (kicker, value, color) => {
          const card = el("article");
          card.append(el("div", "eyebrow", kicker));
          const b = el("b", null, value);
          b.style.color = color;
          card.append(b);
          return card;
        };
        nums.append(chip("HS", (last.hs || 0) + "%", "#ff5fa2"));
        nums.append(chip("ACS", String(last.acs || 0), "#c9a6ff"));
        nums.append(chip("Kills", String(last.kills || 0), "#3dce86"));
        wrap.append(nums);
        const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        svg.setAttribute("viewBox", "0 0 640 292");
        const ns = "http://www.w3.org/2000/svg";
        const add = (name, attrs) => {
          const node = document.createElementNS(ns, name);
          for (const key of Object.keys(attrs)) node.setAttribute(key, attrs[key]);
          svg.append(node);
          return node;
        };
        const series = [
          { key: "hs", color: "#ff5fa2", cap: 100, label: "HS", suffix: "%" },
          { key: "acs", color: "#c9a6ff", cap: 0, label: "ACS", suffix: "" },
          { key: "kills", color: "#3dce86", cap: 0, label: "Kills", suffix: "" },
        ];
        series.forEach((item, band) => {
          const vals = rows.map((day) => Number(day[item.key]) || 0);
          const max = item.cap || Math.max(1, ...vals);
          const top = 16 + band * 92;
          const h = 62;
          const left = 36;
          const right = 600;
          add("text", { x: "4", y: String(top + 12), fill: item.color, "font-size": "11" }).textContent = item.label;
          add("text", { x: "4", y: String(top + h), fill: "#b89aaa", "font-size": "10" }).textContent = "0";
          add("text", { x: String(right + 8), y: String(top + 10), fill: "#fff7fa", "font-size": "12" }).textContent = String(Math.round(max)) + item.suffix;
          for (const step of [0, 0.5, 1]) {
            const y = top + h - step * h;
            add("line", { x1: String(left), y1: String(y), x2: String(right), y2: String(y), stroke: "rgba(255,247,250,.12)", "stroke-width": "1" });
          }
          const pts = vals.map((value, index) => {
            const x = rows.length === 1 ? (left + right) / 2 : left + (index / (rows.length - 1)) * (right - left);
            const y = top + h - (Math.max(0, Math.min(max, value)) / max) * h;
            return [x, y];
          });
          add("polyline", { points: pts.map((p) => p[0].toFixed(1) + "," + p[1].toFixed(1)).join(" "), fill: "none", stroke: item.color, "stroke-width": "2.5", "stroke-linejoin": "round", "stroke-linecap": "round" });
          const end = pts[pts.length - 1];
          add("circle", { cx: end[0].toFixed(1), cy: end[1].toFixed(1), r: "3.5", fill: item.color });
          add("text", { x: String(Math.min(right - 8, end[0] + 8)), y: String(end[1] - 6), fill: "#fff7fa", "font-size": "12" }).textContent = String(Math.round(vals[vals.length - 1])) + item.suffix;
        });
        const labels = rows.length > 8 ? rows.filter((_, i) => i === 0 || i === rows.length - 1 || i % 2 === 0) : rows;
        labels.forEach((day) => {
          const index = rows.indexOf(day);
          const x = rows.length === 1 ? 318 : 36 + (index / (rows.length - 1)) * 564;
          add("text", { x: String(x), y: "286", fill: "#b89aaa", "font-size": "10", "text-anchor": "middle" }).textContent = day.label || "";
        });
        wrap.append(svg);
        const ordered = rows.slice().reverse();
        const shown = trendMore ? ordered : ordered.slice(0, 7);
        const table = document.createElement("table");
        table.className = "trend-table";
        const head = document.createElement("tr");
        for (const name of ["Day", "Games", "HS", "ACS", "Kills"]) {
          const th = document.createElement("th");
          th.textContent = name;
          head.append(th);
        }
        table.append(head);
        for (const day of shown) {
          const tr = document.createElement("tr");
          for (const value of [day.label || day.day, day.games, (day.hs || 0) + "%", day.acs || 0, day.kills || 0]) {
            const td = document.createElement("td");
            td.textContent = String(value);
            tr.append(td);
          }
          table.append(tr);
        }
        wrap.append(table);
        if (ordered.length > 7) {
          wrap.append(viewMore(trendMore, ordered.length - 7, () => {
            trendMore = !trendMore;
            const slot = document.getElementById("insight-slot");
            if (slot) paintInsights(slot);
          }));
        }
        return wrap;
      }
      function friendIds(data) {
        const ids = new Set();
        for (const id of (data && data.friendIds) || []) ids.add(String(id).toLowerCase());
        for (const person of (data && data.friends) || []) {
          if (person.puuid && !person.you) ids.add(String(person.puuid).toLowerCase());
        }
        return ids;
      }
      function insightCard(kicker, title, sub) {
        const card = el("article");
        card.append(el("div", "eyebrow", kicker));
        card.append(el("b", null, title));
        if (sub) card.append(el("div", "sub", sub));
        return card;
      }
      function paintInsights(root) {
        root.replaceChildren();
        const rec = latestState && latestState.records;
        if (rec && rec.games) {
          const grid = el("div", "insight");
          grid.append(insightCard("Saved", rec.wins + "–" + rec.losses, rec.games + " games · " + (rec.kd || "—") + " K/D"));
          const heads = insightCard("Headshots", rec.hs || "—", "Across saved games");
          grid.append(heads);
          grid.append(insightCard("ACS", rec.acs ? String(rec.acs) : "—", "Average"));
          grid.append(insightCard("Best kills", rec.bestKills ? String(rec.bestKills) : "—", rec.bestKillsMap || ""));
          grid.append(insightCard("Best ACS", rec.bestACS ? String(rec.bestACS) : "—", rec.bestACSMap || ""));
          grid.append(insightCard("Best weapon", rec.bestGun || "—", "Most finishing kills"));
          grid.append(insightCard("Best map", rec.bestMap || "—", rec.bestMapRate ? rec.bestMapRate + " wins" : ""));
          root.append(grid);
          const trend = paintTrend(rec.days);
          if (trend) root.append(trend);
          root.append(el("h2", null, "Rival"));
          if (rec.rival && rec.rival.games) root.append(insightCard(rec.rival.wins + "–" + rec.rival.losses, rec.rival.name, rec.rival.games + " games against them"));
          else root.append(el("p", "sub", "A rival needs two games against the same enemy."));
          root.append(el("h2", null, "With friends"));
          if ((rec.friends || []).length) {
            const chem = el("div", "insight");
            for (const row of rec.friends) {
              const played = row.wins + row.losses;
              const pct = played ? Math.round((row.wins / played) * 100) + "% wins" : "Together";
              chem.append(insightCard(pct, row.name, row.wins + "–" + row.losses + " · " + row.games + " games"));
            }
            root.append(chem);
          } else root.append(el("p", "sub", "Only people on your Riot friends list, and only after two games together."));
          if ((rec.guns || []).length) {
            root.append(el("h2", null, "Weapons"));
            const gunGrid = el("div", "insight");
            for (const gun of rec.guns) {
              const hs = gun.shots ? gun.hs + "% HS" : "";
              gunGrid.append(insightCard(gun.kills + " kills", gun.name, [hs, "Finishing blows"].filter(Boolean).join(" · ")));
            }
            root.append(gunGrid);
          }
          if ((rec.maps || []).length) {
            root.append(el("h2", null, "Maps"));
            const mapGrid = el("div", "insight");
            for (const map of rec.maps) {
              const pct = map.games ? Math.round((map.wins / map.games) * 100) + "%" : "—";
              mapGrid.append(insightCard(pct + " wins", map.name, map.wins + "–" + map.losses + " · " + map.games + " games"));
            }
            root.append(mapGrid);
          }
          if (rec.note) root.append(el("p", "sub", rec.note));
          const chemRows = duoChemistry(history);
          if (chemRows.length) {
            root.append(el("h2", null, "Chemistry"));
            const chem = el("div", "insight");
            for (const row of chemRows) {
              const bits = [];
              if (row.pistolN) bits.push(row.pistol + "/" + row.pistolN + " pistols");
              if (row.trades) bits.push(row.trades + " trades");
              if (row.first) bits.push("died first " + row.first);
              chem.append(insightCard(row.wins + "–" + row.losses, row.name, bits.join(" · ") || "Together"));
            }
            root.append(chem);
          }
          const habits = tendencyCards(history);
          if (habits.length) {
            root.append(el("h2", null, "Tendencies"));
            const grid = el("div", "insight");
            for (const card of habits) grid.append(insightCard(card.value, card.label, card.note));
            root.append(grid);
          }
          return;
        }
        const study = studyHistory(history, friendIds(latestState));
        const grid = el("div", "insight");
        grid.append(insightCard("Saved", study.wins + "–" + study.losses, study.games ? study.games + " recent games" : "Waiting for games"));
        grid.append(insightCard("Record", study.bestKills ? study.bestKills.kills + " kills" : "—", study.bestKills ? study.bestKills.map : ""));
        grid.append(insightCard("Best ACS", study.bestACS ? String(study.bestACS.acs) : "—", study.bestACS ? study.bestACS.map : ""));
        grid.append(insightCard("Best weapon", study.guns[0] ? study.guns[0].name : "—", study.guns[0] ? study.guns[0].kills + " kills" : ""));
        root.append(grid);
        root.append(el("h2", null, "Rival"));
        root.append(study.rival
          ? insightCard(study.rival.wins + "–" + study.rival.losses, study.rival.name, study.rival.games + " games")
          : el("p", "sub", "A rival needs two games against the same enemy."));
        root.append(el("h2", null, "With friends"));
        const chemRows = duoChemistry(history);
        if (chemRows.length) {
          const chem = el("div", "insight");
          for (const row of chemRows) {
            const played = row.wins + row.losses;
            const pct = played ? Math.round((row.wins / played) * 100) + "%" : "Together";
            const bits = [row.wins + "–" + row.losses];
            if (row.pistolN) bits.push(row.pistol + "/" + row.pistolN + " pistols");
            if (row.trades) bits.push(row.trades + " trades");
            if (row.first) bits.push(row.first + " first deaths");
            chem.append(insightCard(pct, row.name, bits.join(" · ")));
          }
          root.append(chem);
        } else root.append(el("p", "sub", "Only Riot friends, after two games together. Random teammates are skipped."));
        const habits = tendencyCards(history);
        if (habits.length) {
          root.append(el("h2", null, "Tendencies"));
          const grid = el("div", "insight");
          for (const card of habits) grid.append(insightCard(card.value, card.label, card.note));
          root.append(grid);
        }
        if (study.maps.length) {
          root.append(el("h2", null, "Maps"));
          const mapGrid = el("div", "insight");
          for (const map of study.maps) {
            const pct = map.games ? Math.round((map.wins / map.games) * 100) + "%" : "—";
            mapGrid.append(insightCard(pct, map.name, map.wins + "–" + map.losses));
          }
          root.append(mapGrid);
        }
      }
      function paintProfile(data) {
        const root = document.getElementById("profile");
        root.className = "";
        root.replaceChildren();
          const card = el("div", "calling");
          if (data.card) card.style.backgroundImage = "url(" + data.card + ")";
          if (data.border) {
            const frame = document.createElement("img");
            frame.className = "frame";
            frame.alt = "";
            frame.src = data.border;
            card.append(frame);
          }
          const id = el("div", "id");
          id.append(el("b", null, (data.name || "You") + (data.tag ? " #" + data.tag : "")));
          id.append(el("div", null, data.title || "No title"));
          id.append(el("div", "sub", [data.level ? "Level " + data.level : "Level unavailable", data.kc ? data.kc + " KC" : ""].filter(Boolean).join(" · ")));
          if (data.level) card.append(el("b", "acct", String(data.level)));
          card.append(id);
          root.append(card);
          paintActCompare(root);
          const poster = el("button", "tiny", "Act poster");
          poster.type = "button";
          poster.onclick = () => copyActPoster();
          root.append(poster);
          const slot = el("div");
          slot.id = "insight-slot";
          root.append(slot);
          paintInsights(slot);
          const rrSlot = el("div");
          rrSlot.id = "rr-slot";
          root.append(rrSlot);
          paintRR(rrSlot);
          if (data.error) root.append(el("p", "sub", data.error));
          root.append(el("h2", null, "Agent mastery"));
          root.append(el("p", "sub", "Act level resets each Act. Lifetime level stays. One Act level is one Lifetime level. Portrait accents land at 4, 7 and 10."));
          const list = (data.agents || []).filter(hasMastery);
          if (!list.length) root.append(el("div", "empty", data.error || "No Act or Lifetime levels came back. Open an agent's mastery screen in Valorant once, then open Profile again."));
          for (const agent of list) {
            const row = el("div", "gear-row");
            if (agent.icon) {
              const img = document.createElement("img");
              img.className = "agent";
              img.alt = "";
              img.src = agent.icon;
              row.append(img);
            }
            const text = el("div");
            text.append(el("b", null, agent.name || "Agent"));
            text.append(masteryStats(agent));
            const bits = [agent.accent || "", agent.owned ? "" : "locked"].filter(Boolean);
            if (bits.length) text.append(el("div", "sub", bits.join(" · ")));
            row.append(text);
            root.append(row);
          }
      }
      function actCell(label, games, wins) {
        const cell = el("div", "act-cell");
        cell.append(el("b", null, games + " games"));
        cell.append(el("div", null, wins + "–" + Math.max(0, games - wins)));
        cell.append(el("div", "sub", label));
        return cell;
      }
      function paintActCompare(root) {
        if (!root) return;
        const games = (history || []).length;
        let wins = 0;
        for (const match of history || []) if (match && match.won === true) wins++;
        const box = el("div", "act-compare");
        box.id = "act-compare";
        box.append(el("div", "eyebrow", "Acts"));
        const row = el("div", "act-row");
        row.append(actCell("This act", games, wins));
        const last = latestState && latestState.lastAct;
        if (last && Number(last.games) > 0) row.append(actCell("Last act", Number(last.games) || 0, Number(last.wins) || 0));
        else row.append(el("div", "act-cell", "Last act is not on this PC yet."));
        box.append(row);
        const prev = root.querySelector("#act-compare");
        if (prev) prev.replaceWith(box);
        else {
          const slot = root.querySelector("#insight-slot");
          if (slot) slot.before(box);
          else root.append(box);
        }
      }
      async function loadProfile(live) {
        const root = document.getElementById("profile");
        const key = "valvet-cache-card";
        const saved = readJSON(key);
        const ok = saved && (saved.name || saved.card || saved.level);
        if (ok) paintProfile(saved);
        else if (latestState && (latestState.you || knownLevel)) {
          const raw = String(latestState.you || "");
          const cut = raw.lastIndexOf("#");
          paintProfile({ name: cut > 0 ? raw.slice(0, cut) : raw, tag: cut > 0 ? raw.slice(cut + 1) : "", level: knownLevel || latestState.level || 0 });
        } else if (live) {
          root.className = "";
          root.replaceChildren(el("div", "empty", "Loading your card…"));
        }
        const need = !ok || (live && !cacheFresh(key, 30 * 60 * 1000));
        if (!need) return;
        if (!ok && !live) {
          root.className = "";
          root.replaceChildren(el("div", "empty", "Loading your card…"));
        }
        try {
          const data = await (await fetch("/api/card", { cache: "no-store" })).json();
          writeJSON(key, data);
          stampCache(key);
          if (!saved || JSON.stringify(saved) !== JSON.stringify(data)) paintProfile(data);
        } catch {
          if (!ok) root.replaceChildren(el("div", "empty", "Couldn't load your profile."));
        }
      }
      async function loadAgents(live) {
        const root = document.getElementById("agents");
        const key = "valvet-cache-agents";
        if (!agentCache) {
          const saved = readJSON(key);
          if (Array.isArray(saved) && saved.length) agentCache = saved;
        }
        const ok = Array.isArray(agentCache) && agentCache.length;
        const sig = ok ? agentCache.map((agent) => agent.id || agent.name).join("|") : "";
        if (ok && root.dataset.sig === sig && root.querySelector(".agent-pick")) {
          /* already on screen */
        } else if (ok) {
          drawAgents(agentCache);
          root.dataset.sig = sig;
        } else if (live) {
          root.className = "";
          root.replaceChildren(el("div", "empty", "Loading agents…"));
        }
        const force = live === "force";
        const need = force || !ok || !cacheFresh(key, 7 * 24 * 60 * 60 * 1000);
        if (!need) return;
        if (!ok && !live) {
          root.className = "";
          root.replaceChildren(el("div", "empty", "Loading agents…"));
        }
        try {
          const data = await (await fetch("/api/agents", { cache: "no-store" })).json();
          const list = data.agents || [];
          if (!list.length) {
            if (!agentCache) root.replaceChildren(el("div", "empty", "Couldn't load agents."));
            return;
          }
          const prev = JSON.stringify(agentCache || []);
          const next = JSON.stringify(list);
          agentCache = list;
          writeJSON(key, list);
          stampCache(key);
          const nextSig = list.map((agent) => agent.id || agent.name).join("|");
          if (force || prev !== next) {
            drawAgents(list);
            root.dataset.sig = nextSig;
          }
        } catch {
          if (!agentCache) root.replaceChildren(el("div", "empty", "Couldn't load agents."));
        }
      }
      function missionRow(mission) {
        const row = el("div", "mission");
        const title = el("div");
        title.append(el("b", null, mission.title || "Mission"));
        const meta = [mission.kind, mission.xp ? mission.xp + " XP" : "", mission.ends ? "ends " + mission.ends : "", mission.done ? "Done" : ""].filter(Boolean);
        title.append(el("div", "sub", meta.join(" · ")));
        const goal = Number(mission.goal || 0);
        const progress = Number(mission.progress || 0);
        title.append(el("div", "meta", goal ? progress + " / " + goal : String(progress)));
        const meter = el("div", "meter");
        const fill = document.createElement("span");
        const pct = goal ? Math.max(0, Math.min(100, (progress / goal) * 100)) : (mission.done ? 100 : 0);
        fill.style.width = pct + "%";
        meter.append(fill);
        title.append(meter);
        row.append(title);
        return row;
      }
      function drawPass(data) {
        const root = document.getElementById("pass");
        root.className = "";
        root.replaceChildren();
        savedBar(root, "valvet-cache-pass", () => loadPass("force"));
        if (data.error && !data.level && !(data.missions || []).length) {
          root.append(el("div", "empty", data.error));
          return;
        }
        const head = el("div", "pass");
        const copy = el("div");
        copy.append(el("div", "eyebrow", data.free ? "Free pass" : "Battle pass"));
        copy.append(el("h1", null, data.name || "Battle pass"));
        const goal = Number(data.next || 0);
        const xp = Number(data.xp || 0);
        if (goal) copy.append(el("div", "sub", xp + " / " + goal + " XP"));
        if (data.ends) {
          const left = el("div", "pass-clock");
          left.id = "pass-left";
          left.dataset.ends = data.ends;
          paintActClock(left, data.ends);
          copy.append(left);
        }
        const meter = el("div", "pass-meter");
        const fill = document.createElement("span");
        fill.style.width = (goal ? Math.max(4, Math.min(100, (xp / goal) * 100)) : 8) + "%";
        meter.append(fill);
        copy.append(meter);
        head.append(copy);
        head.append(el("b", "level", String(data.level || 0)));
        root.append(head);
        if (data.error) root.append(el("p", "sub", data.error));
        const rewards = data.rewards || [];
        if (rewards.length) {
          root.append(el("h2", null, "All rewards · " + rewards.length));
          const grid = el("div", "rewards pass-scroll");
          let current = null;
          for (const reward of rewards) {
            const card = el("article", "reward" + (reward.got ? " got" : "") + (reward.level === data.level ? " now" : ""));
            if (reward.level === data.level) current = card;
            const preview = reward.anim && !String(reward.anim).includes(".mp4") ? reward.anim : reward.icon;
            if (preview) {
              const stage = el("div", "pedestal");
              const img = document.createElement("img");
              img.alt = "";
              img.src = preview;
              stage.append(img);
              card.append(stage);
            }
            card.style.cursor = "pointer";
            card.onclick = () => openZoom({ icon: reward.icon, anim: reward.anim, name: "Tier " + reward.level + " · " + (reward.name || "Reward") + (reward.got ? " · claimed" : "") });
            card.append(el("div", "eyebrow", "Tier " + reward.level + (reward.free ? " · free" : "") + (reward.got ? " · claimed" : "")));
            card.append(el("b", null, reward.name || "Reward"));
            grid.append(card);
          }
          root.append(grid);
          if (current) requestAnimationFrame(() => current.scrollIntoView({ block: "center", inline: "nearest" }));
        }
        for (const track of data.tracks || []) {
          const head2 = el("div", "pass");
          const copy2 = el("div");
          copy2.append(el("div", "eyebrow", track.kind || (track.free ? "Free pass" : "Event pass")));
          copy2.append(el("h1", null, track.name || "Pass"));
          const goal2 = Number(track.next || 0);
          const xp2 = Number(track.xp || 0);
          if (goal2) copy2.append(el("div", "sub", xp2 + " / " + goal2 + " XP"));
          const meter2 = el("div", "pass-meter");
          const fill2 = document.createElement("span");
          fill2.style.width = (goal2 ? Math.max(4, Math.min(100, (xp2 / goal2) * 100)) : 8) + "%";
          meter2.append(fill2);
          copy2.append(meter2);
          head2.append(copy2);
          head2.append(el("b", "level", String(track.level || 0)));
          root.append(head2);
          const rewards2 = track.rewards || [];
          if (!rewards2.length) continue;
          root.append(el("h2", null, (track.kind || "Pass") + " rewards · " + rewards2.length));
          const grid2 = el("div", "rewards");
          for (const reward of rewards2) {
            const card = el("article", "reward" + (reward.got ? " got" : "") + (reward.level === track.level ? " now" : ""));
            const preview = reward.anim && !String(reward.anim).includes(".mp4") ? reward.anim : reward.icon;
            if (preview) {
              const stage = el("div", "pedestal");
              const img = document.createElement("img");
              img.alt = "";
              img.src = preview;
              stage.append(img);
              card.append(stage);
            }
            card.style.cursor = "pointer";
            card.onclick = () => openZoom({ icon: reward.icon, anim: reward.anim, name: "Tier " + reward.level + " · " + (reward.name || "Reward") + (reward.free ? " · free" : "") });
            card.append(el("div", "eyebrow", "Tier " + reward.level + (reward.free ? " · free" : "") + (reward.got ? " · claimed" : "")));
            card.append(el("b", null, reward.name || "Reward"));
            grid2.append(card);
          }
          root.append(grid2);
        }
      }
      function drawMissions(data) {
        const root = document.getElementById("missions");
        root.className = "";
        root.replaceChildren();
        if (data.error && !(data.missions || []).length) {
          root.append(el("div", "empty", data.error));
          return;
        }
        const kindOf = (mission) => String(mission.kind || "").toLowerCase();
        const daily = (data.missions || []).filter((mission) => kindOf(mission) === "daily");
        const weekly = (data.missions || []).filter((mission) => kindOf(mission) === "weekly");
        root.append(el("h2", null, "Daily"));
        if (!daily.length) root.append(el("div", "empty", "No daily missions."));
        for (const mission of daily) root.append(missionRow(mission));
        root.append(el("h2", null, "Weekly"));
        if (!weekly.length) root.append(el("div", "empty", "No weekly missions."));
        for (const mission of weekly) root.append(missionRow(mission));
      }
      function readJSON(key) {
        try { return JSON.parse(localStorage.getItem(key) || "null"); } catch { return null; }
      }
      function writeJSON(key, data) {
        try {
          const raw = JSON.stringify(data);
          if (raw && raw.length < 1500000) localStorage.setItem(key, raw);
        } catch {}
      }
      function stampCache(key) {
        try { localStorage.setItem(key + "-at", String(Date.now())); } catch {}
      }
      function cacheFresh(key, ms) {
        const at = Number(localStorage.getItem(key + "-at") || 0);
        return at > 0 && Date.now() - at < ms;
      }
      function savedAgo(key) {
        const at = Number(localStorage.getItem(key + "-at") || 0);
        if (!at) return "";
        const mins = Math.max(0, Math.round((Date.now() - at) / 60000));
        if (mins < 1) return "saved just now";
        if (mins < 60) return "saved " + mins + "m ago";
        const hours = Math.floor(mins / 60);
        if (hours < 48) return "saved " + hours + "h ago";
        return "saved " + Math.floor(hours / 24) + "d ago";
      }
      function savedBar(root, key, refresh) {
        if (!root) return;
        const bar = el("div", "saved-bar");
        bar.append(el("span", null, savedAgo(key) || "Not saved yet"));
        const btn = el("button", "tiny", "Refresh");
        btn.type = "button";
        btn.onclick = (event) => {
          event.preventDefault();
          event.stopPropagation();
          refresh();
        };
        bar.append(btn);
        root.prepend(bar);
      }
      function showCachedPass() {
        const root = document.getElementById("pass");
        if (!root) return;
        const saved = readJSON("valvet-cache-pass");
        if (saved && (saved.level || saved.name || (saved.missions || []).length || (saved.rewards || []).length)) drawPass(saved);
      }
      async function loadPass(live) {
        const root = document.getElementById("pass");
        const key = "valvet-cache-pass";
        const saved = readJSON(key);
        const ok = saved && (saved.level || saved.name || (saved.missions || []).length || (saved.rewards || []).length);
        const sig = ok ? [saved.level, saved.xp, saved.name, (saved.missions || []).map((mission) => (mission.title || "") + (mission.progress || 0)).join(",")].join("|") : "";
        const force = live === "force";
        if (ok && root.dataset.sig === sig && root.querySelector(".pass, .reward, .mission")) {
          /* already on screen */
        } else if (ok) {
          drawPass(saved);
          root.dataset.sig = sig;
        }
        const need = force || !ok || !cacheFresh(key, 30 * 60 * 1000);
        if (!need) return;
        if (!ok) {
          root.className = "";
          root.replaceChildren(el("div", "empty", "Loading battle pass…"));
        }
        try {
          const data = await (await fetch("/api/pass", { cache: "no-store" })).json();
          writeJSON(key, data);
          stampCache(key);
          const nextSig = [data.level, data.xp, data.name, (data.missions || []).map((mission) => (mission.title || "") + (mission.progress || 0)).join(",")].join("|");
          if (force || root.dataset.sig !== nextSig) {
            drawPass(data);
            root.dataset.sig = nextSig;
          }
        } catch {
          if (!ok) root.replaceChildren(el("div", "empty", "Couldn't load the battle pass."));
        }
      }
      document.getElementById("find-player").onsubmit = async (event) => {
        event.preventDefault();
        const note = document.getElementById("find-note");
        const q = document.getElementById("find-q").value.trim();
        note.textContent = "Searching…";
        try {
          const data = await (await fetch("/api/search?q=" + encodeURIComponent(q), { cache: "no-store" })).json();
          if (!data.puuid) {
            note.textContent = data.error || "No player found.";
            if (data.tracker) fetch("/api/open?url=" + encodeURIComponent(data.tracker), { method: "POST" }).catch(() => {});
            return;
          }
          note.textContent = "";
          tone(660, 0.08);
          openPlayer({ puuid: data.puuid, name: data.name, tag: data.tag });
        } catch {
          note.textContent = "Search failed.";
        }
      };
      function sharedGames(player) {
        const id = String((player && player.puuid) || "").toLowerCase();
        if (!id) return [];
        const rows = [];
        for (const match of history || []) {
          const hit = (match.players || []).find((person) => String(person.puuid || "").toLowerCase() === id);
          if (!hit) continue;
          const mine = (match.players || []).find((person) => person.you);
          const same = mine && mine.team && hit.team && mine.team === hit.team;
          let won = null;
          if (match.won === true) won = !!same;
          else if (match.won === false) won = !same;
          const people = (match.players || []).map((person) => ({ ...person, you: String(person.puuid || "").toLowerCase() === id }));
          rows.push({
            map: match.map,
            mode: match.mode,
            agent: hit.agent,
            kills: hit.kills,
            deaths: hit.deaths,
            assists: hit.assists,
            hs: hit.hs,
            score1: match.score1,
            score2: match.score2,
            won,
            players: people,
            rounds: hit.you ? match.rounds : null,
          });
        }
        return rows.slice(0, 8);
      }
      function duoChemistry(matches) {
        const book = new Map();
        for (const match of matches || []) {
          const you = (match.players || []).find((player) => player.you);
          if (!you) continue;
          const youId = String(you.puuid || "").toLowerCase();
          const mates = (match.players || []).filter((player) => !player.you && player.team && you.team && player.team === you.team && player.puuid);
          if (!mates.length) continue;
          for (const mate of mates) {
            const id = String(mate.puuid).toLowerCase();
            const row = book.get(id) || { name: shownName(mate), wins: 0, losses: 0, pistol: 0, pistolN: 0, trades: 0, first: 0 };
            if (mate.name && mate.name !== "Player" && mate.name !== "Hidden") row.name = shownName(mate);
            if (match.won === true) row.wins += 1;
            else if (match.won === false) row.losses += 1;
            book.set(id, row);
          }
          for (const round of match.rounds || []) {
            const n = Number(round.n);
            if (n !== 1 && n !== 13) continue;
            for (const mate of mates) {
              const row = book.get(String(mate.puuid).toLowerCase());
              if (!row) continue;
              row.pistolN += 1;
              if (round.you) row.pistol += 1;
            }
          }
          const seen = new Set();
          for (const edge of (match.edges || []).slice().sort((a, b) => a.round - b.round || a.t - b.t)) {
            if (edge.trade) {
              const row = book.get(String(edge.kid || "").toLowerCase());
              if (row) row.trades += 1;
            }
            if (seen.has(edge.round)) continue;
            seen.add(edge.round);
            const vid = String(edge.vid || "").toLowerCase();
            const youDied = (youId && vid === youId) || /^you\b/i.test(String(edge.victim || ""));
            if (!youDied) continue;
            for (const mate of mates) {
              const row = book.get(String(mate.puuid).toLowerCase());
              if (row) row.first += 1;
            }
          }
        }
        return [...book.values()].filter((row) => row.wins + row.losses >= 2).sort((a, b) => (b.wins + b.losses) - (a.wins + a.losses)).slice(0, 6);
      }
      function tendencyCards(matches) {
        let rounds = 0;
        let firstK = 0;
        let firstD = 0;
        let lost = 0;
        let saved = 0;
        for (const match of matches || []) {
          const you = (match.players || []).find((player) => player.you);
          if (!you || !(match.rounds || []).length) continue;
          const youId = String(you.puuid || "").toLowerCase();
          const byRound = {};
          for (const edge of match.edges || []) {
            const list = byRound[edge.round] || [];
            list.push(edge);
            byRound[edge.round] = list;
          }
          for (const round of match.rounds) {
            rounds += 1;
            const edges = (byRound[round.n] || []).slice().sort((a, b) => a.t - b.t);
            const first = edges.find((edge) => edge.first) || edges[0];
            if (first) {
              const kid = String(first.kid || "").toLowerCase();
              const vid = String(first.vid || "").toLowerCase();
              if ((youId && kid === youId) || /^you\b/i.test(String(first.killer || ""))) firstK += 1;
              if ((youId && vid === youId) || /^you\b/i.test(String(first.victim || ""))) firstD += 1;
            }
            if (round.you === false) {
              lost += 1;
              const died = edges.some((edge) => {
                const vid = String(edge.vid || "").toLowerCase();
                return (youId && vid === youId) || /^you\b/i.test(String(edge.victim || ""));
              });
              if (!died) saved += 1;
            }
          }
        }
        if (!rounds) return [];
        const cards = [
          { value: firstK + " / " + rounds, label: "Swing first", note: "Rounds you took the first kill" },
          { value: firstD + " / " + rounds, label: "Die first", note: "Rounds you were the first death" },
        ];
        if (lost) cards.push({ value: saved + " / " + lost, label: "Saves", note: "Lost rounds you were still alive" });
        return cards;
      }
      function scoutCard(player) {
        const id = String((player && player.puuid) || "").toLowerCase();
        if (!id) return null;
        const rows = [];
        let yours = 0;
        let theirs = 0;
        for (const match of history || []) {
          const hit = (match.players || []).find((person) => String(person.puuid || "").toLowerCase() === id);
          if (!hit) continue;
          const you = (match.players || []).find((person) => person.you);
          const same = !!(you && you.team && hit.team && you.team === hit.team);
          if (rows.length < 5) rows.push({ map: mapName(match.map) || match.map || "Map", agent: hit.agent || "", won: match.won, same });
          if (!same && (match.won === true || match.won === false)) {
            if (match.won === true) yours += 1;
            else theirs += 1;
          }
        }
        if (!rows.length) return null;
        const card = el("div", "scout");
        card.append(el("div", "eyebrow", "Scout"));
        card.append(el("b", null, (yours + theirs) ? "You " + yours + "–" + theirs : "Shared games"));
        const line = el("div", "scout-line");
        for (const row of rows) {
          const mark = row.won === true ? (row.same ? "W" : "L") : row.won === false ? (row.same ? "L" : "W") : "–";
          line.append(el("span", null, mark + "  " + row.map + (row.agent ? " · " + row.agent : "")));
        }
        card.append(line);
        return card;
      }
      function copyActPoster() {
        const canvas = document.createElement("canvas");
        canvas.width = 900;
        canvas.height = 1200;
        const ctx = canvas.getContext("2d");
        if (!ctx) return;
        ctx.fillStyle = "#14080f";
        ctx.fillRect(0, 0, 900, 1200);
        ctx.fillStyle = "#ff5fa2";
        ctx.font = "700 28px Segoe UI, sans-serif";
        ctx.fillText("VALVET", 64, 96);
        const games = (history || []).length;
        let wins = 0;
        const agents = {};
        const maps = {};
        for (const match of history || []) {
          if (match.won === true) wins += 1;
          const you = (match.players || []).find((player) => player.you);
          if (you && you.agent && you.agent !== "Picking") agents[you.agent] = (agents[you.agent] || 0) + 1;
          if (match.map) maps[match.map] = (maps[match.map] || 0) + 1;
        }
        const best = (book) => Object.entries(book).sort((a, b) => b[1] - a[1])[0];
        const agent = best(agents);
        const map = best(maps);
        const last = (rrPoints || [])[(rrPoints || []).length - 1];
        ctx.fillStyle = "#fff7fa";
        ctx.font = "760 68px Segoe UI, sans-serif";
        ctx.fillText(last && last.rank ? last.rank : "This act", 64, 230);
        ctx.fillStyle = "#ffd0e4";
        ctx.font = "650 32px Segoe UI, sans-serif";
        ctx.fillText(wins + " wins   " + Math.max(0, games - wins) + " losses", 64, 310);
        ctx.fillStyle = "#fff7fa";
        ctx.font = "700 40px Segoe UI, sans-serif";
        ctx.fillText(agent ? agent[0] : "—", 64, 460);
        ctx.fillStyle = "#b89aaa";
        ctx.font = "600 22px Segoe UI, sans-serif";
        ctx.fillText("Most played agent", 64, 500);
        ctx.fillStyle = "#fff7fa";
        ctx.font = "700 40px Segoe UI, sans-serif";
        ctx.fillText(map ? (mapName(map[0]) || map[0]) : "—", 64, 620);
        ctx.fillStyle = "#b89aaa";
        ctx.font = "600 22px Segoe UI, sans-serif";
        ctx.fillText("Most played map", 64, 660);
        canvas.toBlob((blob) => {
          if (!blob) return;
          const url = URL.createObjectURL(blob);
          const link = document.createElement("a");
          link.href = url;
          link.download = "valvet-act.png";
          link.click();
          setTimeout(() => URL.revokeObjectURL(url), 1500);
        });
      }
      function attachDeathBeat(match) {
        const host = document.getElementById("post-death");
        if (!host || !match || !match.id) return;
        if (host.dataset.id === String(match.id)) return;
        host.dataset.id = String(match.id);
        host.replaceChildren();
        const you = (match.players || []).find((player) => player.you);
        const youId = String((you && you.puuid) || "").toLowerCase();
        const deaths = (match.edges || []).filter((edge) => {
          const vid = String(edge.vid || "").toLowerCase();
          return (youId && vid === youId) || /^you\b/i.test(String(edge.victim || ""));
        });
        if (!deaths.length && !(match.deaths || []).length) {
          host.hidden = true;
          return;
        }
        host.hidden = false;
        const box = el("div", "death-beat");
        const canvas = document.createElement("canvas");
        canvas.width = 520;
        canvas.height = 168;
        box.append(canvas, el("div", "sub", "Your last death"));
        host.append(box);
        const ctx = canvas.getContext("2d");
        if (!ctx) return;
        const art = MAP_ART[mapName(match.map) || match.map || ""] || "";
        const img = new Image();
        if (art) img.src = art;
        const started = performance.now();
        const draw = (now) => {
          if (host.dataset.id !== String(match.id)) return;
          const t = Math.min(1, (now - started) / 8000);
          ctx.clearRect(0, 0, 520, 168);
          ctx.fillStyle = "#10080d";
          ctx.fillRect(0, 0, 520, 168);
          if (img.complete && img.naturalWidth) {
            ctx.globalAlpha = 0.28;
            ctx.drawImage(img, 0, 0, 520, 168);
            ctx.globalAlpha = 1;
          }
          const x = 36 + t * 430;
          const y = 84 + Math.sin(t * Math.PI) * -26;
          ctx.strokeStyle = "rgba(255,95,162,.9)";
          ctx.lineWidth = 3;
          ctx.lineCap = "round";
          ctx.beginPath();
          ctx.moveTo(36, 96);
          ctx.lineTo(x, y);
          ctx.stroke();
          ctx.fillStyle = "#fff7fa";
          ctx.beginPath();
          ctx.arc(x, y, 6, 0, Math.PI * 2);
          ctx.fill();
          if (t < 1 && document.body.dataset.motion !== "calm") requestAnimationFrame(draw);
        };
        if (document.body.dataset.motion === "calm") draw(started + 8000);
        else requestAnimationFrame(draw);
      }
      function profileRow(match) {
        markAwards(match);
        const you = (match.players || []).find((player) => player.you);
        const wrap = el("div", "pgame");
        const row = el("button", "pmatch" + (match.won === true ? " win" : match.won === false ? " loss" : ""));
        row.type = "button";
        const left = el("div");
        const title = el("div");
        title.append(el("b", null, mapName(match.map) || "Match"));
        if (you && you.mvp) title.append(document.createTextNode(" "), el("span", "badge mvp", "MVP"));
        else if (you && you.svp) title.append(document.createTextNode(" "), el("span", "badge svp", "SVP"));
        left.append(title);
        const kda = match.kills != null && match.kills !== "" ? match.kills + "/" + (match.deaths || "0") + (match.assists != null && match.assists !== "" ? "/" + match.assists : "") : "";
        left.append(el("div", "sub", [match.mode, match.agent, kda, match.hs ? match.hs + " HS" : ""].filter(Boolean).join(" · ")));
        const right = el("div", "pscore");
        if (match.won === true || match.won === false) right.append(catFace(match.won === true));
        right.append(el("b", null, score(match.score1) + "–" + score(match.score2)));
        if (match.won === true) right.append(el("span", null, "Win"));
        else if (match.won === false) right.append(el("span", null, "Loss"));
        row.append(left, right);
        row.onclick = () => openMatchSheet(match);
        wrap.append(row);
        return wrap;
      }
      function dropRoundMark() {
        if (!latestState || latestState.phase !== "In game") return;
        const round = (Number(latestState.score1) || 0) + (Number(latestState.score2) || 0) + 1;
        const rows = readJSON("valvet-marks");
        const marks = Array.isArray(rows) ? rows : [];
        marks.unshift({ at: Date.now(), round, map: latestState.map || "" });
        localStorage.setItem("valvet-marks", JSON.stringify(marks.slice(0, 24)));
        const btn = document.getElementById("mark-round");
        if (btn) {
          btn.textContent = "Marked " + round;
          setTimeout(() => { if (btn.textContent.indexOf("Marked") === 0) btn.textContent = "Mark round"; }, 1200);
        }
      }
      function paintSameLobby(data) {
        const node = document.getElementById("same-lobby");
        if (!node) return;
        const live = data && (data.phase === "Agent select" || data.phase === "In game" || data.phase === "Loading lobby");
        const ids = new Set(((data && data.players) || []).filter((player) => player && player.puuid && !player.you).map((player) => String(player.puuid).toLowerCase()));
        if (!live || ids.size < 3) { node.hidden = true; return; }
        let best = null;
        for (const match of history || []) {
          const names = [];
          const seen = new Set();
          for (const player of match.players || []) {
            const id = String(player.puuid || "").toLowerCase();
            if (!id || player.you || !ids.has(id) || seen.has(id)) continue;
            seen.add(id);
            if (player.name) names.push(player.name);
          }
          if (seen.size >= 3 && (!best || seen.size > best.n)) best = { n: seen.size, names, map: match.map };
        }
        if (!best) { node.hidden = true; return; }
        const text = best.n + " from " + (mapName(best.map) || "an older game") + " · " + best.names.slice(0, 4).join(", ");
        if (node.textContent !== text) node.textContent = text;
        node.hidden = false;
      }
      function markedRoundSet(match) {
        const rows = readJSON("valvet-marks");
        const marks = Array.isArray(rows) ? rows : [];
        const when = Date.parse((match && match.when) || 0);
        const out = new Set();
        for (const mark of marks) {
          if (!mark || !mark.round) continue;
          if (mark.map && match && match.map && mapName(mark.map) !== mapName(match.map)) continue;
          if (when && mark.at && (when < mark.at - 120000 || when > mark.at + 3 * 3600000)) continue;
          out.add(Number(mark.round));
        }
        return out;
      }
      if (!document.body.dataset.markKey) {
        document.body.dataset.markKey = "1";
        document.addEventListener("keydown", (event) => {
          if (event.key !== "b" && event.key !== "B") return;
          if (event.metaKey || event.ctrlKey || event.altKey) return;
          const tag = event.target && event.target.tagName;
          if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
          dropRoundMark();
        });
        const markBtn = document.getElementById("mark-round");
        if (markBtn) markBtn.onclick = () => dropRoundMark();
      }
      function roundScrub(match) {
        const rounds = (match && match.rounds) || [];
        if (!rounds.length) return null;
        const wrap = el("div", "scrub");
        const view = el("div", "scrub-view");
        const row = el("div", "scrub-row");
        const slider = document.createElement("input");
        slider.type = "range";
        slider.min = "1";
        slider.max = String(rounds.length);
        slider.value = String(rounds.length);
        const prev = el("button", "tiny", "Prev");
        const next = el("button", "tiny", "Next");
        prev.type = "button";
        next.type = "button";
        const paint = () => {
          const at = Math.max(1, Math.min(rounds.length, Number(slider.value) || rounds.length));
          slider.value = String(at);
          const round = rounds[at - 1];
          let yours = 0;
          let theirs = 0;
          for (let i = 0; i < at; i++) {
            if (rounds[i].you) yours += 1;
            else theirs += 1;
          }
          if (round.scoreY || round.scoreN) {
            yours = round.scoreY || 0;
            theirs = round.scoreN || 0;
          }
          const youBuy = ((match.buys || []).find((row) => Number(row.round) === Number(round.n) && String(row.id || "").toLowerCase() === String(((match.players || []).find((player) => player.you) || {}).puuid || "").toLowerCase())) || null;
          const buyBit = youBuy && youBuy.buy ? youBuy.buy + (youBuy.gun ? " " + youBuy.gun : "") : "";
          view.replaceChildren(el("b", null, "Round " + (round.n || at)), el("div", "scrub-score", yours + "–" + theirs), el("div", "sub", [round.you ? "Won" : "Lost", round.how, round.side, buyBit, round.vs ? "1v" + round.vs : ""].filter(Boolean).join(" · ")));
          ticks.replaceChildren();
          if (round.plant || round.hold) ticks.append(el("span", "scrub-tick", round.plant ? "Plant" : "Hold"));
          const youId = String((((match.players || []).find((player) => player.you)) || {}).puuid || "").toLowerCase();
          for (const edge of match.edges || []) {
            if (Number(edge.round) !== Number(round.n || at)) continue;
            const kid = String(edge.kid || "").toLowerCase();
            const vid = String(edge.vid || "").toLowerCase();
            const kill = (youId && kid === youId) || /^you\b/i.test(String(edge.killer || ""));
            const death = (youId && vid === youId) || /^you\b/i.test(String(edge.victim || ""));
            if (!kill && !death) continue;
            const tick = el("button", "tiny scrub-tick", kill ? "Kill" : "Death");
            tick.type = "button";
            tick.onclick = (event) => {
              event.preventDefault();
              event.stopPropagation();
              ticks.querySelectorAll(".scrub-tick").forEach((node) => node.classList.remove("on"));
              tick.classList.add("on");
            };
            ticks.append(tick);
          }
        };
        const ticks = el("div", "scrub-ticks");
        prev.onclick = (event) => { event.preventDefault(); event.stopPropagation(); slider.value = String(Number(slider.value) - 1); paint(); };
        next.onclick = (event) => { event.preventDefault(); event.stopPropagation(); slider.value = String(Number(slider.value) + 1); paint(); };
        slider.oninput = paint;
        row.append(prev, slider, next);
        wrap.append(view, row, ticks);
        paint();
        return wrap;
      }
      function roundStrip(rounds, match) {
        if (!rounds || !rounds.length) return null;
        const marks = match ? markedRoundSet(match) : new Set();
        const line = el("div", "round-line");
        for (const round of rounds) {
          const pip = el("i", (round.you ? "win" : "loss") + (marks.has(Number(round.n)) ? " marked" : ""), String(round.n));
          bindHoverTip(pip, (tip) => {
            tip.append(el("div", "tip-kicker", "Round " + round.n));
            tip.append(el("b", null, round.you ? "Won" : "Lost"));
            const extra = [round.how, round.side].filter(Boolean).join(" · ");
            if (extra) tip.append(el("div", "tip-line", extra));
          });
          line.append(pip);
        }
        return line;
      }
      function openMatchSheet(match) {
        const box = document.getElementById("matchbox");
        document.getElementById("sheet-kicker").textContent = [match.mode, match.agent, match.won === true ? "Win" : match.won === false ? "Loss" : ""].filter(Boolean).join(" · ");
        document.getElementById("sheet-title").textContent = (match.map || "Match") + "   " + score(match.score1) + "–" + score(match.score2);
        const body = document.getElementById("sheet-body");
        body.replaceChildren();
        markAwards(match);
        const players = match.players || [];
        if (!players.length) {
          body.append(el("div", "empty", "That game came back without the full scoreboard."));
          box.hidden = false;
          return;
        }
        const sides = el("div", "sides");
        const you = players.find((player) => player.you);
        const colors = partyColors(players);
        for (const [team, list] of groups(players)) {
          const mine = list.some((player) => player.you) || (you && you.team === team);
          const block = sideList(list, team, mine ? "ally" : "enemy", colors);
          if (block) sides.append(block);
        }
        const strip = roundStrip(match.rounds, match);
        const scrub = roundScrub(match);
        if (scrub) body.append(scrub);
        if (strip) body.append(strip);
        const tools = el("div", "set-actions");
        const play = el("button", "tiny solid", "Play the round");
        play.type = "button";
        play.onclick = () => openDirector(match);
        tools.append(play);
        body.append(tools);
        body.append(sides, statTable(players));
        wireHover(body);
        box.hidden = false;
      }
      document.getElementById("sheet-close").onclick = () => {
        const box = document.getElementById("matchbox");
        box.hidden = true;
        box.classList.remove("notes");
      };
      document.getElementById("matchbox").onclick = (event) => { if (event.target.id === "matchbox") event.currentTarget.hidden = true; };
      let comparePair = [];
      function playerSummary(seed, data) {
        const id = String((seed && seed.puuid) || (data && data.puuid) || "").toLowerCase();
        let games = 0;
        let wins = 0;
        let kills = 0;
        let deaths = 0;
        let assists = 0;
        let hs = 0;
        let hsN = 0;
        for (const match of history || []) {
          const hit = (match.players || []).find((person) => String(person.puuid || "").toLowerCase() === id);
          if (!hit) continue;
          games += 1;
          const you = (match.players || []).find((person) => person.you);
          const same = !!(you && you.team && hit.team && you.team === hit.team);
          const won = resultOf(match);
          const theirs = hit.you || same ? won : won == null ? null : !won;
          if (theirs === true) wins += 1;
          kills += Number(hit.kills) || 0;
          deaths += Number(hit.deaths) || 0;
          assists += Number(hit.assists) || 0;
          const rate = Number(String(hit.hs || "").replace(/[^\d.]/g, ""));
          if (rate) { hs += rate; hsN += 1; }
        }
        return {
          id,
          name: (data && data.name) || (seed && seed.name) || "Player",
          tag: (data && data.tag) || (seed && seed.tag) || "",
          rank: (data && data.rank) || (seed && seed.rank) || "",
          games,
          wins,
          kills,
          deaths,
          assists,
          hs: hsN ? Math.round(hs / hsN) : 0,
        };
      }
      function rememberCompare(seed, data) {
        const row = playerSummary(seed, data);
        if (!row.id) return;
        comparePair = comparePair.filter((item) => item.id !== row.id);
        comparePair.push(row);
        if (comparePair.length > 2) comparePair = comparePair.slice(-2);
      }
      function compareBoard() {
        if (!comparePair.length) return null;
        const board = el("div", "compare-board");
        const paint = (row) => {
          const col = el("div", "compare-col");
          col.append(el("b", null, row.name + (row.tag ? " #" + row.tag : "")));
          const rate = row.games ? Math.round((row.wins / row.games) * 100) + "%" : "—";
          col.append(el("div", "sub", [row.rank, row.games + " games", rate + " wins"].filter(Boolean).join(" · ")));
          col.append(el("div", "sub", row.kills + "/" + row.deaths + "/" + row.assists + (row.hs ? " · " + row.hs + "% HS" : "")));
          board.append(col);
        };
        comparePair.forEach(paint);
        if (comparePair.length === 1) {
          const wait = el("div", "compare-col");
          wait.append(el("b", null, "Second player"));
          wait.append(el("div", "sub", "Open someone else and press Compare."));
          board.append(wait);
        }
        return board;
      }
      function renderProfile(seed, data) {
        const slot = document.getElementById("dossier");
        slot.replaceChildren();
        const hero = el("div", "hero");
        const img = face(seed && seed.agentIcon ? seed : (data && data.agentIcon ? data : seed));
        const id = el("div");
        const tools = el("div", "profile-tools");
        const watchBtn = el("button", "tiny solid", onWatchList(seed) ? "On watchlist" : "Add to watchlist");
        watchBtn.type = "button";
        watchBtn.onclick = async (event) => {
          event.preventDefault();
          event.stopPropagation();
          const tag = (data && data.tag) || seed.tag || "";
          const whoName = (data && data.name) || seed.name || "";
          if (!whoName || !tag) {
            watchBtn.textContent = "Needs a name#tag";
            return;
          }
          watchBtn.textContent = "Saving…";
          try {
            const book = await (await fetch("/api/social?q=" + encodeURIComponent(whoName + "#" + tag), { method: "POST" })).json();
            if (book && Array.isArray(book.watch)) watch = book.watch;
            watchBtn.textContent = "On watchlist";
          } catch {
            watchBtn.textContent = "Add to watchlist";
          }
        };
        const shared = el("button", "tiny", "Games together");
        shared.type = "button";
        shared.onclick = (event) => { event.preventDefault(); event.stopPropagation(); showShared(seed); };
        tools.append(watchBtn, shared);
        const riotID = (((data && data.name) || seed.name || "") + ((data && data.tag) || seed.tag ? "#" + ((data && data.tag) || seed.tag || "") : "")).replace(/#$/, "");
        if ((data && data.name && data.tag) || (seed.name && seed.tag)) {
          const copyID = el("button", "tiny", "Copy Riot ID");
          copyID.type = "button";
          copyID.onclick = (event) => {
            event.preventDefault();
            event.stopPropagation();
            const done = () => { copyID.textContent = "Copied"; setTimeout(() => { copyID.textContent = "Copy Riot ID"; }, 900); };
            if (navigator.clipboard && navigator.clipboard.writeText) navigator.clipboard.writeText(riotID).then(done).catch(done);
            else done();
          };
          tools.append(copyID);
        }
        const avoidBtn = el("button", "tiny", isAvoided(seed) ? "Avoiding" : "Avoid");
        avoidBtn.type = "button";
        avoidBtn.onclick = (event) => {
          event.preventDefault();
          event.stopPropagation();
          const whoName = ((data && data.name) || seed.name || "").trim();
          const tag = ((data && data.tag) || seed.tag || "").trim();
          const id = String((data && data.puuid) || seed.puuid || "").toLowerCase();
          if (!whoName) return;
          let rows = avoidList().filter((row) => !((id && String(row.puuid || "").toLowerCase() === id) || (String(row.name || "").toLowerCase() === whoName.toLowerCase() && String(row.tag || "").toLowerCase() === tag.toLowerCase())));
          if (avoidBtn.textContent !== "Avoiding") rows = [{ name: whoName, tag, puuid: id }].concat(rows).slice(0, 40);
          writeJSON("valvet-avoid", rows);
          avoidBtn.textContent = isAvoided({ name: whoName, tag, puuid: id }) ? "Avoiding" : "Avoid";
          if (latestState) paintAvoidLive(latestState);
        };
        tools.append(avoidBtn);
        const compareBtn = el("button", "tiny", "Compare");
        compareBtn.type = "button";
        compareBtn.onclick = (event) => {
          event.preventDefault();
          event.stopPropagation();
          rememberCompare(seed, data);
          compareBtn.textContent = "Added";
          setTimeout(() => { compareBtn.textContent = "Compare"; }, 800);
        };
        tools.append(compareBtn);
        id.append(tools);
        const mine = !!(seed && seed.you);
        const name = mine ? (seed.name || (data && data.name) || "You") : ((data && data.name) || seed.name || "Player");
        const tag = mine ? (seed.tag || (data && data.tag) || "") : ((data && data.tag) || seed.tag || "");
        id.append(el("h3", null, name));
        if (tag) id.append(el("div", "sub", "#" + tag));
        const incoming = (data && data.rank) || "";
        const seedRank = (seed && seed.rank) || "";
        const rank = (incoming && incoming !== "Unranked" ? incoming : "") || (seedRank && seedRank !== "Unranked" ? seedRank : "") || incoming || seedRank;
        const rr = (data && data.rr) || seed.rr;
        const peak = (data && data.peak) || (seed && seed.peak) || "";
        const peakOnly = !!(seed && seed.rankPeak) || (!rr && peak && (!rank || rank === peak));
        const rankIcon = (data && String(data.rankIcon || "").indexOf("/pack/rank/") === 0 && data.rankIcon) || (seed && String(seed.rankIcon || "").indexOf("/pack/rank/") === 0 && seed.rankIcon) || (data && data.rankIcon) || (seed && seed.rankIcon) || "";
        const paintRank = (text) => {
          if (!text && !rankIcon) return;
          const node = el("div", "rankline coin");
          if (rankIcon) {
            const badge = document.createElement("img");
            badge.className = "rankimg";
            badge.alt = "";
            badge.src = String(rankIcon).indexOf("/pack/") === 0 ? rankIcon : cachedArt(rankIcon);
            node.append(badge);
          }
          if (text) node.append(document.createTextNode(text));
          id.append(node);
        };
        if (peakOnly && (rank || peak)) {
          const line = el("div", "rankline coin");
          bindHoverTip(line, (tip) => {
            tip.append(el("div", "tip-kicker", "Peak"));
            tip.append(el("b", null, peak || rank || "Peak"));
            tip.append(el("div", "tip-line", "No active rank. This is their peak, not the current act rank."));
          });
          if (rankIcon) {
            const badge = document.createElement("img");
            badge.className = "rankimg";
            badge.alt = "";
            badge.src = String(rankIcon).indexOf("/pack/") === 0 ? rankIcon : cachedArt(rankIcon);
            line.append(badge);
          }
          line.append(document.createTextNode("Peak " + (peak || rank)));
          line.append(el("i", "peak-flag", "Not active"));
          id.append(line);
        } else {
          const line = [rank ? rank + (rr ? " · " + rr + " RR" : "") : data ? "" : "Loading rank…", peak ? "Peak " + peak : ""].filter(Boolean).join("  ·  ");
          paintRank(line);
        }
        const open = el("button", "tracker");
        open.type = "button";
        open.textContent = "Open on vtl.lol";
        open.onclick = () => openTracker(seed);
        id.append(open);
        hero.append(img, id);
        slot.append(hero);
        const scout = scoutCard(seed);
        if (scout) slot.append(scout);
        const board = compareBoard();
        if (board) slot.append(board);
        const was = (data && data.was) || seed.was || "";
        if (was) {
          const aka = el("div", "aka");
          aka.append(el("div", "kicker", "Old names"));
          for (const part of was.split(", ")) if (part) aka.append(el("div", null, part));
          slot.append(aka);
        }
        const apiGames = (data && data.matches) || [];
        const together = sharedGames(seed);
        const games = apiGames.length ? apiGames : together;
        const head = el("div", "kicker", data && data.cached ? "On this PC" : apiGames.length ? "Recent games" : together.length ? "Games you were in" : data ? "Recent games" : "Loading games…");
        slot.append(head);
        if (!games.length) {
          if (!data) return;
          slot.append(el("div", "sub", data.error ? data.error : "No games came back."));
          return;
        }
        for (const match of games) slot.append(profileRow(match));
      }
      async function openPlayer(player) {
        if (!player || !player.puuid) return;
        const phase = (latestState && latestState.phase) || "";
        const live = phase === "In game" || phase === "Loading lobby" || phase === "Agent select";
        document.getElementById("scrim").hidden = false;
        const local = { matches: sharedGames(player), cached: true };
        renderProfile(player, local);
        if (live) return;
        const job = fetch("/api/player?id=" + encodeURIComponent(player.puuid), { cache: "no-store" });
        try {
          renderProfile(player, await (await job).json());
        } catch {
          renderProfile(player, { error: "Couldn't load that history.", matches: sharedGames(player) });
        }
      }
