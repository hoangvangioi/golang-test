package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/yuin/goldmark"
)

// Config holds application configuration
type Config struct {
	APIKey string
	APIURL string
}

// Request represents incoming prompt request
type Request struct {
	Prompt string `json:"prompt"`
}

// Response represents API response structure
type Response struct {
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// loadConfig initializes configuration from environment variables
func loadConfig() Config {
	_ = godotenv.Load() // Ignore error if .env not found

	apiKey := os.Getenv("GROQ_API_KEY")
	apiURL := os.Getenv("GROQ_API_URL")
	if apiKey == "" {
		log.Fatal("GROQ_API_KEY is required")
	}
	if apiURL == "" {
		apiURL = "https://api.groq.com/openai/v1/chat/completions"
	}

	return Config{
		APIKey: apiKey,
		APIURL: apiURL,
	}
}

// groqClient handles communication with Groq API
type groqClient struct {
	config Config
	client *http.Client
}

func newGroqClient(config Config) *groqClient {
	return &groqClient{
		config: config,
		client: &http.Client{},
	}
}

// callGroqAPI makes request to Groq API and returns parsed response
func (c *groqClient) callGroqAPI(prompt string) (string, error) {
	payload := map[string]interface{}{
		"model":    "deepseek-r1-distill-llama-70b",
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", c.config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response content received")
	}

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(result.Choices[0].Message.Content), &buf); err != nil {
		return "", fmt.Errorf("failed to convert markdown: %v", err)
	}

	return buf.String(), nil
}

func main() {
	app := iris.New()
	config := loadConfig()
	groq := newGroqClient(config)

	// Serve static assets
	app.HandleDir("/views", iris.Dir("./views"))

	// Main page
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("./views/index.html")
	})

	// API endpoint
	app.Post("/api/groq", func(ctx iris.Context) {
		var req Request
		if err := ctx.ReadJSON(&req); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.JSON(Response{Error: "Invalid prompt format"})
			return
		}

		if req.Prompt == "" {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.JSON(Response{Error: "Prompt cannot be empty"})
			return
		}

		result, err := groq.callGroqAPI(req.Prompt)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.JSON(Response{Error: err.Error()})
			return
		}

		ctx.JSON(Response{Content: result})
	})

	log.Println("Server starting on :8080")
	app.Listen(":8080")
}
