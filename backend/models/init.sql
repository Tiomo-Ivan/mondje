-- Extension pour la génération d'UUIDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table de suivi des imports de fichiers
CREATE TABLE IF NOT EXISTS imports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name VARCHAR(255) NOT NULL,
    agency_name VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'IN_PROGRESS',
    total_lines INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table des transactions financières
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(10) NOT NULL CHECK (type IN ('CREDIT', 'DEBIT')),
    montant BIGINT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    import_id UUID REFERENCES imports(id) ON DELETE CASCADE,
    file_line_number INT NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

-- Index pour accélérer les recherches et le filtrage des doublons/soft deletes
CREATE INDEX IF NOT EXISTS idx_transactions_import_id ON transactions(import_id);
CREATE INDEX IF NOT EXISTS idx_transactions_deleted_at ON transactions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);
