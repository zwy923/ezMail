from pydantic_settings import BaseSettings
import os


class Settings(BaseSettings):
    openai_api_key: str = ""
    model_name: str = "gpt-3.5-turbo"
    port: int = 8000

    class Config:
        env_file = ".env"
        # 允许从环境变量读取
        case_sensitive = False

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        # 优先从环境变量读取
        if not self.openai_api_key:
            self.openai_api_key = os.getenv("OPENAI_API_KEY", "")


settings = Settings()

