#!/bin/bash

echo "stop throw 100 seconds"
sleep 100
docker stop load-balancer && docker-compose stop 
