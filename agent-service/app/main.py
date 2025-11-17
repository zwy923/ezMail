from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from app.agent.chain import classify_email
from app.config import settings

app = FastAPI(title="Agent Service", version="1.0.0")


class ClassifyRequest(BaseModel):
    subject: str
    body: str


class ClassifyResponse(BaseModel):
    category: str
    confidence: float


@app.post("/classify", response_model=ClassifyResponse)
async def classify(request: ClassifyRequest):
    """Classify an email using LangChain agent."""
    try:
        result = await classify_email(request.subject, request.body)
        return ClassifyResponse(
            category=result["category"],
            confidence=result["confidence"]
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Classification failed: {str(e)}")


@app.get("/health")
async def health():
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)

