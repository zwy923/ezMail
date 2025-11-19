# app/agent/project_planner_chain.py

import json
from openai import OpenAI
from app.agent.prompt import PROJECT_PLANNER_SYSTEM_MESSAGE
from app.schema import ProjectPlanResponse
import logging
import os

logger = logging.getLogger(__name__)


class ProjectPlannerChain:
    def __init__(self):
        """初始化项目规划链，创建 OpenAI 客户端"""
        from app.config import settings
        
        # 优先使用环境变量中的 API key
        api_key = os.getenv("OPENAI_API_KEY") or settings.openai_api_key
        if api_key:
            self.client = OpenAI(api_key=api_key)
        else:
            # 如果没有设置 API key，使用默认客户端（会从环境变量读取）
            self.client = OpenAI()
            logger.warning("OpenAI API key not found in config or environment")

    async def ainvoke(self, text: str) -> ProjectPlanResponse:
        logger.debug("开始处理项目规划请求", extra={"text_length": len(text)})

        # 使用 f-string 格式化用户消息
        user_message = f"""
Analyze the following project goal and break it down into phases (milestones) and tasks.

Project Goal: {text}

Now output the JSON project plan.
"""

        # 构建消息：system message + user message
        openai_messages = [
            {
                "role": "system",
                "content": PROJECT_PLANNER_SYSTEM_MESSAGE
            },
            {
                "role": "user",
                "content": user_message
            }
        ]

        try:
            logger.debug("正在调用 LLM...")
            response = self.client.chat.completions.create(
                model="gpt-4o-mini",
                temperature=0.2,
                messages=openai_messages
            )
            raw_text = response.choices[0].message.content
            logger.debug("LLM 原始返回: %s", raw_text)
        except Exception as e:
            logger.error("LLM 调用失败: %s", str(e))
            raise

        # 尝试解析 JSON
        try:
            # 清理可能的 markdown 代码块标记
            cleaned_text = raw_text.strip()
            if cleaned_text.startswith("```json"):
                cleaned_text = cleaned_text[7:]
            if cleaned_text.startswith("```"):
                cleaned_text = cleaned_text[3:]
            if cleaned_text.endswith("```"):
                cleaned_text = cleaned_text[:-3]
            cleaned_text = cleaned_text.strip()

            parsed = json.loads(cleaned_text)
            
            # 转换为 ProjectPlanResponse
            from app.schema import ProjectPlan, Milestone as MilestoneSchema, ProjectTask as ProjectTaskSchema
            
            milestones = []
            for m in parsed.get("milestones", []):
                tasks = [ProjectTaskSchema(**task) for task in m.get("tasks", [])]
                milestones.append(MilestoneSchema(
                    title=m["title"],
                    order=m["order"],
                    due_in_days=m["due_in_days"],
                    tasks=tasks
                ))
            
            project_plan = ProjectPlan(
                title=parsed["title"],
                description=parsed.get("description", ""),
                target_days=parsed["target_days"],
                milestones=milestones
            )
            
            logger.debug("JSON 解析成功，阶段数量: %d", len(milestones))
            return ProjectPlanResponse(project=project_plan)

        except Exception as e:
            logger.error("JSON 解析失败 - 错误: %s, 原始文本: %s", str(e), raw_text)
            raise


def build_project_planner_chain() -> ProjectPlannerChain:
    """构建并返回项目规划链实例"""
    return ProjectPlannerChain()

