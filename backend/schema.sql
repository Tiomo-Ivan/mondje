CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table de suivi des lots d'importation
CREATE TABLE IF NOT EXISTS imports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name VARCHAR(255) NOT NULL,
    agency_name VARCHAR(100),
    status VARCHAR(50) DEFAULT 'IN_PROGRESS',
    total_lines INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Table des transactions
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    type VARCHAR(10) NOT NULL CHECK (type IN ('CREDIT', 'DEBIT')),
    montant BIGINT NOT NULL CHECK (montant > 0),
    description VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    import_id UUID REFERENCES imports(id) ON DELETE CASCADE,
    file_line_number INT NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

-- Indexation optimisée
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_uuid_lower ON transactions (LOWER(id::text));
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions (created_at) WHERE deleted_at IS NULL;