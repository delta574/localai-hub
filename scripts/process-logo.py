from PIL import Image
import os

def remove_bg(img_path, out_path, threshold=240, feather=10):
    img = Image.open(img_path).convert("RGBA")
    pixels = img.load()
    w, h = img.size
    for y in range(h):
        for x in range(w):
            r, g, b, a = pixels[x, y]
            # If pixel is close to white, make transparent
            if r > threshold and g > threshold and b > threshold:
                pixels[x, y] = (r, g, b, 0)
    img.save(out_path)
    print(f"Saved transparent PNG: {out_path}")
    return img

def make_dark_variant(img_path, out_path):
    """Create a version suitable for dark backgrounds (add subtle light glow/border)"""
    img = Image.open(img_path).convert("RGBA")
    w, h = img.size
    # composite onto dark background
    dark_bg = Image.new("RGBA", (w, h), (15, 23, 42, 255))  # #0f172a
    composite = Image.alpha_composite(dark_bg, img)
    composite.save(out_path)
    print(f"Saved dark variant: {out_path}")

def make_og_image(logo_path, out_path):
    img = Image.open(logo_path).convert("RGBA")
    # Create 1280x640 gradient-like background (dark blue->teal)
    og = Image.new("RGBA", (1280, 640), (15, 23, 42, 255))
    # Resize logo to fit
    logo_w = int(og.width * 0.5)
    logo_h = int(img.height * (logo_w / img.width))
    logo_resized = img.resize((logo_w, logo_h), Image.Resampling.LANCZOS)
    # Center it
    x = (og.width - logo_w) // 2
    y = (og.height - logo_h) // 2
    og.paste(logo_resized, (x, y), logo_resized)
    og.save(out_path)
    print(f"Saved OG image: {out_path}")

base = "docs/logo"
remove_bg(f"{base}/logo-orig.png", f"{base}/logo.png", threshold=240)
make_dark_variant(f"{base}/logo.png", f"{base}/logo-dark.png")
make_og_image(f"{base}/logo.png", f"{base}/og-image.png")
