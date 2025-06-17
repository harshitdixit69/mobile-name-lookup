package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// MobileNameLookupResponse represents the API response structure
type MobileNameLookupResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		MobileLinkedName string `json:"mobile_linked_name"`
	} `json:"result"`
}

// DigitapClient handles API communication
type DigitapClient struct {
	BaseURL    string
	AuthToken  string
	HTTPClient *http.Client
}

// NewDigitapClient creates a new client instance
func NewDigitapClient(baseURL, authToken string) *DigitapClient {
	return &DigitapClient{
		BaseURL:    baseURL,
		AuthToken:  authToken,
		HTTPClient: &http.Client{},
	}
}

// LookupMobileName performs the mobile name lookup with retry logic
func (c *DigitapClient) LookupMobileName(clientRefNum, mobile, name string) (*MobileNameLookupResponse, error) {
	url := c.BaseURL + "/validation/misc/v1/mobile-name-lookup"

	payload := fmt.Sprintf(`{
		"client_ref_num": "%s",
		"mobile": "%s",
		"name": "%s"
	}`, clientRefNum, mobile, name)

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, strings.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Add("Authorization", "Basic "+c.AuthToken)
		req.Header.Add("Content-Type", "application/json")

		// Set timeout for the request
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			logger.WithError(err).WithField("attempt", attempt+1).Warn("Request failed, retrying...")
			time.Sleep(time.Duration(attempt+1) * time.Second) // Exponential backoff
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			logger.WithError(err).WithField("attempt", attempt+1).Warn("Failed to read response, retrying...")
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		var response MobileNameLookupResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse response: %v", err)
		}

		return &response, nil
	}

	return nil, fmt.Errorf("all retry attempts failed: %v", lastErr)
}

// HTML template for the mobile interface
const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Mobile Name Lookup</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        button {
            background-color: #4CAF50;
            color: white;
            padding: 10px 15px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            width: 100%;
        }
        button:hover {
            background-color: #45a049;
        }
        .result {
            margin-top: 20px;
            padding: 15px;
            border-radius: 4px;
            background-color: #f8f9fa;
            font-size: 18px;
            text-align: center;
        }
        .error {
            color: #dc3545;
            margin-top: 10px;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Mobile Name Lookup</h1>
        <form method="POST" action="/lookup">
            <div class="form-group">
                <label for="mobile">Mobile Number:</label>
                <input type="tel" id="mobile" name="mobile" required>
            </div>
            <button type="submit">Lookup</button>
        </form>
        {{if .Result}}
        <div class="result">
            {{if .Result.Result.MobileLinkedName}}
            <strong>Name:</strong> {{.Result.Result.MobileLinkedName}}
            {{else}}
            No name found for this number
            {{end}}
        </div>
        {{end}}
        {{if .Error}}
        <div class="error">
            {{.Error}}
        </div>
        {{end}}
    </div>
</body>
</html>
`

type PageData struct {
	Result *MobileNameLookupResponse
	Error  string
}

// Logger instance
var logger = logrus.New()

// RateLimiter represents a rate limiter for an IP
type RateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter manages rate limiting by IP address
type IPRateLimiter struct {
	ips   map[string]*RateLimiter
	mu    sync.RWMutex
	rate  rate.Limit
	burst int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips:   make(map[string]*RateLimiter),
		rate:  r,
		burst: b,
	}
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := &RateLimiter{
		limiter:  rate.NewLimiter(i.rate, i.burst),
		lastSeen: time.Now(),
	}

	i.ips[ip] = limiter
	return limiter.limiter
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.ips[ip]

	if !exists {
		i.mu.Unlock()
		return i.AddIP(ip)
	}

	limiter.lastSeen = time.Now()
	i.mu.Unlock()
	return limiter.limiter
}

// Middleware for rate limiting
func rateLimitMiddleware(next http.HandlerFunc, limiter *IPRateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if !limiter.GetLimiter(ip).Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			logger.WithFields(logrus.Fields{
				"ip":     ip,
				"status": "rate_limited",
			}).Warn("Rate limit exceeded")
			return
		}
		next(w, r)
	}
}

func main() {
	// Only try to load .env file if we're not in a cloud environment
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		if err := godotenv.Load(); err != nil {
			logger.WithError(err).Info("No .env file found in development environment")
		}
	}

	// Configure logging
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Get environment variables with defaults
	baseURL := getEnvOrDefault("DIGITAP_BASE_URL", "https://svc.digitap.ai")
	authToken := getEnvOrDefault("DIGITAP_AUTH_TOKEN", "")
	if authToken == "" {
		logger.Fatal("DIGITAP_AUTH_TOKEN environment variable is required")
	}

	// Create HTTP client with custom timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	// Create rate limiter (5 requests per minute per IP)
	limiter := NewIPRateLimiter(rate.Every(12*time.Second), 5)

	// Create client with custom HTTP client
	client := &DigitapClient{
		BaseURL:    baseURL,
		AuthToken:  authToken,
		HTTPClient: httpClient,
	}

	// Parse template
	tmpl := template.Must(template.New("mobile").Parse(htmlTemplate))

	// Handle form submission with rate limiting
	http.HandleFunc("/lookup", rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mobile := r.FormValue("mobile")
		if mobile == "" {
			tmpl.Execute(w, PageData{Error: "Mobile number is required"})
			return
		}

		// Log request
		logger.WithFields(logrus.Fields{
			"mobile": mobile,
			"ip":     r.RemoteAddr,
			"method": r.Method,
		}).Info("Lookup request received")

		// Hardcoded values
		clientRefNum := fmt.Sprintf("REF_%d", time.Now().Unix())
		name := ""

		response, err := client.LookupMobileName(clientRefNum, mobile, name)
		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"mobile":     mobile,
				"client_ref": clientRefNum,
			}).Error("Lookup failed")
			tmpl.Execute(w, PageData{Error: "Service temporarily unavailable"})
			return
		}

		logger.WithFields(logrus.Fields{
			"mobile":     mobile,
			"status":     response.Status,
			"client_ref": clientRefNum,
		}).Info("Lookup successful")

		tmpl.Execute(w, PageData{Result: response})
	}, limiter))

	// Handle root path
	http.HandleFunc("/", rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		tmpl.Execute(w, PageData{})
	}, limiter))

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	logger.WithField("port", port).Info("Server starting")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
