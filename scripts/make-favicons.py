from PIL import Image
import struct, io

def make_ico(png_path, out_path, sizes=[16, 32, 48]):
    img = Image.open(png_path).convert("RGBA")
    icons = []
    for s in sizes:
        resized = img.resize((s, s), Image.Resampling.LANCZOS)
        buf = io.BytesIO()
        resized.save(buf, format="PNG")
        icons.append((s, buf.getvalue()))

    # ICO header
    header = struct.pack("<HHH", 0, 1, len(icons))
    entries = b""
    data = b""
    offset = 6 + 16 * len(icons)
    for s, png_data in icons:
        w = 0 if s >= 256 else s
        h = 0 if s >= 256 else s
        entries += struct.pack("<BBBBHHII", w, h, 0, 0, 1, 32, len(png_data), offset)
        data += png_data
        offset += len(png_data)

    with open(out_path, "wb") as f:
        f.write(header + entries + data)
    print(f"Saved ICO: {out_path}")

def make_png(img_path, out_path, size):
    img = Image.open(img_path).convert("RGBA")
    resized = img.resize((size, size), Image.Resampling.LANCZOS)
    resized.save(out_path)
    print(f"Saved PNG {size}: {out_path}")

base = "docs/logo"
logo = f"{base}/logo.png"

# Favicons
make_ico(logo, f"{base}/favicon.ico", [16, 32, 48])
make_png(logo, f"{base}/favicon-16.png", 16)
make_png(logo, f"{base}/favicon-32.png", 32)
make_png(logo, f"{base}/favicon-192.png", 192)
make_png(logo, f"{base}/favicon-512.png", 512)
make_png(logo, f"{base}/apple-touch-icon.png", 180)
