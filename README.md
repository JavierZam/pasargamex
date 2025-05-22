# PasargameX

PasargameX is a secure marketplace platform for game-related transactions, built with Go using clean architecture principles. It enables users to buy, sell, and trade game accounts, in-game items, and boosting services with secure middleware options.

## Features

- **Authentication System**: Secure user registration, login, token refresh and password management
- **Game Title Management**: Create and manage game categories with custom attributes
- **Product Management**: List, search, and manage gaming products with detailed attributes
- **Transaction Processing**: Multiple delivery methods including instant and middleman options
- **Review System**: Rate and review transactions with reporting functionality
- **File Management**: Upload, store, and manage images for products and verification
- **Admin Dashboard**: Verify users, manage disputes, handle middleman transactions
- **User Verification**: Identity verification process for sellers

## Tech Stack

- **Backend**: Go 1.22.4 with Echo framework
- **Authentication**: Firebase Auth
- **Database**: Firestore
- **Storage**: Google Cloud Storage
- **Containerization**: Docker and Docker Compose
- **Deployment**: Google Cloud Platform (Cloud Run)
- **CI/CD**: Google Cloud Build

## Installation

### Prerequisites

- Go 1.22.4 or higher
- Docker and Docker Compose (for containerized setup)
- Firebase project with Firestore and Storage enabled
- Google Cloud Platform account (for deployment)

### Local Setup

1. Clone the repository:
   ```
   git clone https://github.com/username/pasargamex.git
   cd pasargamex
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Set up environment variables (create a `.env` file):
   ```
   SERVER_PORT=8080
   FIREBASE_PROJECT_ID=your-firebase-project-id
   FIREBASE_API_KEY=your-firebase-api-key
   ENVIRONMENT=development
   JWT_SECRET=your-jwt-secret
   STORAGE_BUCKET=your-gcs-bucket-name
   FIREBASE_SERVICE_ACCOUNT_PATH=./path-to-firebase-service-account.json
   ```

4. Run the application:
   ```
   go run cmd/api/main.go
   ```

### Docker Setup

1. Build and run with Docker Compose:
   ```
   docker-compose up --build
   ```

## Project Structure

```
pasargamex/
├── cmd/
│   └── api/           # Application entry point
├── internal/
│   ├── adapter/       # External adapters (API, repositories)
│   │   ├── api/       # HTTP handlers and middleware
│   │   └── repository/# Database repositories
│   ├── domain/        # Business domain (entities, repositories interfaces)
│   ├── infrastructure/# Technical implementations (Firebase, Storage)
│   └── usecase/       # Business logic
├── pkg/               # Shared packages
├── tests/             # Test files
└── docker-compose.yml # Docker configuration
```

## API Endpoints

The API follows RESTful principles with these main endpoints:

- **Authentication**
  - `POST /v1/auth/register` - Register a new user
  - `POST /v1/auth/login` - Login user
  - `POST /v1/auth/refresh` - Refresh token

- **Users**
  - `GET /v1/users/me` - Get user profile
  - `PATCH /v1/users/me` - Update profile
  - `POST /v1/users/me/verification` - Submit verification

- **Game Titles**
  - `GET /v1/game-titles` - List all game titles
  - `GET /v1/game-titles/:id` - Get game title details
  - `GET /v1/games/:slug` - Get game by slug

- **Products**
  - `GET /v1/products` - List all products
  - `GET /v1/products/search` - Search products
  - `GET /v1/products/:id` - Get product details
  - `GET /v1/my-products` - List user's products
  - `POST /v1/my-products` - Create a product
  - `PUT /v1/my-products/:id` - Update a product

- **Transactions**
  - `POST /v1/transactions` - Create a transaction
  - `GET /v1/transactions` - List transactions
  - `GET /v1/transactions/:id` - Get transaction details
  - `POST /v1/transactions/:id/payment` - Process payment
  - `POST /v1/transactions/:id/confirm` - Confirm delivery

- **Files**
  - `POST /v1/files/upload` - Upload file
  - `GET /v1/files/view/:id` - View file
  - `POST /v1/files/delete` - Delete file

- **Chat & Messaging**
  - 

## Deployment

The application is designed to be deployed on Google Cloud Platform using Cloud Run. The repository includes a Dockerfile and configuration for GCP deployment.

### Google Cloud Platform Deployment

1. Build the Docker image:
   ```
   docker build -t gcr.io/your-project-id/pasargamex .
   ```

2. Push to Google Container Registry:
   ```
   docker push gcr.io/your-project-id/pasargamex
   ```

3. Deploy to Cloud Run:
   ```
   gcloud run deploy pasargamex \
     --image gcr.io/your-project-id/pasargamex \
     --platform managed \
     --region asia-southeast1 \
     --allow-unauthenticated
   ```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.