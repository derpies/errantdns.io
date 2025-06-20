#!/bin/bash


# Test A records
dig @127.0.0.1 -p 10001 test.internal A
dig @127.0.0.1 -p 10001 www.test.internal A
dig @127.0.0.1 -p 10001 mail.test.internal A

# Test AAAA records (IPv6)
dig @127.0.0.1 -p 10001 test.internal AAAA
dig @127.0.0.1 -p 10001 www.test.internal AAAA

# Test CNAME records
dig @127.0.0.1 -p 10001 ftp.test.internal CNAME
dig @127.0.0.1 -p 10001 blog.test.internal CNAME

# Test MX records (should show priority)
dig @127.0.0.1 -p 10001 test.internal MX

# Test TXT records
dig @127.0.0.1 -p 10001 test.internal TXT
dig @127.0.0.1 -p 10001 _dmarc.test.internal TXT

# Test NS records
dig @127.0.0.1 -p 10001 test.internal NS
