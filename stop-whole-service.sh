#!/bin/bash

echo "stop throw 60 seconds"
sleep 300
docker stop load-balancer && docker-compose stop 
