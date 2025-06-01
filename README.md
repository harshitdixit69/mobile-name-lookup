# Mobile Name Lookup Service

A web service that allows users to look up names associated with mobile numbers using the Digitap API.

## Features

- Web interface for mobile number lookups
- Rate limiting (5 requests per minute per IP)
- Structured logging
- Docker support with Docker Compose
- Environment variable configuration
- Health checks and automatic restarts

## Environment Variables

The following environment variables are required:

- `DIGITAP_AUTH_TOKEN`: Your Digitap API authentication token
- `PORT`: Port number for the server (default: 8080)
- `DIGITAP_BASE_URL`: Digitap API base URL (default: https://svc.digitap.ai)

## Local Development

1. Clone the repository
2. Create a `.env` file with the required environment variables
3. Run `go mod download` to install dependencies
4. Run `go run main.go` to start the server

## Docker Deployment

1. Build the Docker image:
   ```bash
   docker build -t mobile-name-lookup .
   ```

2. Run the container:
   ```bash
   docker run -p 8080:8080 --env-file .env mobile-name-lookup
   ```

## Cloud Deployment (Railway)

1. Create a new project on [Railway](https://railway.app)
2. Connect your GitHub repository
3. Add the required environment variables in Railway dashboard
4. Deploy!

## Docker Compose Deployment (Recommended)

1. Create a `.env` file with your Digitap API token:
   ```
   DIGITAP_AUTH_TOKEN=your-token-here
   ```

2. Start the service:
   ```