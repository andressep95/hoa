#!/bin/bash
# Downloads Oracle's pre-converted all-MiniLM-L12-v2 ONNX model (with tokenizer).
# Run once before `docker compose up` on a fresh setup.

MODEL_DIR="$(dirname "$0")/oracle/models"
MODEL_FILE="$MODEL_DIR/all_MiniLM_L12_v2.onnx"

if [ -f "$MODEL_FILE" ]; then
  echo "✓ Model already exists: $MODEL_FILE"
  exit 0
fi

mkdir -p "$MODEL_DIR"

echo "Downloading Oracle pre-converted all-MiniLM-L12-v2 model..."
curl -L -o "$MODEL_DIR/all_MiniLM_L12_v2_augmented.zip" \
  "https://adwc4pm.objectstorage.us-ashburn-1.oci.customer-oci.com/p/TtH6hL2y25EypZ0-rrczRZ1aXp7v1ONbRBfCiT-BDBN8WLKQ3lgyW6RxCfIFLdA6/n/adwc4pm/b/OML-ai-models/o/all_MiniLM_L12_v2_augmented.zip"

if [ $? -ne 0 ]; then
  echo "✗ Download failed"
  exit 1
fi

echo "Extracting..."
cd "$MODEL_DIR"
unzip -o all_MiniLM_L12_v2_augmented.zip
rm -f all_MiniLM_L12_v2_augmented.zip

# Find the .onnx file (may be nested)
ONNX_FILE=$(find . -name "*.onnx" | head -1)
if [ -n "$ONNX_FILE" ] && [ "$ONNX_FILE" != "./all_MiniLM_L12_v2.onnx" ]; then
  mv "$ONNX_FILE" all_MiniLM_L12_v2.onnx
fi

echo "✓ Model ready: $MODEL_FILE"
