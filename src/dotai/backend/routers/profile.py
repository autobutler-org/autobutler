from typing import Optional

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from ..models.profile import Profile

router = APIRouter(prefix="/profile", tags=["Profile"])


class ProfileUpdate(BaseModel):
    name: Optional[str] = None
    phone: Optional[str] = None
    company: Optional[str] = None


class ProfileResponse(BaseModel):
    id: str
    email: str
    name: Optional[str] = None
    phone: Optional[str] = None
    company: Optional[str] = None

    class Config:
        from_attributes = True


@router.get("", response_model=ProfileResponse)
async def get_profile():
    """Get customer profile"""
    # TODO: Implement profile retrieval from database
    # For now, return a mock profile
    mock_profile = Profile(
        id="user_123",
        email="john.doe@example.com",
        name="John Doe",
        phone="+1234567890",
        company="ACME Inc",
    )
    return ProfileResponse.from_orm(mock_profile)


@router.put("", response_model=ProfileResponse)
async def update_profile(profile_update: ProfileUpdate):
    """Update customer profile"""
    if not any([profile_update.name, profile_update.phone, profile_update.company]):
        raise HTTPException(
            status_code=400, detail="At least one field must be provided for update"
        )

    # TODO: Implement profile update in database
    # For now, return updated mock profile
    updated_profile = Profile(
        id="user_123",
        email="john.doe@example.com",
        name=profile_update.name or "John Doe",
        phone=profile_update.phone or "+1234567890",
        company=profile_update.company or "ACME Inc",
    )
    return ProfileResponse.from_orm(updated_profile)
