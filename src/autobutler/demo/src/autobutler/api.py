import uvicorn
from fastapi import Body, FastAPI, Path, Query

app = FastAPI()


@app.get("/api/v1/inv")
async def get_inventory(query: str = Query(None)):
    # Returns all inventory or a specific item based on query
    return {"message": "stub: get inventory", "query": query}


@app.put("/api/v1/inv")
async def update_inventory(data: dict = Body(...)):
    # Updates inventory (called automagically by the butler)
    return {"message": "stub: update inventory", "data": data}


@app.get("/api/v1/dev")
async def get_devices(header_only: bool = Query(False, alias="header-only")):
    # Get all devices, optionally with headers only
    return {"message": "stub: get devices", "header_only": header_only}


@app.get("/api/v1/dev/{device_id}")
async def get_device_by_id(device_id: int = Path(...)):
    # Get a device by id
    return {"message": "stub: get device by id", "device_id": device_id}


@app.post("/api/v1/dev")
async def add_device(data: dict = Body(...)):
    # Add a new device
    return {"message": "stub: add device", "data": data}


@app.put("/api/v1/dev")
async def update_device(data: dict = Body(...)):
    # Update a device
    return {"message": "stub: update device", "data": data}


@app.delete("/api/v1/dev/{device_id}")
async def delete_device(device_id: int = Path(...)):
    # Delete a device
    return {"message": "stub: delete device", "device_id": device_id}


def main():
    uvicorn.run(app, host="0.0.0.0", port=8000)
