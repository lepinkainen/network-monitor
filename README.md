# Network Connectivity Monitor

A focused ping-based network monitoring solution designed for long-term monitoring and ISP issue documentation. Features intelligent data management, pattern detection via heatmap visualization, and comprehensive reporting.

## Features

- **Continuous Monitoring**: Ping multiple targets with configurable intervals
- **Smart Data Retention**: 7 days raw data, 90 days aggregated
- **Pattern Heatmap**: 24-hour overlay showing issues at same times across days
- **Web Dashboard**: Real-time visualizations with D3.js
- **Static Report Generation**: PNG charts for ISP evidence
- **Automatic Maintenance**: Database optimization and data aggregation

## Quick Start

### Option 1: Using Docker (Recommended)

#### Prerequisites

- Docker and Docker Compose installed

#### 1. Clone and navigate to the project

```bash
cd /Users/shrike/projects/network-monitor
```

#### 2. Build and run with Docker Compose

```bash
# Build and start the container
docker-compose up --build

# Or run in detached mode
docker-compose up -d --build
```

#### 3. Access the Dashboard

Open your browser to: <http://localhost:8080>

#### 4. Stop the container

```bash
docker-compose down
```

#### Docker Configuration

- **Database**: Persisted in `./data/network_monitor.db` on host
- **Port**: 8080 (configurable via docker-compose.yml)
- **Health Check**: Automatic monitoring of web interface
- **Auto-restart**: Container restarts automatically unless stopped

#### Custom Configuration

Create a `config` directory and add configuration files, then mount them:

```yaml
volumes:
  - ./config:/app/config:ro
```

### Option 2: Native Installation

#### 1. Install Dependencies

```bash
cd /Users/shrike/projects/network-monitor
go mod download
```

#### 2. Build the Application

```bash
go build -o network-monitor *.go
```

#### 3. Find Your ISP Gateway (Optional but Recommended)

```bash
# On macOS/Linux:
traceroute 8.8.8.8 | head -2

# The first external hop is typically your ISP gateway
```

#### 4. Run the Monitor

```bash
# With default targets (Google DNS, Cloudflare, OpenDNS)
./network-monitor

# With custom targets including your ISP gateway
./network-monitor -targets "8.8.8.8,1.1.1.1,YOUR_ISP_GATEWAY_IP"

# With custom interval (default is 30s)
./network-monitor -interval 60s
```

#### 5. Access the Dashboard

Open your browser to: <http://localhost:8080>

## Command Line Options

- `-targets`: Comma-separated IPs to ping (default: "8.8.8.8,1.1.1.1,208.67.222.222")
- `-interval`: Time between pings (default: 30s)
- `-timeout`: Ping timeout (default: 5s)  
- `-db`: Database path (default: "network_monitor.db")
- `-port`: Web server port (default: 8080)

## Dashboard Features

### Real-time Monitoring

- **Uptime Cards**: Current connectivity status per target
- **Latency Chart**: Response times over selected period
- **Availability Timeline**: Visual up/down status bars

### Pattern Detection Heatmap

- **24-hour View**: See all issues overlaid on single day timeline
- **Color Coding**: Green = good, Red = high failure rate
- **Interactive**: Click any hour for day-by-day breakdown
- **Data Density**: Opacity shows how many days of data

### Outage Tracking

- Lists all connectivity failures (3+ consecutive failed pings)
- Shows duration and timing of outages
- Helps identify patterns

## Long-term Monitoring

The system is designed to run continuously:

- **Hour 1**: Basic connectivity data starts appearing
- **Day 1**: Initial patterns visible in heatmap
- **Week 1**: Clear time-of-day patterns emerge
- **Month 1**: Solid evidence of systematic issues

### Data Management

Automatic maintenance runs hourly:

- Aggregates hourly patterns for heatmap
- Archives old detailed data
- Keeps raw data for 7 days
- Keeps aggregated data for 90 days
- Monthly database vacuum

### Resource Usage

With 30-second intervals and 3 targets:

- **Daily**: ~8,640 pings
- **Monthly**: ~260,000 pings  
- **Storage**: ~10MB/month after aggregation
- **CPU**: <1% average
- **Memory**: 20-50MB

## Running as a Service

### macOS (launchd)

1. Create service file:

```bash
cat > ~/Library/LaunchAgents/com.network.monitor.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.network.monitor</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/shrike/projects/network-monitor/network-monitor</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/Users/shrike/projects/network-monitor</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/shrike/projects/network-monitor/monitor.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/shrike/projects/network-monitor/monitor.error.log</string>
</dict>
</plist>
EOF
```

2. Load and start:

```bash
launchctl load ~/Library/LaunchAgents/com.network.monitor.plist
```

3. Check status:

```bash
launchctl list | grep monitor
```

4. Stop/unload if needed:

```bash
launchctl unload ~/Library/LaunchAgents/com.network.monitor.plist
```

## Using the Heatmap for ISP Evidence

The heatmap is particularly powerful for ISP discussions:

1. **Pattern Recognition**: Issues at same time daily = capacity problem
2. **Peak Hour Correlation**: Problems during typical usage times
3. **Multi-target Proof**: All targets affected = ISP issue
4. **Visual Impact**: Screenshot after 1-2 weeks is compelling evidence

### Example Patterns

- **Evening slowdown**: Red zones 19:00-22:00 = streaming/gaming congestion
- **Workday mornings**: Issues at 09:00 = work-from-home surge
- **Weekend nights**: Failures Fri-Sat nights = entertainment peak

## Database Queries

### Check monitoring health

```bash
# Last ping time
sqlite3 network_monitor.db "SELECT datetime(max(timestamp), 'localtime') FROM ping_results"

# Database size
ls -lh network_monitor.db
```

### Find problem hours

```bash
sqlite3 network_monitor.db "
SELECT 
    printf('%02d:00', hour) as time,
    round(avg(failure_rate), 1) as avg_fail_rate
FROM hourly_patterns 
WHERE date > date('now', '-7 days')
GROUP BY hour 
ORDER BY avg_fail_rate DESC
LIMIT 10"
```

### Export data

```bash
sqlite3 network_monitor.db <<!
.headers on
.mode csv
.output connectivity_export.csv
SELECT * FROM ping_results WHERE timestamp > datetime('now', '-7 days');
.quit
!
```

## Troubleshooting

### No data appearing

- Check if process is running: `ps aux | grep network-monitor`
- Check logs: `tail -f monitor.log`
- Verify network connectivity: `ping 8.8.8.8`

### High CPU usage

- Increase interval: `-interval 60s`
- Reduce targets

### Database growing large

- Check size: `ls -lh network_monitor.db`
- Manual cleanup: `sqlite3 network_monitor.db "DELETE FROM ping_results WHERE timestamp < datetime('now', '-7 days'); VACUUM;"`

## ISP Communication Tips

1. **Collect 48-72 hours minimum** before contacting ISP
2. **Screenshot the heatmap** after 1-2 weeks
3. **Note your service plan** (speed tier, any SLA)
4. **Document correlations** (time of day, weather, etc.)
5. **Export summary statistics** for your records

## Support

For issues or questions about the code, please review the source files:

- `main.go`: Core monitoring logic
- `report.go`: Report generation
- `static/index.html`: Web dashboard

## License

This is a custom solution for network monitoring. Feel free to modify for your needs.
