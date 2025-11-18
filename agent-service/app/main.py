from fastapi import FastAPI, HTTPException
from app.agent.chain import build_decision_chain
from app.schema import EmailInput, AgentDecision
from app.config import settings
import logging
import json

# 配置日志
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

app = FastAPI(title="Agent Service", version="1.0.0")

# Initialize decision chain
logger.info("正在初始化决策链...")
decision_chain = build_decision_chain()
logger.info("决策链初始化完成")


@app.post("/decide", response_model=AgentDecision)
async def decide(payload: EmailInput) -> AgentDecision:
    logger.info(f"收到决策请求 - email_id: {payload.email_id}, user_id: {payload.user_id}")
    logger.debug(f"请求详情 - subject: {payload.subject[:50]}..., body长度: {len(payload.body)}")
    
    try:
        # 调用 LangChain，拿到 LLM 结果（JSON）
        logger.debug("开始调用决策链...")
        result = await decision_chain.ainvoke(payload.dict())
        logger.debug(f"决策链返回结果: {json.dumps(result, ensure_ascii=False, indent=2)}")

        # result 应该是一个 dict，可以直接喂给 AgentDecision
        decision = AgentDecision(**result)
        logger.info(f"决策完成 - email_id: {payload.email_id}, priority: {decision.priority}, categories: {decision.categories}")
        return decision
    except Exception as e:
        logger.error(f"处理决策请求时发生错误 - email_id: {payload.email_id}, 错误: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"处理请求时发生错误: {str(e)}")


@app.get("/health")
async def health():
    logger.debug("健康检查请求")
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn
    logger.info(f"启动 Agent Service - 端口: 8000")
    uvicorn.run(app, host="0.0.0.0", port=8000)

