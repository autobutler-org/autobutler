import sys
from pprint import pprint

import autobutler.config as config
import torch
from pydantic import BaseModel, Field
from transformers import pipeline


class ChatRequest(BaseModel):
    prompt: str = Field(...)


class LLM:
    def __init__(self):
        """
        Initialize the model and tokenizer.
        This function is called when the module is imported.
        """
        print(f"[DEBUG] Initializing {config.LLM.MODEL}...")
        self.model = pipeline(
            task="text-generation",
            model=config.LLM.MODEL,
            torch_dtype=torch.bfloat16,
            device_map="auto",
        )
        print("[DEBUG] LLM loaded with the following config:", repr(config.LLM))
        self.prompt = config.LLM.PROMPT

    def chat(self, request: ChatRequest) -> str:
        try:
            messages = config.LLM.PROMPT + [
                {
                    "role": "user",
                    "content": request.prompt,
                }
            ]
            print("[DEBUG] Applying chat template...")
            prompt = self.model.tokenizer.apply_chat_template(
                messages, tokenize=False, add_generation_prompt=True
            )
            print("[DEBUG] Generating response...")
            outputs = self.model(
                prompt,
                max_new_tokens=config.LLM.MAX_TOKENS,
                do_sample=True,
                temperature=config.LLM.TEMPERATURE,
                top_k=config.LLM.TOP_K,
                top_p=config.LLM.TOP_P,
            )
            response = outputs[0]["generated_text"].split("<|assistant|>")[-1].strip()
            return response
        except Exception as e:
            print(e, file=sys.stderr)
            return "I'm sorry, I couldn't process your request. Please try again later."
