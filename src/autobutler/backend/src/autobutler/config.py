import os
from dataclasses import dataclass

# Configuration
HA_URL = os.getenv(
    "HA_URL", "http://homeassistant.local:8123"
)  # Your Home Assistant URL
HA_TOKEN = os.getenv("HA_TOKEN", None)  # Your Home Assistant access token


@dataclass
class LLMConfig:
    MODEL: str
    PROMPT: list[dict[str, str]]
    MAX_TOKENS: int
    TOP_P: float
    TOP_K: int
    TEMPERATURE: float

    def __repr__(self):
        return (
            f"LLMConfig(MODEL={self.MODEL}, MAX_TOKENS={self.MAX_TOKENS}, "
            f"TOP_P={self.TOP_P}, TOP_K={self.TOP_K}, TEMPERATURE={self.TEMPERATURE})"
        )


LLM = LLMConfig(
    # Source: https://huggingface.co/microsoft/Phi-4-mini-instruct
    MODEL="microsoft/Phi-4-mini-instruct",
    MAX_TOKENS=int(os.getenv("LLM_MAX_TOKENS", 256)),  # Max tokens for LLM response
    PROMPT=[
        {
            "role": "system",
            "content": """
            You are a home butler.
            You are incredibly succinct in your responses.
            Do not provide any additional information or context.
            Do not provide your thought process or reasoning, but simply respond to the user's request.

            Users will have a few different things they ask you to do. For the time being, consider this list to be the only things you can respond to and feel free to tell the user such a thing:
            - they may ask you what inventory items are in their home. Do not worry that you don't yet have real data, but simply answer with some quantity.
            - they may ask you to turn on or off a device in their home. Do not worry that you don't yet have real data, but simply answer as if you did something.

            Answer the user's request in a single sentence, and nothing more.
            If you do not know the answer, simply say "I don't know.".
            """,
        },
    ],
    TOP_P=float(os.getenv("LLM_TOP_P", 0.95)),  # Top P sampling for LLM
    TOP_K=int(os.getenv("LLM_TOP_K", 50)),  # Top K sampling for LLM
    TEMPERATURE=float(os.getenv("LLM_TEMPERATURE", 0.7)),  # Temperature for LLM
)
