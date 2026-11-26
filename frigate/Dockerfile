ARG FRIGATE_VERSION=0.18.3@sha256:d1a3e0008557bb3f3314d442ecd4ba68e9b16ab7872d64612649f7a6e5156635
ARG MODEL_SIZE=s
ARG IMG_SIZE=320

FROM python:3.11@sha256:55c475496e35ad8d898659c85f92b42d3b1c58f25f5d79023baf0840b0b4a462 AS model-prep

ARG MODEL_SIZE
ARG IMG_SIZE
# renovate: datasource=pypi depName=onnx
ARG ONNX_VERSION=1.23.2
# renovate: datasource=git-refs depName=https://github.com/WongKinYiu/yolov9 branch=main
ARG YOLOV9_COMMIT=5b1ea9a8b3f0ffe4fe0e203ec6232d788bb3fcff

RUN <<'EOF'
set -e
apt-get update
apt-get install --no-install-recommends -y cmake libgl1
rm -rf /var/lib/apt/lists/*
EOF
COPY --from=ghcr.io/astral-sh/uv:0.13.10@sha256:a3ea1357342c1a4bf9a0c53a3deb1b55379aefa037338d8113d5771efa01774f /uv /bin/
COPY extract-coco-labels.py /

RUN <<'EOF'
set -e
mkdir -p /models /yolo
cd /yolo
git clone https://github.com/WongKinYiu/yolov9.git .
git checkout ${YOLOV9_COMMIT}
curl -fsSL -o yolov9-${MODEL_SIZE}-converted.pt https://github.com/WongKinYiu/yolov9/releases/download/v0.1/yolov9-${MODEL_SIZE}-converted.pt
/bin/uv pip install --system -r requirements.txt onnx==${ONNX_VERSION} onnxruntime onnxsim onnxscript
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
