# ErrantDNS

This project intends to result in a fully functional Authoritative DNS server that has the following features:

1. Standard Auth DNS functions
1. PostgreSQL backed records for persistence and loading
1. Local in-memory, configurable cache for query HIT/MISS handling
1. In some cases, *nested wildcard support*;  this means DNS records such as `sub.*.mydomain.com`, where appropriate

## Nested Wildcard Support

