#!/bin/bash

# Clear existing DOCKER-USER rules to prevent duplicates on re-execution
iptables -F DOCKER-USER

# Ports to restrict
PORTS=(5000 5432 5672 6333 6334 6379 8080 8123 8888 9000 9001 9009 15672)

for PORT in "${PORTS[@]}"; do
    # Allow traffic from localhost
    iptables -A DOCKER-USER -p tcp -s 127.0.0.1 --dport "$PORT" -j ACCEPT
    # Drop traffic from any other source
    iptables -A DOCKER-USER -p tcp --dport "$PORT" -j DROP
done

# Optional: Save iptables rules if you are also using netfilter-persistent
# sudo netfilter-persistent save
