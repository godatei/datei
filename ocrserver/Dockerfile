FROM debian:bookworm-slim AS models

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# PP-OCRv5 models: language-agnostic detection + Latin-script recognition, which
# covers German (incl. umlauts and ß), English and other Latin-script languages.
# The character dictionary is embedded inside each recognition ONNX model, so no
# separate dict file is required. The textline-orientation classifier (180° fix)
# uses the v4 model because rapidocr_api cannot set the v5 classifier's input
# shape; the classifier is version-agnostic.
WORKDIR /models
COPY models.sha256 /
RUN repo="https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.8.0/onnx" \
    && curl -fsSL "$repo/PP-OCRv5/det/ch_PP-OCRv5_det_mobile.onnx" -o ch_PP-OCRv5_det_mobile.onnx \
    && curl -fsSL "$repo/PP-OCRv4/cls/ch_ppocr_mobile_v2.0_cls_mobile.onnx" -o ch_ppocr_mobile_v2.0_cls_mobile.onnx \
    && curl -fsSL "$repo/PP-OCRv5/rec/latin_PP-OCRv5_rec_mobile.onnx" -o latin_PP-OCRv5_rec_mobile.onnx \
    && sha256sum -c /models.sha256


FROM python:3.11-slim-bookworm

# rapidocr_api pulls in rapidocr; the inference engine (onnxruntime) must be
# installed explicitly. rapidocr declares a dependency on the full opencv-python,
# which needs system GUI libs (libGL); swap it for the headless variant, which
# provides the same cv2 API without them.
RUN pip install --no-cache-dir rapidocr_api onnxruntime \
    && pip uninstall -y opencv-python \
    && pip install --no-cache-dir opencv-python-headless

# rapidocr_api reads these env vars.
ENV det_model_path=/models/ch_PP-OCRv5_det_mobile.onnx
ENV cls_model_path=/models/ch_ppocr_mobile_v2.0_cls_mobile.onnx
ENV rec_model_path=/models/latin_PP-OCRv5_rec_mobile.onnx
COPY --from=models /models /models

# Warm up to validate the models load with the same paths the server uses
RUN python -c "import os; from rapidocr import RapidOCR; RapidOCR(params={'Det.model_path': os.environ['det_model_path'], 'Cls.model_path': os.environ['cls_model_path'], 'Rec.model_path': os.environ['rec_model_path']})"

EXPOSE 9003
ENTRYPOINT ["rapidocr_api", "-ip", "0.0.0.0", "-p", "9003"]
