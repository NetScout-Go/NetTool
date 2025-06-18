# Network Latency Heatmap Plugin

This plugin creates a visual heatmap of network latency to multiple targets over time, allowing you to analyze patterns and identify connectivity issues.

## Features

- **Multi-target Measurement**: Test latency to multiple hosts simultaneously
- **Customizable Sampling**: Adjust sample count and interval
- **Visual Heatmap**: Generate a color-coded heatmap of latency values
- **Statistical Analysis**: Calculate min, max, average, median latency and jitter
- **Packet Loss Tracking**: Track and visualize failed pings

## Use Cases

- **Network Quality Assessment**: Evaluate the quality and stability of your network connection
- **ISP Performance Monitoring**: Compare latency patterns across different destinations
- **Server Selection**: Identify the lowest-latency servers for your location
- **Troubleshooting**: Detect intermittent network issues or patterns in latency spikes

## Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| Target Hosts | Comma-separated list of hosts or IPs to ping | 8.8.8.8,1.1.1.1,google.com |
| Measurement Interval | Time between measurements in seconds | 1 |
| Number of Samples | Total number of samples to collect for each target | 30 |
| Ping Timeout | Maximum time to wait for each ping response | 2 |
| Packet Size | Size of ping packets in bytes | 56 |
| Show Interactive Graph | Generate an interactive graph visualization | true |

## Interpreting Results

The heatmap uses a color gradient where:
- **Green**: Low latency (good)
- **Yellow**: Medium latency
- **Red**: High latency (potential issues)
- **Gray/Black**: Failed pings (packet loss)

Look for patterns such as:
- Consistent high latency to specific targets
- Time-based patterns (e.g., congestion during peak hours)
- Correlation between targets (network-wide issues vs. target-specific)

## Example Usage

To monitor latency to Google DNS, Cloudflare DNS, and your ISP's gateway:

```
Targets: 8.8.8.8,1.1.1.1,192.168.1.1
Samples: 60
Interval: 1
```

This will produce a minute-long sample of latency to these three targets.

## Notes

- Requires ping privileges (may need sudo/root depending on system configuration)
- For accurate results, ensure targets are reachable from your network
- Higher sample counts and longer intervals provide more comprehensive data but take longer to complete
