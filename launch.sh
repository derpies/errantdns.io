#!/bin/bash


DNS_PORT=10001 \
DB_HOST=localhost \
DB_USER=dnsuser \
DB_PASSWORD=dnspass \
CACHE_ENABLED=true \
CACHE_MAX_ENTRIES=50000 \
go run cmd/dns-server/main.go

