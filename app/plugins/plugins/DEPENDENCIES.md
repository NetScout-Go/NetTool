# Setup Dependencies

To ensure that the Iperf3 and Traffic Control (tc) plugins work correctly, you need to install the required dependencies on your Raspberry Pi:

## Installing Iperf3

Iperf3 is required for the Iperf3 Throughput Test plugin:

```bash
sudo apt-get update -y
sudo apt-get install iperf3 -y
```

You can verify the installation by running:

```bash
iperf3 --version
```

## Installing Traffic Control (tc)

The tc command is part of the iproute2 package, which is required for the Traffic Control (QoS) plugin:

```bash
sudo apt-get update
sudo apt-get install iproute2
```

You can verify the installation by running:

```bash
tc -V
```

Note that using the Traffic Control (QoS) plugin requires root privileges. Make sure to run NetScout-Pi with sudo if you plan to use this plugin.
