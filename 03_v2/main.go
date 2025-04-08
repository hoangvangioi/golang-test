package main

import (
	"log"
	"vocabulary/config"
	"vocabulary/database"
	"vocabulary/handlers"

	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	// Enable CORS with default settings or custom options
	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})

	app.UseRouter(crs)

	// Tải cấu hình
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Kết nối cơ sở dữ liệu
	err = database.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB()

	// Register routes
	app.Get("/", handlers.IndexHandler)
	app.Get("/dialog", handlers.GenerateDialogHandler)
	app.Get("/words", handlers.ExtractWordsHandler)
	app.Post("/translate", handlers.TranslateWordsHandler)
	app.Post("/save-words", handlers.SaveWordsHandler)

	// Start server
	err = app.Listen(":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
