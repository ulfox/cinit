# GO CInit

Container sys init


## Please read this first

```
    Do not run this on your personal comupter directly without Devmod, only within a container. 
    If you want to run it from your personal host, run it with -dev mode. 
    Without Devmod, on exit it will send SIGTERM and SIGKILL (on timeout) to all your processes. 
    Check Devmod run section for how to run with Devmod enabled
```

## Build

Build cinitd

```bash
    $> make cinitd

    #./bin/cinit-daemon -h
```

Build cinit cli

```bash
    $> make cli

    #./bin/cinit -h
```

## Devmod run (Not a PID 1 process)


Via Makefile

```bash
    $> make run
```

Manually

```bash
    $> go run cinitd/cinitd.go -dev
```

Expected output

```bash
    $> go run cinitd/cinitd.go -dev
    INFO[0000] Initiated                                     Component=Cinitd
    INFO[0000] Server initializing                           Component=SocketServer Stage=Init
```

## Run as PID 1

To run as PID 1, simply build cinitd and run it **without adding flag: -dev** 

## Using Cinit CLI

Cinit CLI uses http connection to interact with cinitd. In the future this will change to GRCP but for now it's http, sorry for that :(

### List Services

```bash
    $> ./bin/cinit -list
    No services
```

### Register a new service

Check `docs/examples/service.yaml`

```bash
    $> cat docs/examples/service.yaml
    name: command1
    command: sleep
    args: 10000
```

Register the above service

```bash
    $> ./bin/cinit -register -f docs/examples/service.yaml
    Service command1 has been registered
```
Expected cinitd events

```bash
INFO[0015] Registered new task            Component=ProcessPoolManager Part=TaskListener
DEBU[0015] Expanding ProcessPool          Component=ProcessPoolManager Part=Expanding
DEBU[0015] Received task. Pushing to next available process handler  Component=ProcessPoolManager Part=Binding
DEBU[0015] ProcessHandler 6... has been created  Component=ProcessPoolManager Part=ProcessQueue
INFO[0015] Task is being executed         Component=ProcessHandler Name=command1 PID=17584 PUID=6... Part=Running
```

### Get service status

First get a list of services

```bash
    $> ./bin/cinit -list
    {"Services":["command1"]}
```

Get service **command1** status

```bash
    $> ./bin/cinit -status -name command1
    {"action":"status","name":"command1","pid":"17584","status":"running","startTime":"2021-10-17T14:56:02.056110357Z"}
```

### Stop a service


```bash
    $> ./bin/cinit -stop -name command1
    {"action":"stop","name":"command1","status":"stopped","startTime":"2021-10-17T14:56:02.056110357Z","exitTime":"2021-10-17T15:00:14.732705337Z","exitStatus":"signal: killed"}
```

### Start a service


```bash
    $> ./bin/cinit -stop -name command1
    {"action":"start","name":"command1","pid":"17611","status":"running","startTime":"2021-10-17T15:00:52.833677948Z"}
```

### Delete a service

```bash
    $> ./bin/cinit -stop -name command1
    {"action":"delete","name":"command1","status":"deleted"}
```

### Exit cinitd

```bash
    $> kill <pid of cinitd>
    INFO[0009] Interrupted            Component=Cinitd
    INFO[0009] Sent 29 bytes to cinitd                      
    INFO[0009] Bye!            Component=SocketServer
    INFO[0009] Bye!            Component="HTTP Server"
    INFO[0009] Bye!            Component=ServiceOperator Part=ServiceListener
    INFO[0009] Bye!            Component=ServiceOperator
    INFO[0009] Bye!            Component=ProcessPoolManager Part=TaskListener
    INFO[0009] Bye!            Component=ProcessPoolManager Part=TaskOperator
    WARN[0009] ProcessQueue expansion is now forbidden       Component=ProcessPoolManager Part=ProcessQueue
    INFO[0009] Task has finished: signal: terminated         Component=ProcessHandler Name=command1 PUID=9... Part=Exit
    WARN[0009] ProcessQueue base has been shrinked to 0      Component=ProcessPoolManager Part=ProcessQueue
    DEBU[0009] ProcessHandler 9... is shutting down  Component=ProcessPoolManager Part=ProcessQueue
    INFO[0009] Bye!            Component=ProcessPoolManager Part=ZombieKiller
    INFO[0009] Bye!            Component=ProcessPoolManager
    INFO[0009] Bye!            Component=Cinitd
```

