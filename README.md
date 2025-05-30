# Mobile Name Lookup Service

This service provides a web interface for looking up names associated with mobile numbers using the Digitap API.

## Features

- Web interface for mobile number lookups
- Rate limiting (5 requests per minute per IP)
- Structured logging
- Docker support with Docker Compose
- Environment variable configuration
- Health checks and automatic restarts

## Environment Variables

- `PORT` - Port to run the server on (default: 8080)
- `DIGITAP_AUTH_TOKEN` - Your Digitap API authentication token (required)
- `DIGITAP_BASE_URL` - Digitap API base URL (default: https://svc.digitap.ai)

## Local Development

1. Install Go 1.21 or later
2. Set up environment variables:
   ```bash
   export DIGITAP_AUTH_TOKEN="your-token-here"
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Run the server:
   ```bash
   go run main.go
   ```

## Docker Compose Deployment (Recommended)

1. Create a `.env` file with your Digitap API token:
   ```
   DIGITAP_AUTH_TOKEN=your-token-here
   ```

2. Start the service:
   ```bash
   docker-compose up -d
   ```

3. View logs:
   ```bash
   docker-compose logs -f
   ```

4. Stop the service:
   ```bash
   docker-compose down
   ```

## Manual Docker Deployment

1. Build the Docker image:
   ```bash
   docker build -t mobile-lookup-service .
   ```

2. Run the container:
   ```bash
   docker run -p 8080:8080 \
     -e DIGITAP_AUTH_TOKEN="your-token-here" \
     mobile-lookup-service
   ```

## Cloud Deployment

### Google Cloud Run

1. Build and push the image:
   ```bash
   gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/mobile-lookup-service
   ```

2. Deploy to Cloud Run:
   ```bash
   gcloud run deploy mobile-lookup-service \
     --image gcr.io/YOUR_PROJECT_ID/mobile-lookup-service \
     --platform managed \
     --allow-unauthenticated \
     --set-env-vars="DIGITAP_AUTH_TOKEN=your-token-here"
   ```

### Heroku

1. Install Heroku CLI and login
2. Create a new Heroku app:
   ```bash
   heroku create
   ```

3. Set environment variables:
   ```bash
   heroku config:set DIGITAP_AUTH_TOKEN="your-token-here"
   ```

4. Deploy:
   ```bash
   git push heroku main
   ```

## Security Considerations

- The service implements rate limiting to prevent abuse
- Sensitive error messages are not exposed to clients
- All requests are logged for monitoring
- API credentials are managed through environment variables
- HTTPS is required in production (handled by cloud providers)
- Container includes health checks and automatic restarts
- Log rotation is configured to prevent disk space issues

## Monitoring

The service outputs structured JSON logs that can be collected by logging platforms like:
- Google Cloud Logging
- AWS CloudWatch
- ELK Stack
- Datadog

Docker Compose configuration includes:
- Health checks
- Automatic restarts
- Log rotation (max 3 files of 10MB each)

## Support

For issues or questions, please open a GitHub issue. 