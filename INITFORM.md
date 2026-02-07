# INITFORM.md

Documentation for the Montelibero Initiation Form (https://init.panarchy.now/)

## Overview

The Montelibero Initiation Form is a web application that allows Stellar account holders to manage their blockchain identity metadata. It generates unsigned Stellar transactions (XDR) containing ManageData operations to set/update account data entries.

## Form Types

### Participant Data (`/participant`)

For individual persons (MTLAP token holders).

**Fields:**

| Field | Key(s) | Required | Limit | Description |
|-------|--------|----------|-------|-------------|
| Account ID | - | Yes | - | Stellar public key (G...) |
| Name | `Name` | Yes | 64 bytes | Full name |
| About | `About` | Yes | 64 bytes | Short bio/description |
| Website | `Website` | No | 64 bytes | Personal website URL |
| PartOf | `PartOf001`, `PartOf002`, ... | No | - | Organization memberships (account IDs) |
| Telegram User ID | `TelegramUserID` | No | 64 bytes | Telegram numeric user ID |
| Tags | `TagBelgrade`, `TagDeveloper`, etc. | No | - | Profile tags (value = own account ID) |
| Time Token Code | `TimeTokenCode` | No | 64 bytes | Personal time token code (e.g., STAS) |
| Time Token Issuer | `TimeTokenIssuer` | No | 56 bytes | Issuer account ID |
| Time Token Description | `TimeTokenDesc` | No | 64 bytes | Description of services offered |
| Time Token Offer IPFS | `TimeTokenOfferIPFS` | No | - | IPFS hash for detailed offer |

### Corporate Data (`/corporate`)

For organizations/companies (MTLAC token holders).

**Fields:**

| Field | Key(s) | Required | Limit | Description |
|-------|--------|----------|-------|-------------|
| Account ID | - | Yes | - | Stellar public key (G...) |
| Company Name | `Name` | Yes | 64 bytes | Official company name |
| About | `About` | Yes | 64 bytes | Company description |
| Website | `Website` | No | 64 bytes | Company website URL |
| MTLA PII Standard | `MTLA: PII Standard` | No | - | Toggle for PII certification |
| MyPart | `MyPart001`, `MyPart002`, ... | No | - | Organization members (participant account IDs) |
| Telegram Part Chat ID | `TelegramPartChatID` | No | - | Group chat ID for members |
| Tags | `TagBelgrade`, `TagInvestor`, etc. | No | - | Company tags |
| Contract IPFS | `ContractIPFS` | No | - | IPFS hash for company contract |

## Available Tags

Both forms share the same tag set:
- Belgrade, Montenegro (location)
- Blogger, Developer, Designer, Investor, Charity (roles)
- AIAgent (for AI agents participating in the network)
- Blockchain, Crypto, Nft, Defi (tech)
- Startup, Business, Marketing, Sales, Management, Entrepreneur (business)
- Ancap, Libertarian, Panarchist, Anarchist, Accelerationist (ideology)

## How It Currently Works (JS-based)

### Data Flow

1. **Load**: User enters Account ID → `onBlur` event triggers fetch from Stellar Horizon API
2. **Parse**: Response `data_attr` field contains base64-encoded ManageData entries
3. **Populate**: Form fields are filled with decoded values
4. **Track**: Original data is stored for comparison
5. **Edit**: User modifies fields, adds/removes parts, toggles tags
6. **Generate**: "Generate Transaction" compares original vs current, creates XDR with only changed operations
7. **Export**: XDR displayed for copying, SEP-0007 deep link, or MMWB wallet button

### Transaction Generation Logic

- **New/Changed values**: `manageData(key, newValue)` - sets the value
- **Deleted values**: `manageData(key, null)` - removes the key
- **Unchanged values**: Not included in transaction

### Numbered Keys (PartOf, MyPart)

Relations use numbered suffixes with leading zeros preserved:
- `PartOf001`, `PartOf002`, ..., `PartOf014`
- When a relation is deleted, remaining ones shift to fill the gap
- Index format: 3-digit zero-padded (001, 002, etc.)

## Known Issues (Current JS Implementation)

1. **State Reset on Tab Switch**: Form data is lost when switching between Participant/Corporate tabs
2. **Multiple Redundant API Calls**: Same account fetched multiple times
3. **No Offline Support**: Requires constant internet connection
4. **No Error Recovery**: If API call fails, form is left in inconsistent state
5. **No Validation Before Generate**: Can generate invalid transactions
6. **Complex Client-side State**: Tracking original vs current values in JS is error-prone

## Proposed Server-Side Implementation

### Architecture

Replace client-side JS logic with server-rendered HTML and minimal interactivity.

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Browser   │────>│   Server    │────>│   Horizon   │
│  (HTML+CSS) │<────│   (Go)      │<────│   API       │
└─────────────┘     └─────────────┘     └─────────────┘
```

### Routes

```
GET  /init                     → Landing page with account ID form
POST /init/participant         → Load participant data, render form
POST /init/participant/preview → Generate XDR preview
POST /init/corporate           → Load corporate data, render form
POST /init/corporate/preview   → Generate XDR preview
```

### Flow

1. **Landing Page**: Simple form with account ID input and type selector
2. **Load Data**: Form POST → Server fetches from Horizon → Renders pre-filled form
3. **Hidden State**: Original values stored in hidden form fields
4. **Edit**: Standard HTML form editing (no JS needed for basic fields)
5. **Dynamic Parts**: Use `<template>` + minimal JS, or server-side add/remove with form resubmit
6. **Generate**: Form POST → Server compares hidden original vs submitted values → Generates XDR → Renders result page

### Form Structure

```html
<form action="/init/participant/preview" method="POST">
  <!-- Account ID (readonly after load) -->
  <input type="hidden" name="account_id" value="G...">

  <!-- Original values for comparison -->
  <input type="hidden" name="original_name" value="Stanislav Karkavin">
  <input type="hidden" name="original_about" value="...">

  <!-- Editable fields -->
  <input type="text" name="name" value="Stanislav Karkavin" maxlength="64">
  <input type="text" name="about" value="..." maxlength="64">

  <!-- PartOf relations -->
  <input type="hidden" name="original_partof" value="001:G...,002:G...,003:G...">
  <input type="text" name="partof[]" value="G...">
  <input type="text" name="partof[]" value="G...">

  <!-- Tags as checkboxes -->
  <input type="hidden" name="original_tags" value="belgrade,developer,ancap">
  <input type="checkbox" name="tags[]" value="belgrade" checked>
  <input type="checkbox" name="tags[]" value="developer" checked>

  <button type="submit">Generate Transaction</button>
</form>
```

### Advantages of Server-Side Approach

1. **Simpler State Management**: Server compares values on submit, no client tracking
2. **Works Without JS**: Core functionality works with HTML forms only
3. **Single API Call**: Horizon fetched once on initial load
4. **Better Error Handling**: Server can validate and return error pages
5. **Cacheable**: Static assets, predictable responses
6. **Accessible**: Standard form controls work with screen readers
7. **Debuggable**: Server logs, no browser console needed

### Progressive Enhancement

For better UX without requiring JS:

1. **Add/Remove Parts**: Submit form with action parameter, server re-renders
2. **Tag Toggle**: Checkboxes work natively
3. **Byte Counter**: CSS `counter` with `:valid/:invalid` states, or server-side on submit
4. **Tab Switching**: Separate URLs (`/init/participant`, `/init/corporate`)

Optional JS enhancements (graceful degradation):

1. **HTMX**: Partial page updates for add/remove parts
2. **Live Byte Counter**: JS updates character count display
3. **Copy to Clipboard**: JS for one-click XDR copy

### XDR Generation (Server-Side)

```go
func generateXDR(accountID string, original, current FormData) (string, error) {
    var ops []txnbuild.Operation

    // Compare each field
    if current.Name != original.Name {
        ops = append(ops, &txnbuild.ManageData{
            Name:  "Name",
            Value: []byte(current.Name),
        })
    }

    // Handle deletions (original had value, current is empty)
    if original.Website != "" && current.Website == "" {
        ops = append(ops, &txnbuild.ManageData{
            Name:  "Website",
            Value: nil, // nil = delete
        })
    }

    // Handle PartOf changes with index preservation
    // ... (complex logic for numbered keys)

    // Build transaction
    tx, err := txnbuild.NewTransaction(...)
    return tx.Base64()
}
```

### Result Page

```html
<h2>Transaction Generated</h2>

<h3>Operations:</h3>
<ul>
  <li>Set Name: "New Name"</li>
  <li>Delete Website</li>
  <li>Add PartOf015: G...</li>
</ul>

<h3>XDR:</h3>
<textarea readonly>AAAA...base64...</textarea>

<h3>Sign Transaction:</h3>
<a href="web+stellar:tx?xdr=...">Open in Stellar Wallet (SEP-0007)</a>
<a href="https://laboratory.stellar.org/#txsigner?xdr=...">Stellar Laboratory</a>
```

## Data Format Reference

### ManageData Entry Structure

- **Key**: UTF-8 string, max 64 bytes
- **Value**: Binary data, max 64 bytes (or null to delete)

### Common Patterns

| Pattern | Example | Description |
|---------|---------|-------------|
| Simple string | `Name` = "Alice" | Direct value |
| Account reference | `PartOf001` = "G..." | 56-char Stellar address |
| Boolean flag | `MTLA: PII Standard` = (any value) | Presence = true, null = false |
| Self-reference | `TagDeveloper` = (own account ID) | Tag with self as value |

### Byte Counting

UTF-8 encoding means:
- ASCII: 1 byte per character
- Cyrillic/Greek: 2 bytes per character
- Emoji: 4 bytes per character

Form should show byte count, not character count.
