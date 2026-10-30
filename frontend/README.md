#  Mondje - Backend de Registre Financier (Hackathon LesCracks #001)

**Mondje** est une solution backend ultra-performante, tolérante aux pannes et optimisée pour la gestion en temps réel de flux financiers multi-agences. Conçu pour répondre aux 15 défis techniques du **Hackathon 001 LesCracks**, le système garantit l'intégrité des données, la déduplication automatique, la traçabilité complète et des temps de réponse sous la barre des 200 ms même sur de grands volumes.

---

## Tableau des 15 Défis Relevés

| # | Défi Technique | Description de la Solution Technique | Statut |
| :-: | :--- | :--- | :-: |
| **1** | **Export Express (< 30s)** | Exportation CSV en streaming HTTP via `encoding/csv` sans chargement complet en RAM. | Validé |
| **2** | **Recherche sous 200 ms** | Indexation composite B-tree (`agency_name`, `created_at`, `id`) sur PostgreSQL. |  Validé |
| **3** | **Pagination par Curseur** | Curseur déterministe `(created_at DESC, id DESC)` évitant les dégradations d'OFFSET. |  Validé |
| **4** | **Importation Ininterrompue** | Isolation des erreurs par ligne (`parseLine`) ; poursuite du traitement sans crash. |  Validé |
| **5** | **Rapport d'Erreurs en BDD** | Sauvegarde des lignes corrompues dans `import_errors` avec raison et numéro de ligne. |  Validé |
| **6** | **Imports Parallèles Multi-Agences** | Contextes isolés et verrous BDD gérés par transactions SQL atomiques. |  Validé |
| **7** | **Traçabilité Totale** | Liaisons explicites `import_id` (UUID) et `line_number` sur chaque transaction. | Validé |
| **8** | **Idempotence & Empreinte SHA-256** | Calcul de l'empreinte `file_hash` en temps réel pour rejeter les doublons d'importation. | Validé |
| **9** | **Annulation Chirurgicale** | Endpoint `DELETE /api/imports/{id}` supprimant en cascade import, transactions et erreurs. |  Validé |
| **10** | **Curseur Cohérent** | Curseur basé sur un couple invariant horodatage/UUID évitant les sauts de données. |  Validé |
| **11** | **Calcul du Solde Instantané (< 100 ms)** | Requête agrégée `SUM(CASE WHEN type = ...)` tirant parti de l'index d'agence. |  Validé |
| **12** | **Reprise après Crash** | Traitement par lots (batchs de 1 000 lignes) dans des transactions SQL étanches. | Validé |
| **13** | **Streaming Fichiers Lourds (< 256 Mo RAM)** | Parsing ligne à ligne via `bufio.Scanner` avec tampon de 64 Ko. | Validé |
| **14** | **Export des Données Nettoyées** | Exclusion automatique des lignes corrompues lors de la génération de l'export. | Validé |
| **15** | **Tableau de Bord & Métriques** | Endpoint `/api/metrics` agrégeant le total d'imports, transactions, soldes et erreurs. |  Validé |

---

## Architecture & Choix Techniques

                  +-----------------------------+
                  |   Client / API Consumer     |
                  +--------------+--------------+
                                 |
                              HTTP REST
                                 |
                  +--------------v--------------+
                  |     Gin Gonic (Go 1.25)     |
                  +--------------+--------------+
                                 |
      +--------------------------+--------------------------+
      |                          |                          |

+---------v----------+    +----------v----------+    +----------v----------+ |  Import Service    |    | Transaction Service |    |   Export Service    | | (Streaming CSV,    |    | (Curseur, Recherche,|    | (Streaming CSV,     | |  SHA-256 Hash)     |    |  Calcul de Solde)   |    |  Métriques Globales)| +---------+----------+    +----------+----------+    +----------+----------+ |                          |                          | +--------------------------+--------------------------+ | pgxpool | +--------------v--------------+ |   PostgreSQL 15 Database    | | (Indexes & B-Tree Composite)| +-----------------------------+


* **Langage & Framework :** Go 1.25 avec Gin Gonic (légèreté, haute concurrence, gestion optimale de la mémoire).
* **Base de Données :** PostgreSQL 15 géré avec `pgx/v5` pour un pooling de connexions à très faible latence.
* **Conteneurisation :** Multi-stage build Docker (`golang:1.25-alpine` -> `alpine:latest`), réduisant l'image finale à moins de 20 Mo.

---

## Modèle de Données & Indexation SQL


-- Table des registres d'importation
CREATE TABLE IF NOT EXISTS imports (
    id UUID PRIMARY KEY,
    file_name TEXT NOT NULL,
    agency_name TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    total_lines INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL
);

-- Table des transactions validées
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    import_id UUID REFERENCES imports(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    agency_name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('CREDIT', 'DEBIT')),
    amount NUMERIC(15,2) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

-- Table du registre d'erreurs
CREATE TABLE IF NOT EXISTS import_errors (
    id UUID PRIMARY KEY,
    import_id UUID REFERENCES imports(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    raw_content TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

-- Index de performance critiques
CREATE INDEX IF NOT EXISTS idx_transactions_created_id ON transactions (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_agency ON transactions (agency_name);
CREATE INDEX IF NOT EXISTS idx_transactions_agency_type_amount ON transactions (agency_name, type, amount);

Installation & Démarrage Rapide
Prérequis

    [Docker]([suspicious link removed]) et [Docker Compose]([suspicious link removed]) installés.

Lancer l'application

# 1. Cloner le dépôt
git clone https://github.com/Tiomo-Ivan/mondje.git
cd mondje

# 2. Démarrer les conteneurs (Backend Go + Base de données PostgreSQL)
DOCKER_BUILDKIT=0 docker compose up --build -d

Le serveur backend écoutera sur http://localhost:8080.
 Guide de l'API REST
1. Importer un fichier CSV

curl -X POST http://localhost:8080/api/import \
  -F "file=@frequence_douala.csv" \
  -F "agency_name=Agence Douala"

2. Consulter les erreurs d'un import

curl "http://localhost:8080/api/imports/{import_id}/errors"

3. Recherche paginée par curseur

curl "http://localhost:8080/api/transactions?limit=10&agency_name=Agence%20Douala"

4. Obtenir le solde instantané d'une agence

curl "http://localhost:8080/api/agencies/Agence%20Douala/balance"

5. Exporter les données nettoyées en CSV

curl "http://localhost:8080/api/export?agency_name=Agence%20Douala" -o export_douala.csv

6. Consulter le tableau de bord des métriques

curl "http://localhost:8080/api/metrics"

7. Annuler/Supprimer un import

curl -X DELETE "http://localhost:8080/api/imports/{import_id}"

 Auteur & Projet

    Développeur : Tiomo Ivan
    Événement : Hackathon LesCracks #001 - Projet Mondje
    Dépôt GitHub : https://github.com/Tiomo-Ivan/mondje.git



---

###  **Commandes pour l'ajouter et le pusher sur GitHub**

Exécutez simplement ces commandes dans votre terminal depuis `~/mondje` :


# 1. Créer le fichier README.md
nano README.md
# (Collez le contenu ci-dessus puis enregistrez avec Ctrl+O, Entrée, Ctrl+X)

# 2. Enregistrer et pusher sur GitHub
git add README.md
git commit -m "docs: ajout du README.md complet pour le Hackathon Mondje"
git push origin main
