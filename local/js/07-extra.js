      function queueAverage(mode) {
        const rows = (readJSON("valvet-pops") || []).filter((row) => row && row.wait > 0 && (!mode || !row.mode || String(row.mode).toLowerCase() === String(mode).toLowerCase()));
        if (!rows.length) return "";
        const avg = rows.reduce((sum, row) => sum + Number(row.wait || 0), 0) / rows.length;
        const secs = Math.round(avg / 1000);
        if (!secs) return "";
        return "Usually " + (secs >= 60 ? Math.floor(secs / 60) + "m " + (secs % 60) + "s" : secs + "s");
      }
      function mapRates() {
        const book = {};
        for (const match of history || []) {
          const name = (typeof mapName === "function" ? mapName(match.map) : match.map) || match.map;
          if (!name) continue;
          const row = book[name] || { name, games: 0, wins: 0 };
          row.games += 1;
          if (match.won === true) row.wins += 1;
          book[name] = row;
        }
        return Object.values(book).sort((a, b) => b.games - a.games).slice(0, 7);
      }
      function paintVeto(data) {
        const host = document.getElementById("veto");
        if (!host) return;
        const mode = String((data && (data.mode || data.queue)) || "");
        const phase = (data && data.phase) || "";
        const show = phase === "In lobby" && /custom|premier/i.test(mode);
        host.hidden = !show;
        if (!show) return;
        const rows = mapRates();
        const sig = mode + "|" + rows.map((row) => row.name + row.wins + "/" + row.games).join(",");
        if (host.dataset.sig === sig) return;
        host.dataset.sig = sig;
        host.replaceChildren();
        host.append(el("div", "eyebrow", /premier/i.test(mode) ? "Premier maps" : "Custom maps"));
        if (!rows.length) {
          host.append(el("p", "sub", "Map win rates show after a few saved games."));
          return;
        }
        for (const row of rows) {
          const rate = row.games ? Math.round((row.wins / row.games) * 100) : 0;
          host.append(el("div", "veto-row", row.name + " · " + rate + "% · " + row.wins + "-" + (row.games - row.wins)));
        }
      }
