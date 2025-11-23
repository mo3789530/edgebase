#!/bin/bash

# Simple mock server using netcat
echo "Starting mock sync service on port 8080..."

while true; do
  echo -e "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 44\r\n\r\n{\"success\":true,\"inserted\":10,\"failed\":0}" | nc -l -p 8080 -q 1
done
