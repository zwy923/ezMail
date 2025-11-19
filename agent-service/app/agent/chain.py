# app/agent/chain.py

import json
from openai import OpenAI
from app.agent.prompt import JSON_SCHEMA_SYSTEM_MESSAGE
from app.schema import AgentDecision
import logging

logger = logging.getLogger(__name__)

FALLBACK_DECISION = {
    "categories": ["UNKNOWN"],
    "priority": "LOW",
    "summary": "Unable to classify this email.",
    "should_create_task": False,
    "task": None,
    "should_notify": False,
    "notification_channel": None,
    "notification_message": None
}

class DecisionChain:
    def __init__(self):
        """初始化决策链，创建 OpenAI 客户端"""
        import os
        from app.config import settings
        
        # 优先使用环境变量中的 API key
        api_key = os.getenv("OPENAI_API_KEY") or settings.openai_api_key
        if api_key:
            self.client = OpenAI(api_key=api_key)
        else:
            # 如果没有设置 API key，使用默认客户端（会从环境变量读取）
            self.client = OpenAI()
            logger.warning("OpenAI API key not found in config or environment")

    async def ainvoke(self, payload: dict) -> AgentDecision:
        logger.debug("开始处理决策请求 - email_id=%s user_id=%s",
                     payload["email_id"], payload["user_id"])

        # 使用 f-string 进行变量注入（最稳定的方式）
        email_id = payload.get("email_id", "")
        user_id = payload.get("user_id", "")
        subject = payload.get("subject", "")
        body = payload.get("body", "")
        
        # 使用 f-string 格式化用户消息（直接变量注入）
        user_message = f"""
You are an AI assistant that analyzes a single email and makes decisions.

Return ONLY valid JSON.

--------------------
INPUT:
Email ID: {email_id}
User ID: {user_id}

Subject: {subject}
Body: {body}
--------------------

Now output the JSON.
"""

        # 构建消息：system message (JSON schema) + user message (email content)
        openai_messages = [
            {
                "role": "system",
                "content": JSON_SCHEMA_SYSTEM_MESSAGE
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
            return AgentDecision(**FALLBACK_DECISION)

        # 尝试解析 JSON
        try:
            parsed = json.loads(raw_text)
            decision = AgentDecision(**parsed)
            logger.debug("JSON 校验成功")
            return decision

        except Exception as e:
            logger.error("JSON 解析失败，将使用 fallback - 错误: %s", str(e))
            return AgentDecision(**FALLBACK_DECISION)


def build_decision_chain() -> DecisionChain:
    """构建并返回决策链实例"""
    return DecisionChain()
