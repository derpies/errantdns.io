# ErrantDNS Implementation Guide - Handling Edge Cases and Operational Concerns

## Overview

This document outlines how to handle edge cases, validation requirements, and operational concerns for the ErrantDNS positional wildcard system. It complements the core framework specification with practical implementation guidance.

## Domain Processing Model

### ETLD Exclusion
- **ETLD Detection**: Use Mozilla Public Suffix List to identify ETLD boundaries
- **ETLD Storage**: Store separately, never included in wildcard processing
- **Bitmask Scope**: Only applies to labels between subdomain and ETLD boundary

### Example Domain Processing
```
Query: api.service.prod.example.com
ETLD: example.com (excluded from wildcard processing)
Processable Labels: [api, service, prod]
Bitmask Positions: 0=api, 1=service, 2=prod
```

## Edge Case Handling

### 1. Identical Bitmask Patterns

**Scenario**: Two patterns with identical wildcard structure but different exact match values
```
Pattern A: api.service.prod.example.com → labels=[api,service,prod] → bitmask 000
Pattern B: api.backend.prod.example.com → labels=[api,backend,prod] → bitmask 000
Query: api.service.prod.example.com
```

**Resolution Strategy**:
- Apply Tier 1 precedence (exact match check) first
- Only Pattern A matches the query; Pattern B is eliminated during matching phase
- No ambiguity occurs because patterns with identical bitmasks can only conflict if they have identical exact match values (configuration duplicate)

**Implementation**: Standard matching algorithm handles this naturally - no special case logic needed.

### 2. Complex Pattern Overlap Scenarios

**Scenario**: Multiple patterns match same query with complex precedence interactions
```
Pattern A: api.*.prod.example.com    → labels=[api,*,prod] → bitmask 010 → 2 exact matches [0,2]
Pattern B: api.service.*.example.com → labels=[api,service,*] → bitmask 001 → 2 exact matches [0,1]
Query: api.service.prod.example.com → processable=[api,service,prod]
```

**Resolution Strategy**:
1. **Tier 2**: Both have 2 exact matches - tie
2. **Tier 3**: Apply left-to-right positional precedence
   - Both exact at position 0 - continue
   - Pattern B exact at position 1, Pattern A wildcard at position 1
   - **Winner**: Pattern B (earlier exact match at position 1)

**Implementation**: 
- Implement position-by-position comparison from left to right
- First position where patterns differ determines winner
- Stop comparison at first differentiating position

### 3. ETLD Boundary Handling

**Scenario**: Same logical pattern across different ETLD structures
```
Domain A: api.service.example.co.uk → ETLD=example.co.uk → processable=[api,service]
Domain B: api.service.example.com   → ETLD=example.com → processable=[api,service]
```

**Resolution Strategy**:
- Use Mozilla Public Suffix List for authoritative ETLD detection
- Normalize all patterns by excluding ETLD before bitmask processing
- Store ETLD separately for validation but not wildcard matching

**Implementation**:
```
Pre-processing Steps:
1. Extract ETLD using Public Suffix List
2. Split remaining labels for bitmask processing
3. Store: etld="example.co.uk", labels=["api","service"], bitmask=B'01'
```

### 4. Pattern Length Validation

**Scenario**: Query with different processable label count than stored patterns
```
Stored Pattern: api.*.prod.example.com → processable labels=3
Query: api.prod.example.com → processable labels=2
```

**Resolution Strategy**:
- Require exact processable label count matching for pattern consideration
- Pattern eliminated during initial filtering, not matching phase
- This is expected behavior - patterns are structure-specific

**Implementation**:
- Add processable_label_count field to pattern storage
- Filter candidate patterns by processable_label_count before bitmask matching
- Index on (etld, processable_label_count) for query performance

### 5. Malformed Query Handling

**Scenario**: DNS queries with invalid label structures
```
Query: api..prod.example.com    (empty label - double dot)
Query: .api.prod.example.com    (leading dot)
Query: api.prod.example.com.    (trailing dot)
```

**Resolution Strategy**:
- Apply standard DNS validation before pattern matching
- Reject malformed queries at protocol level
- Trailing dots are valid DNS (absolute vs relative) - handle during parsing

**Implementation**:
- Use miekg/dns library validation
- Normalize trailing dots during query parsing
- Return DNS error codes for malformed queries

## Conflict Detection and Validation

### Configuration Time Validation

**Duplicate Pattern Detection**:
```sql
-- Detect exact duplicates
SELECT etld, labels, wildcard_mask, COUNT(*) 
FROM dns_records 
GROUP BY etld, labels, wildcard_mask 
HAVING COUNT(*) > 1;
```

**Overlapping Pattern Analysis**:
```sql
-- Find patterns with identical structure but different exact values
SELECT a.pattern_id, b.pattern_id, a.wildcard_mask
FROM dns_records a, dns_records b 
WHERE a.etld = b.etld 
  AND a.wildcard_mask = b.wildcard_mask 
  AND a.labels != b.labels
  AND a.pattern_id != b.pattern_id;
```

### Runtime Conflict Warning System

**Multi-Match Detection**:
- Log when multiple patterns match same query
- Include precedence reasoning in logs
- Provide operator dashboard for pattern conflicts

**Conflict Resolution Logging**:
```
WARN: Multiple patterns matched query api.service.prod.example.com
  Pattern A (ID: 123): api.*.prod.example.com → precedence score: 2.2
  Pattern B (ID: 456): api.service.*.example.com → precedence score: 2.1
  Resolution: Pattern B selected (earlier exact match at position 1)
```

## Operational Tooling Requirements

### Pattern Testing Interface

**Test Query Feature**:
- Allow operators to input test queries
- Show all matching patterns with precedence scores
- Explain precedence decision reasoning
- Highlight potential conflicts

**Pattern Validation**:
- Validate pattern syntax during creation
- Check for potential conflicts with existing patterns
- Suggest pattern optimizations

### Monitoring and Alerting

**Key Metrics**:
- Pattern match distribution (how often each pattern is used)
- Multi-match frequency (potential configuration issues)
- Query resolution latency by pattern complexity
- Cache hit rates for pattern matching

**Alert Conditions**:
- High frequency of multi-match scenarios
- Patterns that never match queries (unused patterns)
- Performance degradation in pattern matching

### Documentation and Training

**Operator Training Topics**:
1. Understanding left-to-right precedence logic
2. Pattern design best practices
3. Conflict resolution examples
4. Performance implications of pattern complexity

**Configuration Guidelines**:
- Prefer exact matches over wildcards when possible
- Design patterns with clear precedence intent
- Test patterns against expected query patterns
- Document business logic behind pattern choices

## Implementation Checklist

### Core Algorithm
- [ ] Implement three-tier precedence system
- [ ] Add left-to-right positional comparison
- [ ] Include processable label count filtering
- [ ] Precompute exact match counts for performance
- [ ] ETLD extraction and exclusion from wildcard processing

### Validation System
- [ ] ETLD extraction using Public Suffix List
- [ ] Pattern duplicate detection
- [ ] Malformed query rejection
- [ ] Configuration-time conflict analysis

### Operational Tools
- [ ] Pattern testing interface
- [ ] Conflict detection dashboard
- [ ] Performance monitoring
- [ ] Multi-match logging system

### Database Schema
- [ ] Add processable_label_count field for filtering
- [ ] Add exact_match_count for precedence
- [ ] Index on (etld, processable_label_count, exact_match_count)
- [ ] Constraint to prevent true duplicates

## Performance Considerations

### Query Path Optimization
1. **Filter by ETLD** (scope to relevant domain space)
2. **Filter by processable label count** (eliminate impossible matches)
3. **Apply bitmask matching** (check pattern compatibility)
4. **Rank by precedence** (select winner from candidates)

### Caching Strategy
- Cache compiled patterns by (etld, processable_label_count)
- Cache precedence rankings for frequent query patterns
- Use Redis bitmaps for wildcard mask operations
- TTL alignment with DNS record TTLs

### Database Performance
- Partition tables by ETLD for large datasets
- Use partial indexes for wildcard vs exact patterns
- Consider materialized views for complex precedence queries
- Monitor query performance with pattern complexity growth

## ETLD Processing Examples

### Single-Level ETLD
```
Query: api.service.prod.example.com
ETLD: example.com
Processable: [api, service, prod]
Pattern: *.service.prod.example.com → bitmask 100 → 2 exact matches
```

### Multi-Level ETLD
```
Query: api.service.prod.example.co.uk
ETLD: example.co.uk
Processable: [api, service, prod]
Pattern: *.service.prod.example.co.uk → bitmask 100 → 2 exact matches
```

### Corporate/Internal ETLD
```
Query: api.service.prod.corp.amazonaws.com
ETLD: corp.amazonaws.com (if in Public Suffix List)
Processable: [api, service, prod]
Pattern: *.service.prod.corp.amazonaws.com → bitmask 100 → 2 exact matches
```