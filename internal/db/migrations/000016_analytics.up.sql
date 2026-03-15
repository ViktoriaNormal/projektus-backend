-- Analytics cache for heavy reports

CREATE TABLE analytics_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    report_type TEXT NOT NULL,
    parameters JSONB NOT NULL,
    result_data JSONB NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_analytics_cache_project_type ON analytics_cache(project_id, report_type);
CREATE INDEX idx_analytics_cache_expires_at ON analytics_cache(expires_at);

