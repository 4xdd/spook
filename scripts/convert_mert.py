#!/usr/bin/env python3
"""Convert MERT-v1-95M PyTorch weights to Spook's native .mert format.

Usage:
    python3 scripts/convert_mert.py [--input DIR] [--output PATH] [--bits 8]

Downloads pytorch_model.bin from Hugging Face if missing. Quantizes linear and
conv weights to int8 (symmetric, per output channel). Biases and norm params
stay float32.
"""

from __future__ import annotations

import argparse
import json
import struct
import sys
from pathlib import Path

import numpy as np
import torch

MAGIC = b"MERT"
VERSION = 1
DTYPE_F32 = 0
DTYPE_I8 = 1

# Tensors stored as float32 (norms, biases, weight_norm g).
ALWAYS_F32_SUFFIXES = (
    ".bias",
    ".weight_g",
    ".layer_norm.weight",
    ".layer_norm.bias",
    "feature_projection.layer_norm.weight",
    "feature_projection.layer_norm.bias",
    "encoder.layer_norm.weight",
    "encoder.layer_norm.bias",
)


def should_quantize(name: str, arr: np.ndarray, bits: int) -> bool:
    if bits != 8:
        return False
    if arr.dtype != np.float32:
        return False
    if name.endswith(ALWAYS_F32_SUFFIXES):
        return False
    if name.endswith(".weight") or name.endswith(".weight_v"):
        return arr.ndim >= 2
    return False


def quantize_per_output_channel(weight: np.ndarray) -> tuple[np.ndarray, np.ndarray]:
    """Symmetric int8 quant along the first dimension (output channels)."""
    w = weight.astype(np.float32)
    out_dim = w.shape[0]
    flat = w.reshape(out_dim, -1)
    scales = np.max(np.abs(flat), axis=1)
    scales = np.maximum(scales, 1e-8)
    q = np.round(flat / scales[:, None] * 127.0).astype(np.int8)
    q = q.reshape(w.shape)
    return q, scales.astype(np.float32)


def precompute_pos_conv_weight(state: dict[str, torch.Tensor]) -> np.ndarray:
    wg = state["encoder.pos_conv_embed.conv.weight_g"].numpy()
    wv = state["encoder.pos_conv_embed.conv.weight_v"].numpy()
    norm = np.linalg.norm(wv, axis=(0, 1), keepdims=True)
    norm = np.maximum(norm, 1e-8)
    return (wg * wv / norm).astype(np.float32)


def build_tensor_map(state: dict[str, torch.Tensor]) -> dict[str, np.ndarray]:
    out: dict[str, np.ndarray] = {}
    for key, tensor in state.items():
        if key == "masked_spec_embed":
            continue  # training-only
        if key == "encoder.pos_conv_embed.conv.weight_v":
            continue  # folded into effective weight
        if key == "encoder.pos_conv_embed.conv.weight_g":
            out["encoder.pos_conv_embed.conv.weight"] = precompute_pos_conv_weight(state)
            continue
        out[key] = tensor.detach().cpu().numpy().astype(np.float32)
    return out


def write_tensor(f, name: str, arr: np.ndarray, bits: int) -> None:
    name_b = name.encode("utf-8")
    if len(name_b) > 65535:
        raise ValueError(f"tensor name too long: {name}")

    ndim = arr.ndim
    if ndim > 4:
        raise ValueError(f"unsupported ndim {ndim} for {name}")

    if should_quantize(name, arr, bits):
        q, scales = quantize_per_output_channel(arr)
        f.write(struct.pack("<H", len(name_b)))
        f.write(name_b)
        f.write(struct.pack("<B", ndim))
        dims = list(arr.shape) + [0] * (4 - ndim)
        f.write(struct.pack("<4I", *dims[:4]))
        f.write(struct.pack("<B", DTYPE_I8))
        f.write(struct.pack("<I", len(scales)))
        f.write(scales.astype("<f4").tobytes())
        f.write(struct.pack("<I", q.size))
        f.write(q.astype(np.int8).tobytes())
        return

    flat = arr.astype("<f4", copy=False)
    f.write(struct.pack("<H", len(name_b)))
    f.write(name_b)
    f.write(struct.pack("<B", ndim))
    dims = list(arr.shape) + [0] * (4 - ndim)
    f.write(struct.pack("<4I", *dims[:4]))
    f.write(struct.pack("<B", DTYPE_F32))
    f.write(struct.pack("<I", 0))  # no scales
    f.write(struct.pack("<I", flat.size))
    f.write(flat.tobytes())


def convert(input_dir: Path, output_path: Path, bits: int) -> None:
    bin_path = input_dir / "pytorch_model.bin"
    config_path = input_dir / "config.json"
    if not bin_path.exists():
        from huggingface_hub import hf_hub_download

        print(f"downloading weights to {input_dir} ...")
        hf_hub_download("m-a-p/MERT-v1-95M", "pytorch_model.bin", local_dir=str(input_dir))
    if not config_path.exists():
        from huggingface_hub import hf_hub_download

        hf_hub_download("m-a-p/MERT-v1-95M", "config.json", local_dir=str(input_dir))

    config = json.loads(config_path.read_text())
    state = torch.load(bin_path, map_location="cpu", weights_only=True)
    tensors = build_tensor_map(state)

    output_path.parent.mkdir(parents=True, exist_ok=True)
    config_bytes = json.dumps(config, separators=(",", ":")).encode("utf-8")

    with output_path.open("wb") as f:
        f.write(MAGIC)
        f.write(struct.pack("<III", VERSION, len(config_bytes), len(tensors)))
        f.write(b"\x00" * 16)
        f.write(config_bytes)
        for name in sorted(tensors.keys()):
            write_tensor(f, name, tensors[name], bits)

    size_mb = output_path.stat().st_size / (1024 * 1024)
    print(f"wrote {output_path} ({size_mb:.1f} MiB, {len(tensors)} tensors, {bits}-bit)")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--input",
        type=Path,
        default=Path("/tmp/mert-work/MERT-v1-95M"),
        help="directory containing pytorch_model.bin and config.json",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("/tmp/mert-work/mert-v1-95m.mert"),
        help="output .mert file path",
    )
    parser.add_argument("--bits", type=int, default=8, choices=(8, 32), help="quantization bits")
    args = parser.parse_args()
    convert(args.input, args.output, args.bits)
    return 0


if __name__ == "__main__":
    sys.exit(main())
