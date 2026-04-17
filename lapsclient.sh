#!/bin/sh

if [ "$FQDN" = "" ]; then
  FQDN=$(hostname -f)
fi

ipv6_addr=$(ip -6 addr show scope global |grep inet6 |grep -v temporary |awk '{print $2}' |head -n 1 |cut -f 1 -d '/')

ipv4_addr=$(ip -4 addr show scope global |grep inet |awk '{print $2}' |head -n 1 |cut -f 1 -d '/')

timestamp=$(date +%s)

signature=$(printf "%s|%s|%s|%s|%s" "$FQDN" "$ipv6_addr" "$ipv4_addr" "$timestamp" "abcd1234" | sha256sum | awk '{print $1}')

echo $FQDN
echo $ipv6_addr
echo $ipv4_addr
echo $timestamp

echo $signature

request_body=$(jq -n --arg fqdn "$FQDN" \
                    --arg ipv6 "$ipv6_addr" \
                    --arg ipv4 "$ipv4_addr" \
                    --argjson timestamp "$timestamp" \
                    --arg signature "$signature" \
                    '{
                      hostname: $fqdn,
                      ipv6_addr: $ipv6,
                      ipv4_addr: $ipv4,
                      timestamp: $timestamp,
                      signature: $signature
                    }')

echo $request_body

curl -X POST -H "Content-Type: application/json" -d "$request_body" http://localhost:8080/update_dns
