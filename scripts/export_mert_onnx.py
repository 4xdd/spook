#!/usr/bin/env python3
"""Export MERT-v1-95M embedding model to ONNX for ONNX Runtime inference.

Produces a model that maps raw mono waveform [batch, samples] @ 24 kHz to a
768-d embedding [batch, hidden]. Matches Spook's native Go inference path
(input_values without Wav2Vec2FeatureExtractor normalization).

Usage:
    python3 scripts/export_mert_onnx.py [--input DIR] [--output PATH]
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import torch
from torch import nn


class MERTEmbed(nn.Module):
    """Mean-pooled track embedding from raw waveform."""

    def __init__(self, backbone: nn.Module) -> None:
        super().__init__()
        self.backbone = backbone

    def forward(self, input_values: torch.Tensor) -> torch.Tensor:
        out = self.backbone(input_values=input_values)
        hidden = out.last_hidden_state
        return hidden.mean(dim=1)


def export(input_dir: Path, output_path: Path, opset: int) -> None:
    from transformers import AutoModel

    bin_path = input_dir / "pytorch_model.bin"
    config_path = input_dir / "config.json"
    if not bin_path.exists() or not config_path.exists():
        from huggingface_hub import hf_hub_download

        print(f"downloading weights to {input_dir} ...")
        input_dir.mkdir(parents=True, exist_ok=True)
        hf_hub_download("m-a-p/MERT-v1-95M", "pytorch_model.bin", local_dir=str(input_dir))
        hf_hub_download("m-a-p/MERT-v1-95M", "config.json", local_dir=str(input_dir))

    print("loading MERT-v1-95M ...")
    backbone = AutoModel.from_pretrained(str(input_dir), trust_remote_code=True)
    backbone.eval()
    model = MERTEmbed(backbone)
    model.eval()

    # 12 s clip @ 24 kHz — same default as the embed worker.
    sample_len = 24_000 * 12
    dummy = torch.randn(1, sample_len, dtype=torch.float32)

    output_path.parent.mkdir(parents=True, exist_ok=True)
    print(f"exporting ONNX → {output_path} (opset {opset}) ...")
    torch.onnx.export(
        model,
        dummy,
        str(output_path),
        input_names=["input_values"],
        output_names=["embedding"],
        dynamic_axes={
            "input_values": {0: "batch", 1: "samples"},
            "embedding": {0: "batch"},
        },
        opset_version=opset,
        do_constant_folding=True,
    )

    try:
        import onnx

        onnx.checker.check_model(str(output_path))
    except ImportError:
        print("onnx package not installed; skipping checker")
    else:
        print("onnx checker OK")

    size_mb = output_path.stat().st_size / (1024 * 1024)
    print(f"wrote {output_path} ({size_mb:.1f} MiB)")


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
        default=Path.home() / ".local/share/spook/models/mert-v1-95m.onnx",
        help="output .onnx file path",
    )
    parser.add_argument("--opset", type=int, default=18, help="ONNX opset version")
    args = parser.parse_args()
    export(args.input, args.output, args.opset)
    return 0


if __name__ == "__main__":
    sys.exit(main())
