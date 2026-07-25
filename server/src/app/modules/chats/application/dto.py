from pydantic import BaseModel, ConfigDict, Field


class MessageDTO(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    from_account: str = Field(alias="from")
    to_account: str = Field(alias="to")
    text: str
    at: str
