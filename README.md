# balancer

### Usage

**build and run containers with balancer and target servers:** ```make build ```

*at 100 seconds, target-web2 is shutdown, at 200 second runing again. The logs show 
the load distribution between other servers*

**To show the logs** ```cat logs/balancer.log | less```

