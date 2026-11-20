"""Antre 3 generate WAN 2.1 T2V (13 frame, 832x480) untuk aset NOTARIUS,
tunggu selesai, lalu salin frame terbaik ke web/assets/landing/.
Pakai: python run.py  (ComfyUI harus jalan di 127.0.0.1:8188)
"""
import json
import os
import shutil
import time
import urllib.request

COMFY = "http://127.0.0.1:8188"
OUT = r"D:\ComfyUI\output"
DEST = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
                    "..", "web", "assets", "landing")

JOBS = [
    ("hero",
     "Professional Indonesian notary office desk close-up, brass scales of justice, "
     "stack of legal deed documents, luxury fountain pen, warm morning window light, "
     "shallow depth of field, premium corporate photography, photorealistic, ultra detailed",
     21),
    ("office",
     "Modern notary office interior in dark navy and gold, professional team reviewing "
     "documents with laptop showing dashboard, floor-to-ceiling windows, city view, "
     "premium corporate photography, photorealistic, ultra detailed",
     22),
    ("sign",
     "Close-up of hands signing a notarial deed with gold pen, red wax seal stamp nearby, "
     "scales of justice softly blurred in background, warm premium light, "
     "professional legal photography, photorealistic, ultra detailed",
     23),
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


def wait_ready(timeout=300):
    t0 = time.time()
    while time.time() - t0 < timeout:
        try:
            urllib.request.urlopen(COMFY + "/system_stats", timeout=5).read()
            return True
        except Exception:
            time.sleep(5)
    return False


def main():
    assert wait_ready(), "ComfyUI tidak merespons di " + COMFY
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
        print("done:", name, hist[pid]["status"], flush=True)
        # salin frame tengah (paling stabil) sebagai aset final
        outs = hist[pid]["outputs"]
        for node_out in outs.values():
            for img in node_out.get("images", []):
                if img["filename"].startswith(prefix):
                    src = os.path.join(OUT, img.get("subfolder", ""), img["filename"])
                    if "_00007_" in img["filename"]:
                        dst = os.path.join(DEST, f"notarius-{name}.png")
                        shutil.copy2(src, dst)
                        print("saved:", dst, flush=True)
    print("ALL DONE ->", DEST)


if __name__ == "__main__":
    main()
