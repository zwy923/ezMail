from langchain_openai import ChatOpenAI
from langchain_core.prompts import ChatPromptTemplate
from .prompt import DECISION_PROMPT
import json
import logging

logger = logging.getLogger(__name__)


def build_decision_chain():
    logger.debug("开始构建决策链...")
    llm = ChatOpenAI(
        model="gpt-4o-mini",
        temperature=0.1,
    )
    logger.debug(f"LLM 模型已初始化 - model: gpt-4o-mini, temperature: 0.1")

    prompt = ChatPromptTemplate.from_template(DECISION_PROMPT)
    logger.debug("提示词模板已创建")

    async def ainvoke(payload: dict) -> dict:
        logger.debug(f"开始处理决策请求 - email_id: {payload.get('email_id')}, user_id: {payload.get('user_id')}")
        content = None
        
        try:
            # 只提取 prompt 需要的字段，过滤其他字段
            safe_payload = {
                "email_id": payload["email_id"],
                "user_id": payload["user_id"],
                "subject": payload["subject"],
                "body": payload["body"],
            }
            
            # 格式化提示词
            logger.debug("正在格式化提示词消息...")
            messages = prompt.format_messages(**safe_payload)
            logger.debug(f"提示词消息已格式化，消息数量: {len(messages)}")
            
            # 记录发送给 LLM 的提示词（仅记录部分内容以避免日志过长）
            if messages:
                first_msg = str(messages[0])[:200] if len(str(messages[0])) > 200 else str(messages[0])
                logger.debug(f"提示词预览: {first_msg}...")
            
            # 调用 LLM
            logger.debug("正在调用 LLM...")
            resp = await llm.ainvoke(messages)
            logger.debug(f"LLM 响应已接收 - 响应长度: {len(resp.content)} 字符")
            
            # LLM 输出必须是 JSON 字符串
            content = resp.content.strip()
            logger.debug(f"LLM 原始响应内容: {content[:500]}..." if len(content) > 500 else f"LLM 原始响应内容: {content}")
            
            # 解析 JSON
            logger.debug("正在解析 JSON 响应...")
            result = json.loads(content)
            logger.debug(f"JSON 解析成功 - 包含字段: {list(result.keys())}")
            
            return result
        except json.JSONDecodeError as e:
            error_msg = f"JSON 解析失败"
            if content:
                error_msg += f" - 原始内容: {content[:500]}"
            logger.error(error_msg, exc_info=True)
            raise
        except Exception as e:
            logger.error(f"决策链处理过程中发生错误: {str(e)}", exc_info=True)
            raise

    # 这里只返回一个"自定义 async 函数"也可以，不一定非要搞 LangChain Runnable
    class DecisionChain:
        async def ainvoke(self, payload):
            return await ainvoke(payload)

    logger.debug("决策链构建完成")
    return DecisionChain()
