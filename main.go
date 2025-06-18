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
	"regexp"
	"strings"
	"sync"
	"time"

	"mobile-name-lookup/db"

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
        .db-record {
            margin-top: 20px;
            padding: 15px;
            border-radius: 4px;
            background-color: #e9ecef;
            font-size: 16px;
        }
        .db-record strong {
            color: #495057;
        }
        .timestamp {
            font-size: 14px;
            color: #6c757d;
            margin-top: 5px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Mobile Name Lookup</h1>
        <form method="POST" action="/lookup">
            <div class="form-group">
                <label for="mobile">Mobile Number:</label>
                <input type="tel" id="mobile" name="mobile" required 
                       placeholder="e.g., 8318090007 or +91 83180 90007" 
                       title="Enter a 10-digit mobile number. Country codes and spaces are automatically handled.">
                <small style="color: #6c757d; font-size: 0.875em;">
                    Supports formats: 8318090007, +91 83180 90007, +91-83180-90007
                </small>
            </div>
            <button type="submit">Lookup</button>
        </form>
        {{if .Record}}
        <div class="db-record">
            <strong>Name:</strong> {{.Record.Name}}<br>
            <strong>Mobile:</strong> {{.Record.Mobile}}<br>
        </div>
        {{end}}
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

// PageData represents the data passed to the template
type PageData struct {
	Result *MobileNameLookupResponse
	Error  string
	Record *db.MobileRecord
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

// cleanPhoneNumber removes all non-digit characters and handles country codes
func cleanPhoneNumber(phone string) (string, error) {
	// Remove all non-digit characters
	re := regexp.MustCompile(`[^\d]`)
	digits := re.ReplaceAllString(phone, "")

	// Handle different formats
	if len(digits) == 0 {
		return "", fmt.Errorf("no digits found in phone number")
	}

	// If it starts with country code (e.g., 91 for India), remove it
	if len(digits) > 10 {
		// Common country codes: 91 (India), 1 (US/Canada), 44 (UK), etc.
		if strings.HasPrefix(digits, "91") && len(digits) == 12 {
			digits = digits[2:] // Remove 91
		} else if strings.HasPrefix(digits, "1") && len(digits) == 11 {
			digits = digits[1:] // Remove 1
		} else if strings.HasPrefix(digits, "44") && len(digits) == 12 {
			digits = digits[2:] // Remove 44
		} else if len(digits) > 10 {
			// For other country codes, try to extract the last 10 digits
			digits = digits[len(digits)-10:]
		}
	}

	// Validate the final number
	if len(digits) != 10 {
		return "", fmt.Errorf("invalid phone number length: %d digits (expected 10)", len(digits))
	}

	// Check if it's a valid Indian mobile number (starts with 6, 7, 8, 9)
	if !regexp.MustCompile(`^[6-9]\d{9}$`).MatchString(digits) {
		return "", fmt.Errorf("invalid mobile number format")
	}

	return digits, nil
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

	// Initialize database
	database, err := db.NewDB()
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer database.Close()

	// Test database connection
	if err := database.TestConnection(); err != nil {
		logger.WithError(err).Fatal("Failed to test database connection")
	}
	logger.Info("Successfully connected to database")

	// Initialize database schema
	if err := database.InitDB(); err != nil {
		logger.WithError(err).Fatal("Failed to initialize database")
	}
	logger.Info("Successfully initialized database schema")

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

	// Handle root path - GET request to show the form
	http.HandleFunc("/", rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Show empty form
		tmpl.Execute(w, PageData{})
	}, limiter))

	// Handle form submission - POST request
	http.HandleFunc("/lookup", rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Redirect GET requests to home page
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		case http.MethodPost:
			// Handle POST request
			if err := r.ParseForm(); err != nil {
				logger.WithError(err).Error("Failed to parse form")
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}

			rawMobile := r.FormValue("mobile")
			if rawMobile == "" {
				tmpl.Execute(w, PageData{Error: "Mobile number is required"})
				return
			}

			// Clean and validate mobile number
			mobile, err := cleanPhoneNumber(rawMobile)
			if err != nil {
				tmpl.Execute(w, PageData{Error: fmt.Sprintf("Invalid mobile number: %v", err)})
				return
			}

			// Log request
			logger.WithFields(logrus.Fields{
				"raw_mobile":   rawMobile,
				"clean_mobile": mobile,
				"ip":           r.RemoteAddr,
				"method":       r.Method,
			}).Info("Lookup request received")

			// First, check if we have the record in our database
			record, err := database.GetMobileRecord(mobile)
			if err != nil {
				logger.WithError(err).Error("Failed to query database")
				tmpl.Execute(w, PageData{Error: "Database error occurred"})
				return
			}

			if record != nil {
				// We found the record in our database
				logger.WithFields(logrus.Fields{
					"mobile": mobile,
					"name":   record.Name,
				}).Info("Found record in database")
				tmpl.Execute(w, PageData{Record: record})
				return
			}

			// If not in database, query the API
			clientRefNum := fmt.Sprintf("REF_%d", time.Now().Unix())
			name := ""

			response, err := client.LookupMobileName(clientRefNum, mobile, name)
			if err != nil {
				logger.WithError(err).WithFields(logrus.Fields{
					"mobile":     mobile,
					"client_ref": clientRefNum,
				}).Error("Lookup failed")
				tmpl.Execute(w, PageData{Error: "Service temporarily unavailable. Please try again."})
				return
			}

			// If we got a name from the API, save it to our database
			if response.Result.MobileLinkedName != "" {
				if err := database.SaveMobileRecord(mobile, response.Result.MobileLinkedName); err != nil {
					logger.WithError(err).Error("Failed to save record to database")
				}
			}

			logger.WithFields(logrus.Fields{
				"mobile":     mobile,
				"status":     response.Status,
				"client_ref": clientRefNum,
			}).Info("Lookup successful")

			tmpl.Execute(w, PageData{Result: response})
			return
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
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
