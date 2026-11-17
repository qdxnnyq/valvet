package main

const broadcastHTML = `<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<title>Valvet broadcast</title>
<style>
  html, body { margin: 0; background: transparent; color: #fff7fa; font-family: "Segoe UI", sans-serif; }
  body { min-height: 100vh; display: grid; place-items: end stretch; padding: 28px; box-sizing: border-box; }
  .bug { display: flex; justify-content: space-between; gap: 18px; align-items: flex-end; padding: 16px 18px; border-radius: 18px; background: linear-gradient(90deg, rgba(16,8,14,.92), rgba(16,8,14,.55)); border: 1px solid rgba(255,95,162,.35); }
  .map { font-size: 42px; letter-spacing: -0.05em; font-weight: 760; }
  .sub { color: #e7b7cc; letter-spacing: .08em; text-transform: uppercase; font-size: 12px; }
  .score { font-size: 64px; font-weight: 780; letter-spacing: -0.06em; font-variant-numeric: tabular-nums; }
  .names { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 8px; }
  .names span { padding: 4px 8px; border-radius: 999px; background: rgba(255,255,255,.08); font-size: 13px; }
  .names span.you { background: rgba(255,95,162,.28); }
</style>
</head>
<body>
  <div class="bug">
    <div>
      <div class="sub" id="mode">Valvet</div>
      <div class="map" id="map">Waiting</div>
      <div class="names" id="names"></div>
    </div>
    <div class="score" id="score">–</div>
  </div>
<script>
async function tick() {
  try {
    const data = await (await fetch("/api/state", { cache: "no-store" })).json();
    document.getElementById("mode").textContent = [data.phase, data.mode].filter(Boolean).join(" · ") || "Valvet";
    document.getElementById("map").textContent = data.map || "No game";
    const a = data.score1 == null ? "–" : data.score1;
    const b = data.score2 == null ? "–" : data.score2;
    document.getElementById("score").textContent = a + "  " + b;
    const names = document.getElementById("names");
    names.replaceChildren();
    for (const player of data.players || []) {
      const span = document.createElement("span");
      if (player.you) span.className = "you";
      span.textContent = (player.agent || player.name || "Player");
      names.append(span);
    }
  } catch {}
}
tick();
setInterval(tick, 1000);
</script>
</body>
</html>`
