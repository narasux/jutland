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
    .type-buttons { display: inline-flex; flex-wrap: wrap; gap: 6px; }
    .type-count { color: var(--muted); margin-left: 5px; }
    .type.active .type-count { color: var(--text); }
    input[type="number"] { width: 74px; }
    input[type="checkbox"] { accent-color: var(--accent); }
    main {
      display: grid;
      grid-template-columns: minmax(0, 1fr);
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
    #arcPreview {
      position: absolute;
      inset: 0;
      width: 100%;
      height: 100%;
      overflow: visible;
    }
    .arc-circle {
      fill: none;
      stroke: rgba(255, 255, 255, .38);
      stroke-width: 1;
      stroke-dasharray: 4 4;
      vector-effect: non-scaling-stroke;
    }
    .arc-sector {
      stroke-width: 1.5;
      vector-effect: non-scaling-stroke;
    }
    .arc-sector.starboard {
      fill: rgba(98, 214, 167, .18);
      stroke: rgba(98, 214, 167, .95);
    }
    .arc-sector.port {
      fill: rgba(101, 183, 255, .18);
      stroke: rgba(101, 183, 255, .95);
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
      background: #d0d0d0;
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
      display: grid;
      grid-template-columns: minmax(280px, 1fr) minmax(340px, 1.2fr);
      gap: 12px;
      align-items: start;
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
    .arc-field {
      display: flex;
      flex-direction: column;
      gap: 3px;
      min-width: 0;
    }
    .arc-field label {
      color: var(--muted);
    }
    .arc-field input { width: 100%; }
    .angle-presets {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      margin-top: 2px;
    }
    .angle-presets button {
      min-width: 38px;
      padding: 3px 6px;
      color: var(--muted);
      background: #1d222c;
      font-size: 12px;
      line-height: 1.25;
    }
    .angle-presets button:hover { color: var(--text); }
    .angle-presets button:disabled { cursor: default; opacity: .45; }
    .actions { display: flex; flex-wrap: wrap; gap: 7px; }
    .footnote { color: var(--muted); font-size: 12px; }
    @media (max-width: 900px) {
      main { grid-template-columns: 1fr; }
      .side { grid-template-columns: 1fr; }
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
    <span id="typeButtons" class="type-buttons"></span>
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
        <h2>射界参考</h2>
        <p class="hint">选中炮位后在地图上实时显示半透明射界，并写入项目 JSON；推荐值也可人工修正。</p>
        <div class="actions">
          <button id="recommendArc">按当前位置推荐</button>
          <button id="applyArc">应用到所选标记</button>
        </div>
        <div class="arc-grid">
          <div class="arc-field">
            <label for="rightStart">右舷起</label>
            <input id="rightStart" type="number" step="1">
            <div class="angle-presets" aria-label="右舷起常用角度">
              <button type="button" data-input="rightStart" data-angle="0">0°</button>
              <button type="button" data-input="rightStart" data-angle="30">30°</button>
              <button type="button" data-input="rightStart" data-angle="45">45°</button>
              <button type="button" data-input="rightStart" data-angle="90">90°</button>
              <button type="button" data-input="rightStart" data-angle="135">135°</button>
              <button type="button" data-input="rightStart" data-angle="150">150°</button>
              <button type="button" data-input="rightStart" data-angle="180">180°</button>
            </div>
          </div>
          <div class="arc-field">
            <label for="rightEnd">右舷止</label>
            <input id="rightEnd" type="number" step="1">
            <div class="angle-presets" aria-label="右舷止常用角度">
              <button type="button" data-input="rightEnd" data-angle="0">0°</button>
              <button type="button" data-input="rightEnd" data-angle="30">30°</button>
              <button type="button" data-input="rightEnd" data-angle="45">45°</button>
              <button type="button" data-input="rightEnd" data-angle="90">90°</button>
              <button type="button" data-input="rightEnd" data-angle="135">135°</button>
              <button type="button" data-input="rightEnd" data-angle="150">150°</button>
              <button type="button" data-input="rightEnd" data-angle="180">180°</button>
            </div>
          </div>
          <div class="arc-field">
            <label for="leftStart">左舷起</label>
            <input id="leftStart" type="number" step="1">
            <div class="angle-presets" aria-label="左舷起常用角度">
              <button type="button" data-input="leftStart" data-angle="180">180°</button>
              <button type="button" data-input="leftStart" data-angle="210">210°</button>
              <button type="button" data-input="leftStart" data-angle="225">225°</button>
              <button type="button" data-input="leftStart" data-angle="270">270°</button>
              <button type="button" data-input="leftStart" data-angle="315">315°</button>
              <button type="button" data-input="leftStart" data-angle="330">330°</button>
              <button type="button" data-input="leftStart" data-angle="360">360°</button>
            </div>
          </div>
          <div class="arc-field">
            <label for="leftEnd">左舷止</label>
            <input id="leftEnd" type="number" step="1">
            <div class="angle-presets" aria-label="左舷止常用角度">
              <button type="button" data-input="leftEnd" data-angle="180">180°</button>
              <button type="button" data-input="leftEnd" data-angle="210">210°</button>
              <button type="button" data-input="leftEnd" data-angle="225">225°</button>
              <button type="button" data-input="leftEnd" data-angle="270">270°</button>
              <button type="button" data-input="leftEnd" data-angle="315">315°</button>
              <button type="button" data-input="leftEnd" data-angle="330">330°</button>
              <button type="button" data-input="leftEnd" data-angle="360">360°</button>
            </div>
          </div>
        </div>
        <div class="actions" style="margin-top: 8px;">
          <button id="copyProject" class="primary">复制项目 JSON</button>
        </div>
        <div id="copyStatus" class="footnote"></div>
      </section>
    </aside>
  </main>

  <script>
    const meta = __META__;
    const typeDefs = meta.types;
    const storageKey = `jutland-turret-marker:${meta.key}`;
    const arcRulesVersion = 2;
    const state = {
      markers: [],
      type: typeDefs[0]?.name ?? "main",
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
    const copyStatus = document.getElementById("copyStatus");
    const typeButtons = document.getElementById("typeButtons");
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

    function typeDefinition(name) {
      return typeDefs.find(item => item.name === name) ?? {name, label: name, expected: null};
    }

    function markerCount(name) {
      return state.markers.filter(marker => marker.type === name).length;
    }

    function sideFor(x, y) {
      const transverse = meta.axis === "y" ? x : y;
      const span = meta.axis === "y" ? meta.width : meta.height;
      const center = span / 2;
      const tolerance = Math.max(2, span * 0.025);
      if (Math.abs(transverse - center) <= tolerance) return "center";
      const lowSideIsPort = meta.axis === "y" ? meta.bow === "top" : meta.bow === "right";
      return (transverse < center) === lowSideIsPort ? "port" : "starboard";
    }

    function projectData() {
      return {
        image: meta.name,
        imagePath: meta.path,
        bow: meta.bow,
        axis: meta.axis,
        bounds: {start: state.start, end: state.end},
        precision: Number(precision.value),
        types: typeDefs.map(item => ({
          ...item,
          actual: markerCount(item.name)
        })),
        markers: state.markers.map((marker, index) => ({
          index: index + 1,
          id: marker.id,
          type: marker.type,
          side: marker.side,
          x: Number(marker.x.toFixed(2)),
          y: Number(marker.y.toFixed(2)),
          posPercent: Number(fmt(marker.pos)),
          arcs: marker.arcs,
          snapped: marker.snapped
        }))
      };
    }

    function saveState() {
      try {
        localStorage.setItem(storageKey, JSON.stringify({
          markers: state.markers,
          start: state.start,
          end: state.end,
          type: state.type,
          arcRulesVersion
        }));
      } catch {
        copyStatus.textContent = "浏览器无法保存标注状态。";
      }
    }

    function restoreState() {
      try {
        const saved = JSON.parse(localStorage.getItem(storageKey));
        if (!saved || !Array.isArray(saved.markers)) return;
        state.markers = saved.markers.map(marker => {
          const side = marker.side ?? sideFor(marker.x, marker.y);
          return {
            ...marker,
            side,
            arcs: saved.arcRulesVersion === arcRulesVersion
              ? marker.arcs
              : recommendArc(marker.type, marker.pos, side)
          };
        });
        state.start = Number(saved.start ?? meta.start);
        state.end = Number(saved.end ?? meta.end);
        for (const marker of state.markers) {
          if (!typeDefs.some(item => item.name === marker.type)) {
            typeDefs.push({name: marker.type, label: marker.type, expected: null});
          }
        }
        if (typeDefs.some(item => item.name === saved.type)) state.type = saved.type;
      } catch {
        localStorage.removeItem(storageKey);
      }
    }

    function renderTypeButtons() {
      typeButtons.innerHTML = "";
      for (const definition of typeDefs) {
        const count = markerCount(definition.name);
        const button = document.createElement("button");
        button.className = `type${state.type === definition.name ? " active" : ""}`;
        button.dataset.type = definition.name;
        button.textContent = definition.expected == null
          ? `${definition.label} ${count}`
          : `${definition.label} ${count}/${definition.expected}`;
        typeButtons.appendChild(button);
      }
    }

    function recommendArc(type, pos, side = "center") {
      let arcs;
      if (type === "torpedo") {
        arcs = { rightStart: 30, rightEnd: 150, leftStart: 210, leftEnd: 330 };
      } else if (type === "main") {
        if (side !== "center") {
          arcs = { rightStart: 0, rightEnd: 180, leftStart: 180, leftEnd: 360 };
        } else if (pos > 0.25) {
          arcs = { rightStart: 0, rightEnd: 150, leftStart: 210, leftEnd: 360 };
        } else if (pos < -0.25) {
          arcs = { rightStart: 30, rightEnd: 180, leftStart: 180, leftEnd: 330 };
        } else {
          arcs = { rightStart: 45, rightEnd: 135, leftStart: 225, leftEnd: 315 };
        }
      } else if (pos > 0.25) {
        arcs = { rightStart: 30, rightEnd: 150, leftStart: 210, leftEnd: 330 };
      } else if (pos < -0.25) {
        arcs = { rightStart: 45, rightEnd: 180, leftStart: 180, leftEnd: 315 };
      } else {
        arcs = { rightStart: 0, rightEnd: 180, leftStart: 180, leftEnd: 360 };
      }
      if (side === "starboard") {
        return { ...arcs, leftStart: 360, leftEnd: 360 };
      }
      if (side === "port") {
        return { ...arcs, rightStart: 0, rightEnd: 0 };
      }
      return arcs;
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

    function nearestOppositeSameType(raw, side) {
      if (!snap.checked || side === "center") return null;
      const tolerance = Number(snapTolerance.value);
      let best = null;
      let bestDistance = Infinity;
      for (const marker of state.markers) {
        if (marker.type !== state.type || marker.side === side || marker.side === "center") continue;
        const distance = Math.abs(marker.raw - raw);
        if (distance <= tolerance && distance < bestDistance) {
          best = marker;
          bestDistance = distance;
        }
      }
      return best;
    }

    function mirrorArc(start, end) {
      if (start === 0 && end === 0) return [360, 360];
      if (start === 360 && end === 360) return [0, 0];
      // 按舰体艏艉轴镜像：舰艏/舰艉方向不变，左右舷互换。
      return [360 - end, 360 - start];
    }

    function mirrorArcs(arcs) {
      const [rightStart, rightEnd] = mirrorArc(arcs.leftStart, arcs.leftEnd);
      const [leftStart, leftEnd] = mirrorArc(arcs.rightStart, arcs.rightEnd);
      return { rightStart, rightEnd, leftStart, leftEnd };
    }

    function addMarker(x, y) {
      const raw = calcPos(longitudinalCoord(x, y));
      const nearest = nearestSameType(raw);
      const side = sideFor(x, y);
      const opposite = nearestOppositeSameType(raw, side);
      const pos = nearest ? nearest.pos : Number(raw.toFixed(Number(precision.value)));
      let arcs = recommendArc(state.type, pos, side);
      if (nearest) {
        const arcSource = opposite ?? nearest;
        arcs = arcSource.side !== side && arcSource.side !== "center" && side !== "center"
          ? mirrorArcs(arcSource.arcs)
          : { ...arcSource.arcs };
      }
      const marker = {
        id: crypto.randomUUID(),
        type: state.type,
        x,
        y,
        raw,
        pos,
        side,
        snapped: Boolean(nearest),
        arcs
      };
      state.markers.push(marker);
      state.selected = marker.id;
      status.textContent = nearest
        ? `已吸附到已有 ${state.type} 标记，posPercent=${fmt(pos)}${opposite ? "，射界已按对侧镜像" : ""}`
        : `已添加 ${state.type}，posPercent=${fmt(pos)}`;
      render();
      saveState();
    }

    function renderOverlay() {
      overlay.innerHTML = "";
      const marker = selectedMarker();
      if (marker) {
        const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        svg.id = "arcPreview";
        svg.setAttribute("viewBox", `0 0 ${meta.width} ${meta.height}`);
        svg.setAttribute("preserveAspectRatio", "none");

        // 射界圆周至少延伸到离当前炮位最远的画布角落。
        const radius = Math.max(
          Math.hypot(marker.x, marker.y),
          Math.hypot(meta.width - marker.x, marker.y),
          Math.hypot(marker.x, meta.height - marker.y),
          Math.hypot(meta.width - marker.x, meta.height - marker.y),
        );
        const circle = document.createElementNS("http://www.w3.org/2000/svg", "circle");
        circle.setAttribute("cx", marker.x);
        circle.setAttribute("cy", marker.y);
        circle.setAttribute("r", radius);
        circle.setAttribute("class", "arc-circle");
        svg.appendChild(circle);

        const point = angle => {
          const radians = angle * Math.PI / 180;
          return [marker.x + radius * Math.cos(radians), marker.y + radius * Math.sin(radians)];
        };
        const addSector = (start, end, side) => {
          start = Math.max(0, Math.min(360, Number(start)));
          end = Math.max(0, Math.min(360, Number(end)));
          if (end <= start) return;
          const [startX, startY] = point(start);
          const [endX, endY] = point(end);
          const largeArc = end - start > 180 ? 1 : 0;
          const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
          path.setAttribute(
            "d",
            `M ${marker.x} ${marker.y} L ${startX} ${startY} A ${radius} ${radius} 0 ${largeArc} 1 ${endX} ${endY} Z`,
          );
          path.setAttribute("class", `arc-sector ${side}`);
          svg.appendChild(path);
        };
        addSector(marker.arcs.rightStart, marker.arcs.rightEnd, "starboard");
        addSector(marker.arcs.leftStart, marker.arcs.leftEnd, "port");
        overlay.appendChild(svg);
      }
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
        label.textContent = `${index + 1}. ${marker.type} [${marker.side ?? sideFor(marker.x, marker.y)}]: ${fmt(marker.pos)}${marker.snapped ? " *" : ""}`;
        label.style.cursor = "pointer";
        label.addEventListener("click", () => selectMarker(marker.id));
        const remove = document.createElement("button");
        remove.textContent = "删除";
        remove.addEventListener("click", event => {
          event.stopPropagation();
          state.markers = state.markers.filter(item => item.id !== marker.id);
          if (state.selected === marker.id) state.selected = state.markers.at(-1)?.id ?? null;
          render();
          saveState();
        });
        row.append(label, remove);
        item.appendChild(row);
        markerList.appendChild(item);
      });
    }

    function fillArcs(marker) {
      document.querySelectorAll(".angle-presets button").forEach(button => {
        button.disabled = !marker;
      });
      if (!marker) {
        Object.values(arcInputs).forEach(input => { input.value = ""; });
        return;
      }
      arcInputs.rightStart.value = marker.arcs.rightStart;
      arcInputs.rightEnd.value = marker.arcs.rightEnd;
      arcInputs.leftStart.value = marker.arcs.leftStart;
      arcInputs.leftEnd.value = marker.arcs.leftEnd;
    }

    function syncArcInputs() {
      const marker = selectedMarker();
      if (!marker) return;
      marker.arcs = {
        rightStart: Number(arcInputs.rightStart.value),
        rightEnd: Number(arcInputs.rightEnd.value),
        leftStart: Number(arcInputs.leftStart.value),
        leftEnd: Number(arcInputs.leftEnd.value)
      };
      renderOverlay();
      saveState();
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
      renderTypeButtons();
      renderOverlay();
      renderList();
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
          saveState();
        } else {
          status.textContent = "已记录第一个校准点，请点击另一个首尾端点。";
        }
        return;
      }
      addMarker(x, y);
    });

    typeButtons.addEventListener("click", event => {
      const button = event.target.closest(".type");
      if (!button) return;
      state.type = button.dataset.type;
      status.textContent = `当前类型：${typeDefinition(state.type).label}`;
      render();
      saveState();
    });

    document.getElementById("autoCalibrate").addEventListener("click", () => {
      state.start = meta.start;
      state.end = meta.end;
      state.calibrating = false;
      state.calPoints = [];
      status.textContent = `自动校准完成：${state.start} -> ${state.end}`;
      render();
      saveState();
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
      saveState();
    });

    document.getElementById("recommendArc").addEventListener("click", () => {
      const marker = selectedMarker();
      if (!marker) {
        status.textContent = "请先选择一个标记。";
        return;
      }
      marker.arcs = recommendArc(marker.type, marker.pos, marker.side);
      fillArcs(marker);
      status.textContent = "已按当前位置重新推荐射界。";
    });

    document.getElementById("applyArc").addEventListener("click", () => {
      const marker = selectedMarker();
      if (!marker) {
        status.textContent = "请先选择一个标记。";
        return;
      }
      syncArcInputs();
      status.textContent = "已应用人工修正的射界（仅保留在当前会话）。";
      render();
      saveState();
    });

    Object.values(arcInputs).forEach(input => {
      input.addEventListener("input", syncArcInputs);
    });

    document.querySelector(".arc-grid").addEventListener("click", event => {
      const button = event.target.closest("button[data-angle]");
      if (!button || !selectedMarker()) return;
      const input = arcInputs[button.dataset.input];
      if (!input) return;
      input.value = button.dataset.angle;
      input.dispatchEvent(new Event("input"));
    });

    async function copyText(text, label) {
      try {
        await navigator.clipboard.writeText(text);
        copyStatus.textContent = `${label}已复制。`;
      } catch {
        const helper = document.createElement("textarea");
        helper.value = text;
        helper.style.position = "fixed";
        helper.style.opacity = "0";
        document.body.appendChild(helper);
        helper.focus();
        helper.select();
        document.execCommand("copy");
        helper.remove();
        copyStatus.textContent = `${label}已尝试复制。`;
      }
    }

    document.getElementById("copyProject").addEventListener("click", () => {
      const mismatches = typeDefs
        .filter(item => item.expected != null && markerCount(item.name) !== item.expected)
        .map(item => `${item.label} ${markerCount(item.name)}/${item.expected}`);
      const label = mismatches.length > 0
        ? `项目 JSON（数量提示：${mismatches.join("，")}）`
        : "项目 JSON";
      copyText(JSON.stringify(projectData(), null, 2), label);
    });

    precision.addEventListener("change", render);
    snapTolerance.addEventListener("change", render);
    restoreState();
    status.textContent = meta.edgeWarning
      ? `警告：${meta.edgeWarning}；自动边界：${state.start} -> ${state.end}；舰艏方向：${meta.bow}`
      : `自动边界：${state.start} -> ${state.end}；舰艏方向：${meta.bow}`;
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
    parser.add_argument(
        "--types",
        default="main,secondary,aa,torpedo,rocket,custom",
        help="comma-separated type names with optional expected counts, e.g. main:5,aa75:8",
    )
    parser.add_argument("--check", action="store_true", help="print image metadata and exit")
    parser.add_argument("--no-open", action="store_true", help="do not open the browser automatically")
    return parser.parse_args()


TYPE_LABELS = {
    "main": "主炮",
    "secondary": "副炮",
    "aa": "防空炮",
    "aa75": "75mm 高炮",
    "aa37": "37mm 高炮",
    "aa100": "100mm 高炮",
    "aa40": "40mm 高炮",
    "aa132": "13.2mm 机枪",
    "aa132q": "13.2mm 四联装机枪",
    "aa132t": "13.2mm 双联装机枪",
    "aa20": "20mm 机炮",
    "aa20d": "20mm 双联",
    "aa20q": "20mm 四联",
    "torpedo": "鱼雷",
    "rocket": "火箭炮",
    "custom": "自定义",
}


def parse_types(value: str) -> list[dict]:
    """Parse comma-separated marker types and optional expected counts."""
    result: list[dict] = []
    seen: set[str] = set()
    for raw_item in value.split(","):
        item = raw_item.strip()
        if not item:
            continue
        name, separator, count_text = item.partition(":")
        name = name.strip()
        if not name or name in seen:
            continue
        expected = None
        if separator:
            try:
                expected = int(count_text)
            except ValueError as exc:
                raise ValueError(f"invalid expected count for type {name!r}: {count_text!r}") from exc
            if expected < 0:
                raise ValueError(f"expected count for type {name!r} must be non-negative")
        seen.add(name)
        result.append({
            "name": name,
            "label": TYPE_LABELS.get(name, name),
            "expected": expected,
        })
    if not result:
        raise ValueError("--types must include at least one type")
    return result


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
    axis_length = image.height if axis == "y" else image.width
    edge_warnings = []
    if start <= 1:
        edge_warnings.append("舰艉或首个端点贴边")
    if end >= axis_length - 2:
        edge_warnings.append("舰艏或末端端点贴边")
    return {
        "name": name,
        "width": image.width,
        "height": image.height,
        "axis": axis,
        "start": start,
        "end": end,
        "bow": bow,
        "edgeWarning": "；".join(edge_warnings),
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
    meta["path"] = str(image_path)
    meta["key"] = str(image_path)
    meta["types"] = parse_types(args.types)
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
