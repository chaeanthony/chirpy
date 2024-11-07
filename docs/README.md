# Chirpy

Chirpy is a social network similar to Twitter. Built using Go (Golang) and PostgreSQL. Follows RESTful principles.

## Table of Contents

- [Getting Started](#getting-started)
- [API](#api)
  - [User Management](#user-management)
  - [Chirps](#chirp-management)

## Getting Started

### Prerequisites

This is a learning project. To run this API, you need:

- Go (this project was built using 1.23)
- PostgreSQL

### Installation

1. Clone the repository:

```bash
git clone https://github.com/chaeanthony/chirpy.git
cd chirpy
```

2. Install dependencies:

```bash
go mod tidy
```

3. Create .env:

```
DB_URL = "database url"
PLATFORM = "dev" (prevent dangerous endpoints from being accessed in production)
JWT_SECRET = "jwt secret"
POLKA_KEY = "payment api key"
```

## API

#### Base URL

The base URL for the API is `/api`.

#### Middleware

The application uses middleware for metrics collection. The metrics are incremented for each request to the API.

#### Static File Serving

Static files are served from the `/app/` directory. The prefix `/app` is stripped from the request path before serving files.

#### Static File Endpoint

- **Path**: `/app/`
- **Method**: `GET`
- **Description**: Serves static files from the specified directory.

#### Health Check

- **Path**: `/api/healthz`
- **Method**: `GET`
- **Description**: Returns API ready.

### User Management

#### Create User

- **Path**: `/api/users`
- **Method**: `POST`
- **Parameters**: {"email": "test@email.com", "password": "123456"}
- **Description**: Creates a new user.

#### Update User

- **Path**: `/api/users`
- **Method**: `PUT`
- **Parameters**: {"email": "test@email.com", "password": "123456"}
- **Description**: Updates an existing user's email and/or password.

#### User Login

- **Path**: `/api/login`
- **Method**: `POST`
- **Parameters**: {"email": "test@email.com", "password": "123456"}
- **Description**: Authenticates a user and returns a session token.

#### Refresh Token

- **Path**: `/api/refresh`
- **Method**: `POST`
- **Description**: Accepts Bearer token in authorization header (this is a refresh token). Uses refresh token to create a new jwt token.

#### Revoke Token

- **Path**: `/api/revoke`
- **Method**: `POST`
- **Description**: Revokes the user's refresh token.

### Chirp Management

#### Get All Chirps

- **Path**: `/api/chirps?sort=asc&author_id=2`
- **Method**: `GET`
- **Description**: Retrieves a list of all chirps. _Optional sort and author_id url paramters._

#### Get Specific Chirp

- **Path**: `/api/chirps/{chirpId}`
- **Method**: `GET`
- **Description**: Retrieves a specific chirp by its ID.

#### Create Chirp

- **Path**: `/api/chirps`
- **Method**: `POST`
- **Paramters**: {"body": "paragraph"}
- **Description**: Creates a new chirp.

#### Delete Chirp

- **Path**: `/api/chirps/{chirpId}`
- **Method**: `DELETE`
- **Description**: Deletes a specific chirp by its ID.

### Polka Integration

#### Upgrade User to Red

- **Path**: `/api/polka/webhooks`
- **Method**: `POST`
- **Description**: Handles webhooks to upgrade a user to red status.

### Admin Routes

#### Metrics

- **Path**: `/admin/metrics`
- **Method**: `GET`
- **Description**: Retrieves metrics for the application.

#### Reset Admin

- **Path**: `/admin/reset`
- **Method**: `POST`
- **Description**: Resets admin-related data or state.
