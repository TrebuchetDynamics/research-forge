// ResearchForge interactive citation graph — vendored, dependency-free, offline.
//
// Progressive enhancement: the server renders an accessible static SVG fallback
// inside the mount element. When this script runs it fetches the graph data and
// replaces the fallback with an interactive SVG supporting pan (drag), zoom
// (wheel / +/- keys), and click-through (node -> its /papers/{id} page). With
// JavaScript disabled the static fallback remains fully usable.

(function () {
  "use strict";

  var SVG_NS = "http://www.w3.org/2000/svg";

  // Deterministic circular layout so the same graph always renders the same way.
  function layout(nodes, width, height) {
    var cx = width / 2;
    var cy = height / 2;
    var radius = Math.min(width, height) / 2 - 60;
    var positions = {};
    var n = nodes.length;
    for (var i = 0; i < n; i++) {
      var angle = (2 * Math.PI * i) / Math.max(n, 1) - Math.PI / 2;
      positions[nodes[i].id] = {
        x: n === 1 ? cx : cx + radius * Math.cos(angle),
        y: n === 1 ? cy : cy + radius * Math.sin(angle),
      };
    }
    return positions;
  }

  function el(name, attrs) {
    var node = document.createElementNS(SVG_NS, name);
    for (var key in attrs) {
      if (Object.prototype.hasOwnProperty.call(attrs, key)) {
        node.setAttribute(key, attrs[key]);
      }
    }
    return node;
  }

  function renderCitationGraph(mount, graph) {
    var width = 720;
    var height = 420;
    var nodes = graph.nodes || [];
    var edges = graph.edges || [];
    var positions = layout(nodes, width, height);

    var svg = el("svg", {
      viewBox: "0 0 " + width + " " + height,
      class: "citation-graph-svg citation-graph-interactive",
      role: "application",
      "aria-label": "Interactive citation graph",
      tabindex: "0",
    });
    var viewport = el("g", {});
    svg.appendChild(viewport);

    edges.forEach(function (edge) {
      var s = positions[edge.source];
      var t = positions[edge.target];
      if (!s || !t) return;
      viewport.appendChild(
        el("line", {
          x1: s.x, y1: s.y, x2: t.x, y2: t.y,
          stroke: "currentColor", "stroke-width": "1",
        })
      );
    });

    nodes.forEach(function (node) {
      var p = positions[node.id];
      if (!p) return;
      var anchor = el("a", { href: node.href || "#", "data-stem": node.stem || "" });
      anchor.appendChild(
        el("circle", {
          cx: p.x, cy: p.y, r: "18",
          fill: "rgba(143,211,255,0.12)", stroke: "currentColor", "stroke-width": "2",
          style: "cursor:pointer",
        })
      );
      var text = el("text", { x: p.x, y: p.y + 34, "text-anchor": "middle" });
      text.textContent = node.label || node.id;
      anchor.appendChild(text);
      viewport.appendChild(anchor);
    });

    // Pan + zoom state applied as a transform on the viewport group.
    var scale = 1, tx = 0, ty = 0, dragging = false, lastX = 0, lastY = 0;
    function apply() {
      viewport.setAttribute(
        "transform",
        "translate(" + tx + "," + ty + ") scale(" + scale + ")"
      );
    }
    function zoom(factor) {
      scale = Math.max(0.3, Math.min(4, scale * factor));
      apply();
    }

    svg.addEventListener("wheel", function (e) {
      e.preventDefault();
      zoom(e.deltaY < 0 ? 1.1 : 0.9);
    }, { passive: false });

    svg.addEventListener("mousedown", function (e) {
      dragging = true; lastX = e.clientX; lastY = e.clientY;
    });
    window.addEventListener("mouseup", function () { dragging = false; });
    window.addEventListener("mousemove", function (e) {
      if (!dragging) return;
      tx += e.clientX - lastX; ty += e.clientY - lastY;
      lastX = e.clientX; lastY = e.clientY;
      apply();
    });

    svg.addEventListener("keydown", function (e) {
      if (e.key === "+" || e.key === "=") { zoom(1.1); }
      else if (e.key === "-") { zoom(0.9); }
      else if (e.key === "ArrowLeft") { tx += 20; apply(); }
      else if (e.key === "ArrowRight") { tx -= 20; apply(); }
      else if (e.key === "ArrowUp") { ty += 20; apply(); }
      else if (e.key === "ArrowDown") { ty -= 20; apply(); }
      else { return; }
      e.preventDefault();
    });

    mount.textContent = "";
    mount.appendChild(svg);
  }

  function enhance(mount) {
    var src = mount.getAttribute("data-src");
    if (!src || !window.fetch) return; // keep the static fallback
    fetch(src)
      .then(function (resp) { return resp.ok ? resp.json() : null; })
      .then(function (graph) {
        if (graph && graph.nodes && graph.nodes.length) {
          renderCitationGraph(mount, graph);
        }
      })
      .catch(function () { /* leave the static SVG fallback in place */ });
  }

  function init() {
    var mounts = document.querySelectorAll("[data-citation-graph]");
    Array.prototype.forEach.call(mounts, enhance);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  // Exposed for testing / reuse.
  if (typeof window !== "undefined") {
    window.renderCitationGraph = renderCitationGraph;
  }
})();
