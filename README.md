# Enterprise E-commerce Backend

A production-ready e-commerce backend built with Go, following clean architecture principles.

## 🚀 Quick Start

1. **Clone and setup the project:**
   ```bash
   # After running the setup script, you're already in the project directory
   make setup
   ```

2. **Configure environment:**
   ```bash
   # Edit .env file with your configuration
   vim .env
   ```

3. **Start development environment:**
   ```bash
   make dev
   ```

4. **Access the application:**
   - API: http://localhost:8080
   - Database Admin: http://localhost:8081
   - Redis Admin: http://localhost:8082

## 📁 Project Structure

```
ecommerce-backend/
├── cmd/                    # Application entry points
├── internal/               # Private application code
├── configs/                # Configuration files
├── scripts/                # Database and deployment scripts
├── docs/                   # Documentation
└── tests/                  # Test files
```

## 🛠 Development Commands

```bash
make help          # Show all available commands
make dev           # Start development environment
make logs          # Show application logs
make shell         # Access backend container
make tools         # Start admin tools
```

## 📚 Documentation

- [API Documentation](docs/api.md)
- [Database Schema](docs/database.md)
- [Deployment Guide](docs/deployment.md)

## 🧪 Testing

```bash
make test          # Run all tests
make test-coverage # Run tests with coverage
```

## 📦 Production Build

```bash
make docker-build  # Build production Docker image
```
# thesheunit
