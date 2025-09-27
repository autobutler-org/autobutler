CREATE TABLE
    IF NOT EXISTS emails (
        mail_id INTEGER PRIMARY KEY AUTOINCREMENT,
        date DATETIME,
        "to" TEXT,
        "from" TEXT,
        subject TEXT,
        body TEXT,
        mail TEXT,
        spam_score REAL,
        hash TEXT,
        content_type TEXT,
        recipient TEXT,
        has_attach BOOLEAN,
        ip_addr TEXT,
        return_path TEXT,
        is_tls BOOLEAN,
        message_id TEXT,
        reply_to TEXT,
        sender TEXT
    );

CREATE TABLE
    IF NOT EXISTS inventory (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        amount REAL NOT NULL,
        unit TEXT NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS calendars (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS calendar_events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        description TEXT,
        start_time DATETIME NOT NULL,
        end_time DATETIME,
        all_day BOOLEAN NOT NULL DEFAULT 0,
        location TEXT,
        calendar_id INTEGER NOT NULL,
        FOREIGN KEY (calendar_id) REFERENCES calendars (id)
    );
