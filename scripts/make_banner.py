#!/usr/bin/env python3
# Делает картинку для репозитория: assets/banner.png (1600x520).
# Запускать из корня проекта: python scripts/make_banner.py
from PIL import Image, ImageDraw, ImageFont

W, H = 1600, 520
BG = (26, 30, 36)
CARD = (36, 41, 49)
LINE = (58, 65, 76)
TEXT = (238, 241, 245)
DIM = (150, 160, 172)
ACCENT = (122, 176, 226)
OK = (122, 196, 148)
BAD = (232, 138, 126)

img = Image.new("RGB", (W, H), BG)
d = ImageDraw.Draw(img)


def font(size):
    return ImageFont.truetype("internal/gui/InterVariable.ttf", size)


d.rectangle([0, 0, W, 6], fill=ACCENT)
d.rectangle([0, H - 6, W, H], fill=LINE)

d.text((70, 82), "Менеджер модов", font=font(64), fill=TEXT)
d.text((70, 162), "GTA San Andreas", font=font(64), fill=ACCENT)
d.text((70, 262), "Включай и выключай моды одним кликом.", font=font(30), fill=TEXT)
d.text((70, 306), "Видно, какие моды делят одни и те же файлы.", font=font(30), fill=TEXT)
d.text((70, 350), "Профили сборок: САМП, одиночка, всё сразу.", font=font(30), fill=TEXT)

x0, y0, x1, y1 = 940, 70, 1530, 450
d.rounded_rectangle([x0, y0, x1, y1], radius=16, fill=CARD, outline=LINE, width=2)

rows = [
    ("SilentPatch", "заменяет 4 стандартных файла", True, OK),
    ("Тюнинг тачек", "делит файлы с другими: 1", True, BAD),
    ("Пак машин", "1024 модели, текстуры", True, None),
    ("Ночной город", "заменяет файл игры", False, None),
]
ry = y0 + 26
for name, sub, on, mark in rows:
    d.rounded_rectangle([x0 + 20, ry, x1 - 20, ry + 74], radius=12, fill=(44, 50, 60))
    d.text((x0 + 42, ry + 12), name, font=font(26), fill=TEXT)
    col = mark or DIM
    d.text((x0 + 42, ry + 44), sub, font=font(18), fill=col)
    tw = 52
    tx = x1 - 92
    ty = ry + 26
    track = (122, 176, 226) if on else (78, 86, 98)
    d.rounded_rectangle([tx, ty, tx + tw, ty + 24], radius=12, fill=track)
    knob = tx + tw - 24 if on else tx + 2
    d.ellipse([knob, ty + 2, knob + 20, ty + 22], fill=(245, 247, 250))
    ry += 86

d.text((70, 430), "sa-mod-manager  ·  Windows  ·  MIT", font=font(24), fill=DIM)

img.save("assets/banner.png")
print("assets/banner.png готов")
