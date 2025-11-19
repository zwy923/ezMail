# app/agent/text_to_tasks_chain.py

import json
from openai import OpenAI
from app.agent.prompt import TEXT_TO_TASKS_SYSTEM_MESSAGE
from app.schema import TaskListResponse, TaskItem, HabitItem
import logging
import os

logger = logging.getLogger(__name__)


class TextToTasksChain:
    def __init__(self):
        """初始化文本转任务链，创建 OpenAI 客户端"""
        from app.config import settings
        
        # 优先使用环境变量中的 API key
        api_key = os.getenv("OPENAI_API_KEY") or settings.openai_api_key
        if api_key:
            self.client = OpenAI(api_key=api_key)
        else:
            # 如果没有设置 API key，使用默认客户端（会从环境变量读取）
            self.client = OpenAI()
            logger.warning("OpenAI API key not found in config or environment")

    async def ainvoke(self, text: str) -> TaskListResponse:
        logger.debug("开始处理文本转任务请求", extra={"text_length": len(text)})

        # 使用 f-string 格式化用户消息
        user_message = f"""
Parse the following text and extract all tasks with their due dates.

Text: {text}

Now output the JSON array of tasks.
"""

        # 构建消息：system message + user message
        openai_messages = [
            {
                "role": "system",
                "content": TEXT_TO_TASKS_SYSTEM_MESSAGE
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
            return TaskListResponse(tasks=[])

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
            
            # 处理新的格式：包含 tasks 和 habits 的对象
            if isinstance(parsed, dict):
                tasks = [TaskItem(**task) for task in parsed.get("tasks", [])]
                habits = [HabitItem(**habit) for habit in parsed.get("habits", [])]
            elif isinstance(parsed, list):
                # 兼容旧格式：只有任务列表
                tasks = [TaskItem(**task) for task in parsed]
                habits = []
            else:
                tasks = []
                habits = []
            
            logger.debug("JSON 解析成功，任务数量: %d, 习惯数量: %d", len(tasks), len(habits))
            return TaskListResponse(tasks=tasks, habits=habits)

        except Exception as e:
            logger.error("JSON 解析失败 - 错误: %s, 原始文本: %s", str(e), raw_text)
            return TaskListResponse(tasks=[])


def build_text_to_tasks_chain() -> TextToTasksChain:
    """构建并返回文本转任务链实例"""
    return TextToTasksChain()

