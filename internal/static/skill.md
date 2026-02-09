---
name: lore
version: 1.0.0
description: Montelibero Association blockchain explorer. Query MTLA members, relationships, reputation scores, and on-chain identity via REST API.
homepage: https://lore.mtlprog.xyz
---

# Lore — Montelibero Association Explorer

Lore is a Stellar blockchain token explorer for the Montelibero Association. It indexes all MTLAP (Persons), MTLAC (Companies), and MTLAX (Synthetic) token holders and exposes their on-chain identity, relationships, reputation scores, and portfolio data via a REST API.

**Base URL:** `https://lore.mtlprog.xyz`
**Swagger UI:** `https://lore.mtlprog.xyz/swagger/index.html`
**OpenAPI spec:** `https://lore.mtlprog.xyz/swagger/doc.json`

---

## What is Montelibero?

Montelibero is a libertarian movement and community focused on building voluntary, decentralized social and economic structures. Its core principles:

- **Self-ownership**: People belong only to themselves; the organization expands personal sovereignty
- **Non-aggression**: Adherence to the non-aggression principle; opposing aggression against members
- **Freedom of association**: Right to join and leave freely, with fair property division
- **Pluralism**: No monopoly on truth; respect for individual choices
- **Subsidiarity**: Decisions made at the lowest effective level, involving those affected
- **Transparency**: Open operation of the Association while respecting privacy and security
- **Solidarity**: Peaceful conflict resolution with a presumption of good faith

## Montelibero Association (MTLA)

The Association is the formal structure within the Montelibero movement. It has fixed membership tracked on the Stellar blockchain through tokens issued from a multisig Council account.

**Key governance features:**
- Liquid democracy model with delegation of voting power
- Council of 20 verified members with the most delegated votes
- All membership statuses recorded as token balances on Stellar
- Council account (multisig): `GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA`

### Token Types

| Token | Purpose | Issuer |
|-------|---------|--------|
| **MTLAP** | Individual membership (Persons) | `GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA` |
| **MTLAC** | Corporate membership (Companies) | `GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA` |
| **MTLAX** | Synthetic accounts (bots, services) | `GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA` |

### Individual Membership Levels (MTLAP balance)

| Balance | Status | Requirements |
|---------|--------|-------------|
| 1 | New member | Accepted by Secretariat |
| 2 | Verified | Personal data verified |
| 3 | Creditworthy | Personal token >= 1000 EURMTL or equivalent assets |
| 4 | Active contributor | Consistent governance participation or valuable projects |
| 5 | Major donor | Significant financial contributions |

### Corporate Membership Levels (MTLAC balance)

| Balance | Status | Requirements |
|---------|--------|-------------|
| 1 | Registered | Accepted by Secretariat |
| 2 | Certified | Passed certification |
| 3 | Active | Signed commercial contracts with other members |
| 4 | Donor | Significant corporate contributions |

### How to Join

1. Create a Stellar account (wallet)
2. Add a trustline to the MTLAP token (issuer: `GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA`)
3. Submit an application via the Telegram bot with your Stellar public address
4. Get a recommendation from a verified member (they set `RecommendToMTLA` tag pointing to your account)
5. Secretariat reviews and issues the token

---

## On-Chain Identity (ManageData)

All identity and relationship data is stored as ManageData entries on each Stellar account. Values are stored in base64 on the blockchain.

### Self-Presentation Fields

| Key | Description | Example value |
|-----|-------------|---------------|
| `Name` | Display name | `Ivan Petrov` |
| `About` | Bio / description | `Developer and MTL enthusiast` |
| `Website0`, `Website1`, ... | Websites (numbered) | `https://example.com` |
| `TagBelgrade`, `TagProgrammer`, ... | Tags (prefixed with `Tag`) | Account ID (value ignored for display) |

### Delegation

| Key | Value | Meaning |
|-----|-------|---------|
| `mtla_delegate` | Account ID | Delegates general voting power to that account |
| `mtla_c_delegate` | `ready` | Declares willingness to serve on Council |
| `mtla_c_delegate` | Account ID | Delegates council vote to that account |

---

## Relationships (Blockchainization Standard)

Relationships between accounts are declared via ManageData entries. The key is the relationship type (optionally with an index digit 0-9 for multiple relationships of the same type), and the value is the target account ID.

**Important:** Complementary and symmetric relationships only have legal force when confirmed by BOTH parties. A one-way declaration may be considered false information.

### Complementary Pairs (require both sides for confirmation)

| Side A declares | Side B declares | Meaning |
|----------------|----------------|---------|
| `MyPart` (org -> person) | `PartOf` (person -> org) | Membership in organization |
| `Guardian` (guardian -> ward) | `Ward` (ward -> guardian) | Guardianship |
| `OwnershipFull` (corp -> owner) | `Owner` (owner -> corp) | Full ownership (95%+) |
| `OwnershipMajority` (corp -> owner) | `OwnerMajority` (owner -> corp) | Majority ownership (25-95%) |
| `OwnershipMinority` (corp -> owner) | `OwnerMinority` (owner -> corp) | Minority ownership (<25%) |
| `Employer` (employer -> employee) | `Employee` (employee -> employer) | Employment |

### Symmetric Types (both must set the same tag = mutual)

These are only displayed when BOTH parties have declared. One-way declarations are hidden.

- `Spouse` — Marriage
- `OneFamily` — Family membership
- `Partnership` — Business partnership
- `Collaboration` — Collaboration
- `FactionMember` — Faction membership

### Unilateral Tags (no confirmation needed)

- `A`, `B`, `C`, `D` — Reputation rating (see Reputation section below)
- `Sympathy` — Sympathy declaration
- `Love` — Love declaration
- `Divorce` — Divorce declaration
- `Contractor` — Contractor relationship
- `Client` — Client relationship
- `WelcomeGuest` — Welcome guest invitation
- `RecommendToMTLA` — Recommendation for MTLA membership

### Relationship Categories

| Category | Types | Color |
|----------|-------|-------|
| FAMILY | OneFamily, Spouse, Guardian, Ward, Sympathy, Love, Divorce | Red |
| WORK | Employer, Employee, Contractor, Client | Blue |
| NETWORK | Partnership, Collaboration, MyPart, PartOf, RecommendToMTLA | Purple |
| OWNERSHIP | OwnershipFull/Majority/Minority, Owner/OwnerMajority/OwnerMinority | Gold |
| SOCIAL | WelcomeGuest, FactionMember | Green |

---

## Reputation System

Reputation in Montelibero is based on ABCD credit ratings that members assign to each other on-chain.

### Rating Values

| Rating | Numeric Value | Meaning |
|--------|--------------|---------|
| **A** | 4.0 | Highest trust — equivalent to guaranteeing 1000+ EURMTL |
| **B** | 3.0 | Trusted — good standing |
| **C** | 2.0 | Neutral — no strong opinion |
| **D** | 1.0 | Untrusted — serious debt violations |

### How Reputation Score is Calculated

Lore computes a **weighted reputation score** where each rater's vote is weighted by their portfolio size and social connections:

```
Weight = log10(portfolio_xlm + 1) * sqrt(connections + 1)
```

- **Portfolio weight**: Logarithmic scaling prevents whale dominance (10 XLM = 1.0, 100 XLM = 2.0, 1000 XLM = 3.0)
- **Connection weight**: Square root scaling gives diminishing returns for highly connected accounts
- Minimum weight: 1.0 (every vote counts)
- Maximum weight: 100.0 (cap to prevent outliers)

**Weighted Score** = sum(rating_value * weight) / sum(weight)
**Base Score** = sum(rating_value) / count(ratings)

### Grade Conversion

| Score Range | Grade |
|-------------|-------|
| 3.50 - 4.00 | A |
| 2.50 - 3.49 | B |
| 1.50 - 2.49 | C |
| 0.01 - 1.49 | D |

### Reputation Graph

Lore builds a 2-level reputation graph for each account:
- **Level 1**: Direct raters (accounts that gave A/B/C/D ratings to the target)
- **Level 2**: Raters of the Level 1 raters (shows transitive trust)

Each node includes: rating given, weight, portfolio in XLM, connection count, own reputation score, and distance (1 or 2).

---

## Lore REST API

**Base URL:** `https://lore.mtlprog.xyz/api/v1`

All responses are JSON. Pagination uses `limit` (default: 20, max: 100) and `offset` parameters.

### GET /api/v1/stats

Returns aggregate statistics for the Montelibero Association.

```bash
curl https://lore.mtlprog.xyz/api/v1/stats
```

Response:
```json
{
  "total_accounts": 394,
  "total_persons": 205,
  "total_companies": 75,
  "total_synthetic": 1,
  "total_xlm_value": 34023011.33
}
```

### GET /api/v1/accounts

List accounts with optional type filter.

**Parameters:**
- `type` — Filter: `person`, `corporate`, or `synthetic`
- `limit` — Results per page (default: 20, max: 100)
- `offset` — Pagination offset

```bash
# List all persons
curl "https://lore.mtlprog.xyz/api/v1/accounts?type=person&limit=5"

# List all corporate accounts
curl "https://lore.mtlprog.xyz/api/v1/accounts?type=corporate"

# List all accounts (no filter)
curl "https://lore.mtlprog.xyz/api/v1/accounts?limit=10&offset=20"
```

Response:
```json
{
  "data": [
    {
      "id": "GABCD...",
      "name": "Ivan Petrov",
      "type": "person",
      "mtlap_balance": 3,
      "mtlac_balance": 0,
      "mtlax_balance": 0,
      "total_xlm_value": 15000.5,
      "reputation_score": 3.75,
      "reputation_grade": "A",
      "is_council_ready": true,
      "received_votes": 12
    }
  ],
  "pagination": {
    "limit": 5,
    "offset": 0,
    "total": 205
  }
}
```

### GET /api/v1/accounts/{id}

Full account detail including metadata, trustlines, LP shares, ratings, reputation, and relationships.

```bash
curl https://lore.mtlprog.xyz/api/v1/accounts/GABCD...
```

Response:
```json
{
  "id": "GABCD...",
  "name": "Ivan Petrov",
  "about": "Developer and MTL enthusiast",
  "websites": ["https://example.com"],
  "tags": ["Belgrade", "Programmer"],
  "is_corporate": false,
  "total_xlm_value": 15000.5,
  "trustlines": [
    {"asset_code": "MTLAP", "asset_issuer": "GCNVDZ...", "balance": "3.0000000"}
  ],
  "lp_shares": [
    {
      "pool_id": "abc123...",
      "share_balance": "100.0000000",
      "share_percent": "0.05%",
      "reserve_a": {"asset_code": "EURMTL", "asset_issuer": "GCNVDZ...", "amount": "50.0000"},
      "reserve_b": {"asset_code": "native", "asset_issuer": "", "amount": "1000.0000"},
      "xlm_value": 2000.0
    }
  ],
  "trust_rating": {
    "count_a": 5,
    "count_b": 3,
    "count_c": 1,
    "count_d": 0,
    "total": 9,
    "score": 3.44,
    "grade": "B+"
  },
  "reputation": {
    "weighted_score": 3.62,
    "base_score": 3.44,
    "grade": "A",
    "rating_count_a": 5,
    "rating_count_b": 3,
    "rating_count_c": 1,
    "rating_count_d": 0,
    "total_ratings": 9,
    "total_weight": 45.2
  },
  "categories": [
    {
      "name": "WORK",
      "color": "#58a6ff",
      "relationships": [
        {
          "type": "Employee",
          "target_id": "GXYZ...",
          "target_name": "MTL Corp",
          "direction": "outgoing",
          "is_mutual": false,
          "is_confirmed": true
        }
      ]
    }
  ]
}
```

### GET /api/v1/accounts/{id}/reputation

Returns the 2-level reputation graph.

```bash
curl https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../reputation
```

Response:
```json
{
  "target_account_id": "GABCD...",
  "target_name": "Ivan Petrov",
  "score": {
    "weighted_score": 3.62,
    "base_score": 3.44,
    "grade": "A",
    "rating_count_a": 5,
    "rating_count_b": 3,
    "rating_count_c": 1,
    "rating_count_d": 0,
    "total_ratings": 9,
    "total_weight": 45.2
  },
  "level1_nodes": [
    {
      "account_id": "GRATER1...",
      "name": "Alice",
      "rating": "A",
      "weight": 12.5,
      "portfolio_xlm": 50000.0,
      "connections": 15,
      "own_score": 3.8,
      "distance": 1
    }
  ],
  "level2_nodes": [
    {
      "account_id": "GRATER2...",
      "name": "Bob",
      "rating": "B",
      "weight": 8.3,
      "portfolio_xlm": 20000.0,
      "connections": 8,
      "own_score": 3.2,
      "distance": 2
    }
  ]
}
```

### GET /api/v1/accounts/{id}/relationships

Returns relationships grouped by category with optional filters.

**Parameters:**
- `type` — Filter by relationship type (e.g., `Spouse`, `Employer`, `PartOf`)
- `confirmed` — `true` to show only confirmed relationships
- `mutual` — `true` to show only mutual relationships

```bash
# All relationships
curl https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../relationships

# Only confirmed relationships
curl "https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../relationships?confirmed=true"

# Only employer/employee relationships
curl "https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../relationships?type=Employee"
```

### GET /api/v1/search

Search accounts by name, account ID, or tags.

**Parameters:**
- `q` — Search query (min 2 characters)
- `tags` — Comma-separated tag names (e.g., `Belgrade,Programmer`)
- `sort` — Sort by `balance` (default) or `reputation`
- `limit` — Results per page (default: 20, max: 100)
- `offset` — Pagination offset

```bash
# Search by name
curl "https://lore.mtlprog.xyz/api/v1/search?q=Ivan"

# Search by tag
curl "https://lore.mtlprog.xyz/api/v1/search?tags=Belgrade"

# Search by name and tag, sorted by reputation
curl "https://lore.mtlprog.xyz/api/v1/search?q=Ivan&tags=Programmer&sort=reputation"

# Search by Stellar account ID
curl "https://lore.mtlprog.xyz/api/v1/search?q=GABCD"
```

---

## Using stellar-cli for On-Chain Operations

The `stellar` CLI can be used to set identity, relationships, delegation, and reputation ratings on the Stellar blockchain.

### Prerequisites

Install stellar-cli: https://developers.stellar.org/docs/tools/developer-tools/cli/install-stellar-cli

```bash
# Generate a new identity (or import existing)
stellar keys generate myidentity --network main

# Check your address
stellar keys address myidentity
```

### Setting Identity (ManageData)

Values must be hex-encoded. Use `echo -n "value" | xxd -p | tr -d '\n'` to convert.

```bash
# Set your display name
echo -n "Ivan Petrov" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "Name" --data-value {} \
    --source-account myidentity --network main

# Set your bio
echo -n "Developer and MTL enthusiast" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "About" --data-value {} \
    --source-account myidentity --network main

# Set website (numbered — use 0, 1, 2... for multiple)
echo -n "https://example.com" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "Website0" --data-value {} \
    --source-account myidentity --network main

# Set a tag (value is your own account ID)
echo -n "GABCDEF..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "TagProgrammer" --data-value {} \
    --source-account myidentity --network main
```

### Adding Trustline for MTLA Tokens

Before joining the Association, you must add a trustline to the token:

```bash
# Add MTLAP trustline (for individuals)
stellar tx new change-trust \
  --line MTLAP:GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA \
  --source-account myidentity --network main

# Add MTLAC trustline (for companies)
stellar tx new change-trust \
  --line MTLAC:GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA \
  --source-account myidentity --network main
```

### Setting Relationships

The value is always the target account ID, hex-encoded. Append an index digit (0-9) for multiple relationships of the same type.

```bash
# Rate someone A (highest trust)
echo -n "GTARGET..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "A0" --data-value {} \
    --source-account myidentity --network main

# Declare membership in an organization (PartOf)
echo -n "GORG..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "PartOf0" --data-value {} \
    --source-account myidentity --network main

# Declare spouse (symmetric — both parties must set this)
echo -n "GSPOUSE..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "Spouse0" --data-value {} \
    --source-account myidentity --network main

# Declare employment (Employee side)
echo -n "GEMPLOYER..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "Employee0" --data-value {} \
    --source-account myidentity --network main
```

### Setting Delegation

```bash
# Delegate general voting power
echo -n "GDELEGATE..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "mtla_delegate" --data-value {} \
    --source-account myidentity --network main

# Declare council readiness
echo -n "ready" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "mtla_c_delegate" --data-value {} \
    --source-account myidentity --network main

# Delegate council vote to someone
echo -n "GCOUNCIL..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "mtla_c_delegate" --data-value {} \
    --source-account myidentity --network main
```

### Removing Data Entries

Omit `--data-value` to delete a ManageData entry:

```bash
# Remove a relationship
stellar tx new manage-data \
  --data-name "A0" \
  --source-account myidentity --network main

# Remove delegation
stellar tx new manage-data \
  --data-name "mtla_delegate" \
  --source-account myidentity --network main
```

### Testing on Testnet

Always test on Stellar testnet first:

```bash
# Generate testnet identity
stellar keys generate testperson --network testnet

# Fund via friendbot
curl "https://friendbot.stellar.org/?addr=$(stellar keys address testperson)"

# Run any command above with --network testnet instead of --network main
echo -n "Test Person" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "Name" --data-value {} \
    --source-account testperson --network testnet
```

---

## Quick Reference: Account Types

| Type | Token | How to identify via API |
|------|-------|------------------------|
| **Person** | MTLAP (1-5) | `type: "person"` or `mtlap_balance > 0` |
| **Corporate** | MTLAC (1-4) | `type: "corporate"` or `mtlac_balance > 0` |
| **Synthetic** | MTLAX (has trustline, no MTLAP) | `type: "synthetic"` or `mtlax_balance > 0 && mtlap_balance == 0` |

## Quick Reference: Common API Queries

```bash
# How many members does Montelibero have?
curl https://lore.mtlprog.xyz/api/v1/stats

# Who are the top council candidates?
curl "https://lore.mtlprog.xyz/api/v1/accounts?type=person&limit=20"

# Find someone by name
curl "https://lore.mtlprog.xyz/api/v1/search?q=Ivan"

# Find all people in Belgrade
curl "https://lore.mtlprog.xyz/api/v1/search?tags=Belgrade"

# Get full profile of a specific account
curl https://lore.mtlprog.xyz/api/v1/accounts/GABCD...

# See who trusts someone and how much
curl https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../reputation

# Check only confirmed business relationships
curl "https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../relationships?confirmed=true&type=Employee"
```

---

## Error Handling

All errors return JSON:

```json
{
  "error": "account not found",
  "code": 404
}
```

Common error codes:
- `400` — Invalid parameters (bad account ID format, invalid type filter, query too short)
- `404` — Account not found
- `500` — Internal server error
- `503` — Reputation feature not available (reputation data not yet calculated)

## Stellar Account ID Format

Stellar account IDs are 56-character strings starting with `G`. Example: `GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA`. Always validate before sending to the API.

## Data Freshness

Lore syncs data from the Stellar Horizon API periodically. Data may be up to a few minutes behind the live blockchain. The reputation scores are recalculated during each sync cycle.
