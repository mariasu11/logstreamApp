# LogStream

A high-performance log aggregation and analysis tool built in Go that demonstrates concurrent processing, plugin architecture, and robust API design.

[![CI](https://github.com/yourusername/logstream/actions/workflows/ci.yml/badge.svg)](https://github.com/yourusername/logstream/actions/workflows/ci.yml)

## Features

- **Distributed Log Collection**: Collect logs from multiple sources including files and HTTP endpoints
- **Concurrent Processing**: Process logs in parallel using worker pools and goroutines
- **Customizable Log Processing**: Filter and transform logs using customizable rules
- **Plugin System**: Extend functionality with custom log processors
- **Robust Storage**: Store processed logs for analysis in memory or disk
- **Query Capabilities**: Powerful API for searching and analyzing logs
- **CLI Interface**: Comprehensive command-line interface
- **Performance Optimized**: Designed for high throughput
- **Prometheus Integration**: Built-in metrics for monitoring

## Getting Started

### Installation

```bash
# Install using Go
go install github.com/yourusername/logstream/cmd/logstream

# Or download binary from releases
curl -LO https://github.com/yourusername/logstream/releases/latest/download/logstream-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x logstream-*
mv logstream-* /usr/local/bin/logstream
```

## Usage

### CLI Commands

LogStream provides a comprehensive CLI interface with the following commands:

```
# Start the API server
logstream serve --host=0.0.0.0 --port=8000

# Start collecting logs
logstream collect --sources=file:///var/log/syslog,https://api.example.com/logs

# Query collected logs
logstream query "level:error" --limit=100 --from="2025-05-01T00:00:00Z"
```

### Configuration

LogStream can be configured using a YAML configuration file. See [config.yaml.example](config.yaml.example) for a complete example.

```bash
# Use a specific config file
logstream serve --config=/path/to/config.yaml
```

## API Documentation

LogStream provides a comprehensive REST API for log ingestion, querying, and analysis.

### Endpoints

#### Authentication

All API requests are unauthenticated by default. For production deployments, it's recommended to secure the API using an API gateway or a reverse proxy that handles authentication.

#### Log Ingestion

##### Store a single log entry

```
POST /api/v1/logs
Content-Type: application/json
```

Request Body:
```json
{
  "timestamp": "2025-05-13T00:01:00Z",
  "source": "system",
  "level": "info",
  "message": "System startup complete",
  "fields": {
    "host": "server1.example.com",
    "datacenter": "us-west-1"
  }
}
```

Response:
```json
{
  "status": "ok"
}
```

##### Store multiple log entries

```
POST /api/v1/logs/batch
Content-Type: application/json
```

Request Body:
```json
[
  {
    "timestamp": "2025-05-13T00:01:15Z",
    "source": "auth",
    "level": "warn",
    "message": "Failed login attempt",
    "fields": {
      "ip": "192.168.1.100",
      "username": "admin"
    }
  },
  {
    "timestamp": "2025-05-13T00:01:30Z",
    "source": "database",
    "level": "error",
    "message": "Connection timeout",
    "fields": {
      "db_host": "db-primary-01"
    }
  }
]
```

Response:
```json
{
  "status": "ok",
  "count": "2"
}
```

#### Log Querying

##### Get log entries

```
GET /api/v1/logs?source=auth&level=warn&limit=10
```

Response:
```json
[
  {
    "timestamp": "2025-05-13T00:01:15Z",
    "source": "auth",
    "level": "warn",
    "message": "Failed login attempt",
    "fields": {
      "ip": "192.168.1.100",
      "username": "admin"
    }
  }
]
```

##### Execute a custom query

```
POST /api/v1/query
Content-Type: application/json
```

Request Body:
```json
{
  "filter": "source:auth",
  "limit": 10,
  "from": "2025-05-13T00:00:00Z",
  "to": "2025-05-13T01:00:00Z",
  "levels": ["warn", "error"],
  "sort_by": "timestamp",
  "sort_order": "desc"
}
```

Response:
```json
[
  {
    "timestamp": "2025-05-13T00:01:15Z",
    "source": "auth",
    "level": "warn",
    "message": "Failed login attempt",
    "fields": {
      "ip": "192.168.1.100",
      "username": "admin"
    }
  }
]
```

#### Log Analysis

##### Perform log analysis

```
POST /api/v1/query/analyze
Content-Type: application/json
```

Request Body:
```json
{
  "analysis": {
    "type": "frequency",
    "time_range": {
      "from": "2025-05-13T00:00:00Z",
      "to": "2025-05-13T01:00:00Z"
    },
    "group_by": "source"
  }
}
```

Response:
```json
{
  "type": "frequency",
  "time_range": {
    "from": "2025-05-13T00:00:00Z",
    "to": "2025-05-13T01:00:00Z"
  },
  "count": 4,
  "frequency": {
    "auth": 1,
    "cache": 1,
    "database": 1,
    "user": 1
  }
}
```

#### Metadata

##### Get log sources

```
GET /api/v1/logs/sources
```

Response:
```json
[
  "auth",
  "cache",
  "database",
  "system",
  "user"
]
```

##### Get storage statistics

```
GET /api/v1/logs/stats
```

Response:
```json
{
  "TotalEntries": 5,
  "OldestEntry": "2025-05-13T00:01:00Z",
  "NewestEntry": "2025-05-13T00:02:00Z",
  "EntriesBySource": {
    "auth": 1,
    "cache": 1,
    "database": 1,
    "system": 1,
    "user": 1
  },
  "EntriesByLevel": {
    "debug": 1,
    "error": 1,
    "info": 2,
    "warn": 1
  },
  "StorageSize": 2500,
  "CompressionRatio": 1
}
```

##### Check API health

```
GET /api/v1/health
```

Response:
```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

#### Metrics

```
GET /metrics
```

Response:
```
# HELP logstream_logs_received_total Total number of log entries received
# TYPE logstream_logs_received_total counter
logstream_logs_received_total{source="auth"} 1
logstream_logs_received_total{source="cache"} 1
logstream_logs_received_total{source="database"} 1
logstream_logs_received_total{source="system"} 1
logstream_logs_received_total{source="user"} 1

# HELP logstream_logs_processed_total Total number of log entries processed
# TYPE logstream_logs_processed_total counter
logstream_logs_processed_total 5

# HELP logstream_query_duration_seconds Time spent executing queries
# TYPE logstream_query_duration_seconds histogram
logstream_query_duration_seconds_bucket{le="0.005"} 3
logstream_query_duration_seconds_bucket{le="0.01"} 5
logstream_query_duration_seconds_bucket{le="0.025"} 7
logstream_query_duration_seconds_bucket{le="0.05"} 9
logstream_query_duration_seconds_bucket{le="0.1"} 10
logstream_query_duration_seconds_bucket{le="0.25"} 10
logstream_query_duration_seconds_bucket{le="0.5"} 10
logstream_query_duration_seconds_bucket{le="1.0"} 10
logstream_query_duration_seconds_bucket{le="2.5"} 10
logstream_query_duration_seconds_bucket{le="5.0"} 10
logstream_query_duration_seconds_bucket{le="10.0"} 10
logstream_query_duration_seconds_bucket{le="+Inf"} 10
logstream_query_duration_seconds_sum 0.1234
logstream_query_duration_seconds_count 10
```

## Architecture

LogStream is built using a modular architecture with the following components:

- **Collectors**: Responsible for collecting logs from various sources
- **Processor**: Parses, filters, and transforms log entries
- **Storage**: Stores processed log entries for querying and analysis
- **API Server**: Provides HTTP API for log ingestion and querying
- **Query Engine**: Executes log queries and analysis
- **CLI Interface**: Provides command-line access to all functionality
- **Plugin System**: Allows extending functionality through plugins

The application is designed to be highly concurrent, leveraging Go's goroutines and channels for parallel processing.
