"""Gambar tutorial: KTP masuk -> OCR -> review -> dokumen legalitas sah.
Pakai: D:\\python\\python.exe scripts/wan_gen/docs.py (ComfyUI :8188 jalan)
Output: web/assets/docs/notarius-doc-*.jpg (q82)
"""
import json
import os
import time
import urllib.request

from PIL import Image

COMFY = "http://127.0.0.1:8188"
OUT = r"D:\ComfyUI\output"
DEST = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..",
                    "web", "assets", "docs")

JOBS = [
    ("doc-ktp", "Indonesian national ID card KTP lying on notary desk next to smartphone taking photo of it, shallow depth of field, warm office light, premium photography, photorealistic", 41),
    ("doc-ocr", "Flatbed scanner light beam moving over stacked legal documents, subtle digital grid overlay, dark modern office, premium technology photography, photorealistic", 42),
    ("doc-review", "Notary staff comparing paper dossier with computer screen showing verification form, checkmarks, office desk, warm light, premium photography, photorealistic", 43),
    ("doc-legal", "Signed notarial deed with red wax seal and gold round stamp, scales of justice softly blurred behind, warm premium light, professional legal photography, photorealistic", 44),
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
                    Image.open(src).convert("RGB").save(
                        dst, "JPEG", quality=82, progressive=True, optimize=True)
                    print("saved:", dst, os.path.getsize(dst) // 1024, "KB", flush=True)
    print("ALL DONE ->", DEST)


if __name__ == "__main__":
    main()
