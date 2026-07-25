ARG FRIGATE_VERSION=0.17.1
ARG MODEL_SIZE=m
ARG IMG_SIZE=320

FROM python:3.11 AS model-prep

ARG MODEL_SIZE
ARG IMG_SIZE

RUN <<'EOF'
set -e
apt-get update
apt-get install --no-install-recommends -y cmake libgl1
rm -rf /var/lib/apt/lists/*
EOF
COPY --from=ghcr.io/astral-sh/uv:0.10.4 /uv /bin/
COPY extract-coco-labels.py /

RUN <<'EOF'
set -e
mkdir -p /models /yolo
cd /yolo
git clone https://github.com/WongKinYiu/yolov9.git .
curl -fsSL -o yolov9-${MODEL_SIZE}-converted.pt https://github.com/WongKinYiu/yolov9/releases/download/v0.1/yolov9-${MODEL_SIZE}-converted.pt
/bin/uv pip install --system -r requirements.txt onnx==1.18.0 onnxruntime onnxsim onnxscript
sed -i "s/ckpt = torch.load(attempt_download(w), map_location='cpu')/ckpt = torch.load(attempt_download(w), map_location='cpu', weights_only=False)/g" models/experimental.py
python3 export.py --weights ./yolov9-${MODEL_SIZE}-converted.pt --imgsz ${IMG_SIZE} --simplify --include onnx
mv yolov9-${MODEL_SIZE}*.onnx /models/yolov9-${MODEL_SIZE}-${IMG_SIZE}.onnx
python3 /extract-coco-labels.py data/coco.yaml /models/coco.txt
rm -rf /yolo
EOF

FROM ghcr.io/blakeblackshear/frigate:${FRIGATE_VERSION}

ARG MODEL_SIZE
ARG IMG_SIZE

RUN <<'EOF'
rm -rf /openvino-model/
EOF

COPY --from=model-prep /models/yolov9-${MODEL_SIZE}-${IMG_SIZE}.onnx /models/yolov9-${MODEL_SIZE}-${IMG_SIZE}.onnx
COPY --from=model-prep /models/coco.txt /models/coco.txt
