package handler

import (
	"fmt"
	"os"

	"github.com/resend/resend-go/v2"
)

func sendWelcomeEmail(toEmail, toName string) error {
	client := resend.NewClient(os.Getenv("RESEND_API_KEY"))

	params := &resend.SendEmailRequest{
		From:    "Chai Shop <onboarding@resend.dev>",
		To:      []string{toEmail},
		Subject: "চায়ের দোকানে স্বাগতম! ☕",
		Html: fmt.Sprintf(`
			<h1>স্বাগতম, %s!</h1>
			<p>চায়ের দোকানে আপনাকে স্বাগত জানাই।</p>
			<p>আপনার account সফলভাবে তৈরি হয়েছে।</p>
			<p>এখন আপনি আমাদের menu দেখতে পারবেন।</p>
			<br>
			<p>ধন্যবাদ ☕</p>
		`, toName),
	}

	_, err := client.Emails.Send(params)
	return err
}