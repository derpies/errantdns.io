# ErrantDNS Positional Wildcard Framework

## Overview

ErrantDNS implements a positional wildcard system using bitmasks to efficiently handle nested wildcard patterns in DNS resolution. This framework addresses client-side demands for flexible DNS pattern matching within controlled domain structures.

## Core Concepts

### Problem Definition

- **Not a pattern matching problem**: This is a domain list matching problem
- **Client-controlled scope**: Patterns are defined for specific domain structures that clients control
- **Positional wildcards**: Each label position can be either exact match or wildcard

### Domain Decomposition
DNS queries are decomposed left-to-right into positional arrays:

```

Query: api.service.prod.example.com
Decomposed: [api, service, prod, example, com]
Positions:   0    1       2     3       4
```

### Bitmask Representation

Each stored pattern uses a bitmask where:
- `0` = exact match required at this position
- `1` = wildcard (any value accepted) at this position

**Examples:**

```
Pattern: api.service.prod.example.com → [api, service, prod, example, com] → bitmask 00000
Pattern: *.service.prod.example.com  → [*, service, prod, example, com]   → bitmask 10000
Pattern: api.*.*.example.com         → [api, *, *, example, com]          → bitmask 01100
Pattern: *.*.prod.example.com        → [*, *, prod, example, com]         → bitmask 11000
```

## Matching Logic

### Template Matching Process

1. Decompose incoming DNS query into positional array
2. For each stored pattern, compare array positions:
   - If pattern bitmask has `0` at position: query label must exactly match stored label
   - If pattern bitmask has `1` at position: any query label value is accepted
3. Pattern matches if all position comparisons succeed

### Example Matching

```
Query: api.service.prod.example.com → [api, service, prod, example, com]

Pattern: *.service.prod.example.com → bitmask 10000
- Position 0: 1 (wildcard) → "api" accepted ✓
- Position 1: 0 (exact) → "service" == "service" ✓  
- Position 2: 0 (exact) → "prod" == "prod" ✓
- Position 3: 0 (exact) → "example" == "example" ✓
- Position 4: 0 (exact) → "com" == "com" ✓
Result: MATCH
```

## Precedence Framework

When multiple patterns match the same query, precedence is determined through a three-tier hierarchy:

### Tier 1: Exact Match Priority

Patterns with all exact matches (bitmask all `0`s) always win over patterns with any wildcards.

### Tier 2: Total Exact Match Count

Among wildcard patterns, those with more total exact matches win over those with fewer exact matches.

**Calculation**: Count `0` bits in bitmask (number of exact match positions)

### Tier 3: Left-to-Right Positional Precedence

When patterns have equal exact match counts, earlier exact matches (leftmost positions) indicate higher specificity.

**Rationale**: Human domain reading is left-to-right, with leftmost labels being most specific identifiers.

### Precedence Examples

**Example 1: Different Exact Match Counts**

```
Query: api.service.prod.example.com

Pattern A: *.service.prod.example.com → bitmask 10000 → 4 exact matches
Pattern B: api.*.*.example.com       → bitmask 01100 → 3 exact matches

Winner: Pattern A (more exact matches)
```

**Example 2: Equal Exact Match Counts, Different Positions**

```
Query: api.service.prod.example.com

Pattern X: api.*.prod.example.com    → bitmask 01010 → 3 exact matches (positions 0,2,3,4)
Pattern Y: *.service.*.example.com   → bitmask 10100 → 3 exact matches (positions 1,3,4)

Winner: Pattern X (exact match at position 0 is more specific than position 1)
```

## Technical Implementation

### ETLD Handling

- Use Mozilla Public Suffix List for authoritative ETLD detection
- Exclude ETLD components from wildcard processing
- ETLD always treated as exact match requirement

### Storage Schema

```sql
-- Conceptual schema elements
etld: "example.com"
labels: ["api", "service", "prod"] 
wildcard_mask: B'010'  -- PostgreSQL bitstring
exact_match_count: 2   -- Precomputed for performance
```

### Performance Characteristics

- **Matching**: Simple bitwise operations and string comparisons
- **Precedence**: Integer comparison of precomputed exact match counts
- **Storage**: Compact bitmask representation
- **Caching**: Redis-friendly bitstring operations

## Scope and Limitations

### What This Solves

- Client-defined positional wildcard patterns
- Efficient matching against controlled domain structures
- Clear precedence for overlapping patterns
- High-performance DNS resolution

### What This Doesn't Solve

- Universal pattern matching across arbitrary domain structures
- Variable-depth wildcard patterns
- Traditional DNS wildcard semantics (`*.example.com` matching any subdomain depth)

### Design Philosophy

This system prioritizes:
1. **Operational clarity** over universal flexibility
2. **Performance** over pattern complexity
3. **Client control** over automatic pattern inference
4. **Deterministic behavior** over heuristic matching

## Edge Cases and Tiebreaking

### Identical Patterns

Patterns with identical bitmasks and labels represent configuration duplicates and should be flagged during validation.

### Complex Precedence Ties

In rare cases where patterns have identical precedence scores, use deterministic tiebreakers:

- Pattern creation timestamp
- Lexicographic pattern ordering
- Explicit priority field

### Validation Requirements

- Ensure patterns are valid for intended domain structures
- Detect conflicting patterns during configuration
- Provide tooling for pattern testing and validation
