from dataclasses import dataclass


@dataclass
class PostgresConfig:
    host: str
    port: int
    username: str
    password: str
    dbname: str
    pool_size: int
