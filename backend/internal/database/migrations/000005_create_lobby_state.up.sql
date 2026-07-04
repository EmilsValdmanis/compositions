CREATE TABLE lobby_state (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    state JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lobby_state_singleton CHECK (id)
);
