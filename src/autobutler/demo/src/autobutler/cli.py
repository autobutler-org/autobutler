import sys

import autobutler.config as config
import requests
from autobutler.device.light import LightState, lightctl


def get_state(entity_id):
    """Fetch the state of an entity from Home Assistant."""
    url = f"{config.HA_URL}/api/states/{entity_id}"
    headers = {"Authorization": f"Bearer {config.HA_TOKEN}"}
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
    url = f"{config.OLLAMA_URL}/api/generate"
    data = {"model": "deepseek-r1", "prompt": prompt}  # Adjust model name as needed
    response = requests.post(url, json=data)
    return response.json()["response"]  # Adjust based on Ollama's API response


def usage():
    print("Usage: butlerctl", file=sys.stderr)
    print("  --help            Show this help message", file=sys.stderr)
    print("Enter queries in the format: topic: question", file=sys.stderr)
    print("Available topics:", file=sys.stderr)
    for topic in config.TOPICS:
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
            if topic in config.TOPICS:
                if topic == "light":
                    lower_question = question.lower()
                    if "on" in lower_question:
                        lightctl(LightState.ON)
                    elif "off" in lower_question:
                        lightctl(LightState.OFF)
                    else:
                        print("Invalid command for light. Use 'on' or 'off'.")
                        continue
                entity_ids = config.TOPICS[topic]
                context = get_context(entity_ids)
                prompt = f"You are AutoButler, a home assistant. Use the following context to answer the user's question.\n\nContext: {context}\n\nUser: {question}\n\nAssistant:"
                response = get_ai_response(prompt)
                print(response)
            else:
                print("Topic not supported.", file=sys.stderr)
        except ValueError:
            print(
                "Invalid format. Use 'topic: question' (e.g., 'fridge: What's in the fridge?')",
                file=sys.stderr,
            )

    print("It was a joy serving you. :)")
    return 0
