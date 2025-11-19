from fastapi import FastAPI, HTTPException
from app.agent.chain import build_decision_chain
from app.agent.text_to_tasks_chain import build_text_to_tasks_chain
from app.agent.project_planner_chain import build_project_planner_chain
from app.schema import EmailInput, AgentDecision, TaskListResponse, TaskTextInput, ProjectPlanResponse, ProjectTextInput
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

# Initialize text-to-tasks chain
logger.info("正在初始化文本转任务链...")
text_to_tasks_chain = build_text_to_tasks_chain()
logger.info("文本转任务链初始化完成")

# Initialize project planner chain
logger.info("正在初始化项目规划链...")
project_planner_chain = build_project_planner_chain()
logger.info("项目规划链初始化完成")


@app.post("/decide", response_model=AgentDecision)
async def decide(payload: EmailInput) -> AgentDecision:
    logger.info(f"收到决策请求 - email_id: {payload.email_id}, user_id: {payload.user_id}")
    logger.debug(f"请求详情 - subject: {payload.subject[:50]}..., body长度: {len(payload.body)}")

    try:
        logger.debug("开始调用决策链...")

        # ⭐ 关键：decision_chain.ainvoke() 已返回 AgentDecision 对象，不是 dict
        # Pydantic v2 使用 model_dump()，v1 使用 dict()
        payload_dict = payload.model_dump() if hasattr(payload, 'model_dump') else payload.dict()
        decision: AgentDecision = await decision_chain.ainvoke(payload_dict)

        # ⭐ 使用 model_dump() 或 dict() 才能 JSON dump
        decision_dict = decision.model_dump() if hasattr(decision, 'model_dump') else decision.dict()
        logger.debug(
            "决策链返回结果: %s",
            json.dumps(decision_dict, ensure_ascii=False, indent=2)
        )

        logger.info(
            "决策完成 - email_id: %s, priority: %s, categories: %s",
            payload.email_id, decision.priority, decision.categories
        )

        # FastAPI 直接能序列化 AgentDecision
        return decision

    except Exception as e:
        logger.error(
            f"处理决策请求时发生错误 - email_id: {payload.email_id}, 错误: {str(e)}",
            exc_info=True
        )
        raise HTTPException(
            status_code=500,
            detail=f"处理请求时发生错误: {str(e)}"
        )


@app.post("/text-to-tasks", response_model=TaskListResponse)
async def text_to_tasks(payload: TaskTextInput) -> TaskListResponse:
    logger.info(f"收到文本转任务请求 - user_id: {payload.user_id}, text长度: {len(payload.text)}")
    logger.debug(f"请求文本: {payload.text[:100]}...")

    try:
        logger.debug("开始调用文本转任务链...")
        result = await text_to_tasks_chain.ainvoke(payload.text)
        
        logger.info(
            "文本转任务完成 - user_id: %s, 任务数量: %d",
            payload.user_id, len(result.tasks)
        )
        
        return result

    except Exception as e:
        logger.error(
            f"处理文本转任务请求时发生错误 - user_id: {payload.user_id}, 错误: {str(e)}",
            exc_info=True
        )
        raise HTTPException(
            status_code=500,
            detail=f"处理请求时发生错误: {str(e)}"
        )


@app.post("/plan-project", response_model=ProjectPlanResponse)
async def plan_project(payload: ProjectTextInput) -> ProjectPlanResponse:
    logger.info(f"收到项目规划请求 - user_id: {payload.user_id}, text长度: {len(payload.text)}")
    logger.debug(f"请求文本: {payload.text[:100]}...")

    try:
        logger.debug("开始调用项目规划链...")
        result = await project_planner_chain.ainvoke(payload.text)
        
        logger.info(
            "项目规划完成 - user_id: %s, 阶段数量: %d",
            payload.user_id, len(result.project.milestones)
        )
        
        return result

    except Exception as e:
        logger.error(
            f"处理项目规划请求时发生错误 - user_id: {payload.user_id}, 错误: {str(e)}",
            exc_info=True
        )
        raise HTTPException(
            status_code=500,
            detail=f"处理请求时发生错误: {str(e)}"
        )


@app.get("/health")
async def health():
    logger.debug("健康检查请求")
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn
    logger.info(f"启动 Agent Service - 端口: 8000")
    uvicorn.run(app, host="0.0.0.0", port=8000)

