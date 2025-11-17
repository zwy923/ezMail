from pydantic import BaseModel


class ClassifyRequest(BaseModel):
    subject: str
    body: str


class ClassifyResponse(BaseModel):
    category: str
    confidence: float

