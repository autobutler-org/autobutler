import os

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
    "light": True,
}
