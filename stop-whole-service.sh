#!/bin/bash

echo "stop throw 300 seconds"
sleep 300
docker stop load-balancer && docker-compose stop 
