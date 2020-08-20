# Experimental Opera

This version of Opera is a testing/benchmarking platform, it is not safe to use.

## Building the source

Building `miniopera` requires both a Go (version 1.13 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
go build -o ./build/miniopera ./cmd/miniopera
```
The build output is ```build/miniopera``` executable.

Do not clone the project into $GOPATH, due to the Go Modules. Instead, use any other location.

## Running `miniopera`

Going through all the possible command line flags is out of scope here,
but we've enumerated a few common parameter combos to get you up to speed quickly
on how you can run your own `miniopera` instance.

### Configuration

As an alternative to passing the numerous flags to the `miniopera` binary, you can also pass a
configuration file via:

```shell
$ miniopera --config /path/to/your_config.toml
```

To get an idea how the file should look like you can use the `dumpconfig` subcommand to
export your existing configuration:

```shell
$ miniopera --your-favourite-flags dumpconfig
```

## Dev

### Testing

Use the Go tool to run tests:
```shell
go test ./...
```

If everything goes well, it should output something along these lines:
```
?   	github.com/Fantom-foundation/miniopera/api	[no test files]
?   	github.com/Fantom-foundation/miniopera/miniopera	[no test files]
?   	github.com/Fantom-foundation/miniopera/miniopera/genesis	[no test files]
?   	github.com/Fantom-foundation/miniopera/cmd/miniopera	[no test files]
?   	github.com/Fantom-foundation/miniopera/cmd/miniopera/metrics	[no test files]
?   	github.com/Fantom-foundation/miniopera/debug	[no test files]
?   	github.com/Fantom-foundation/miniopera/eventcheck	[no test files]
?   	github.com/Fantom-foundation/miniopera/eventcheck/basiccheck	[no test files]
?   	github.com/Fantom-foundation/miniopera/eventcheck/heavycheck	[no test files]
?   	github.com/Fantom-foundation/miniopera/eventcheck/parentscheck	[no test files]
?   	github.com/Fantom-foundation/miniopera/gossip	[no test files]
?   	github.com/Fantom-foundation/miniopera/gossip/emitter	[no test files]
ok  	github.com/Fantom-foundation/miniopera/integration	10.078s
ok  	github.com/Fantom-foundation/miniopera/inter	(cached)
?   	github.com/Fantom-foundation/miniopera/logger	[no test files]
?   	github.com/Fantom-foundation/miniopera/metrics/prometheus	[no test files]
ok  	github.com/Fantom-foundation/miniopera/utils	(cached)
?   	github.com/Fantom-foundation/miniopera/utils/errlock	[no test files]
ok  	github.com/Fantom-foundation/miniopera/utils/fast	(cached)
ok  	github.com/Fantom-foundation/miniopera/utils/migration	(cached)
?   	github.com/Fantom-foundation/miniopera/utils/throughput	[no test files]
?   	github.com/Fantom-foundation/miniopera/version	[no test files]
```

### Operating a private network (fakenet)

Fakenet is a private network optimized for your private testing.
It'll generate a genesis containing N validators with equal stakes.
To launch a validator in this network, all you need to do is specify a validator ID you're willing to launch.

Pay attention that validator's private keys are deterministically generated in this network, so you must use it only for private testing.

Maintaining your own private network is more involved as a lot of configurations taken for
granted in the official networks need to be manually set up.

To run the fakenet with just one validator (which will work practically as a PoA blockchain), use:
```shell
$ miniopera --fakenet 1/1
```

To run the fakenet with 5 validators, run the command for each validator:
```shell
$ miniopera --fakenet 1/5 # first node, use 2/5 for second node
```

If you have to launch a non-validator node in fakenet, use 0 as ID:
```shell
$ miniopera --fakenet 0/5
```

After that, you have to connect your nodes. Either connect them statically or specify a bootnode:
```shell
$ miniopera --fakenet 1/5 --bootnodes "enode://verylonghex@1.2.3.4:5050"
```

### Testing event payload
```shell
$ miniopera --bps=600000 --txpayload=100000
```
This combination of flags means "emit no more than 600000 bytes per second, place 100000 bytes of payload into each event"

### Running the demo

For the testing purposes, the full demo may be launched using:
```shell
cd demo/
./start.sh # start the fakenet instances
./stop.sh # stop the demo
```
