CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE documents(
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    owner_id INT REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE events(
    id SERIAL PRIMARY KEY,
    document_id INT REFERENCES documents(id),
    user_id INT REFERENCES users(id),
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);