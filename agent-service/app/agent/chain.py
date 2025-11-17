from langchain_openai import ChatOpenAI
from langchain_core.prompts import ChatPromptTemplate
from app.agent.prompt import CLASSIFICATION_PROMPT
from app.config import settings


async def classify_email(subject: str, body: str) -> dict:
    """
    Classify email using LangChain.
    Returns dict with 'category' and 'confidence'.
    """
    # Initialize LLM
    llm = ChatOpenAI(
        model=settings.model_name,
        api_key=settings.openai_api_key,
        temperature=0,
    )

    # Create prompt template
    prompt = ChatPromptTemplate.from_template(CLASSIFICATION_PROMPT)
    chain = prompt | llm

    # Invoke chain
    try:
        response = await chain.ainvoke({
            "subject": subject,
            "body": body[:500],  # Limit body length
        })

        # Parse response
        category_text = response.content.strip().lower()
        
        # Extract category
        category = "other"
        if "finance" in category_text:
            category = "finance"
        elif "schedule" in category_text:
            category = "schedule"
        
        # Simple confidence calculation (can be improved)
        confidence = 0.9 if category != "other" else 0.7

        return {
            "category": category,
            "confidence": confidence,
        }
    except Exception as e:
        # Fallback to simple rule-based classification if LLM fails
        category = "other"
        if any(word in subject.lower() for word in ["payment", "invoice", "bill", "bank"]):
            category = "finance"
        elif any(word in subject.lower() for word in ["meeting", "appointment", "schedule"]):
            category = "schedule"
        
        return {
            "category": category,
            "confidence": 0.6,
        }

