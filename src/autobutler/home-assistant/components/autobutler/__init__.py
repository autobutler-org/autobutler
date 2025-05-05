DOMAIN = "autobutler"


def setup(hass, config):
    hass.states.set(f"{DOMAIN}.hello.world", "from the autobutler")

    # Return boolean to indicate that initialization was successful.
    return True
