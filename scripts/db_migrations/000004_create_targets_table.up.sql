CREATE TABLE IF NOT EXISTS targets
(
    name VARCHAR(80) NOT NULL,
    project VARCHAR(80) NOT NULL,
    properties JSONB,
    type VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT targets_pkey PRIMARY KEY(name, project),
    CONSTRAINT targets_project_fkey FOREIGN KEY (project) REFERENCES projects(project) ON DELETE CASCADE ON UPDATE CASCADE
);

GRANT ALL PRIVILEGES ON targets TO cello;
