import os

import pytest
from flowtools.db import Database


@pytest.fixture
def db():
    """
    Fixture to create a Database instance for testing.
    """
    db_instance = Database("test")
    db_instance.open()

    db_instance.query(
        "CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, value TEXT)"
    )
    db_instance.query(
        "CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, value TEXT)"
    )
    db_instance.query("INSERT INTO test (value) VALUES ('test_value')")

    yield db_instance

    db_instance.close()
    os.remove("test.db")


def test_set_and_get(db):
    """
    Test setting and getting a key-value pair in the database.
    """
    db.set("test_key", "test_value")
    value = db.get("test_key")
    assert value == b"test_value"


def test_get_non_existent_key(db):
    """
    Test getting a non-existent key returns None.
    """
    value = db.get("non_existent_key")
    assert value is None


def test_query(db):
    """
    Test executing a simple query.
    """
    results = db.query("SELECT * FROM test")
    assert results == [(1, "test_value")]


def test_query_non_existent_key(db):
    """
    Test querying a non-existent key returns an empty list.
    """
    results = db.query("SELECT * FROM test where id = 999")
    assert results == []
