"""Generate 8 ilustrasi fitur NOTARIUS via WAN 2.1 + konversi JPEG ringan.
Pakai: D:\\python\\python.exe scripts/wan_gen/feat.py (ComfyUI :8188 jalan)
Output: web/assets/landing/notarius-feat-*.jpg (q82, progresif)
"""
import json
import os
import time
import urllib.request

from PIL import Image

COMFY = "http://127.0.0.1:8188"
OUT = r"D:\ComfyUI\output"
DEST = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..",
                    "web", "assets", "landing")

JOBS = [
    ("feat-klien", "Professional notary staff verifying client identity documents at elegant desk, KTP cards and folders, warm premium light, photorealistic", 31),
    ("feat-akta", "Notary round seal stamp pressed on official deed document beside gold pen, scales of justice blurred background, premium legal photography, photorealistic", 32),
    ("feat-repertorium", "Grand archive library shelves full of labeled navy and gold legal binders, ladder, warm light, premium photography, photorealistic", 33),
    ("feat-portal", "Laptop and smartphone showing client dashboard on notary desk, login screen glow, dark navy office evening, premium photography, photorealistic", 34),
    ("feat-keuangan", "Invoice documents with calculator and fountain pen on dark wood desk, gold coins stack blurred, premium finance photography, photorealistic", 35),
    ("feat-laporan", "Computer monitor displaying business bar charts and reports in elegant notary office, navy and gold interior, premium photography, photorealistic", 36),
    ("feat-keamanan", "Brass padlock resting on stack of confidential legal documents tied with ribbon, shield bokeh background, premium security photography, photorealistic", 37),
    ("feat-dokumen", "Neatly organized archive boxes and folders with labels on shelves, document storage room, warm light, premium photography, photorealistic", 38),
]

NEG = ("blurry, low quality, watermark, text, logo, cartoon, painting, illustration, "
       "distorted hands, extra fingers, deformed, ugly, noisy")


def workflow(prompt, seed, prefix):
    return {
        "1": {"class_type": "UNETLoader",
              "inputs": {"unet_name": "wan2.1_t2v_1.3B_fp16.safetensors",
                         "weight_dtype": "default"}},
        "2": {"class_type": "CLIPLoader",
              "inputs": {"clip_name": "umt5_xxl_fp8_e4m3fn_scaled.safetensors",
                         "type": "wan", "device": "default"}},
        "3": {"class_type": "VAELoader",
              "inputs": {"vae_name": "wan_2.1_vae.safetensors"}},
        "4": {"class_type": "CLIPTextEncode",
              "inputs": {"text": prompt, "clip": ["2", 0]}},
        "5": {"class_type": "CLIPTextEncode",
              "inputs": {"text": NEG, "clip": ["2", 0]}},
        "6": {"class_type": "EmptyHunyuanLatentVideo",
              "inputs": {"width": 832, "height": 480, "length": 13, "batch_size": 1}},
        "7": {"class_type": "ModelSamplingSD3",
              "inputs": {"model": ["1", 0], "shift": 8.0}},
        "8": {"class_type": "KSampler",
              "inputs": {"model": ["7", 0], "positive": ["4", 0], "negative": ["5", 0],
                         "latent_image": ["6", 0], "seed": seed, "steps": 20,
                         "cfg": 6.0, "sampler_name": "euler", "scheduler": "simple",
                         "denoise": 1.0}},
        "9": {"class_type": "VAEDecode",
              "inputs": {"samples": ["8", 0], "vae": ["3", 0]}},
        "10": {"class_type": "SaveImage",
               "inputs": {"images": ["9", 0], "filename_prefix": prefix}},
    }


def api(path, data=None):
    req = urllib.request.Request(
        COMFY + path,
        data=json.dumps(data).encode() if data is not None else None,
        headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=600) as r:
        return json.load(r)


def main():
    os.makedirs(DEST, exist_ok=True)
    for name, prompt, seed in JOBS:
        prefix = "NOTARIUS_" + name
        print("queue:", name, flush=True)
        pid = api("/prompt", {"prompt": workflow(prompt, seed, prefix)})["prompt_id"]
        while True:
            time.sleep(10)
            try:
                hist = api(f"/history/{pid}")
            except Exception:
                continue
            if pid in hist:
                break
        print("done:", name, flush=True)
        for node_out in hist[pid]["outputs"].values():
            for img in node_out.get("images", []):
                if img["filename"].startswith(prefix) and "_00007_" in img["filename"]:
                    src = os.path.join(OUT, img.get("subfolder", ""), img["filename"])
                    dst = os.path.join(DEST, f"notarius-{name}.jpg")
                    im = Image.open(src).convert("RGB")
                    im.save(dst, "JPEG", quality=82, progressive=True, optimize=True)
                    print("saved:", dst, os.path.getsize(dst) // 1024, "KB", flush=True)
    print("ALL DONE ->", DEST)


if __name__ == "__main__":
    main()
