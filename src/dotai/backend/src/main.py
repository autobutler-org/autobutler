import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from dotai.backend.config import DOTAI

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


@app.post("/api/v1/health")
async def health_check():
    return {
        "response": "This is a dummy response from the DotAI API.",
        "health": "OK",
    }


def main():
    print("Starting DotAI API...")
    print("DotAI API started.")
    uvicorn.run(app, host=DOTAI.HOST, port=DOTAI.PORT)


if __name__ == "__main__":
    main()
