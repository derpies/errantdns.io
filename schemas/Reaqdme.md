# SQL Schemas

## Domain Naming Convention for Test Domains

### Primary Test Domain

- **`test.internal`** - Main test domain for general DNS functionality
- **`.internal` TLD** - Reserved for internal use, never publicly routable
- **Descriptive subdomains** - Clear naming that indicates the test purpose

### Specialized Test Domains

- **`errant-dns-test.internal`** - Project-specific testing
- **`dns-cache-test.internal`** - Cache behavior testing
- **`short-ttl.internal`** - TTL expiration testing
- **`long-ttl.internal`** - Long-lived cache testing
- **`priority-test.internal`** - Priority/preference testing
- **`wildcard-parent.internal`** - Future wildcard pattern testing

## IP Address Organization

### 10.0.0.x - Core Infrastructure

Primary `test.internal` domain services:

- **10.0.0.10** - Main domain (`test.internal`, `www.test.internal`)
- **10.0.0.20** - Mail services (`mail.test.internal`)
- **10.0.0.30** - API services (`api.test.internal`)

### 10.0.1.x - TTL Testing

Domains with varying TTL values for cache testing:

- **10.0.1.10** - Standard TTL test domain
- **10.0.1.20** - Short TTL (30 seconds)
- **10.0.1.30** - Long TTL (3600 seconds)

### 10.0.2.x - Cache and Priority Testing

Specialized behavior testing:

- **10.0.2.10** - Cache behavior testing
- **10.0.2.20** - Priority test (high priority record)
- **10.0.2.21** - Priority test (low priority record)

### 10.0.3.x - Wildcard Testing

Reserved for future wildcard pattern implementation:

- **10.0.3.10** - Parent domain for wildcard tests
- **10.0.3.11** - First subdomain variant
- **10.0.3.12** - Second subdomain variant

### IPv6 Addressing

- **fd00::1** - Main IPv6 address for core domains
- **fd00::x** - Future expansion using unique local addresses (RFC 4193)

## Expansion Guidelines

### Adding New Test Categories

1. **Choose next available 10.0.x.0 subnet**
2. **Use .10, .20, .30 pattern** for primary, secondary, tertiary services
3. **Document the subnet purpose** in this strategy guide
4. **Use descriptive domain names** that clearly indicate test purpose

### IP Assignment Rules

- **10.0.0.x** - Reserved for core infrastructure
- **10.0.1.x** - TTL and timing-related tests
- **10.0.2.x** - Behavior and logic tests
- **10.0.3.x** - Wildcard and pattern tests
- **10.0.4.x** - Available for DNS protocol tests
- **10.0.5.x** - Available for performance tests
- **10.0.6.x** - Available for security tests

### Domain Naming Rules

- **Use `.internal` TLD** for all test domains
- **Include test purpose** in subdomain name when possible
- **Avoid real company/brand names**  
- **Use hyphens for readability** (`dns-cache-test` vs `dnscachetest`)

## Benefits

### Development Benefits

- **No DNS conflicts** with real domains
- **Clear test intent** - obvious these are test records
- **Organized troubleshooting** - IP address indicates test type
- **Systematic expansion** - clear pattern for adding new tests

### Debugging Benefits

- **Network trace analysis** - immediately identify test type by IP
- **Log file clarity** - domain names indicate test purpose
- **Error isolation** - easy to separate test categories
- **Performance analysis** - can group metrics by IP subnet

### Testing Benefits

- **Isolated test environments** - each subnet serves different test purposes
- **Reproducible results** - consistent IP assignments
- **Scalable test design** - room for expansion without conflicts
- **Clear test documentation** - IP/domain mapping is self-documenting
