"""Make Raven logo backgrounds transparent (alpha)."""
from __future__ import annotations

from collections import deque
from pathlib import Path

from PIL import Image


def flood_transparent(op, w: int, h: int, starts: list[tuple[int, int]], predicate) -> None:
    visited = [[False] * w for _ in range(h)]
    q: deque[tuple[int, int]] = deque(starts)
    while q:
        x, y = q.popleft()
        if x < 0 or y < 0 or x >= w or y >= h or visited[y][x]:
            continue
        visited[y][x] = True
        r, g, b, a = op[x, y]
        if a == 0:
            q.extend([(x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)])
            continue
        if not predicate(r, g, b):
            continue
        op[x, y] = (r, g, b, 0)
        q.extend([(x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)])


def edge_neighbors(op, w: int, h: int) -> list[tuple[int, int]]:
    starts: list[tuple[int, int]] = []
    for x in range(w):
        for y in range(h):
            if op[x, y][3] != 0:
                continue
            for nx, ny in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)):
                if 0 <= nx < w and 0 <= ny < h and op[nx, ny][3] != 0:
                    starts.append((nx, ny))
    return starts


def make_transparent(src: Path) -> None:
    im = Image.open(src).convert("RGBA")
    op = im.load()
    w, h = im.size
    corners = [op[0, 0], op[w - 1, 0], op[0, h - 1], op[w - 1, h - 1]]
    flood_transparent(
        op,
        w,
        h,
        [(0, 0), (w - 1, 0), (0, h - 1), (w - 1, h - 1)],
        lambda r, g, b: max(r, g, b) < 30,
    )
    flood_transparent(op, w, h, edge_neighbors(op, w, h), lambda r, g, b: r > 230 and g > 230 and b > 230)
    flood_transparent(
        op,
        w,
        h,
        edge_neighbors(op, w, h),
        lambda r, g, b: max(r, g, b) < 42 and abs(r - g) < 12 and abs(g - b) < 18,
    )
    # Outer-band near-white (light theme squircle with no black fringe).
    if sum(c[0] for c in corners) / 4 > 200:
        margin = int(min(w, h) * 0.08)
        band: list[tuple[int, int]] = []
        for x in range(w):
            for y in range(h):
                if x < margin or y < margin or x >= w - margin or y >= h - margin:
                    r, g, b, a = op[x, y]
                    if a and r > 230 and g > 230 and b > 230:
                        band.append((x, y))
        flood_transparent(op, w, h, band, lambda r, g, b: r > 230 and g > 230 and b > 230)
    im.save(src)
    print(f"wrote {src}")


def main() -> None:
    base = Path(__file__).resolve().parents[1] / "webui" / "public"
    for name in ("dark-theme-logo.png", "white-theme-logo.png"):
        make_transparent(base / name)


if __name__ == "__main__":
    main()
