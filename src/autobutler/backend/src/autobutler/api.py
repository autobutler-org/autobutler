import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

import autobutler.llm as llm

origins = [
    "http://localhost",
    "http://localhost:3000",
]

app = FastAPI()
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

model: llm.LLM = None


@app.post("/api/v1/health")
async def health_check():
    return {
        "response": "This is a dummy response from the AutoButler API.",
        "health": "OK",
    }


@app.post("/api/v1/chat")
async def chat(request: llm.ChatRequest):
    response = model.chat(request)
    return {"response": response}


def main():
    global model
    print("Starting AutoButler API...")
    model = llm.LLM()
    print("AutoButler API started.")
    uvicorn.run(app, host="0.0.0.0", port=8000)
