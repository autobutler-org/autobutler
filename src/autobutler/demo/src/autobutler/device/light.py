__all__ = ["lightctl", "LightState"]

import os
import requests

import autobutler.config as config

from enum import Enum

SMARTBULB_ENTITY = os.getenv("SMARTBULB_ENTITY", "light.smartbulb")

class LightState(Enum):
    OFF = "off"
    ON = "on"

def call_light_service(service: str):
    """Call a Home Assistant light service."""
    url = f"{config.HA_URL}/api/services/light/{service}"
    headers = {
        "Authorization": f"Bearer {config.HA_TOKEN}",
        "Content-Type": "application/json",
    }
    data = {"entity_id": SMARTBULB_ENTITY}
    response = requests.post(url, headers=headers, json=data)
    if response.ok:
        print(f"Smart light {service.replace('_', ' ')} command executed.")
    else:
        print(f"Failed to execute command: {response.content}")

def lightctl(state: LightState):
    """Turn the smart light on using Home Assistant service call."""
    if state == LightState.OFF:
        call_light_service("turn_off")
    elif state == LightState.ON:
        call_light_service("turn_on")
