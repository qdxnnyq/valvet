      const rankSeen = new Map();
      function liveRank(player) {
        player = player || {};
        const peaked = !!player.rankPeak || ((!player.rank || player.rank === "Unranked") && !!player.peak);
        const name = peaked ? (player.peak || player.rank || "Unranked") : (player.rank || "Unranked");
        const icon = peaked ? (player.peakIcon || player.rankIcon) : player.rankIcon;
        const box = el("button", "live-rank coin" + (peaked ? " peaked" : ""));
        box.type = "button";
        const token = String(player.puuid || player.name || "") ;
        const coin = token + ":" + name + ":" + (player.rr || "");
        if (token && rankSeen.has(token) && rankSeen.get(token) !== coin) box.classList.add("flip");
        if (token) rankSeen.set(token, coin);
        if (icon) {
          const img = document.createElement("img");
          img.alt = "";
          img.src = icon;
          box.append(img);
        }
        const copy = el("div");
        const title = el("b");
        title.textContent = name;
        copy.append(title);
        if (peaked) copy.append(el("span", "peak-flag", "Peak"));
        else if (player.rr) copy.append(el("span", null, player.rr + " RR"));
        box.append(copy);
        box.onclick = (event) => {
          event.preventDefault();
          event.stopPropagation();
          openTracker(player);
        };
        return box;
      }
      function boardShape(players, mode) {
        const text = String(mode || "").toLowerCase();
        const grouped = groups(players);
        const counts = grouped.map((pair) => pair[1].length);
        if (/deathmatch/.test(text) || (grouped.length === 1 && (counts[0] || 0) >= 8)) return "ffa";
        if (/gauntlet|glitched/.test(text)) return "pairs";
        if (/skirmish/.test(text) || (grouped.length === 2 && counts.every((n) => n <= 2) && counts.reduce((sum, n) => sum + n, 0) <= 4)) return "duo";
        return "sides";
      }
      function applyLiveLayout(shape) {
        const pane = document.getElementById("pane-live");
        if (!pane) return;
        pane.classList.toggle("layout-ffa", shape === "ffa");
        pane.classList.toggle("layout-duo", shape === "duo");
        pane.classList.toggle("layout-pairs", shape === "pairs");
      }
      function partyColors(players) {
        const order = [];
        const you = (players || []).find((player) => player && player.you && player.party);
        if (you) order.push(you.party);
        for (const player of players || []) if (player && player.party && !order.includes(player.party)) order.push(player.party);
        const palette = "ABCDEFGH";
        const map = {};
        order.forEach((id, index) => { map[id] = palette[index] || "H"; });
        return map;
      }
      function meanTier(list) {
        const tiers = (list || []).map((player) => Number(player.tier) || 0).filter((n) => n >= 3);
        if (!list || !list.length || tiers.length < 2 || tiers.length * 2 < list.length) return 0;
        return tiers.reduce((sum, n) => sum + n, 0) / tiers.length;
      }
      function expectedWin(players) {
        const grouped = groups(players);
        if (grouped.length < 2) return null;
        let yours = null;
        let other = null;
        for (const pair of grouped) {
          if (pair[1].some((player) => player.you)) yours = pair[1];
          else if (!other) other = pair[1];
        }
        if (!yours || !other) return null;
        const a = meanTier(yours);
        const b = meanTier(other);
        if (!a || !b) return null;
        const chance = 1 / (1 + Math.exp(-(a - b) * 0.22));
        return Math.max(1, Math.min(99, Math.round(chance * 100)));
      }
      function roster(players, mode) {
        const shape = boardShape(players, mode);
        applyLiveLayout(shape);
        const grid = el("div", "grid");
        const grouped = groups(players);
        grouped.sort((a, b) => Number(b[1].some((player) => player.you)) - Number(a[1].some((player) => player.you)));
        grouped.forEach(([team, list], index) => {
          const yours = list.some((player) => player.you);
          const enemy = list.some((player) => player.enemy) && !yours;
          const fighting = list.some((player) => player.fight);
          const card = el("section", "team" + (yours ? " ally" : enemy ? " enemy" : "") + (fighting ? " now" : ""));
          const head = el("div", "team-head");
          const left = el("div");
          const title = shape === "duo" ? (yours ? "Your duo" : "Their duo") : shape === "pairs" ? (yours ? "Your duo" : fighting ? "Fighting now" : "Duo " + index) : team;
          left.append(el("div", "eyebrow", shape === "ffa" ? "Everyone" : yours ? "Your side" : enemy ? "Enemy" : "Players"));
          left.append(el("div", "team-name", title + " · " + list.length));
          const avg = averageRank(list);
          const avgBox = el("div", "team-avg");
          avgBox.append(el("span", null, "Average"));
          avgBox.append(el("b", null, avg || "—"));
          head.append(left, avgBox);
          card.append(head);
          const host = shape === "ffa" ? el("div", "ffa-grid") : card;
          if (shape === "ffa") card.append(host);
          const colors = partyColors(players);
          for (const player of list) {
            const tone = colors[player.party] || "";
            const row = el("div", "row" + (playerTracked(player) ? " tracked" : "") + (tone ? " p" + tone : ""));
            const img = face(player);
            img.className = "agent";
            const id = el("div", "live-id");
            const top = el("div", "live-sub");
            top.append(nameButton(player));
            id.append(top);
            const agent = player.agent && player.agent !== "Picking" ? player.agent : "";
            if (agent) id.append(el("div", "live-agent", agent));
            const partyCell = el("div", "live-party");
            if (tone) partyCell.append(el("span", "squad s" + tone, "Party " + tone));
            const side = el("div", "live-side");
            side.append(liveRank(player));
            row.append(img, id, partyCell, side);
            host.append(row);
          }
          grid.append(card);
        });
        const chance = expectedWin(players);
        if (chance != null) {
          const banner = el("div", "expect");
          banner.append(el("b", null, chance + "%"));
          banner.append(el("span", null, "expected"));
          grid.prepend(banner);
        }
        return grid;
      }
      function playedCount(puuid) {
        const id = String(puuid || "").toLowerCase();
        if (!id) return 0;
        let n = 0;
        for (const match of history || []) {
          if ((match.players || []).some((player) => String(player.puuid || "").toLowerCase() === id)) n += 1;
        }
        return n;
      }
      function tally(score1, score2, team1, team2, yours) {
        const wrap = el("div", "line");
        const n1 = score1 == null ? null : Number(score1);
        const n2 = score2 == null ? null : Number(score2);
        const ok = n1 != null && n2 != null && !Number.isNaN(n1) && !Number.isNaN(n2);
        const add = (n, team, lead, you) => {
          wrap.append(el("b", lead ? "hi" : "", score(n)));
          const label = sideLabel(team);
          if (label) wrap.append(el("span", "lbl", label));
        };
        add(n1, team1, ok && n1 > n2, yours === team1);
        wrap.append(el("span", "dash", "–"));
        add(n2, team2, ok && n2 > n1, yours === team2);
        return wrap;
      }
      function kdaValue(player) {
        const kills = Number(player && player.kills) || 0;
        const deaths = Number(player && player.deaths) || 0;
        const assists = Number(player && player.assists) || 0;
        return ((kills + assists) / Math.max(1, deaths)) * 1000 + kills * 3 + assists;
      }
      function byKDA(list) {
        return [...(list || [])].sort((a, b) => kdaValue(b) - kdaValue(a) || String(a.puuid || a.name || "").localeCompare(String(b.puuid || b.name || "")));
      }
      function markAwards(match) {
        const players = (match && match.players) || [];
        const s1 = Number(match && match.score1);
        const s2 = Number(match && match.score2);
        let win = "";
        let lose = "";
        if (Number.isFinite(s1) && Number.isFinite(s2) && s1 !== s2) {
          win = s1 > s2 ? match.team1 : match.team2;
          lose = s1 > s2 ? match.team2 : match.team1;
        } else if (match && (match.won === true || match.won === false)) {
          const you = players.find((player) => player.you);
          const yours = (you && you.team) || "";
          const other = ((players.find((player) => player.team && player.team !== yours) || {}).team) || "";
          win = match.won ? yours : other;
          lose = match.won ? other : yours;
        }
        const best = (team) => byKDA(players.filter((player) => !team || player.team === team))[0];
        const mvp = best(win);
        const svp = best(lose);
        const mvpKey = mvp ? whoKey(mvp) : "";
        const svpKey = svp ? whoKey(svp) : "";
        for (const player of players) {
          const key = whoKey(player);
          player.mvp = !!mvpKey && key === mvpKey;
          player.svp = !!svpKey && key === svpKey && !player.mvp;
        }
      }
      const nameMemory = new Map();
      function keepNames(rows) {
        return (rows || []).map((match) => {
          if (!match || typeof match !== "object") return match;
          const players = (match.players || []).filter(Boolean).map((player) => {
            const key = (match.id || "") + "|" + String(player.puuid || "").toLowerCase();
            const name = String(player.name || "").trim();
            if (name && name !== "Player" && name !== "Hidden") {
              nameMemory.set(key, { name: player.name, tag: player.tag || "" });
              return player;
            }
            const saved = nameMemory.get(key);
            return saved ? { ...player, name: saved.name, tag: player.tag || saved.tag } : player;
          });
          return { ...match, players };
        });
      }
      function rankFace(icon, label) {
        const face = el("span", "face");
        if (icon) {
          const badge = document.createElement("img");
          badge.className = "rankimg";
          badge.alt = "";
          badge.src = cachedArt(icon);
          face.append(badge);
        }
        face.append(el("b", null, label || "—"));
        return face;
      }
      function rankCoin(player) {
        const rank = el("button", "rank coin" + (player.rankPeak ? " peak" : ""));
        rank.type = "button";
        bindHoverTip(rank, (tip) => {
          tip.append(el("div", "tip-kicker", player.rankPeak ? "Peak" : "Rank"));
          tip.append(el("b", null, rankText(player)));
          const line = player.rankPeak ? "No active rank. This is the peak." : (player.peak && player.peak !== player.rank ? "Click to flip the peak." : "");
          if (line) tip.append(el("div", "tip-line", line));
        });
        const inner = el("span", "coin-inner");
        const front = rankFace(player.rankIcon, player.rankPeak ? "Peak" : (player.rank || "—"));
        const back = rankFace(player.peakIcon || player.rankIcon, player.peak || (player.rankPeak ? player.rank : "Peak"));
        back.classList.add("back");
        inner.append(front, back);
        rank.append(inner);
        return rank;
      }
      function rankText(player) {
        if (!player || !player.rank) return "—";
        const name = player.rankPeak ? "Peak " + player.rank : player.rank;
        if (player.rr && !player.rankPeak) return name + "  " + player.rr + " RR";
        return name;
      }
      function shortRank(name) {
        const text = String(name || "").trim();
        if (!text || text === "Unranked") return text;
        const parts = text.split(/\s+/);
        const word = { Iron: "Iron", Bronze: "Bronze", Silver: "Silver", Gold: "Gold", Platinum: "Plat", Diamond: "Diamond", Ascendant: "Asc", Immortal: "Imm", Radiant: "Radiant" };
        const head = word[parts[0]] || parts[0];
        return parts[1] ? head + " " + parts[1] : head;
      }
      function rankMark(player) {
        if (!player) return null;
        const peakName = player.peak || (player.rankPeak ? player.rank : "");
        const currentName = player.rankPeak ? "" : (player.rank && player.rank !== "Unranked" ? player.rank : "");
        const currentIcon = player.rankPeak ? "" : (player.rankIcon || "");
        const peakIcon = player.peakIcon || (player.rankPeak ? player.rankIcon : "");
        const same = peakName && currentName && peakName === currentName;
        if (!currentName && !currentIcon && !peakName && !peakIcon) {
          const box = el("span", "ranks");
          box.append(el("span", "hrank", "Unranked"));
          return box;
        }
        const box = el("span", "ranks");
        const pill = (icon, text, peak) => {
          const wrap = el("span", "hrank" + (peak ? " is-peak" : ""));
          if (icon) {
            const img = document.createElement("img");
            img.className = "rankimg";
            img.alt = "";
            img.src = String(icon).indexOf("/pack/") === 0 ? icon : cachedArt(icon);
            wrap.append(img);
          }
          if (text) wrap.append(document.createTextNode(shortRank(text)));
          return wrap;
        };
        if (currentName || currentIcon) box.append(pill(currentIcon, currentName, false));
        else box.append(pill("", "Unranked", false));
        if (peakName && !same) {
          const node = pill(peakIcon, peakName, true);
          bindHoverTip(node, (tip) => {
            tip.append(el("div", "tip-kicker", "Peak"));
            tip.append(el("b", null, peakName));
          });
          box.append(node);
        }
        return box.childNodes.length ? box : null;
      }
      function chipRow(player, colors) {
        const chips = el("span", "chips");
        if (player && player.mvp) chips.append(el("span", "badge mvp", "MVP"));
        else if (player && player.svp) chips.append(el("span", "badge svp", "SVP"));
        const rank = rankMark(player);
        if (rank) chips.append(rank);
        if (player && player.party && colors && colors[player.party]) {
          const mark = colors[player.party];
          chips.append(el("span", "squad s" + mark, "Party " + mark));
        }
        return chips;
      }
      function sideList(players, team, tone, colors, titleText, sortFn) {
        const raw = Array.isArray(team) ? [...team] : (players || []).filter((player) => player.team === team);
        const list = (sortFn || byKDA)(raw);
        const label = Array.isArray(team) ? "" : sideLabel(team);
        if (!list.length) return null;
        const block = el("div", "sidecol " + (tone || ""));
        const avg = averageRank(list);
        const title = titleText != null ? titleText : ((label || (tone === "ally" ? "Your side" : tone === "enemy" ? "Enemy" : "Players")) + (avg ? " · avg " + avg : ""));
        if (title) block.append(el("div", "kicker", title));
        for (const player of list) {
          const button = el("button", "person" + (player.you ? " me" : ""));
          button.type = "button";
          button.dataset.who = whoKey(player);
          if (player.agentIcon) button.append(face(player));
          button.append(el("span", "nm", shownName(player)));
          button.append(chipRow(player, colors));
          button.append(el("b", "kda", (player.kills || "0") + "/" + (player.deaths || "0") + "/" + (player.assists || "0")));
          button.onclick = (event) => { event.preventDefault(); event.stopPropagation(); openPlayer(player); };
          block.append(button);
        }
        return block;
      }
      function statusClass(state) {
        if (String(state || "").toLowerCase() === "offline") return "offline";
        if (state === "In game") return "ingame";
        if (state === "In queue") return "queue";
        if (state === "Agent select") return "select";
        if (state === "AFK" || state === "Away") return "afk";
        return "lobby";
      }
      function company(person) {
        const n = Number(person.partySize || 0);
        if (!n) return "";
        const max = Number(person.maxParty || 0) || 5;
        return n + "/" + max;
      }
      function brandArt(kind) {
        const img = document.createElement("img");
        img.className = "art";
        img.alt = kind === "league" ? "League" : "Riot";
        img.src = "/brand/" + (kind === "league" ? "league" : "riot");
        img.onerror = () => img.remove();
        return img;
      }
      function personCard(person) {
        const offline = String(person.state || "").toLowerCase() === "offline";
        const card = el("button", "pcard" + (person.you ? " you" : "") + (offline ? " offline" : ""));
        card.type = "button";
        if (offline || person.product === "league" || person.product === "riot") {
          card.append(brandArt(offline ? "riot" : person.product));
        } else if (person.card && String(person.card).startsWith("http")) {
          const img = document.createElement("img");
          img.className = "art";
          img.alt = "";
          img.src = cachedArt(person.card);
          img.onerror = () => {
            if (img.dataset.fallback) return;
            img.dataset.fallback = "1";
          img.src = cachedArt(String(person.card).replace("/wideart.png", "/smallart.png"));
          };
          card.append(img);
        } else {
          card.append(el("div", "art"));
        }
        const body = el("div", "body");
        const row = el("div", "who-row");
        const name = el("b");
        name.append(document.createTextNode(person.name || "Player"));
        if (person.tag) name.title = person.name + "#" + person.tag;
        row.append(name);
        body.append(row);
        const withWho = company(person);
        const modeName = String(person.mode || "").trim();
        const status = el("div", "status-line");
        status.append(el("span", "pill " + statusClass(person.state), person.state || "Online"));
        body.append(status);
        const bits = el("div", "live-bits");
        if (modeName && modeName.toLowerCase() !== "match") bits.append(el("span", null, modeName));
        if (person.level) bits.append(el("span", null, "Lv " + person.level));
        if (withWho) bits.append(el("span", null, withWho));
        if (bits.childNodes.length) body.append(bits);
        const note = friendNote(person.puuid);
        if (note) body.append(el("div", "friend-note", note));
        const extra = [person.map, person.score, person.rank].filter(Boolean);
        const together = person.you ? "" : lastTogether(person.puuid, history);
        if (together) extra.push(together);
        if (extra.length) body.append(el("div", "subline", extra.join(" · ")));
        card.append(body);
        card.onclick = () => openPlayer(person);
        const wrap = el("div", "pcard-wrap");
        wrap.append(card);
        if (person.puuid && !person.you) {
          const pin = el("button", "note-pin" + (note ? " on" : ""), "Note");
          pin.type = "button";
          pin.setAttribute("aria-label", note ? "Edit note" : "Add note");
          pin.onclick = (event) => {
            event.preventDefault();
            event.stopPropagation();
            openFriendNote(person);
          };
          wrap.append(pin);
        }
        return wrap;
      }
      function closeFriendNote() {
        const pop = document.getElementById("note-pop");
        if (pop) pop.hidden = true;
      }
      function openFriendNote(person) {
        const pop = document.getElementById("note-pop");
        const field = document.getElementById("note-text");
        const title = document.getElementById("note-title");
        if (!pop || !field || !person || !person.puuid) return;
        if (!pop.dataset.bound) {
          pop.dataset.bound = "1";
          pop.addEventListener("click", (event) => { if (event.target === pop) closeFriendNote(); });
          document.addEventListener("keydown", (event) => {
            if (event.key === "Escape" && !pop.hidden) {
              event.preventDefault();
              closeFriendNote();
            }
          });
          const commit = (text) => {
            saveFriendNote(pop.dataset.puuid || "", text);
            closeFriendNote();
            partyKey = "";
            if (lastPeople) drawPeople(lastPeople);
          };
          const saveBtn = document.getElementById("note-save");
          const clearBtn = document.getElementById("note-clear");
          const cancelBtn = document.getElementById("note-cancel");
          if (saveBtn) saveBtn.onclick = () => commit((field.value || "").trim());
          if (clearBtn) clearBtn.onclick = () => { field.value = ""; commit(""); };
          if (cancelBtn) cancelBtn.onclick = () => closeFriendNote();
        }
        pop.dataset.puuid = person.puuid;
        if (title) title.textContent = person.name || "Friend";
        field.value = friendNote(person.puuid);
        pop.hidden = false;
        field.focus();
      }
      function paintSocial(partyId, friendsId, partyTitle, friendsTitle, listParty, listFriends) {
        const party = document.getElementById(partyId);
        const friends = document.getElementById(friendsId);
        if (!party || !friends) return;
        party.replaceChildren();
        friends.replaceChildren();
        const title = document.getElementById(partyTitle);
        if (title) title.textContent = listParty.length ? "Your lobby" : "Lobby";
        if (!listParty.length) party.append(el("div", "empty", "Open the Riot Client. Your lobby shows up here even if Valorant is closed."));
        for (const person of listParty) party.append(personCard(person));
        const friendsHead = document.getElementById(friendsTitle);
        const band = friends.closest(".friend-band");
        if (!listFriends.length) {
          if (band) band.hidden = true;
          return;
        }
        if (band) band.hidden = false;
        if (friendsHead) friendsHead.textContent = "Valorant · " + listFriends.length;
        const grid = el("div", "friends");
        for (const person of listFriends) grid.append(personCard(person));
        friends.append(grid);
      }
      function isLeague(person) {
        return String((person && person.product) || "").toLowerCase() === "league";
      }
      function isRiot(person) {
        return String((person && person.product) || "").toLowerCase() === "riot";
      }
      function paintGroup(rootId, titleId, label, list) {
        const root = document.getElementById(rootId);
        const title = document.getElementById(titleId);
        if (!root || !title) return;
        const card = root.closest(".desk-card");
        root.replaceChildren();
        if (!list.length) {
          if (card) card.hidden = true;
          title.textContent = label;
          return;
        }
        if (card) card.hidden = false;
        title.textContent = label + " · " + list.length;
        const grid = el("div", "friends");
        for (const person of list) grid.append(personCard(person));
        root.append(grid);
      }
      function paintLeague(list) {
        paintGroup("league", "league-title", "League", list);
      }
      function paintLiveLeague(list) {
        const root = document.getElementById("live-league");
        const title = document.getElementById("live-league-title");
        const band = document.getElementById("live-league-band");
        if (!root) return;
        root.replaceChildren();
        if (!list.length) {
          if (band) band.hidden = true;
          return;
        }
        if (band) band.hidden = false;
        if (title) title.textContent = "League · " + list.length;
        const grid = el("div", "friends");
        for (const person of list) grid.append(personCard(person));
        root.append(grid);
      }
      function drawPeople(data) {
        lastPeople = data;
        const partySrc = data.party || [];
        const allFriends = data.friends || [];
        const offlineSrc = allFriends.filter((person) => String(person.state || "").toLowerCase() === "offline");
        const friendsSrc = allFriends.filter((person) => String(person.state || "").toLowerCase() !== "offline");
        if (knownLevel) {
          for (const person of partySrc) if (person.you && !person.level) person.level = knownLevel;
        }
        const friendQuery = document.getElementById("friend-query");
        if (friendQuery && !friendQuery.dataset.bound) {
          friendQuery.dataset.bound = "1";
          friendQuery.addEventListener("input", () => { partyKey = ""; if (lastPeople) drawPeople(lastPeople); });
        }
        const q = ((friendQuery && friendQuery.value) || "").trim().toLowerCase();
        const hit = (person) => !q || ((person.name || "") + " #" + (person.tag || "") + " " + (person.rank || "")).toLowerCase().includes(q);
        const together = [...partySrc, ...friendsSrc, ...offlineSrc].map((person) => (person.puuid || "") + ":" + lastTogether(person.puuid, history)).join("|");
        const noteSig = [...partySrc, ...friendsSrc, ...offlineSrc].map((person) => friendNote(person.puuid)).join("\n");
        const key = JSON.stringify([partySrc, friendsSrc, offlineSrc, knownLevel, together, showOfflineFriends, noteSig, q]);
        stampSeen([].concat(partySrc, allFriends), data && data.ok !== false);
        if (key === partyKey) return;
        partyKey = key;
        const party = partySrc.filter((person) => !isLeague(person));
        const online = (data.friends || []).filter((person) => !isLeague(person) && !isRiot(person));
        const mates = party.filter((person) => !person.you);
        const valorantLive = friendsSrc.filter((person) => !isLeague(person) && !isRiot(person));
        const leagueLive = friendsSrc.filter(isLeague);
        const valorant = [];
        const seenFriend = new Set();
        for (const person of valorantLive.concat(mates).concat(online)) {
          const id = String(person.puuid || person.name || "").toLowerCase();
          if (!id || seenFriend.has(id)) continue;
          seenFriend.add(id);
          valorant.push(person);
        }
        const league = leagueLive.filter(hit);
        const riot = friendsSrc.filter((person) => isRiot(person) && !seenFriend.has(String(person.puuid || person.name || "").toLowerCase())).filter(hit);
        const valorantShown = valorant.filter(hit);
        const offlineShown = offlineSrc.filter(hit);
        paintSocial("party", "friends", "party-title", "friends-title", party, valorantLive);
        const friends2 = document.getElementById("friends-2");
        const valorantHead = document.getElementById("friends-title-2");
        const valorantCard = friends2 ? friends2.closest(".desk-card") : null;
        const leagueCard = document.getElementById("league") ? document.getElementById("league").closest(".desk-card") : null;
        if (friends2) {
          friends2.replaceChildren();
          if (valorantHead) valorantHead.textContent = valorantShown.length ? "Valorant · " + valorantShown.length : "Valorant";
          if (valorantShown.length) {
            const grid = el("div", "friends");
            for (const person of valorantShown) grid.append(personCard(person));
            friends2.append(grid);
          } else if (q) friends2.append(el("div", "empty", "No friends match that."));
        }
        if (valorantCard) valorantCard.hidden = !valorantShown.length && !q;
        paintLeague(league);
        if (leagueCard) leagueCard.hidden = !league.length;
        paintLiveLeague(leagueLive);
        paintGroup("riot", "riot-title", "Riot", riot.filter((person) => !isRiotMobile(person)));
        paintGroup("mobile", "mobile-title", "Riot Mobile", riot.filter((person) => isRiotMobile(person)));
        const offlineCard = document.getElementById("offline-card");
        if (showOfflineFriends) {
          for (const card of document.querySelectorAll("#pane-friends .desk-card")) {
            if (card === valorantCard || card === leagueCard) continue;
            card.hidden = card.id !== "offline-card";
          }
          if (valorantCard) valorantCard.hidden = !valorant.length;
          if (leagueCard) leagueCard.hidden = !league.length;
          if (offlineCard) {
            offlineCard.hidden = false;
            const title = document.getElementById("offline-title");
            const root = document.getElementById("offline-friends");
            if (title) title.textContent = offlineShown.length ? "Offline · " + offlineShown.length : "Offline";
            if (root) {
              root.replaceChildren();
              if (!offlineShown.length) {
                offlineCard.hidden = true;
              } else {
                const grid = el("div", "friends");
                for (const person of offlineShown) grid.append(personCard(person));
                root.append(grid);
              }
            }
          }
        } else if (offlineCard) offlineCard.hidden = true;
        const onlineBtn = document.getElementById("friends-online");
        const offlineBtn = document.getElementById("friends-offline");
        if (onlineBtn && offlineBtn && !onlineBtn.dataset.bound) {
          onlineBtn.dataset.bound = "1";
          onlineBtn.onclick = () => {
            showOfflineFriends = false;
            localStorage.setItem("valvet-friends-offline", "off");
            partyKey = "";
            if (lastPeople) drawPeople(lastPeople);
          };
          offlineBtn.onclick = () => {
            showOfflineFriends = true;
            localStorage.setItem("valvet-friends-offline", "on");
            partyKey = "";
            if (lastPeople) drawPeople(lastPeople);
          };
        }
        if (onlineBtn) onlineBtn.classList.toggle("solid", !showOfflineFriends);
        if (offlineBtn) offlineBtn.classList.toggle("solid", showOfflineFriends);
        pingCareQueue([...party, ...online, ...league]);
      }
      function drawStore(shop) {
        nightEnds = Number(shop && shop.nightEnds) || 0;
        nightOpen = ((shop && shop.night) || []).length > 0;
        const key = JSON.stringify(shop || {}) + "|" + ownedNames.join("|");
        if (key === storeKey) {
          paintNightClock();
          return;
        }
        storeKey = key;
        const offerSig = ((shop && shop.offers) || []).map((item) => item.id || item.name).filter(Boolean).join("|");
        markFresh("store", offerSig);
        const seenStore = readJSON("valvet-store-seen") || { sig: "", names: [] };
        const offerNames = ((shop && shop.offers) || []).map((item) => String(item.name || "").toLowerCase()).filter(Boolean);
        const freshNames = new Set();
        if (seenStore.sig && seenStore.sig !== offerSig) {
          const old = new Set(seenStore.names || []);
          for (const name of offerNames) if (!old.has(name)) freshNames.add(name);
        }
        if (seenStore.sig !== offerSig) writeJSON("valvet-store-seen", { sig: offerSig, names: offerNames });
        const root = document.getElementById("store");
        root.replaceChildren();
        const offers = ((shop && shop.offers) || []).filter((item) => !(hideOwned && owns(item.name)));
        const hero = el("div", "shop-hero");
        const copy = el("div", "copy");
        copy.append(el("div", "eyebrow", "Daily store"));
        copy.append(el("h1", null, "Featured"));
        const wallet = el("div", "wallet");
        if (shop && shop.vp) wallet.append(el("span", null, shop.vp + " VP"));
        if (shop && shop.radianite) wallet.append(el("span", null, shop.radianite + " Radianite"));
        if (shop && shop.kc) wallet.append(el("span", null, shop.kc + " KC"));
        if (shop && shop.resets) wallet.append(el("span", null, "Resets " + shop.resets));
        copy.append(wallet);
        const hits = wishlistHits(shop);
        if (hits.length) copy.append(el("p", "wish-note", "Wishlist in the store · " + hits.join(", ")));
        const ownedBtn = el("button", "tiny", hideOwned ? "Show owned" : "Hide owned");
        ownedBtn.type = "button";
        ownedBtn.onclick = () => {
          hideOwned = !hideOwned;
          localStorage.setItem("valvet-hide-owned", hideOwned ? "on" : "off");
          storeKey = "";
          const wishRoot = document.getElementById("wish-list");
          if (wishRoot) wishRoot.dataset.key = "";
          if (lastShop) drawStore(lastShop);
        };
        copy.append(ownedBtn);
        paintWishList(shop);
        hero.append(copy);
        root.append(hero);
        const pinned = offers.filter((item) => wished(item.name));
        const rest = offers.filter((item) => !wished(item.name));
        if (pinned.length) {
          root.append(el("h2", null, "Wishlist"));
          const pinGrid = el("div", "shop");
          for (const item of pinned) pinGrid.append(skinCard({ ...item, fresh: freshNames.has(String(item.name || "").toLowerCase()) }));
          root.append(pinGrid);
        }
        const grid = el("div", "shop");
        if (!offers.length) grid.append(el("div", "empty", (shop && shop.error) || "Store loads once Valorant is open."));
        for (const item of rest) grid.append(skinCard({ ...item, fresh: freshNames.has(String(item.name || "").toLowerCase()) }));
        root.append(grid);
        const kingdom = ((shop && shop.kingdom) || []).filter((item) => !(hideOwned && owns(item.name)));
        root.append(el("h2", null, ["Kingdom credits", shop && shop.kc ? shop.kc + " KC" : "", shop && shop.kingdomLeft ? shop.kingdomLeft + " left" : ""].filter(Boolean).join(" · ")));
        const kingdomGrid = el("div", "shop");
        if (!kingdom.length) kingdomGrid.append(el("div", "empty", "No Kingdom Credit offers right now."));
        for (const item of kingdom) kingdomGrid.append(skinCard(item));
        root.append(kingdomGrid);
        const night = ((shop && shop.night) || []).filter((item) => !(hideOwned && owns(item.name)));
        const nightHead = el("div", "night-head");
        nightHead.append(el("h2", null, "Night market"));
        const clock = el("div", "night-clock", "");
        clock.id = "night-clock";
        nightHead.append(clock);
        root.append(nightHead);
        paintNightClock();
        if (night.length) {
          const nightGrid = el("div", "shop");
          for (const item of night) nightGrid.append(skinCard(item));
          root.append(nightGrid);
        }
      }
      function ownedKey(name) {
        return String(name || "").trim().toLowerCase().replace(/ level \d+$/, "");
      }
      function localOwned() {
        try {
          const rows = JSON.parse(localStorage.getItem("valvet-bought") || "[]");
          return Array.isArray(rows) ? rows : [];
        } catch { return []; }
      }
      function rememberOwned(name) {
        const key = ownedKey(name);
        if (!key) return;
        const rows = localOwned().filter((item) => ownedKey(item) !== key);
        rows.push(String(name).trim());
        localStorage.setItem("valvet-bought", JSON.stringify(rows.slice(-200)));
      }
      function owns(name) {
        const key = ownedKey(name);
        if (!key) return false;
        return ownedNames.concat(localOwned()).some((item) => ownedKey(item) === key);
      }
      function wished(name) {
        const key = String(name || "").trim().toLowerCase();
        if (!key) return false;
        return loadWish().some((item) => {
          const want = item.toLowerCase();
          return want === key || key.startsWith(want + " level");
        });
      }
      function loadWish() {
        try { return JSON.parse(localStorage.getItem("valvet-wish") || "[]"); } catch { return []; }
      }
      function toggleWish(name) {
        const clean = String(name || "").trim();
        if (!clean) return;
        const next = loadWish().filter((item) => item.toLowerCase() !== clean.toLowerCase());
        if (next.length === loadWish().length) next.push(clean);
        localStorage.setItem("valvet-wish", JSON.stringify(next.slice(0, 24)));
        localStorage.removeItem("valvet-wish-sig");
      }
      function wishMatches(shop) {
        const want = new Set(loadWish().map((item) => item.toLowerCase()));
        const hits = [];
        for (const item of [...((shop && shop.offers) || []), ...((shop && shop.night) || [])]) {
          if (item.name && want.has(String(item.name).toLowerCase()) && !hits.includes(item.name)) hits.push(item.name);
        }
        return hits;
      }
      function wishlistHits(shop) {
        const hits = wishMatches(shop);
        const sig = hits.slice().sort().join("|") + "|" + ((shop && shop.resets) || "");
        if (hits.length && localStorage.getItem("valvet-wish-sig") !== sig) {
          localStorage.setItem("valvet-wish-sig", sig);
          playWish();
          if (localStorage.getItem("valvet-wish-note") === "on") {
            fetch("/api/notify", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ title: "Valvet", text: hits.slice(0, 3).join(", ") + (hits.length > 1 ? " are in the store" : " is in the store") }),
            }).catch(() => {});
          }
        }
        return hits;
      }
      let kcItems = [];
      let wishAll = false;
      let hideOwned = localStorage.getItem("valvet-hide-owned") === "on";
      function wishPool(shop) {
        const map = new Map();
        const add = (item, tag) => {
          const name = item && item.name;
          if (!name || name === "Kingdom offer") return;
          const key = name.toLowerCase();
          const prev = map.get(key);
          if (!prev) map.set(key, { name, icon: item.icon || "", tag });
          else if (!prev.icon && item.icon) prev.icon = item.icon;
        };
        for (const item of (shop && shop.kingdom) || []) add(item, "In KC store");
        for (const item of (shop && shop.offers) || []) add(item, "In daily store");
        for (const item of (shop && shop.night) || []) add(item, "In night market");
        for (const item of kcItems) add(item, item.currency || "Kingdom");
        return [...map.values()];
      }
      function paintWishList(shop) {
        const root = document.getElementById("wish-list");
        const tools = document.getElementById("wish-tools");
        if (!root) return;
        const q = (document.getElementById("wish-q") && document.getElementById("wish-q").value || "").trim().toLowerCase();
        const hits = new Set(wishMatches(shop).map((name) => name.toLowerCase()));
        let rows = wishPool(shop).filter((item) => !q || item.name.toLowerCase().includes(q));
        if (hideOwned) rows = rows.filter((item) => !owns(item.name));
        rows.sort((a, b) => {
          const aw = wished(a.name) ? 0 : 1;
          const bw = wished(b.name) ? 0 : 1;
          if (aw !== bw) return aw - bw;
          const ah = hits.has(a.name.toLowerCase()) ? 0 : 1;
          const bh = hits.has(b.name.toLowerCase()) ? 0 : 1;
          if (ah !== bh) return ah - bh;
          return a.name.localeCompare(b.name);
        });
        const total = rows.length;
        if (!q && !wishAll) rows = rows.slice(0, 80);
        const key = q + "|" + wishAll + "|" + (hideOwned ? "hide" : "show") + "|" + ownedNames.length + "|" + rows.map((item) => item.name + (wished(item.name) ? "*" : "") + (owns(item.name) ? "#" : "")).join("|") + "|" + [...hits].sort().join("|");
        if (root.dataset.key === key) return;
        root.dataset.key = key;
        root.replaceChildren();
        if (tools) {
          tools.replaceChildren();
          const ownedBtn = el("button", "tiny", hideOwned ? "Show owned" : "Hide owned");
          ownedBtn.type = "button";
          ownedBtn.onclick = () => {
            hideOwned = !hideOwned;
            localStorage.setItem("valvet-hide-owned", hideOwned ? "on" : "off");
            root.dataset.key = "";
            storeKey = "";
            if (lastShop) drawStore(lastShop);
            else paintWishList(null);
          };
          tools.append(ownedBtn);
          if (!q && total > 80) {
            const more = el("button", "tiny", wishAll ? "Show fewer" : "Show all " + total);
            more.type = "button";
            more.onclick = () => { wishAll = !wishAll; root.dataset.key = ""; paintWishList(lastShop); };
            tools.append(more);
          }
        }
        if (!rows.length) {
          root.append(el("div", "empty", kcItems.length ? "Nothing matches that search." : "Open Valorant once so the Kingdom list can load."));
          return;
        }
        for (const item of rows) {
          const on = wished(item.name);
          const live = hits.has(item.name.toLowerCase());
          const mine = owns(item.name);
          const card = el("article", "wish-pick" + (on ? " on" : "") + (live ? " hot" : "") + (mine ? " owned" : ""));
          if (item.icon) {
            const img = document.createElement("img");
            img.alt = "";
            img.loading = "lazy";
            img.src = item.icon;
            card.append(img);
          } else card.append(el("div", "art"));
          card.append(el("b", null, item.name));
          card.append(el("span", null, mine ? "Owned" : live ? "In the store now" : on ? "Wished" : item.tag));
          const pin = el("button", "tiny pin", on ? "Wished" : "Wish");
          pin.type = "button";
          pin.onclick = (event) => {
            event.preventDefault();
            event.stopPropagation();
            toggleWish(item.name);
            root.dataset.key = "";
            storeKey = "";
            if (lastShop) drawStore(lastShop);
            else paintWishList(null);
          };
          card.append(pin);
          card.onclick = () => openZoom({ icon: item.icon, name: item.name, wishName: item.name });
          root.append(card);
        }
      }
      function openZoom(item) {
        const src = item && (item.anim || item.icon);
        if (!item || (!src && !item.name)) return;
        const img = document.getElementById("zoom-img");
        const vid = document.getElementById("zoom-vid");
        const video = src && (String(src).includes(".mp4") || String(src).includes("video"));
        if (video) {
          img.hidden = true;
          vid.hidden = false;
          vid.src = src;
          vid.play().catch(() => {});
        } else {
          vid.pause();
          vid.hidden = true;
          vid.removeAttribute("src");
          img.hidden = !src;
          if (src) img.src = src;
        }
        document.getElementById("zoom-name").textContent = [item.name, item.price ? item.price + (item.currency === "KC" ? " KC" : " VP") : ""].filter(Boolean).join(" · ");
        const wish = document.getElementById("zoom-wish");
        if (wish) {
          if (item.wishName) {
            wish.hidden = false;
            const paint = () => { wish.textContent = wished(item.wishName) ? "Remove wish" : "Add to wishlist"; };
            paint();
            wish.onclick = (event) => {
              event.stopPropagation();
              toggleWish(item.wishName);
              paint();
              const root = document.getElementById("wish-list");
              if (root) root.dataset.key = "";
              storeKey = "";
              if (lastShop) drawStore(lastShop);
              else paintWishList(null);
            };
          } else {
            wish.hidden = true;
            wish.onclick = null;
          }
        }
        document.getElementById("zoom").hidden = false;
      }
      function skinCard(item) {
        const card = el("article", "skin");
        if (item.icon) {
          const img = document.createElement("img");
          img.alt = "";
          img.src = item.icon;
          card.append(img);
        }
        const body = el("div");
        const mine = owns(item.name);
        if (mine) card.classList.add("owned");
        if (item.fresh) {
          card.classList.add("fresh-skin");
          body.append(el("span", "new-tag", "New"));
        }
        body.append(el("b", null, item.name || "Skin"));
        const price = el("div", "price");
        const unit = item.currency === "KC" ? " KC" : " VP";
        if (mine) price.textContent = "Owned";
        else {
          if (item.was && item.was !== item.price) price.append(el("s", null, item.was + unit));
          price.append(document.createTextNode((item.price || "—") + (item.price ? unit : "")));
        }
        body.append(price);
        if (!mine && item.currency === "KC" && item.offer) {
          const buy = el("button", "tiny", "Buy");
          buy.type = "button";
          buy.onclick = async (event) => {
            event.preventDefault();
            event.stopPropagation();
            if (buy.dataset.arm !== "1") {
              buy.dataset.arm = "1";
              buy.textContent = "Confirm";
              return;
            }
            buy.textContent = "Buying…";
            try {
              const data = await (await fetch("/api/buy", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ offer: item.offer, price: item.price }) })).json();
              buy.textContent = data.error || "Owned";
              if (!data.error) {
                rememberOwned(item.name);
                ownedNames = ownedNames.concat([item.name]);
                storeKey = "";
                const wishRoot = document.getElementById("wish-list");
                if (wishRoot) wishRoot.dataset.key = "";
                if (lastShop) drawStore(lastShop);
              }
            } catch {
              buy.textContent = "Failed";
            }
          };
          body.append(buy);
        }
        card.append(body);
        const wish = el("button", "tiny wish", wished(item.name) ? "Wished" : "Wish");
        wish.type = "button";
        if (wished(item.name)) card.classList.add("wished");
        wish.onclick = (event) => {
          event.preventDefault();
          event.stopPropagation();
          toggleWish(item.name);
          storeKey = "";
          if (lastShop) drawStore(lastShop);
        };
        body.append(wish);
        if (item.icon) card.onclick = () => openZoom(item);
        return card;
      }
