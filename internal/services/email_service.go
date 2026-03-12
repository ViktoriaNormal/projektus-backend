package services

import (
	"context"
	"log"
)

// EmailService is a simple abstraction for sending emails.
// For now it's a stub that just logs messages.
type EmailService interface {
	SendEmail(ctx context.Context, to string, subject string, body string) error
}

type logEmailService struct{}

func NewEmailService() EmailService {
	return &logEmailService{}
}

func (s *logEmailService) SendEmail(ctx context.Context, to string, subject string, body string) error {
	log.Printf("[EMAIL STUB] To=%s Subject=%s Body=%s", to, subject, body)
	return nil
}

