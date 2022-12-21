# balancer

### Usage

**build and run containers with balancer and target servers:** ```make build ```

**stop balanser and target servers:** ```make stop```

**To show the log from balancer** ```docker exec -it load-balancer cat logs/balancer_log.log```

**send request:** ```curl hthp://localhost:3030```
