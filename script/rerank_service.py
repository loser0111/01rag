"""
python -m pip install fastapi uvicorn torch transformers sentence-transformers pydantic regex --default-timeout=1000 -i https://pypi.tuna.tsinghua.edu.cn/simple
"""
from fastapi import FastAPI
from pydantic import BaseModel, Field
from typing import List, Optional, Any
import torch
from transformers import AutoModelForSequenceClassification, AutoTokenizer

app = FastAPI(title="Simple Rerank Service")

# 加载rerank模型，BGE-Reranker-v2-M3
MODEL_NAME = "BAAI/bge-reranker-v2-m3"
device = "cuda" if torch.cuda.is_available() else "cpu"

tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
model = AutoModelForSequenceClassification.from_pretrained(MODEL_NAME).to(device)
model.eval()


class RerankRequest(BaseModel):
    query: str
    documents: List[str]
    top_n: Optional[int] = Field(default=None, description="返回topN结果")
    return_documents: bool = Field(default=True, description="是否返回原文")


class RerankResultItem(BaseModel):
    index: int
    document: Optional[str]
    relevance_score: float


class RerankResponse(BaseModel):
    results: List[RerankResultItem]


def run_rerank(query: str, docs: List[str]):
    """本地执行rerank打分"""
    pairs = [[query, doc] for doc in docs]
    with torch.no_grad():
        inputs = tokenizer(
            pairs,
            padding=True,
            truncation=True,
            return_tensors="pt"
        ).to(device)
        outputs = model(**inputs)
        scores = outputs.logits.squeeze(-1).float().cpu().numpy().tolist()

    # 组装 (原始下标, 文档, 分数)
    indexed = [(i, docs[i], scores[i]) for i in range(len(docs))]
    # 按分数降序
    indexed.sort(key=lambda x: x[2], reverse=True)
    return indexed


@app.post("/rerank", response_model=RerankResponse)
async def rerank(req: RerankRequest):
    indexed_list = run_rerank(req.query, req.documents)

    # top_n截断
    if req.top_n is not None:
        indexed_list = indexed_list[: req.top_n]

    res_items = []
    for idx, doc, score in indexed_list:
        item = RerankResultItem(
            index=idx,
            document=doc if req.return_documents else None,
            relevance_score=float(score)
        )
        res_items.append(item)
    return {"results": res_items}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run("main:app", host="0.0.0.0", port=8001)