package notification

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/resend/resend-go/v3"
)

type resendEmail struct {
	From    string
	To      []string
	Subject string
	HTML    string
	ReplyTo []string
}

type ResendSender struct {
	fromEmail string
	fromName  string
	sendEmail func(ctx context.Context, msg resendEmail) error
}

func NewResendSender(apiKey, fromEmail, fromName string) (*ResendSender, error) {
	apiKey = strings.TrimSpace(apiKey)
	fromEmail = strings.TrimSpace(fromEmail)
	fromName = strings.TrimSpace(fromName)

	switch {
	case apiKey == "":
		return nil, errors.New("RESEND_API_KEY is required")
	case fromEmail == "":
		return nil, errors.New("EMAIL_FROM is required")
	case fromName == "":
		return nil, errors.New("EMAIL_FROM_NAME is required")
	}

	client := resend.NewClient(apiKey)

	return &ResendSender{
		fromEmail: fromEmail,
		fromName:  fromName,
		sendEmail: func(_ context.Context, msg resendEmail) error {
			req := &resend.SendEmailRequest{
				From:    msg.From,
				To:      msg.To,
				Subject: msg.Subject,
				Html:    msg.HTML,
			}
			if len(msg.ReplyTo) > 0 {
				req.ReplyTo = msg.ReplyTo[0]
			}
			_, err := client.Emails.Send(req)
			return err
		},
	}, nil
}

func (s *ResendSender) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	message := resendEmail{
		From:    s.fromHeader(),
		To:      []string{strings.TrimSpace(to)},
		Subject: "Reset your password",
		HTML:    passwordResetHTML(resetURL),
		ReplyTo: []string{s.fromEmail},
	}
	return s.sendEmail(ctx, message)
}

func (s *ResendSender) fromHeader() string {
	return fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
}

func passwordResetHTML(resetURL string) string {
	safeURL := html.EscapeString(resetURL)
	return fmt.Sprintf(`
<p>You requested a password reset.</p>
<p><a href="%s">Reset your password</a></p>
<p>This link expires in 1 hour. If you did not request this, you can ignore this email.</p>
`, safeURL)
}
