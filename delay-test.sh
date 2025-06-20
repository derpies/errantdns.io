#!/bin/bash


# Test with 35-second intervals to cross time boundaries
dig @127.0.0.1 -p 10001 +short round-robin.internal A
sleep 35
dig @127.0.0.1 -p 10001 +short round-robin.internal A
sleep 35
dig @127.0.0.1 -p 10001 +short round-robin.internal A
