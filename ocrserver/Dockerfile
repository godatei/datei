FROM python:3.11-slim-bookworm

# Runtime libs required by onnxruntime/opencv used by rapidocr; curl fetches models
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 \
    libglib2.0-0 \
    curl \
    && rm -rf /var/lib/apt/lists/*

# rapidocr_api pulls in rapidocr; the inference engine (onnxruntime) must be
# installed explicitly.
RUN pip install --no-cache-dir rapidocr_api onnxruntime

# PP-OCRv5 models: language-agnostic detection + Latin-script recognition, which
# covers German (incl. umlauts and ß), English and other Latin-script languages.
# The character dictionary is embedded inside each recognition ONNX model, so no
# separate dict file is required. The textline-orientation classifier (180° fix)
# uses the v4 model because rapidocr_api cannot set the v5 classifier's input
# shape; the classifier is version-agnostic. rapidocr_api reads these env vars.
ENV det_model_path=/models/ch_PP-OCRv5_det_mobile.onnx
ENV cls_model_path=/models/ch_ppocr_mobile_v2.0_cls_mobile.onnx
ENV rec_model_path=/models/latin_PP-OCRv5_rec_mobile.onnx
RUN mkdir -p /models \
    && repo="https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.8.0/onnx" \
    && curl -fsSL "$repo/PP-OCRv5/det/ch_PP-OCRv5_det_mobile.onnx" -o "$det_model_path" \
    && curl -fsSL "$repo/PP-OCRv4/cls/ch_ppocr_mobile_v2.0_cls_mobile.onnx" -o "$cls_model_path" \
    && curl -fsSL "$repo/PP-OCRv5/rec/latin_PP-OCRv5_rec_mobile.onnx" -o "$rec_model_path" \
    && printf '%s\n' \
        "4d97c44a20d30a81aad087d6a396b08f786c4635742afc391f6621f5c6ae78ae  $det_model_path" \
        "e47acedf663230f8863ff1ab0e64dd2d82b838fceb5957146dab185a89d6215c  $cls_model_path" \
        "b20bd37c168a570f583afbc8cd7925603890efbcdc000a59e22c269d160b5f5a  $rec_model_path" \
        | sha256sum -c -

# Warm up to validate the models load with the same paths the server uses
RUN python -c "import os; from rapidocr import RapidOCR; RapidOCR(params={'Det.model_path': os.environ['det_model_path'], 'Cls.model_path': os.environ['cls_model_path'], 'Rec.model_path': os.environ['rec_model_path']})"

EXPOSE 9003
ENTRYPOINT ["rapidocr_api", "-ip", "0.0.0.0", "-p", "9003"]
