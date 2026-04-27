#!/usr/bin/env python3
"""Post-processor for E3 RTT × loss matrix.

Reads the latest run-*.json under eval/results/E3/ (or an explicit path)
and renders the 3×3 P99-latency heatmap for Fig. 4.

Cell value: P99 latency in ms. Cell annotation: "<P99 ms>\n<fast-path %>".
SLO bands: red where P99 > 2000 ms (clinical SLO breach per CLAUDE.md
§12); green where P99 < 500 ms (chart-pull SLO comfortable). Otherwise
the per-cell color follows the chosen sequential colormap (`plasma`).

Why `plasma` and not `viridis`: the SLO story is monotone in *latency*
(brighter = worse). `plasma` runs dark-purple → orange → yellow which
intuitively reads as cool → hot, matching the "WAN gets worse" axis.
`viridis` is dark → green → yellow, which is also fine but less
emotionally charged. The decision is documented here so the figure
caption can stay compact.

CLAUDE.md §8 contract: figure must be reproducible from the raw JSON.
Re-running on the same input produces byte-identical output.
"""
from __future__ import annotations
import argparse
import json
import sys
from pathlib import Path

import matplotlib
matplotlib.use("Agg")  # headless
import matplotlib.colors as mcolors
import matplotlib.pyplot as plt
import numpy as np


# SLO thresholds — CLAUDE.md §12 ("anchor every claim ... to a specific
# clinical SLO: radiology open <2 s; chart pull <500 ms"). The 2 s band
# is the IEC 60601-1-8-derived fast-path SLO; the 500 ms band is the
# chart-pull comfort threshold from the same paragraph.
SLO_BREACH_MS = 2000.0
SLO_COMFORT_MS = 500.0


def find_latest_run(out_dir: Path) -> Path:
    runs = sorted(out_dir.glob("run-*.json"))
    if not runs:
        raise FileNotFoundError(f"no run-*.json in {out_dir}")
    return runs[-1]  # timestamped prefix sorts lexicographically


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--run", type=Path, help="explicit run-*.json path")
    p.add_argument("--out-dir", type=Path, default=Path("eval/results/E3"),
                   help="search dir when --run is omitted")
    p.add_argument("--fig-dir", type=Path, default=Path("paper/figures"),
                   help="output directory for PDF + PNG")
    args = p.parse_args()

    run_path = args.run if args.run else find_latest_run(args.out_dir)
    rec = json.loads(run_path.read_text())

    cfg = rec.get("config", {})
    rtts = cfg.get("rtt_cells_ms", [10, 50, 200])
    losses = cfg.get("loss_cells", [0.0, 0.01, 0.05])
    slo_ms = cfg.get("slo_budget_ms", 2000)
    n_bundles = cfg.get("num_bundles", 0)
    reps = cfg.get("reps_per_cell", 0)

    # Index the cells into a (loss, rtt) grid. Loss is the row axis (so
    # the heatmap reads top-to-bottom = increasing loss); RTT is the
    # column axis (left-to-right = increasing RTT).
    cells = rec.get("cells", [])
    by_key: dict[tuple[int, float], dict] = {}
    for c in cells:
        by_key[(c["rtt_ms"], c["loss_rate"])] = c

    p99_grid = np.zeros((len(losses), len(rtts)), dtype=float)
    fast_grid = np.zeros((len(losses), len(rtts)), dtype=float)
    for i, loss in enumerate(losses):
        for j, rtt in enumerate(rtts):
            c = by_key.get((rtt, loss))
            if c is None:
                print(f"! missing cell rtt={rtt} loss={loss}", file=sys.stderr)
                p99_grid[i, j] = float("nan")
                fast_grid[i, j] = float("nan")
                continue
            st = c.get("stats", {})
            p99_grid[i, j] = float(st.get("p99_ms", float("nan")))
            fast_grid[i, j] = float(st.get("fast_path_pct", float("nan")))

    # Print 3×3 ASCII rendering. Useful for terminal sanity-check before
    # opening the PDF.
    print(f"E3 run: {rec.get('run_id', '?')}  ({n_bundles} bundles × {reps} reps × 9 cells)")
    print(f"  source: {run_path}")
    print(f"  WAN mechanism: {cfg.get('wan_mechanism', '?')}")
    print(f"  SLO budget: {slo_ms} ms")
    print()
    print("P99 latency (ms) and fast-path % matrix:")
    header = f"{'loss \\ rtt':<10}" + "".join(f"  {f'{r} ms':<22}" for r in rtts)
    print(header)
    for i, loss in enumerate(losses):
        row = f"{f'{int(loss*100)}%':<10}"
        for j, _rtt in enumerate(rtts):
            cell_str = f"P99={p99_grid[i, j]:>6.0f} fast={fast_grid[i, j]:>5.1f}%"
            row += f"  {cell_str:<22}"
        print(row)

    # --- Heatmap rendering -------------------------------------------------
    fig, ax = plt.subplots(figsize=(7.0, 4.6))

    # Use plasma; clip to the data range so colour-spread is informative.
    finite = p99_grid[np.isfinite(p99_grid)]
    if finite.size == 0:
        print("! no finite P99 values to plot", file=sys.stderr)
        return 1
    vmin = float(finite.min())
    vmax = float(finite.max())
    cmap = plt.get_cmap("plasma").copy()
    norm = mcolors.Normalize(vmin=vmin, vmax=vmax)

    im = ax.imshow(p99_grid, aspect="auto", cmap=cmap, norm=norm, origin="upper")

    # Axis ticks + labels.
    ax.set_xticks(np.arange(len(rtts)))
    ax.set_xticklabels([f"{r} ms" for r in rtts])
    ax.set_yticks(np.arange(len(losses)))
    ax.set_yticklabels([f"{int(l*100)}%" for l in losses])
    ax.set_xlabel("Added RTT (per Get, applied uniformly across tiers)")
    ax.set_ylabel("Per-Get loss probability")
    ax.set_title(
        f"LBVR-Med fast-path P99 — WAN conditions matrix\n"
        f"({n_bundles} bundles × {reps} reps per cell; SLO budget = {slo_ms} ms)"
    )

    # Cell annotations. Choose text colour for contrast: dark text on
    # light cells, white text on dark cells. The plasma colormap ramps
    # from very dark to bright yellow, so simple normalisation works.
    for i in range(len(losses)):
        for j in range(len(rtts)):
            v = p99_grid[i, j]
            f = fast_grid[i, j]
            if not np.isfinite(v):
                ax.text(j, i, "n/a", ha="center", va="center", fontsize=10, color="white")
                continue
            colour = "white" if norm(v) > 0.55 else "white" if norm(v) > 0.4 else "black"
            ax.text(
                j, i,
                f"{v:.0f} ms\n{f:.1f}% fast",
                ha="center", va="center", fontsize=10, color=colour, fontweight="bold",
            )

    # SLO band shading: overlay a translucent rectangle on cells that
    # breach (red) or comfortably meet (green) the clinical SLO. Rendered
    # AFTER imshow so the band is visible; alpha kept low so the colour
    # ramp underneath still reads.
    for i in range(len(losses)):
        for j in range(len(rtts)):
            v = p99_grid[i, j]
            if not np.isfinite(v):
                continue
            if v > SLO_BREACH_MS:
                ax.add_patch(plt.Rectangle(
                    (j - 0.5, i - 0.5), 1, 1,
                    fill=True, facecolor="red", alpha=0.22,
                    edgecolor="darkred", linewidth=2.0, zorder=3,
                ))
            elif v < SLO_COMFORT_MS:
                ax.add_patch(plt.Rectangle(
                    (j - 0.5, i - 0.5), 1, 1,
                    fill=True, facecolor="green", alpha=0.22,
                    edgecolor="darkgreen", linewidth=2.0, zorder=3,
                ))

    # Colorbar.
    cbar = fig.colorbar(im, ax=ax, fraction=0.05, pad=0.04)
    cbar.set_label("P99 retrieval latency (ms)")

    # Legend explaining SLO bands.
    breach_patch = plt.Rectangle((0, 0), 1, 1, facecolor="red", alpha=0.22,
                                 edgecolor="darkred", linewidth=2.0,
                                 label=f"P99 > {int(SLO_BREACH_MS)} ms (SLO breach)")
    comfort_patch = plt.Rectangle((0, 0), 1, 1, facecolor="green", alpha=0.22,
                                  edgecolor="darkgreen", linewidth=2.0,
                                  label=f"P99 < {int(SLO_COMFORT_MS)} ms (chart-pull comfort)")
    ax.legend(handles=[breach_patch, comfort_patch],
              loc="upper center", bbox_to_anchor=(0.5, -0.18),
              ncol=2, frameon=True, fontsize=9)

    fig.tight_layout()

    args.fig_dir.mkdir(parents=True, exist_ok=True)
    pdf_path = args.fig_dir / "E3_rtt_loss_heatmap.pdf"
    png_path = args.fig_dir / "E3_rtt_loss_heatmap.png"
    fig.savefig(pdf_path, format="pdf", bbox_inches="tight")
    fig.savefig(png_path, format="png", dpi=150, bbox_inches="tight")
    plt.close(fig)
    print()
    print(f"-> wrote {pdf_path}")
    print(f"-> wrote {png_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
