# Restaurant Service System

An open-source system for restaurant order management, designed to run on minimal hardware while providing robust features for order taking, routing, and kitchen display.

## Features

- Menu management with categories, items, and modifiers
- Order creation and routing to appropriate preparation areas
- Real-time updates via WebSockets
- Support for multiple output devices (printers, displays)
- Responsive web UI for POS and kitchen display
- Admin dashboard for configuration

## Technology Stack

- Backend: Go
- Database: PostgreSQL
- Frontend: Progressive Web App
- Real-time: WebSockets
- Deployment: Docker (optional)

## Getting Started

### Prerequisites

- Go 1.18 or higher
- PostgreSQL 14 or higher
- Node.js 16 or higher (for UI development)

### Development Setup

1. Clone this repository:
    git clone https://github.com/yourusername/restaurant-service.git
    cd restaurant-service
2. Install dependencies:
    go mod download
3. Copy example configuration:
    cp configs/config.yaml.example configs/development.yaml
4. Start PostgreSQL (using Docker):
    make docker-up
5. Run database migrations:
    make migrate-up
6. Start the development server:
    make run
    
## Project Structure

- `/cmd`: Entry points for applications
- `/configs`: Configuration files
- `/internal`: Private application code
- `/migrations`: Database migrations
- `/pkg`: Public libraries
- `/scripts`: Utility scripts
- `/ui`: Web user interfaces
- `/web`: Web server assets

## Documentation

See the [docs](./docs) directory for detailed documentation.

## License

This project is licensed under the MIT License - see the LICENSE file for details.