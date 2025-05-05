from __future__ import annotations

from homeassistant.core import HomeAssistant
from homeassistant.helpers.typing import ConfigType

DOMAIN = "autobutler"


def setup(hass: HomeAssistant, config: ConfigType):
    hass.states.set(f"{DOMAIN}.hello.world", "from the autobutler")

    # Return boolean to indicate that initialization was successful.
    return True
