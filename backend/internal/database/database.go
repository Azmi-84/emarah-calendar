package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Global variable to hold the database connection
var db *gorm.DB

// ConnectDatabase initializes and checks the database connection
func ConnectDatabase() error {
	// Database connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),     // Database username
		os.Getenv("DB_PASSWORD"), // Database password
		os.Getenv("DB_HOST"),     // Database host
		os.Getenv("DB_NAME"),     // Database name
	)

	// Attempt to open a connection to the database
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Ping the database to ensure the connection is established
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}

	if err = sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	log.Println("Database connected successfully")
	return nil
}

// DBMiddleware checks and reconnects the database connection for each request
func DBMiddleware(c *fiber.Ctx) error {
	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		log.Println("Reconnecting to the database...")
		if err := ConnectDatabase(); err != nil {
			log.Printf("Failed to reconnect to the database: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Database connection error")
		}
	}
	return c.Next()
}

// GracefulShutdown closes the database connection on application termination
func GracefulShutdown() {
	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		} else {
			log.Println("Database connection closed successfully!")
		}
	}
}

func main() {
	// Initialize Fiber app
	app := fiber.New()

	// Connect to the database initially
	if err := ConnectDatabase(); err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	// Use the database middleware
	app.Use(DBMiddleware)

	// Sample route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, Calendar!")
	})

	// Setup graceful shutdown
	go func() {
		if err := app.Listen(":3000"); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Handle graceful shutdown on system signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	// Perform cleanup
	log.Println("Shutting down successfully...")
	GracefulShutdown()
	app.Shutdown()
}
