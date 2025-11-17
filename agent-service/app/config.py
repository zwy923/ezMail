from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    openai_api_key: str = ""
    model_name: str = "gpt-3.5-turbo"
    port: int = 8000

    class Config:
        env_file = ".env"


settings = Settings()

