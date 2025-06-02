__all__ = [
    "Database",
]

import dbm.sqlite3 as dbm
import sqlite3

from flowtools.types import SupportsStr


class Database:
    """
    A simple wrapper around dbm to handle sqlite-based storage.

    This class uses dbm to store key-value pairs in a file-based database.
    It also supports a simple query interface for SQL-like operations.

    By default, it provides a context manager interface for opening and closing the database.
    Usage:
    ```python
    from autobutler.db import Database
    with Database("path/to/db") as db:
        db.set("key", "value")
        value = db.get("key")
    ```
    """

    def __init__(self, db_path: str):
        """
        Initialize the Database with a path to the database file.
        Does not open the database for you.

        :param db_path: Path to the database file.
        """

        self.db_path = db_path
        self.db = None
        self.conn = None
        self.cursor = None

    def get(self, key: str) -> bytes | None:
        """
        Get the value associated with a key from the database.
        If the key does not exist, it returns None.

        :param key: The key to retrieve from the database.
        """
        if self.db is None:
            raise ValueError(
                "Database not opened yet. Either use a 'with' statement or `self.open()`"
            )
        return self.db.get(key.encode())

    def set(self, key: str, value: SupportsStr) -> None:
        """
        Set a key-value pair in the database.

        :param key: The key to set in the database.
        :param value: The value to associate with the key. Must be convertible to a string.
        """
        if self.db is None:
            raise ValueError(
                "Database not opened yet. Either use a 'with' statement or `self.open()`"
            )
        self.db[key.encode()] = str(value).encode()

    def query(self, query: str) -> list[tuple] | None:
        """
        Execute a complex query via the sqlite3 interface.

        :param query: The SQL query to execute.
        """
        if self.db is None:
            raise ValueError(
                "Database not opened yet. Either use a 'with' statement or `self.open()`"
            )
        results = []
        try:
            self.cursor = self.conn.cursor()
            self.cursor.execute(query)
            results = self.cursor.fetchall()
            self.conn.commit()
        finally:
            if self.cursor:
                self.cursor.close()
        return results

    def open(self):
        """
        Open the database for reading and writing. If the database does not exist, it will be created.
        """
        self.db = dbm.open(self.db_path, "c")
        self.conn = sqlite3.connect(self.db_path)

    def close(self):
        """
        Close the database if it is open.
        """
        if self.db is not None:
            self.db.close()
            self.conn.close()

    """
    Context manager methods for opening and closing the database.
    """

    def __enter__(self):
        """
        Open the database when entering the context manager.
        """
        self.open()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """
        Close the database when exiting the context manager.
        """
        self.close()


if __name__ == "__main__":
    """
    Example usage of the Database class.
    """
    import os

    with Database("example.db") as db:
        db.set("key1", "value1")
        print("From dbm: ", db.get("key1"))
        db.query("CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, value TEXT)")
        db.query("INSERT INTO test (value) VALUES ('test_value')")
        print("From query:", db.query("SELECT * FROM test"))
    os.remove("example.db")
