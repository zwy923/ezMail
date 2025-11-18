# app/agent/chain.py

import json
from openai import OpenAI
from langchain.prompts import ChatPromptTemplate
from app.agent.prompt import DECISION_PROMPT
from app.agent.schema import AgentDecision
import logging

logger = logging.getLogger(__name__)

client = OpenAI()

prompt = ChatPromptTemplate.from_messages([
    ("user", DECISION_PROMPT)
])

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

    async def ainvoke(self, payload: dict) -> AgentDecision:
        logger.debug("开始处理决策请求 - email_id=%s user_id=%s",
                     payload["email_id"], payload["user_id"])

        safe_payload = {
            "email_id": payload.get("email_id"),
            "user_id": payload.get("user_id"),
            "subject": payload.get("subject", ""),
            "body": payload.get("body", ""),
        }

        logger.debug("正在格式化提示词消息...")
        messages = prompt.format_messages(**safe_payload)
        logger.debug("提示词格式化完成")

        try:
            logger.debug("正在调用 LLM...")
            response = client.chat.completions.create(
                model="gpt-4o-mini",
                temperature=0.2,
                messages=[m.to_dict() for m in messages]
            )
            raw_text = response.choices[0].message["content"]
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
