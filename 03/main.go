package main

import (
	"log"

	"groq-iris-english/config"
	"groq-iris-english/database"
	"groq-iris-english/handlers"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	// Thiết lập view engine (HTML)
	app.RegisterView(iris.HTML("./views", ".html"))

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

	// Routes
	app.Get("/", handlers.IndexHandler)
	app.Post("/process", handlers.ProcessHandler)

	// Chạy ứng dụng trên cổng 8080
	err = app.Listen(":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
