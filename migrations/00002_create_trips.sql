-- +goose Up
-- +goose StatementBegin

CREATE TYPE trip_status AS ENUM (
    'draft',
    'published',
    'partially_filled',
    'full',
    'cancelled',
    'completed'
);

CREATE TABLE trips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL,
    from_point TEXT NOT NULL,
    to_point TEXT NOT NULL,
    departure_time TIMESTAMPTZ NOT NULL,
    available_seats INT NOT NULL CHECK (available_seats > 0),
    status trip_status NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE trip_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    from_status trip_status,
    to_status trip_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS trip_history;
DROP TABLE IF EXISTS trips;
DROP TYPE IF EXISTS trip_status;

-- +goose StatementEnd
