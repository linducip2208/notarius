"""Logo SaktiNotaris: 3 opsi emblem futuristik (huruf S, timbangan+circuit, shield+quill).
Pakai: D:\\python\\python.exe scripts/wan_gen/logo.py (ComfyUI :8188 jalan)
Output: web/assets/brand/saktinotaris-logo-{a,b,c}.png (frame tengah, PNG)
"""
import json
import os
import shutil
import time
import urllib.request

COMFY = "http://127.0.0.1:8188"
OUT = r"D:\ComfyUI\output"
DEST = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..",
                    "web", "assets", "brand")

JOBS = [
    ("logo-a", "Minimalist futuristic letter S logo mark, brushed gold metal with glowing cyan edges on deep navy background, premium tech emblem, symmetrical, centered, flat vector logo style, no text, ultra clean", 51),
    ("logo-b", "Abstract scales of justice merged with circuit board lines, gold and navy premium tech logo emblem on dark background, symmetrical, centered, minimal flat vector style, no text", 52),
    ("logo-c", "Futuristic shield emblem with elegant quill pen and subtle circuit pattern, navy and gold premium tech logo on dark background, centered, minimal flat vector style, no text", 53),
]

NEG = "blurry, low quality, watermark, text, letters, words, photo, realistic, noisy, asymmetric"


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
                         "latent_image": ["6", 0], "seed": seed, "steps": 25,
                         "cfg": 7.0, "sampler_name": "euler", "scheduler": "simple",
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
        prefix = "SNLOGO_" + name
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
                    dst = os.path.join(DEST, f"saktinotaris-{name}.png")
                    shutil.copy2(src, dst)
                    print("saved:", dst, os.path.getsize(dst) // 1024, "KB", flush=True)
    print("ALL DONE ->", DEST)


if __name__ == "__main__":
    main()
