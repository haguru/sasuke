# sasuke

This project is designed to help gain a better understanding of authentication using JWT (JSON Web Tokens) in a Go backend environment. It demonstrates secure user registration, login, and session management using JWTs, bcrypt password hashing, and best practices for structuring Go web services.

## Features

- User registration and login with hashed passwords (bcrypt)
- JWT-based authentication for secure session management
- Configurable via YAML files
- Modular Go project structure
- Makefile for streamlined development tasks

## Usage

### Prerequisites

- Go 1.24.2 or newer
- [Make](https://www.gnu.org/software/make/)
- (Optional) MongoDB for persistent user storage

### Common Makefile Commands

Build the project:
```bash
make build
```

Run all tests:
```bash
make test
```

Format the code:
```bash
make fmt
```

Clean build artifacts:
```bash
make clean
```

### Example: Register and Login

1. Start the server (after building):
    ```bash
    ./bin/sasuke
    ```

2. Register a user:
    ```bash
    curl -X POST -H "Content-Type: application/json" \
      -d '{"username":"testuser","password":"testpass"}' \
      http://localhost:50051/register
    ```

3. Login to receive a JWT:
    ```bash
    curl -X POST -H "Content-Type: application/json" \
      -d '{"username":"testuser","password":"testpass"}' \
      http://localhost:50051/login
    ```

4. Use the JWT in subsequent requests as an Authorization header.

## Project Structure

```
sasuke/
├── cmd/                # Application entry points
├── internal/           # Core application logic
│   ├── auth/
│   ├── routes/
│   └── userservice/
├── config/             # Configuration management
├── res/                # Resource files (YAML config, keys)
├── Makefile
├── go.mod
└── README.md
```

## License

MIT