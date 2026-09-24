#!/usr/bin/env python3
"""Local browser tool for marking ship weapon positions on a top view."""

from __future__ import annotations

import argparse
import io
import json
import mimetypes
import webbrowser
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse

from PIL import Image


HTML = r"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Jutland posPercent Marker</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #111318;
      --panel: #1b1f27;
      --line: #343b48;
      --text: #e8edf5;
      --muted: #9ba7b8;
      --accent: #65b7ff;
      --danger: #ff7272;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font: 14px/1.45 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    header {
      padding: 14px 18px;
      border-bottom: 1px solid var(--line);
      background: var(--panel);
    }
    h1 { margin: 0 0 4px; font-size: 18px; }
    .subtitle { color: var(--muted); }
    .toolbar {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      align-items: center;
      padding: 12px 18px;
      border-bottom: 1px solid var(--line);
      background: #151922;
    }
    .toolbar label { color: var(--muted); }
    button, select, input {
      border: 1px solid var(--line);
      border-radius: 5px;
      background: #222834;
      color: var(--text);
      padding: 6px 9px;
      font: inherit;
    }
    button { cursor: pointer; }
    button:hover { border-color: var(--accent); }
    button.primary { background: #1d4f78; border-color: #2c76ad; }
    button.active { background: #285f8d; border-color: var(--accent); }
    input[type="number"] { width: 74px; }
    input[type="checkbox"] { accent-color: var(--accent); }
    main {
      display: grid;
      grid-template-columns: minmax(0, 1fr) 340px;
      gap: 14px;
      padding: 14px 18px 24px;
    }
    .panel {
      min-width: 0;
      border: 1px solid var(--line);
      border-radius: 7px;
      background: var(--panel);
      padding: 12px;
    }
    .canvas-panel { overflow: auto; }
    #stage {
      position: relative;
      display: block;
      width: 100%;
      background:
        linear-gradient(45deg, #202631 25%, transparent 25%) 0 0 / 18px 18px,
        linear-gradient(-45deg, #202631 25%, transparent 25%) 0 9px / 18px 18px,
        linear-gradient(45deg, transparent 75%, #202631 75%) 9px -9px / 18px 18px,
        linear-gradient(-45deg, transparent 75%, #202631 75%) -9px 0 / 18px 18px,
        #171b23;
    }
    #ship {
      display: block;
      width: 100%;
      max-width: none;
      height: auto;
      cursor: crosshair;
      user-select: none;
    }
    #overlay {
      position: absolute;
      inset: 0;
      pointer-events: none;
    }
    .cal-line {
      position: absolute;
      background: var(--accent);
      box-shadow: 0 0 0 1px rgba(101, 183, 255, .25);
    }
    .cal-line.vertical { top: 0; bottom: 0; width: 1px; }
    .cal-line.horizontal { left: 0; right: 0; height: 1px; }
    .marker {
      position: absolute;
      width: 15px;
      height: 15px;
      border: 2px solid #fff;
      border-radius: 50%;
      transform: translate(-50%, -50%);
      box-shadow: 0 0 0 2px #111, 0 0 8px rgba(255,255,255,.55);
    }
    .marker.main { background: #ff5f6d; }
    .marker.secondary { background: #ffbd59; }
    .marker.aa { background: #62d6a7; }
    .marker.torpedo { background: #7aa7ff; }
    .marker.rocket { background: #c68cff; }
    .marker.custom { background: #f1f1f1; }
    .marker.selected { outline: 2px solid var(--accent); outline-offset: 3px; }
    .status {
      min-height: 24px;
      margin-top: 9px;
      color: var(--muted);
    }
    .side {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    h2 { margin: 0 0 8px; font-size: 15px; }
    .hint { color: var(--muted); margin: 0 0 8px; }
    #markerList {
      max-height: 300px;
      overflow: auto;
      margin: 0;
      padding-left: 22px;
    }
    #markerList li { margin: 5px 0; }
    #markerList li.selected { color: var(--accent); }
    .marker-row {
      display: flex;
      gap: 6px;
      align-items: center;
      justify-content: space-between;
    }
    .marker-row button {
      padding: 2px 7px;
      color: var(--danger);
    }
    .arc-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 7px;
      margin-top: 8px;
    }
    .arc-grid label {
      display: flex;
      flex-direction: column;
      gap: 3px;
      color: var(--muted);
    }
    .arc-grid input { width: 100%; }
    .actions { display: flex; flex-wrap: wrap; gap: 7px; }
    textarea {
      width: 100%;
      min-height: 92px;
      resize: vertical;
      border: 1px solid var(--line);
      border-radius: 5px;
      background: #11151c;
      color: var(--text);
      padding: 8px;
      font: 13px/1.5 ui-monospace, SFMono-Regular, Menlo, monospace;
    }
    .footnote { color: var(--muted); font-size: 12px; }
    @media (max-width: 900px) {
      main { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <h1>Jutland posPercent Marker</h1>
    <div class="subtitle">点击俯视图标记火炮纵向位置；图片会按比例放大到画布最大宽度。</div>
  </header>

  <div class="toolbar">
    <span>武器类型：</span>
    <button class="type active" data-type="main">主炮</button>
    <button class="type" data-type="secondary">副炮</button>
    <button class="type" data-type="aa">防空炮</button>
    <button class="type" data-type="torpedo">鱼雷</button>
    <button class="type" data-type="rocket">火箭炮</button>
    <button class="type" data-type="custom">自定义</button>
    <label><input id="snap" type="checkbox" checked> 对称吸附</label>
    <label>吸附阈值 <input id="snapTolerance" type="number" min="0" max="0.2" step="0.005" value="0.03"></label>
    <label>精度 <input id="precision" type="number" min="1" max="6" step="1" value="2"></label>
    <button id="autoCalibrate" class="primary">自动识别</button>
    <button id="manualCalibrate">手动校准</button>
    <button id="clear">清空标记</button>
  </div>

  <main>
    <section class="panel canvas-panel">
      <div id="stage">
        <img id="ship" alt="ship top view">
        <div id="overlay"></div>
      </div>
      <div id="status" class="status">正在加载图片...</div>
    </section>

    <aside class="side">
      <section class="panel">
        <h2>标记列表</h2>
        <p class="hint">选择类型后点击图片；同一类型的近距离重复点击会吸附到已有 posPercent。</p>
        <ol id="markerList"></ol>
      </section>

      <section class="panel">
        <h2>射界参考（不导出）</h2>
        <p class="hint">按同类舰常见配置推荐，可人工修正；主输出仍然只有 posPercent。</p>
        <div class="actions">
          <button id="recommendArc">按当前位置推荐</button>
          <button id="applyArc">应用到所选标记</button>
        </div>
        <div class="arc-grid">
          <label>右舷起<input id="rightStart" type="number" step="1"></label>
          <label>右舷止<input id="rightEnd" type="number" step="1"></label>
          <label>左舷起<input id="leftStart" type="number" step="1"></label>
          <label>左舷止<input id="leftEnd" type="number" step="1"></label>
        </div>
      </section>

      <section class="panel">
        <h2>当前类型 posPercent</h2>
        <textarea id="output" readonly></textarea>
        <div class="actions">
          <button id="copy" class="primary">复制当前类型</button>
          <button id="copyAll">复制全部</button>
        </div>
        <div id="copyStatus" class="footnote"></div>
      </section>
    </aside>
  </main>

  <script>
    const meta = __META__;
    const state = {
      markers: [],
      type: "main",
      selected: null,
      calibrating: false,
      calPoints: [],
      start: meta.start,
      end: meta.end
    };
    const ship = document.getElementById("ship");
    const overlay = document.getElementById("overlay");
    const status = document.getElementById("status");
    const markerList = document.getElementById("markerList");
    const output = document.getElementById("output");
    const copyStatus = document.getElementById("copyStatus");
    const snap = document.getElementById("snap");
    const snapTolerance = document.getElementById("snapTolerance");
    const precision = document.getElementById("precision");
    const arcInputs = {
      rightStart: document.getElementById("rightStart"),
      rightEnd: document.getElementById("rightEnd"),
      leftStart: document.getElementById("leftStart"),
      leftEnd: document.getElementById("leftEnd")
    };

    ship.src = "/image";
    ship.alt = meta.name;

    function fmt(value) {
      return Number(value).toFixed(Number(precision.value));
    }

    function longitudinalCoord(x, y) {
      return meta.axis === "y" ? y : x;
    }

    function calcPos(coord) {
      const center = (state.start + state.end) / 2;
      const half = Math.max(1, (state.end - state.start) / 2);
      let ratio = (coord - center) / half;
      if (meta.bow === "top" || meta.bow === "left") ratio = -ratio;
      return Math.max(-1, Math.min(1, ratio));
    }

    function recommendArc(type, pos) {
      if (type === "torpedo") {
        return { rightStart: 30, rightEnd: 150, leftStart: 210, leftEnd: 330 };
      }
      if (type === "main") {
        if (pos > 0.25) return { rightStart: 0, rightEnd: 150, leftStart: 210, leftEnd: 360 };
        if (pos < -0.25) return { rightStart: 30, rightEnd: 180, leftStart: 180, leftEnd: 330 };
        return { rightStart: 45, rightEnd: 135, leftStart: 225, leftEnd: 315 };
      }
      if (pos > 0.25) return { rightStart: 0, rightEnd: 150, leftStart: 210, leftEnd: 360 };
      if (pos < -0.25) return { rightStart: 30, rightEnd: 180, leftStart: 180, leftEnd: 330 };
      return { rightStart: 0, rightEnd: 180, leftStart: 180, leftEnd: 360 };
    }

    function nearestSameType(raw) {
      if (!snap.checked) return null;
      const tolerance = Number(snapTolerance.value);
      let best = null;
      let bestDistance = Infinity;
      for (const marker of state.markers) {
        if (marker.type !== state.type) continue;
        const distance = Math.abs(marker.raw - raw);
        if (distance <= tolerance && distance < bestDistance) {
          best = marker;
          bestDistance = distance;
        }
      }
      return best;
    }

    function addMarker(x, y) {
      const raw = calcPos(longitudinalCoord(x, y));
      const nearest = nearestSameType(raw);
      const pos = nearest ? nearest.pos : Number(raw.toFixed(Number(precision.value)));
      const marker = {
        id: crypto.randomUUID(),
        type: state.type,
        x,
        y,
        raw,
        pos,
        snapped: Boolean(nearest),
        arcs: recommendArc(state.type, pos)
      };
      state.markers.push(marker);
      state.selected = marker.id;
      status.textContent = nearest
        ? `已吸附到已有 ${state.type} 标记，posPercent=${fmt(pos)}`
        : `已添加 ${state.type}，posPercent=${fmt(pos)}`;
      render();
    }

    function updateOutput() {
      const values = state.markers
        .filter(marker => marker.type === state.type)
        .map(marker => fmt(marker.pos));
      output.value = values.join("\n");
    }

    function renderOverlay() {
      overlay.innerHTML = "";
      const span = meta.axis === "y" ? meta.height : meta.width;
      for (const [coord, className] of [[state.start, "start"], [state.end, "end"]]) {
        const line = document.createElement("div");
        const percent = `${(coord / span) * 100}%`;
        if (meta.axis === "y") {
          line.className = "cal-line horizontal";
          line.style.top = percent;
        } else {
          line.className = "cal-line vertical";
          line.style.left = percent;
        }
        line.dataset.side = className;
        overlay.appendChild(line);
      }
      state.markers.forEach((marker, index) => {
        const node = document.createElement("div");
        node.className = `marker ${marker.type}${marker.id === state.selected ? " selected" : ""}`;
        node.style.left = `${(marker.x / meta.width) * 100}%`;
        node.style.top = `${(marker.y / meta.height) * 100}%`;
        node.title = `${index + 1}. ${marker.type}: ${fmt(marker.pos)}${marker.snapped ? " (snapped)" : ""}`;
        overlay.appendChild(node);
      });
    }

    function renderList() {
      markerList.innerHTML = "";
      state.markers.forEach((marker, index) => {
        const item = document.createElement("li");
        if (marker.id === state.selected) item.classList.add("selected");
        const row = document.createElement("div");
        row.className = "marker-row";
        const label = document.createElement("span");
        label.textContent = `${index + 1}. ${marker.type}: ${fmt(marker.pos)}${marker.snapped ? " *" : ""}`;
        label.style.cursor = "pointer";
        label.addEventListener("click", () => selectMarker(marker.id));
        const remove = document.createElement("button");
        remove.textContent = "删除";
        remove.addEventListener("click", event => {
          event.stopPropagation();
          state.markers = state.markers.filter(item => item.id !== marker.id);
          if (state.selected === marker.id) state.selected = state.markers.at(-1)?.id ?? null;
          render();
        });
        row.append(label, remove);
        item.appendChild(row);
        markerList.appendChild(item);
      });
    }

    function fillArcs(marker) {
      if (!marker) {
        Object.values(arcInputs).forEach(input => { input.value = ""; });
        return;
      }
      arcInputs.rightStart.value = marker.arcs.rightStart;
      arcInputs.rightEnd.value = marker.arcs.rightEnd;
      arcInputs.leftStart.value = marker.arcs.leftStart;
      arcInputs.leftEnd.value = marker.arcs.leftEnd;
    }

    function selectedMarker() {
      return state.markers.find(marker => marker.id === state.selected) ?? null;
    }

    function selectMarker(id) {
      state.selected = id;
      fillArcs(selectedMarker());
      render();
    }

    function render() {
      renderOverlay();
      renderList();
      updateOutput();
      fillArcs(selectedMarker());
    }

    ship.addEventListener("click", event => {
      const rect = ship.getBoundingClientRect();
      const x = (event.clientX - rect.left) * meta.width / rect.width;
      const y = (event.clientY - rect.top) * meta.height / rect.height;
      const coord = longitudinalCoord(x, y);
      if (state.calibrating) {
        state.calPoints.push(coord);
        if (state.calPoints.length === 2) {
          state.start = Math.min(...state.calPoints);
          state.end = Math.max(...state.calPoints);
          state.calPoints = [];
          state.calibrating = false;
          status.textContent = `手动校准完成：${state.start} -> ${state.end}`;
          render();
        } else {
          status.textContent = "已记录第一个校准点，请点击另一个首尾端点。";
        }
        return;
      }
      addMarker(x, y);
    });

    document.querySelectorAll(".type").forEach(button => {
      button.addEventListener("click", () => {
        document.querySelectorAll(".type").forEach(item => item.classList.remove("active"));
        button.classList.add("active");
        state.type = button.dataset.type;
        status.textContent = `当前类型：${button.textContent}`;
        render();
      });
    });

    document.getElementById("autoCalibrate").addEventListener("click", () => {
      state.start = meta.start;
      state.end = meta.end;
      state.calibrating = false;
      state.calPoints = [];
      status.textContent = `自动校准完成：${state.start} -> ${state.end}`;
      render();
    });

    document.getElementById("manualCalibrate").addEventListener("click", () => {
      state.calibrating = true;
      state.calPoints = [];
      status.textContent = "请点击俯视图首尾两个端点，顺序不限。";
      render();
    });

    document.getElementById("clear").addEventListener("click", () => {
      state.markers = [];
      state.selected = null;
      status.textContent = "已清空标记。";
      render();
    });

    document.getElementById("recommendArc").addEventListener("click", () => {
      const marker = selectedMarker();
      if (!marker) {
        status.textContent = "请先选择一个标记。";
        return;
      }
      marker.arcs = recommendArc(marker.type, marker.pos);
      fillArcs(marker);
      status.textContent = "已按当前位置重新推荐射界。";
    });

    document.getElementById("applyArc").addEventListener("click", () => {
      const marker = selectedMarker();
      if (!marker) {
        status.textContent = "请先选择一个标记。";
        return;
      }
      marker.arcs = {
        rightStart: Number(arcInputs.rightStart.value),
        rightEnd: Number(arcInputs.rightEnd.value),
        leftStart: Number(arcInputs.leftStart.value),
        leftEnd: Number(arcInputs.leftEnd.value)
      };
      status.textContent = "已应用人工修正的射界（仅保留在当前会话）。";
      renderList();
    });

    async function copyText(text, label) {
      try {
        await navigator.clipboard.writeText(text);
        copyStatus.textContent = `${label}已复制。`;
      } catch {
        output.focus();
        output.select();
        document.execCommand("copy");
        copyStatus.textContent = `${label}已尝试复制。`;
      }
    }

    document.getElementById("copy").addEventListener("click", () => {
      copyText(output.value, "当前类型 posPercent");
    });

    document.getElementById("copyAll").addEventListener("click", () => {
      const text = state.markers.map(marker => fmt(marker.pos)).join("\n");
      copyText(text, "全部 posPercent");
    });

    precision.addEventListener("change", render);
    snapTolerance.addEventListener("change", render);
    status.textContent = `自动边界：${meta.start} -> ${meta.end}；舰艏方向：${meta.bow}`;
    render();
  </script>
</body>
</html>
"""


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Mark ship weapon posPercent values in a browser.")
    parser.add_argument("image", type=Path, help="top-view PNG path")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8765)
    parser.add_argument("--bow", choices=("top", "bottom", "left", "right"), default="top")
    parser.add_argument(
        "--rotate",
        type=int,
        choices=(0, 90, 180, 270),
        default=0,
        help="rotate the browser preview clockwise without changing the source image",
    )
    parser.add_argument("--check", action="store_true", help="print image metadata and exit")
    parser.add_argument("--no-open", action="store_true", help="do not open the browser automatically")
    return parser.parse_args()


def rotated_bow(bow: str, rotation: int) -> str:
    """Return the bow direction after rotating the preview clockwise."""
    directions = ("top", "right", "bottom", "left")
    return directions[(directions.index(bow) + rotation // 90) % 4]


def load_image(path: Path, rotation: int) -> Image.Image:
    """Load a PNG as RGBA and optionally rotate the in-memory preview."""
    if path.suffix.lower() != ".png":
        raise ValueError("image must be a PNG")
    with Image.open(path) as opened:
        if opened.format != "PNG":
            raise ValueError("image must be a PNG")
        image = opened.convert("RGBA")
        if rotation:
            image = image.rotate(-rotation, expand=True)
        return image


def load_meta(image: Image.Image, name: str, bow: str) -> dict:
    """Calculate the longitudinal bounds and bow direction for the preview."""
    bbox = image.getchannel("A").getbbox()
    if bbox is None:
        raise ValueError("image has no visible pixels")
    left, top, right, bottom = bbox
    right -= 1
    bottom -= 1
    axis = "y" if bow in ("top", "bottom") else "x"
    start = top if axis == "y" else left
    end = bottom if axis == "y" else right
    if end <= start:
        raise ValueError("image has no usable longitudinal extent")
    return {
        "name": name,
        "width": image.width,
        "height": image.height,
        "axis": axis,
        "start": start,
        "end": end,
        "bow": bow,
    }


def encode_png(image: Image.Image) -> bytes:
    """Encode the in-memory preview as PNG bytes for the local web server."""
    output = io.BytesIO()
    image.save(output, format="PNG")
    return output.getvalue()


def make_handler(image_bytes: bytes, image_type: str, meta: dict) -> type[BaseHTTPRequestHandler]:
    html = HTML.replace("__META__", json.dumps(meta, ensure_ascii=False))

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            path = urlparse(self.path).path
            if path == "/":
                payload = html.encode("utf-8")
                content_type = "text/html; charset=utf-8"
            elif path == "/meta":
                payload = json.dumps(meta, ensure_ascii=False).encode("utf-8")
                content_type = "application/json; charset=utf-8"
            elif path == "/image":
                payload = image_bytes
                content_type = image_type
            else:
                self.send_error(404)
                return
            self.send_response(200)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Length", str(len(payload)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(payload)

        def log_message(self, format: str, *args: object) -> None:
            return

    return Handler


def main() -> None:
    args = parse_args()
    image_path = args.image.resolve()
    if not image_path.is_file():
        raise FileNotFoundError(image_path)
    image = load_image(image_path, args.rotate)
    meta = load_meta(image, image_path.name, rotated_bow(args.bow, args.rotate))
    if args.check:
        print(json.dumps(meta, ensure_ascii=False, indent=2))
        return

    image_bytes = encode_png(image)
    image_type = mimetypes.guess_type(image_path.name)[0] or "application/octet-stream"
    handler = make_handler(image_bytes, image_type, meta)
    server = ThreadingHTTPServer((args.host, args.port), handler)
    actual_port = server.server_address[1]
    display_host = "127.0.0.1" if args.host in ("0.0.0.0", "::") else args.host
    url = f"http://{display_host}:{actual_port}/"
    print(f"Open {url}")
    if not args.no_open:
        webbrowser.open(url)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
