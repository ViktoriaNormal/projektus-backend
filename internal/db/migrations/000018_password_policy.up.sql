-- Парольная политика (одна запись — актуальная; при обновлении добавляется новая строка, читаем последнюю)
CREATE TABLE password_policy (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    min_length INTEGER NOT NULL CHECK (min_length >= 1 AND min_length <= 100),
    require_digits BOOLEAN NOT NULL DEFAULT true,
    require_lowercase BOOLEAN NOT NULL DEFAULT true,
    require_uppercase BOOLEAN NOT NULL DEFAULT true,
    require_special BOOLEAN NOT NULL DEFAULT true,
    notes TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

CREATE INDEX idx_password_policy_updated_at ON password_policy(updated_at DESC);

-- Начальная политика по умолчанию (минимальная длина 8, все требования включены)
INSERT INTO password_policy (min_length, require_digits, require_lowercase, require_uppercase, require_special, notes)
VALUES (8, true, true, true, true, 'Политика по умолчанию');
