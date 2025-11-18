# app/agent/schema.py
from pydantic import BaseModel
from typing import List, Optional

class TaskDecision(BaseModel):
    title: str
    due_in_days: int

class AgentDecision(BaseModel):
    categories: List[str]
    priority: str
    summary: str

    should_create_task: bool
    task: Optional[TaskDecision]

    should_notify: bool
    notification_channel: Optional[str]
    notification_message: Optional[str]
