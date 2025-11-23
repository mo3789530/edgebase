# EdgeBase

A distributed edge computing platform with database synchronization, serverless functions, and control plane management.

## Project Structure

- **db/** - Database service with migrations and sync capabilities
- **functions/** - Serverless function runtime and edge runner
- **platform/** - Control plane and platform services

## Quick Start

### Database
```bash
cd db
cargo build
```
See [db/README.md](db/README.md) for details.

### Functions
```bash
cd functions
cargo build
```
See [functions/README.md](functions/README.md) for details.

### Platform
```bash
cd platform
```

## Requirements

- Rust 1.70+
- Cargo

## License

See LICENSE file for details.
