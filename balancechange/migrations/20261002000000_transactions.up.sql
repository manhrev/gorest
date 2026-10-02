CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    note TEXT NOT NULL DEFAULT '',
    -- VND, always positive; direction is carried by type.
    amount BIGINT NOT NULL,
    type TEXT NOT NULL,
    category TEXT NOT NULL,
    owner TEXT NOT NULL,
    -- kept so a bad categorization can be re-parsed later.
    raw_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- value lists mirror internal/enum; change both together.
    CONSTRAINT transactions_amount_check CHECK (amount > 0),
    CONSTRAINT transactions_type_check CHECK (type IN ('income', 'expense')),
    CONSTRAINT transactions_category_check CHECK (category IN ('food_drink', 'entertainment', 'shopping', 'other')),
    CONSTRAINT transactions_owner_check CHECK (owner IN ('van', 'manh'))
);

-- phone automations retry: same notification at same time = same row.
CREATE UNIQUE INDEX transactions_dedupe_key ON transactions (created_at, md5(raw_text));
CREATE INDEX idx_transactions_created_at ON transactions (created_at);
