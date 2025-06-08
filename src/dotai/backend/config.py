import os
from dataclasses import dataclass


@dataclass
class DOTAIConfig:
    HOST: str = os.getenv("DOTAI_HOST", "localhost")
    PORT: int = int(os.getenv("DOTAI_PORT", "3001"))

    @property
    def endpoint(self) -> str:
        return f"http://{self.HOST}:{self.PORT}"


DOTAI = DOTAIConfig()
