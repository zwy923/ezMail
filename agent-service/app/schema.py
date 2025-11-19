# app/schema.py
from pydantic import BaseModel
from typing import List, Optional


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
    priority: str
    summary: str

    should_create_task: bool
    task: Optional[TaskDecision]

    should_notify: bool
    notification_channel: Optional[str]
    notification_message: Optional[str]


class TaskItem(BaseModel):
    title: str
    due_in_days: int


class TaskTextInput(BaseModel):
    user_id: int
    text: str


class HabitItem(BaseModel):
    title: str
    recurrence_pattern: str  # "weekly Wednesday", "daily", "monthly 1"


class TaskListResponse(BaseModel):
    tasks: List[TaskItem]
    habits: Optional[List[HabitItem]] = []  # 新增：习惯列表


class ProjectTask(BaseModel):
    title: str
    due_in_days: int
    priority: str  # LOW / MEDIUM / HIGH
    depends_on: List[str] = []  # List of task titles this task depends on


class Milestone(BaseModel):
    title: str
    order: int
    due_in_days: int
    tasks: List[ProjectTask]


class ProjectPlan(BaseModel):
    title: str
    description: str
    target_days: int
    milestones: List[Milestone]


class ProjectTextInput(BaseModel):
    user_id: int
    text: str


class ProjectPlanResponse(BaseModel):
    project: ProjectPlan
