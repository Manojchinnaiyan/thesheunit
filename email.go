package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/pkg/email"
)

func main() {
	godotenv.Load()
	cfg, _ := config.Load()
	emailService := email.NewEmailService(cfg)

	// Test connection
	if err := emailService.TestSMTPConnection(); err != nil {
		log.Fatal("SMTP failed:", err)
	}

	// Send test email
	ctx := context.Background()
	testEmail := &email.Email{
		To:          []string{"manojchinnaiyan000@gmail.com"},
		Subject:     "Test Email from Go Backend",
		HTMLContent: "<h1>Success!</h1><p>SMTP is working with Gmail!</p>",
		Type:        "test",
	}

	if err := emailService.SendEmail(ctx, testEmail); err != nil {
		log.Fatal("Send failed:", err)
	}

	log.Println("✅ Email sent successfully!")
}
