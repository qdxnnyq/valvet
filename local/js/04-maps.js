      function calloutLabel(call) {
        const region = String((call && call.regionName) || "").trim();
        const area = String((call && call.superRegionName) || "").trim();
        if (!region) return "";
        const side = area === "Attacker Side" ? "Attack" : area === "Defender Side" ? "Defense" : area;
        return side && side !== region ? side + " " + region : region;
      }
      function calloutPins(map) {
        const pins = [];
        const seen = new Set();
        for (const call of (map && map.callouts) || []) {
          const label = calloutLabel(call);
          if (!label || seen.has(label)) continue;
          const loc = call.location || {};
          const spot = plotSpot(map, { x: loc.x, y: loc.y });
          if (!spot) continue;
          seen.add(label);
          pins.push({ label, x: spot.x, y: spot.y });
        }
        return pins;
      }
      function mapFrame(map, pins, paper, tone) {
        const stage = el("div", "map-stage");
        const img = document.createElement("img");
        img.className = "map-img";
        img.alt = "";
        img.src = (map && map.displayIcon) || "";
        stage.append(img);
        const layer = el("div", "map-pins");
        const marks = (pins || []).some((pin) => pin.label) ? spreadPins(pins) : (pins || []);
        for (const pin of marks) {
          const mark = el(pin.label ? "span" : "i", pin.label ? "call" : "ink", pin.label || "");
          mark.style.left = pin.x + "%";
          mark.style.top = pin.y + "%";
          layer.append(mark);
        }
        stage.append(layer);
        if (paper && paper.length) stage.append(heatLayer(paper, tone || "deaths"));
        const bigger = el("button", "tiny map-bigger", "Bigger");
        bigger.type = "button";
        bigger.onclick = () => openMapStage(stage);
        stage.append(bigger);
        return stage;
      }
      function heatLayer(spots, tone) {
        const size = 320;
        const canvas = document.createElement("canvas");
        canvas.width = size;
        canvas.height = size;
        const ctx = canvas.getContext("2d");
        const kill = tone === "kills";
        ctx.clearRect(0, 0, size, size);
        ctx.globalCompositeOperation = "lighter";
        for (const spot of spots || []) {
          const x = (Number(spot.x) / 100) * size;
          const y = (Number(spot.y) / 100) * size;
          if (!Number.isFinite(x) || !Number.isFinite(y)) continue;
          const radius = 28;
          const glow = ctx.createRadialGradient(x, y, 0, x, y, radius);
          if (kill) {
            glow.addColorStop(0, "rgba(255,255,240,0.95)");
            glow.addColorStop(0.16, "rgba(255,214,64,0.92)");
            glow.addColorStop(0.42, "rgba(255,120,16,0.55)");
            glow.addColorStop(1, "rgba(255,70,0,0)");
          } else {
            glow.addColorStop(0, "rgba(255,236,240,0.95)");
            glow.addColorStop(0.18, "rgba(255,48,82,0.9)");
            glow.addColorStop(0.48, "rgba(150,8,32,0.5)");
            glow.addColorStop(1, "rgba(80,0,16,0)");
          }
          ctx.fillStyle = glow;
          ctx.beginPath();
          ctx.arc(x, y, radius, 0, Math.PI * 2);
          ctx.fill();
        }
        ctx.globalCompositeOperation = "source-over";
        for (const spot of spots || []) {
          const x = (Number(spot.x) / 100) * size;
          const y = (Number(spot.y) / 100) * size;
          if (!Number.isFinite(x) || !Number.isFinite(y)) continue;
          ctx.beginPath();
          ctx.fillStyle = kill ? "#fff6c2" : "#ffe4ea";
          ctx.arc(x, y, 3.2, 0, Math.PI * 2);
          ctx.fill();
          ctx.lineWidth = 1.5;
          ctx.strokeStyle = kill ? "rgba(80,40,0,.85)" : "rgba(40,0,10,.8)";
          ctx.stroke();
        }
        const image = document.createElement("img");
        image.className = "heat";
        image.alt = "";
        image.src = canvas.toDataURL("image/png");
        return image;
      }
      function spreadPins(pins) {
        const weight = (label) => /site|spawn|main|mid/i.test(String(label || "")) ? 2 : 0;
        const ordered = [...(pins || [])].sort((a, b) => weight(b.label) - weight(a.label));
        const placed = [];
        for (const pin of ordered) {
          let x = Number(pin.x);
          let y = Number(pin.y);
          if (!Number.isFinite(x) || !Number.isFinite(y)) continue;
          let kept = false;
          for (let n = 0; n < 10; n++) {
            const dist = n === 0 ? 0 : 4 + n * 1.6;
            const ang = n * 1.15;
            const nx = Math.max(6, Math.min(94, x + Math.cos(ang) * dist));
            const ny = Math.max(8, Math.min(92, y + Math.sin(ang) * dist * 0.72));
            const clear = placed.every((other) => {
              const dx = other.x - nx;
              const dy = (other.y - ny) * 1.7;
              return dx * dx + dy * dy > 36;
            });
            if (clear) {
              x = nx;
              y = ny;
              kept = true;
              break;
            }
          }
          if (!kept && weight(pin.label) < 2) continue;
          placed.push({ ...pin, x, y });
        }
        return placed;
      }
      async function paintMaps() {
        const root = document.getElementById("maps");
        if (!root) return;
        const rows = mapRows().map((row) => ({ ...row, name: mapName(row.name) }));
        if (!root.querySelector(".map-stage")) root.replaceChildren(el("div", "empty", "Loading minimaps…"));
        const api = await mapBook();
        if (tab !== "maps") return;
        const stats = {};
        for (const row of rows) stats[row.name] = row;
        const names = [];
        const seen = new Set();
        for (const row of rows) if (row.name && !seen.has(row.name)) { seen.add(row.name); names.push(row.name); }
        for (const name of Object.keys(api || {})) {
          const map = api[name];
          if (!map || !map.displayIcon || !Number(map.xMultiplier) || seen.has(name)) continue;
          seen.add(name);
          names.push(name);
        }
        const sig = names.map((name) => name + ":" + ((stats[name] && stats[name].games) || 0)).join("|");
        if (root.dataset.sig === sig && root.querySelector(".map-stage")) return;
        root.dataset.sig = sig;
        root.className = "";
        root.replaceChildren();
        if (!names.length) {
          root.append(el("div", "empty", "No minimaps loaded."));
          return;
        }
        for (const name of names) {
          const map = api[name] || {};
          const row = stats[name];
          const rate = row && row.games ? Math.round((row.wins / row.games) * 100) + "%" : "—";
          const card = el("article", "desk-card");
          card.append(el("div", "eyebrow", "Minimap"));
          card.append(el("h2", null, name + " · " + rate));
          if (row) card.append(el("p", "sub", row.wins + "–" + row.losses + " · " + row.games + " saved games"));
          else card.append(el("p", "sub", "No saved games on this map yet."));
          if (!map.displayIcon) {
            card.append(el("p", "sub", "No minimap for this name yet."));
            root.append(card);
            continue;
          }
          warmMap(map.displayIcon);
          const pins = calloutPins(map);
          card.append(mapFrame(map, pins, null));
          root.append(card);
        }
      }
      const warmedMaps = new Set();
      function warmMap(url) {
        if (!url || warmedMaps.has(url)) return;
        warmedMaps.add(url);
        const img = new Image();
        img.decoding = "async";
        img.src = url;
      }
      let mapBookPromise = null;
      function mapsFrom(data) {
        const book = {};
        for (const map of (data && data.data) || []) {
          if (!map.displayName) continue;
          book[mapName(map.displayName)] = map;
          warmMap(map.displayIcon);
        }
        return book;
      }
      function mapBook() {
        if (!mapBookPromise) {
          mapBookPromise = fetch("https://valorant-api.com/v1/maps").then((res) => res.json()).then((data) => {
            const book = mapsFrom(data);
            if (Object.keys(book).length) return book;
            throw new Error("empty");
          }).catch(() => fetch("/pack/maps.json", { cache: "no-store" }).then((res) => res.json()).then(mapsFrom).catch(() => ({})));
        }
        return mapBookPromise;
      }
