# [Autobutler](https://autobutler.ai)

Automated home assistant

## Features

### Inventory

Track an inventory for the whole home, from food to tools. Initially,
would integrate as a receipt import app, where it can scan the receipt
and track that you own the thing. Eventually, we should integrate with
existng IoT sensors in appliances like refrigerators, freezers, and
generic webcam setups in a pantry.

### [Home Assistant](https://www.home-assistant.io/)

Integrates with Home Assistant to allow for automated home management tasks,
and allows for agentic management of home smart devices.

## Examples/Situations

- Resident is boiling something on the stove and walks away from the stove. The user wants to be notified when the stove is on and unattended.
  - Determine with combination of temp sensor and a camera or floor pressure sensor
  - Send a push notification to the user's phone that lets them approve or reject the help
  - Say the user intended to leave the stove unattended this one time, then the user should be able to simply state that out loud and
    Autobutler loads that into its context, not notifying the user.
- Resident has left the sink turned on for a cat to drink and everyone has walked away and left the sink running
  - Send a push to the user that the sink is still running
  - Again, allow user to preemptively override the context
