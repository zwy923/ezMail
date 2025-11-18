from pydantic import BaseModel
from typing import List, Literal, Optional


class EmailInput(BaseModel):
    email_id: int
    user_id: int
    subject: str
    body: str


class TaskDecision(BaseModel):
    title: str
    due_in_days: int


class AgentDecision(BaseModel):
    categories: List[str]
    priority: Literal["LOW", "MEDIUM", "HIGH"]
    summary: str

    should_create_task: bool
    task: Optional[TaskDecision] = None

    should_notify: bool
    notification_channel: Optional[Literal["EMAIL"]] = None
    notification_message: Optional[str] = None
