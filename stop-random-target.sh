#!/bin/bash

sleep 100
containerName="target-web2"
docker stop $containerName
echo "Container $containerName was stoped"
sleep 100
docker start $containerName
echo "Container $containerName was running"
