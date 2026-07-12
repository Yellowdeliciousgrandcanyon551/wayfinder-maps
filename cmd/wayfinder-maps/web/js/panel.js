// The slide-in ticket detail panel: header, status chips, rendered markdown
// body, and cross-ticket links that jump the star-map to another node.
import {S} from "./state.js";
import {el, pad2} from "./util.js";
import {mdToHtml} from "./markdown.js";

// edgeChip builds one "blocked by 01 12" / "blocks 04" chip: a neutral pill
// holding the label plus one link per ticket number, each coloured by that
// ticket's status and clicking through to it. A number whose ticket is missing
// from the graph (a dangling edge) renders red and inert.
function edgeChip(label, nums) {
  var chip = el("span", "c");
  chip.appendChild(document.createTextNode(label + " "));
  nums.forEach(function(num, i) {
    if (i) chip.appendChild(document.createTextNode(" "));
    var t = S.byNum[num];
    var cls = t ? (t.status === "out_of_scope" ? "oos" : t.status) : "blocked";
    var a = el("span", "tl " + cls, pad2(num));
    if (t) {
      a.title = t.title;
      a.onclick = function() { openPanel(t); };
    }
    chip.appendChild(a);
  });
  return chip;
}

export function fillPanel(n) {
  var p = document.getElementById("panel");
  var h = p.querySelector("h2"); h.innerHTML = ""; h.appendChild(el("span", "num", pad2(n.num))); h.appendChild(document.createTextNode(n.title));
  var meta = p.querySelector(".meta"); meta.innerHTML = "";
  meta.appendChild(el("span", "c " + (n.status === "out_of_scope" ? "oos" : n.status), n.status.replace(/_/g, " ")));
  if (n.type) meta.appendChild(el("span", "c type", n.type));
  if ((n.blockers || []).length) meta.appendChild(edgeChip("blocked by", n.blockers));
  var blocks = S.nodes.filter(function(o) { return (o.blockers || []).indexOf(n.num) >= 0; })
                      .map(function(o) { return o.num; });
  if (blocks.length) meta.appendChild(edgeChip("blocks", blocks));
  if (n.undermined) meta.appendChild(el("span", "c undermined", "undermined"));
  p.querySelector(".md").innerHTML = mdToHtml(n.body);
}

export function openPanel(n) {
  S.selected = n;
  // Ease the camera so the star sits centred in the space left of the panel.
  var p = document.getElementById("panel"), pw = p.offsetWidth || 0, vx = (innerWidth - pw) / 2, vy = innerHeight / 2;
  S.goal.x = vx - n.x * S.goal.s; S.goal.y = vy - n.y * S.goal.s;
  fillPanel(n); p.classList.add("open"); document.body.classList.add("panelopen");
}

export function closePanel() {
  S.selected = null;
  document.getElementById("panel").classList.remove("open");
  document.body.classList.remove("panelopen");
}

document.querySelector("#panel .x").onclick = closePanel;
// Cross-ticket links in the rendered body jump the star-map to that node.
document.querySelector("#panel .md").addEventListener("click", function(ev) {
  var a = ev.target.closest ? ev.target.closest("[data-goto]") : null;
  if (a) { ev.preventDefault(); var n = S.byNum[parseInt(a.getAttribute("data-goto"), 10)]; if (n) openPanel(n); }
});
