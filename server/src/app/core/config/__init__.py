import os

from app.core.config.postgres import PostgresConfig

postgres_config = PostgresConfig(
    host=os.environ.get("POSTGRES_DB_HOST", "localhost"),
    port=int(os.environ.get("POSTGRES_DB_PORT", 5432)),
    username=os.environ.get("POSTGRES_DB_LOGIN", "postgres"),
    password=os.environ.get("POSTGRES_DB_PASSWORD", "postgres"),
    dbname=os.environ.get("POSTGRES_DB_NAME", "statusphere"),
    pool_size=int(os.environ.get("POSTGRES_POOL_SIZE", 10)),
)
