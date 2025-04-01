import os
import requests
import sys

# Configuration
HA_URL = os.getenv(
    "HA_URL", "http://homeassistant.local:8123"
)  # Your Home Assistant URL
HA_TOKEN = os.getenv("HA_TOKEN", None)  # Your Home Assistant access token
OLLAMA_URL = os.getenv("OLLAMA_URL", "http://localhost:11434")  # Ollama API URL

# Define topics and their corresponding Home Assistant entity IDs
TOPICS = {
    "fridge": ["sensor.fridge_milk", "sensor.fridge_eggs", "sensor.fridge_cheese"],
    "temperature": ["sensor.living_room_temperature", "sensor.bedroom_temperature"],
}


def get_state(entity_id):
    """Fetch the state of an entity from Home Assistant."""
    url = f"{HA_URL}/api/states/{entity_id}"
    headers = {"Authorization": f"Bearer {HA_TOKEN}"}
    response = requests.get(url, headers=headers)
    return response.json()


def get_context(entity_ids):
    """Compile context from entity states."""
    context = ""
    for entity_id in entity_ids:
        state = get_state(entity_id)
        name = state["attributes"].get("friendly_name", entity_id)
        value = state["state"]
        unit = state["attributes"].get("unit_of_measurement", "")
        context += f"{name}: {value} {unit}\n"
    return context


def get_ai_response(prompt):
    """Get a response from the AI model via Ollama."""
    url = f"{OLLAMA_URL}/api/generate"
    data = {"model": "deepseek-r1", "prompt": prompt}  # Adjust model name as needed
    response = requests.post(url, json=data)
    return response.json()["response"]  # Adjust based on Ollama's API response


def usage():
    print("Usage: butlerctl", file=sys.stderr)
    print("  --help            Show this help message", file=sys.stderr)
    print("Enter queries in the format: topic: question", file=sys.stderr)
    print("Available topics:", file=sys.stderr)
    for topic in TOPICS:
        print(f"  - {topic}", file=sys.stderr)


def main() -> int:
    is_running = True
    if "--help" in sys.argv:
        usage()
        is_running = False
    while is_running:
        query = input("Ask AutoButler (format: topic: question): ")
        if query.lower() == "exit":
            is_running = False
            continue
        try:
            topic, question = query.split(":", 1)
            topic = topic.strip().lower()
            question = question.strip()
            if topic in TOPICS:
                entity_ids = TOPICS[topic]
                context = get_context(entity_ids)
                prompt = f"You are AutoButler, a home assistant. Use the following context to answer the user's question.\n\nContext: {context}\n\nUser: {question}\n\nAssistant:"
                response = get_ai_response(prompt)
                print(response)
            else:
                print("Topic not supported.", file=sys.stderr)
        except ValueError:
            print(
                "Invalid format. Use 'topic: question' (e.g., 'fridge: What’s in the fridge?')",
                file=sys.stderr,
            )

    print("It was a joy serving you. :)")
    return 0
