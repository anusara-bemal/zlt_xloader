#!/usr/bin/env python3
"""Dependency-free generator for the README illustration assets.

Uses only the Python standard library to emit PNGs (zlib + struct) so the
repository can rebuild its own images without any external tool or service.

Run:  python docs/assets/generate_assets.py
"""

import math
import os
import struct
import zlib

HERE = os.path.dirname(os.path.abspath(__file__))

# Minimal 5x7 bitmap font: uppercase, digits and the punctuation used in the
# terminal preview. Lowercase input is uppercased by draw_text().
FONT = {
    " ": ["00000"] * 7,
    "A": ["01110", "10001", "10001", "11111", "10001", "10001", "10001"],
    "B": ["11110", "10001", "10001", "11110", "10001", "10001", "11110"],
    "C": ["01111", "10000", "10000", "10000", "10000", "10000", "01111"],
    "D": ["11110", "10001", "10001", "10001", "10001", "10001", "11110"],
    "E": ["11111", "10000", "10000", "11110", "10000", "10000", "11111"],
    "F": ["11111", "10000", "10000", "11110", "10000", "10000", "10000"],
    "G": ["01111", "10000", "10000", "10111", "10001", "10001", "01111"],
    "H": ["10001", "10001", "10001", "11111", "10001", "10001", "10001"],
    "I": ["11111", "00100", "00100", "00100", "00100", "00100", "11111"],
    "J": ["00111", "00010", "00010", "00010", "00010", "10010", "01100"],
    "K": ["10001", "10010", "10100", "11000", "10100", "10010", "10001"],
    "L": ["10000", "10000", "10000", "10000", "10000", "10000", "11111"],
    "M": ["10001", "11011", "10101", "10101", "10001", "10001", "10001"],
    "N": ["10001", "11001", "10101", "10011", "10001", "10001", "10001"],
    "O": ["01110", "10001", "10001", "10001", "10001", "10001", "01110"],
    "P": ["11110", "10001", "10001", "11110", "10000", "10000", "10000"],
    "Q": ["01110", "10001", "10001", "10001", "10101", "10010", "01101"],
    "R": ["11110", "10001", "10001", "11110", "10100", "10010", "10001"],
    "S": ["01111", "10000", "10000", "01110", "00001", "00001", "11110"],
    "T": ["11111", "00100", "00100", "00100", "00100", "00100", "00100"],
    "U": ["10001", "10001", "10001", "10001", "10001", "10001", "01110"],
    "V": ["10001", "10001", "10001", "10001", "10001", "01010", "00100"],
    "W": ["10001", "10001", "10001", "10101", "10101", "10101", "01010"],
    "X": ["10001", "10001", "01010", "00100", "01010", "10001", "10001"],
    "Y": ["10001", "10001", "01010", "00100", "00100", "00100", "00100"],
    "Z": ["11111", "00001", "00010", "00100", "01000", "10000", "11111"],
    "0": ["01110", "10011", "10011", "10101", "11001", "11001", "01110"],
    "1": ["00100", "01100", "00100", "00100", "00100", "00100", "01110"],
    "2": ["01110", "10001", "00001", "00010", "00100", "01000", "11111"],
    "3": ["11110", "00001", "00001", "01110", "00001", "00001", "11110"],
    "4": ["00010", "00110", "01010", "10010", "11111", "00010", "00010"],
    "5": ["11111", "10000", "10000", "11110", "00001", "00001", "11110"],
    "6": ["00110", "01000", "10000", "11110", "10001", "10001", "01110"],
    "7": ["11111", "00001", "00010", "00100", "01000", "01000", "01000"],
    "8": ["01110", "10001", "10001", "01110", "10001", "10001", "01110"],
    "9": ["01110", "10001", "10001", "01111", "00001", "00001", "01110"],
    ".": ["00000", "00000", "00000", "00000", "00000", "01100", "01100"],
    ":": ["00000", "01100", "01100", "00000", "01100", "01100", "00000"],
    "/": ["00001", "00001", "00010", "00100", "01000", "10000", "10000"],
    "-": ["00000", "00000", "00000", "11111", "00000", "00000", "00000"],
    "_": ["00000", "00000", "00000", "00000", "00000", "00000", "11111"],
    ">": ["10000", "01000", "00100", "00010", "00100", "01000", "10000"],
    "$": ["00100", "01111", "10100", "01110", "00101", "11110", "00100"],
    "#": ["01010", "01010", "11111", "01010", "11111", "01010", "01010"],
}

CYAN = (34, 211, 238)
MAGENTA = (236, 72, 153)
GREEN = (74, 222, 128)
GRAY = (110, 125, 145)
WHITE = (226, 232, 240)
DARK = (11, 15, 25)


class Canvas:
    def __init__(self, w, h, fill=(0, 0, 0, 255)):
        self.w, self.h = w, h
        self.px = [[fill[0], fill[1], fill[2], fill[3]] for _ in range(w * h)]

    def set(self, x, y, color):
        if 0 <= x < self.w and 0 <= y < self.h:
            if len(color) == 3:
                color = (color[0], color[1], color[2], 255)
            self.px[y * self.w + x] = list(color)

    def gradient(self, top, bottom):
        for y in range(self.h):
            t = y / max(1, self.h - 1)
            c = tuple(int(top[i] + (bottom[i] - top[i]) * t) for i in range(3))
            for x in range(self.w):
                self.set(x, y, c + (255,))

    def glow(self, cx, cy, radius, color, strength=0.55):
        r2 = radius * radius
        for y in range(max(0, cy - radius), min(self.h, cy + radius)):
            for x in range(max(0, cx - radius), min(self.w, cx + radius)):
                d2 = (x - cx) ** 2 + (y - cy) ** 2
                if d2 >= r2:
                    continue
                a = strength * (1 - d2 / r2) ** 2
                old = self.px[y * self.w + x]
                self.set(x, y, tuple(int(old[i] + (color[i] - old[i]) * a) for i in range(3)) + (255,))

    def rect(self, x, y, w, h, color):
        for yy in range(int(y), int(y + h)):
            for xx in range(int(x), int(x + w)):
                self.set(xx, yy, color)

    def rounded(self, x, y, w, h, r, color):
        for yy in range(int(y), int(y + h)):
            for xx in range(int(x), int(x + w)):
                dx = max(x + r - xx, 0, xx - (x + w - r))
                dy = max(y + r - yy, 0, yy - (y + h - r))
                if dx * dx + dy * dy <= r * r:
                    self.set(xx, yy, color)

    def circle(self, cx, cy, r, color):
        for yy in range(int(cy - r), int(cy + r + 1)):
            for xx in range(int(cx - r), int(cx + r + 1)):
                if (xx - cx) ** 2 + (yy - cy) ** 2 <= r * r:
                    self.set(xx, yy, color)

    def ring(self, cx, cy, r, thickness, color):
        inner = (r - thickness) ** 2
        outer = r * r
        for yy in range(int(cy - r), int(cy + r + 1)):
            for xx in range(int(cx - r), int(cx + r + 1)):
                d2 = (xx - cx) ** 2 + (yy - cy) ** 2
                if inner <= d2 <= outer:
                    self.set(xx, yy, color)

    def line(self, x0, y0, x1, y1, width, color):
        steps = int(max(abs(x1 - x0), abs(y1 - y0)))
        for i in range(steps + 1):
            t = i / max(1, steps)
            x = x0 + (x1 - x0) * t
            y = y0 + (y1 - y0) * t
            for dy in range(-width // 2, width // 2 + 1):
                for dx in range(-width // 2, width // 2 + 1):
                    self.set(int(x + dx), int(y + dy), color)

    def text(self, x, y, s, scale=2, color=WHITE, spacing=1):
        for ch in s.upper():
            glyph = FONT.get(ch, FONT[" "])
            for row, bits in enumerate(glyph):
                for col, bit in enumerate(bits):
                    if bit == "1":
                        self.rect(x + col * scale, y + row * scale, scale, scale, color)
            x += (5 + spacing) * scale

    def save(self, path):
        raw = bytearray()
        for y in range(self.h):
            raw.append(0)
            for x in range(self.w):
                raw.extend(self.px[y * self.w + x])

        def chunk(tag, data):
            return (struct.pack(">I", len(data)) + tag + data +
                    struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF))

        out = b"\x89PNG\r\n\x1a\n"
        out += chunk(b"IHDR", struct.pack(">IIBBBBB", self.w, self.h, 8, 6, 0, 0, 0))
        out += chunk(b"IDAT", zlib.compress(bytes(raw), 9))
        out += chunk(b"IEND", b"")
        with open(path, "wb") as f:
            f.write(out)
        print("wrote", os.path.basename(path), f"{self.w}x{self.h}")


def router(c, cx, cy, scale=1.0, antenna_h=70):
    """Draw a stylised white LTE router centred on (cx, cy)."""
    w, h = int(150 * scale), int(96 * scale)
    x, y = int(cx - w / 2), int(cy - h / 2)
    # shadow
    c.rounded(x + 8, y + h + 4, w - 16, 12, 6, (0, 0, 0, 255))
    # antennas
    for sx in (x + int(28 * scale), x + int(w - 28 * scale)):
        c.line(sx, y + 4, sx - int(12 * scale), y - int(antenna_h * scale),
               max(2, int(9 * scale)), (226, 232, 240, 255))
        c.circle(int(sx - int(12 * scale)), int(y - int(antenna_h * scale)),
                 max(2, int(5 * scale)), (200, 210, 225, 255))
    # body
    c.rounded(x, y, w, h, int(18 * scale), (233, 238, 245, 255))
    c.rounded(x, y + h - int(22 * scale), w, int(22 * scale), int(12 * scale), (205, 214, 228, 255))
    # front LEDs
    for i in range(4):
        c.circle(int(x + w / 2 - 30 * scale + i * 20 * scale), int(y + h - 11 * scale),
                 max(2, int(3 * scale)), CYAN if i == 0 else (120, 140, 165, 255))


def make_logo():
    c = Canvas(256, 256, DARK + (255,))
    c.glow(128, 120, 110, CYAN, 0.22)
    router(c, 128, 150, scale=1.05, antenna_h=58)
    # open padlock in front
    c.rounded(104, 132, 48, 40, 8, CYAN + (255,))
    c.ring(128, 130, 18, 7, CYAN + (255,))
    c.rect(112, 112, 34, 22, DARK + (255,))
    c.circle(128, 152, 5, DARK + (255,))
    c.save(os.path.join(HERE, "logo.png"))


def make_hero():
    c = Canvas(760, 520)
    c.gradient((13, 19, 33), (5, 8, 15))
    c.glow(380, 250, 300, (20, 90, 130), 0.35)
    c.glow(600, 120, 180, MAGENTA, 0.10)
    router(c, 380, 310, scale=2.0, antenna_h=70)
    c.save(os.path.join(HERE, "hero-router.png"))


def make_cli():
    c = Canvas(880, 400, (1, 4, 10, 255))
    c.glow(440, 200, 420, (15, 45, 70), 0.35)
    # window
    c.rounded(24, 20, 832, 360, 14, (13, 17, 23, 255))
    c.rounded(24, 20, 832, 44, 14, (22, 27, 34, 255))
    c.rect(24, 48, 832, 16, (22, 27, 34, 255))
    for i, col in enumerate(((248, 113, 113), (248, 196, 91), (100, 214, 140))):
        c.circle(48 + i * 24, 42, 7, col + (255,))
    c.text(700, 34, "ZLT-UNLOCKER", scale=2, color=GRAY)
    c.text(52, 82, "$ ./ZLT-UNLOCKER --TARGET 192.168.1.1 SCAN", scale=2, color=CYAN)

    lines = [
        ("TARGET: 192.168.1.1  (PROFILE: TOZED ZLT S20 1.12.7)", WHITE),
        ("  TCP/23     CLOSED", GRAY),
        ("  TCP/80     OPEN", GREEN),
        ("  TCP/8080   CLOSED", GRAY),
        ("  TCP/7547   OPEN", GREEN),
        ("  PROFILE    TOZED ZLT S20 1.12.7", WHITE),
        ("  FIRMWARE   S20_1.12.7", WHITE),
        ("  MODEL      ZLT S20", WHITE),
        ("  DRY_RUN    TRUE", CYAN),
    ]
    y = 116
    for text, color in lines:
        c.text(52, y, text, scale=2, color=color)
        y += 28
    c.save(os.path.join(HERE, "cli-preview.png"))


if __name__ == "__main__":
    make_logo()
    make_hero()
    make_cli()
