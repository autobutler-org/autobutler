from dataclasses import dataclass


@dataclass
class DOTAIConfig:
    HOST: str = "localhost"
    PORT: int = 8001

    @property
    def endpoint(self) -> str:
        return f"http://{self.HOST}:{self.PORT}"


DOTAI = DOTAIConfig()
