# Fire-Alarm

A system to monitor transit incidents and notify subscribers about problems affecting their preferred stations.

## Overview

Fire-Alarm is a two-component system:

- **Service**: A CLI tool that processes incident data and sends email alerts to subscribed users based on their station preferences
- **Website**: A web server where users subscribe, verify their email, and manage their alert preferences

Incidents are matched against user preferences by station name, and alerts are sent via email to affected subscribers.

## Quick Start

### 1. Prerequisites

- Rust 2024 or later
- SQLite, MySQL, or PostgreSQL database
- SMTP relay credentials (e.g., Gmail App Password, SendGrid, etc.)
- DC Metro station data (included; see Setup)

### 2. Setup

Initialize the database schema and populate station/rail line data:

```bash
./setup.sh
```

This script:

- Creates a temporary SQLite database
- Runs `setup.sql` to create tables (users, stations, rail lines, subscriptions)
- Generates ORM code for both `service/` and `website/` using `sea-orm-cli`

### 3. Configure Environment

Create a `.env` file with your SMTP credentials:

```bash
# .env
PASSWORD=your_smtp_password
RELAY=smtp.gmail.com:587
ADDRESS=alerts@example.com
DATABASE=sqlite://db.sqlite
```

Alternatively, export environment variables when running each component.

### 4. Run the Website

```bash
cd website
cargo run --release --features env
# Server listens on http://127.0.0.1:8080
```

Users can now:

- Visit the web interface
- Enter their email and select stations
- Confirm their subscription via a verification code sent by email

### 5. Run the Service

Process incidents and send alerts:

```bash
cd service
cat incidents.json | cargo run --release --features env
```

Or integrate into a scheduled job (e.g., cron) to periodically fetch incidents and send notifications.

## Project Structure

```
fire-alarm/
├── service/              # CLI tool for sending alerts
│   ├── src/
│   ├── Cargo.toml
│   └── README.md
├── website/              # Web interface for subscriptions
│   ├── src/
│   ├── index.html        # Web UI template
│   ├── style.css
│   ├── index.js
│   ├── Cargo.toml
│   └── README.md
├── setup.sql             # Database schema
├── setup.sh              # Initialization script
├── csv2sql.go            # Helper to populate station data
├── wmata.sqlite          # Embedded station/rail line reference data
└── LICENSE               # MIT License
```

## Components

### [Service](./service/README.md)

A Rust CLI that:

1. Reads incident data from stdin (JSON format)
2. Queries the database for user subscriptions
3. Matches incidents to stations in user preferences
4. Sends email notifications via SMTP
5. Tracks the last check timestamp to avoid duplicates

**Use cases:**

- Run on a schedule to periodically fetch and process incidents
- Integrate with an incident detection system
- Batch process incident streams

### [Website](./website/README.md)

A Rust web server (using Axum) that:

1. Serves a subscription form where users enter their email and select stations
2. Validates email addresses with one-time passcodes (OTP)
3. Persists verified subscriptions to the database
4. Handles unsubscribe requests

**Features:**

- Email verification to ensure valid addresses
- Configurable OTP timeout (default: 5 minutes)
- Graceful shutdown handling
- Support for SQLite, MySQL, and PostgreSQL

## Configuration

Each component (service and website) can be configured via:

1. **Command-line arguments** — Use `--help` to see all options
2. **Environment variables** — Compile with `--features env`, then set vars like `PASSWORD`, `RELAY`, `ADDRESS`, `DATABASE`, etc.

See individual READMEs for detailed configuration options:

- [Service Configuration](./service/README.md#configuration)
- [Website Configuration](./website/README.md#configuration)

## Database Schema

The system uses these core tables:

- **Users** — Email addresses and verification status
- **Stations** — Transit station names and IDs
- **RailLines** — Rail line names (Red, Blue, Green, etc.)
- **UserStations** — Which stations each user is subscribed to

See [setup.sql](./setup.sql) for the complete schema.

## Development

### Build Both Components

```bash
cd service && cargo build --release
cd ../website && cargo build --release
```

### Run Tests

```bash
# Service tests
cd service && cargo test

# Website tests
cd website && cargo test
```

### Generate ORM Code

If you modify the database schema:

```bash
./setup.sh
```

This regenerates the ORM bindings used by both components.

## License

MIT License — Copyright (c) 2026 Tarun Singh

See [LICENSE](./LICENSE) for details.

## Related Files

- [setup.sql](./setup.sql) — Database schema
- [csv2sql.go](./csv2sql.go) — Helper tool to import station data from CSV
- [wmata.sqlite](./wmata.sqlite) — Reference WMATA station and rail line data
