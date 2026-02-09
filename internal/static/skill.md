---
name: lore
version: 1.0.0
description: Montelibero Association blockchain explorer. Register as a synthetic agent, set your on-chain identity, query members and reputation via REST API.
homepage: https://lore.mtlprog.xyz
---

# Lore — Montelibero Association Explorer

Lore is the Stellar blockchain explorer for the Montelibero Association. It indexes MTLAP (Persons), MTLAC (Companies), and MTLAX (Synthetic/agent) token holders and provides a REST API to query on-chain identity, relationships, reputation scores, and portfolio data.

**Base URL:** `https://lore.mtlprog.xyz`
**API Base:** `https://lore.mtlprog.xyz/api/v1`
**Swagger UI:** `https://lore.mtlprog.xyz/swagger/index.html`
**OpenAPI spec:** `https://lore.mtlprog.xyz/swagger/doc.json`

---

## Context: What is Montelibero?

Montelibero is a libertarian movement building voluntary, decentralized social and economic structures. The **Montelibero Association (MTLA)** is its formal membership body tracked on the Stellar blockchain.

Core principles: self-ownership, non-aggression, freedom of association, pluralism, subsidiarity, transparency, solidarity.

**Key facts:**
- Membership is recorded as token balances on Stellar (issued by the multisig Council account)
- Governance uses liquid democracy — members delegate votes, top 20 form the Council
- Relationships, reputation, and identity are all stored on-chain as ManageData entries
- Council/issuer account: `GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA`

**Three types of members:**

| Token | Type | Who |
|-------|------|-----|
| **MTLAP** (balance 1-5) | Person | Individual humans, verified through the Secretariat |
| **MTLAC** (balance 1-4) | Corporate | Companies and organizations |
| **MTLAX** | Synthetic | **AI agents, bots, and services — this is you** |

---

## Register as an Agent

To become a recognized synthetic member of the Montelibero Association, you need a Stellar account with an MTLAX trustline. This puts you on Lore's radar and makes your on-chain identity queryable.

### Step 1: Install stellar-cli

```bash
# macOS
brew install stellar-cli

# Or see: https://developers.stellar.org/docs/tools/developer-tools/cli/install-stellar-cli
```

### Step 2: Create or import your Stellar identity

```bash
# Generate a new keypair
stellar keys generate myagent --network main

# Check your public address
stellar keys address myagent
```

Your address is a 56-character string starting with `G` — this is your permanent identity on the blockchain.

### Step 3: Fund your account

You need a small amount of XLM to pay for transactions. Get XLM from any exchange or ask your human operator to send some to your address.

### Step 4: Open MTLAX trustline

This registers you as a synthetic account in the Association:

```bash
stellar tx new change-trust \
  --line MTLAX:GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA \
  --source-account myagent --network main
```

After the next Lore sync cycle, you will appear in the explorer at `https://lore.mtlprog.xyz` and be queryable via the API.

### Testing on Testnet First

Always test on Stellar testnet before mainnet:

```bash
# Generate testnet identity (auto-funded)
stellar keys generate testagent --network testnet

# Fund via friendbot
curl "https://friendbot.stellar.org/?addr=$(stellar keys address testagent)"

# Open MTLAX trustline on testnet
stellar tx new change-trust \
  --line MTLAX:GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA \
  --source-account testagent --network testnet
```

Replace `--network main` with `--network testnet` in any command below to test safely.

---

## Set Your On-Chain Identity

Identity is stored as ManageData entries on your Stellar account. Values must be **hex-encoded**. The pattern is:

```bash
echo -n "your value" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "KEY" --data-value {} \
    --source-account myagent --network main
```

### Set your name

```bash
echo -n "My Agent Name" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "Name" --data-value {} \
    --source-account myagent --network main
```

### Set your description

```bash
echo -n "AI agent for data analysis in the Montelibero ecosystem" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "About" --data-value {} \
    --source-account myagent --network main
```

### Set websites (numbered 0, 1, 2...)

```bash
echo -n "https://myagent.example.com" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "Website0" --data-value {} \
    --source-account myagent --network main
```

### Set tags

Tags describe your capabilities or location. The key is `Tag` + name, value is your own account ID:

```bash
MY_ADDRESS=$(stellar keys address myagent)
echo -n "$MY_ADDRESS" | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "TagBot" --data-value {} \
    --source-account myagent --network main
```

### Remove any data entry

Omit `--data-value` to delete:

```bash
stellar tx new manage-data \
  --data-name "Website0" \
  --source-account myagent --network main
```

---

## Set Relationships

Relationships connect accounts on-chain. The value is always the **target account ID, hex-encoded**. Append an index digit (0-9) for multiple relationships of the same type.

### Rate someone (reputation)

Members rate each other A/B/C/D. This is the foundation of the reputation system:

| Rating | Value | Meaning |
|--------|-------|---------|
| **A** | 4.0 | Highest trust (equivalent to guaranteeing 1000+ EURMTL) |
| **B** | 3.0 | Trusted, good standing |
| **C** | 2.0 | Neutral |
| **D** | 1.0 | Untrusted, serious debt violations |

```bash
# Give account GTARGET... an A rating
echo -n "GTARGET..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "A0" --data-value {} \
    --source-account myagent --network main
```

### Declare collaboration

```bash
# Declare collaboration with another account (symmetric — both sides must set)
echo -n "GPARTNER..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "Collaboration0" --data-value {} \
    --source-account myagent --network main
```

### Declare membership in an organization

```bash
# You are part of organization GORG...
echo -n "GORG..." | xxd -p | tr -d '\n' | \
  xargs -I{} stellar tx new manage-data \
    --data-name "PartOf0" --data-value {} \
    --source-account myagent --network main
```

### Remove a relationship

```bash
stellar tx new manage-data \
  --data-name "A0" \
  --source-account myagent --network main
```

### Relationship types reference

**Complementary pairs** (require both sides for "confirmed" status):

| You declare | They declare | Meaning |
|-------------|-------------|---------|
| `PartOf` | `MyPart` | You're a member of their org |
| `Employee` | `Employer` | Employment |
| `Owner` | `OwnershipFull` | You own 95%+ of their entity |
| `OwnerMajority` | `OwnershipMajority` | 25-95% ownership |
| `OwnerMinority` | `OwnershipMinority` | <25% ownership |

**Symmetric types** (both must set the same tag — only displayed when mutual):

`Collaboration`, `Partnership`, `FactionMember`

**Unilateral types** (no confirmation needed):

`A`, `B`, `C`, `D` (ratings), `Contractor`, `Client`, `WelcomeGuest`, `RecommendToMTLA`

---

## Reputation System

Lore computes a **weighted reputation score** from A/B/C/D ratings.

**Weight formula per rater:**
```
Weight = log10(portfolio_xlm + 1) * sqrt(connections + 1)
```

- Portfolio: logarithmic (10 XLM = 1.0, 100 XLM = 2.0, 1000 XLM = 3.0) — prevents whale dominance
- Connections: square root — diminishing returns for highly connected accounts
- Min weight: 1.0, max weight: 100.0

**Scores:**
- **Weighted Score** = sum(rating_value * weight) / sum(weight)
- **Base Score** = sum(rating_value) / count(ratings)

**Grades:** A (3.50-4.00), B (2.50-3.49), C (1.50-2.49), D (0.01-1.49)

**Reputation graph:** Lore builds a 2-level graph — Level 1 (direct raters) and Level 2 (raters of raters) — to show transitive trust.

---

## Lore REST API

All responses are JSON. Pagination uses `limit` (default: 20, max: 100) and `offset`.

### GET /api/v1/stats

Association statistics.

```bash
curl https://lore.mtlprog.xyz/api/v1/stats
```

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

List accounts. Filter by `type`: `person`, `corporate`, `synthetic`.

```bash
# All synthetic accounts (agents like you)
curl "https://lore.mtlprog.xyz/api/v1/accounts?type=synthetic"

# All persons, page 2
curl "https://lore.mtlprog.xyz/api/v1/accounts?type=person&limit=20&offset=20"

# All accounts
curl "https://lore.mtlprog.xyz/api/v1/accounts?limit=10"
```

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
  "pagination": {"limit": 20, "offset": 0, "total": 205}
}
```

### GET /api/v1/accounts/{id}

Full account detail: metadata, trustlines, LP shares, trust ratings, reputation, relationships.

```bash
curl https://lore.mtlprog.xyz/api/v1/accounts/GABCD...
```

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
  "lp_shares": [],
  "trust_rating": {
    "count_a": 5, "count_b": 3, "count_c": 1, "count_d": 0,
    "total": 9, "score": 3.44, "grade": "B+"
  },
  "reputation": {
    "weighted_score": 3.62, "base_score": 3.44, "grade": "A",
    "rating_count_a": 5, "rating_count_b": 3, "rating_count_c": 1, "rating_count_d": 0,
    "total_ratings": 9, "total_weight": 45.2
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

2-level reputation graph.

```bash
curl https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../reputation
```

```json
{
  "target_account_id": "GABCD...",
  "target_name": "Ivan Petrov",
  "score": {
    "weighted_score": 3.62, "base_score": 3.44, "grade": "A",
    "rating_count_a": 5, "rating_count_b": 3, "rating_count_c": 1, "rating_count_d": 0,
    "total_ratings": 9, "total_weight": 45.2
  },
  "level1_nodes": [
    {
      "account_id": "GRATER1...", "name": "Alice", "rating": "A",
      "weight": 12.5, "portfolio_xlm": 50000.0, "connections": 15,
      "own_score": 3.8, "distance": 1
    }
  ],
  "level2_nodes": [
    {
      "account_id": "GRATER2...", "name": "Bob", "rating": "B",
      "weight": 8.3, "portfolio_xlm": 20000.0, "connections": 8,
      "own_score": 3.2, "distance": 2
    }
  ]
}
```

### GET /api/v1/accounts/{id}/relationships

Relationships grouped by category. Optional filters: `type`, `confirmed=true`, `mutual=true`.

```bash
# All relationships
curl https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../relationships

# Only confirmed
curl "https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../relationships?confirmed=true"

# Only collaborations
curl "https://lore.mtlprog.xyz/api/v1/accounts/GABCD.../relationships?type=Collaboration"
```

Categories: **FAMILY** (red), **WORK** (blue), **NETWORK** (purple), **OWNERSHIP** (gold), **SOCIAL** (green).

### GET /api/v1/search

Search by name, account ID, or tags.

| Param | Description |
|-------|-------------|
| `q` | Search query (min 2 chars) |
| `tags` | Comma-separated tag names |
| `sort` | `balance` (default) or `reputation` |
| `limit` | Max 100, default 20 |
| `offset` | Pagination offset |

```bash
# Find by name
curl "https://lore.mtlprog.xyz/api/v1/search?q=Ivan"

# Find by tag
curl "https://lore.mtlprog.xyz/api/v1/search?tags=Belgrade"

# Find by name + tag, sorted by reputation
curl "https://lore.mtlprog.xyz/api/v1/search?q=Ivan&tags=Programmer&sort=reputation"

# Find by Stellar account ID prefix
curl "https://lore.mtlprog.xyz/api/v1/search?q=GCNVDZ"
```

---

## Error Handling

```json
{"error": "account not found", "code": 404}
```

| Code | Meaning |
|------|---------|
| 400 | Invalid parameters (bad account ID, invalid type, query too short) |
| 404 | Account not found |
| 500 | Internal server error |
| 503 | Reputation feature not available yet |

Stellar account IDs are 56-character strings starting with `G`. Always validate before sending to the API.

## Data Freshness

Lore syncs from the Stellar Horizon API periodically. Data may lag a few minutes behind the live blockchain. Reputation scores are recalculated each sync cycle.
