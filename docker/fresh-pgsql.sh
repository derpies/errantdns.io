#!/bin/bash


docker run -d \
  --name errantdns-postgres \
  -e POSTGRES_DB=dnsdb \
  -e POSTGRES_USER=dnsuser \
  -e POSTGRES_PASSWORD=dnspass \
  -p 5432:5432 \
  -v errantdns_data:/var/lib/postgresql/data \
  postgres:15
