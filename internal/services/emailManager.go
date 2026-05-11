package services

import (
	"fmt"

	"github.com/resend/resend-go/v3"
)

const (
	CLIENT_TOKEN string = "re_UapR6ViR_34ED5n3Cy71Mzf6NrEcpuqot"
)

// re_UapR6ViR_34ED5n3Cy71Mzf6NrEcpuqot
func SendMail(destination, confimation_token, email_type string) error {
	client := resend.NewClient(CLIENT_TOKEN)
	var email_type_value string
	var displayMessage string
	// var destination string
	var destination_router string

	switch email_type {
	case "ResetPassword":
		email_type_value = "setpassword"
		displayMessage = "Restablecer contraseña"
		destination_router = "localhost:5173"
		break
	case "Register":
		destination_router = "127.0.0.1:3006/v1"
		email_type_value = "verifyaccount"
		displayMessage = "Verificar cuenta"
		break
	}

	params := &resend.SendEmailRequest{
		From: "Acme <onboarding@resend.dev>",
		To:   []string{destination},
		Html: fmt.Sprintf(`<a href="http://%s/auth/%s?token=%s">%s c:</a>`,
			destination_router, email_type_value, confimation_token, displayMessage),
		Subject: "Confirmación de cuenta para polizas-tracker",
		// Cc:      []string{"cc@example.com"},
		// Bcc:     []string{"bcc@example.com"},
		// ReplyTo: "replyto@example.com",
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	fmt.Println(sent.Id)

	return nil
}

func SendCustomMail(destination, message string) error {
	client := resend.NewClient(CLIENT_TOKEN)

	params := &resend.SendEmailRequest{
		From:    "Acme <onboarding@resend.dev>",
		To:      []string{destination},
		Html:    fmt.Sprintf(`<p>%s</p>`, message),
		Subject: "Confirmación de cambio de contraseña",
		// Cc:      []string{"cc@example.com"},
		// Bcc:     []string{"bcc@example.com"},
		// ReplyTo: "replyto@example.com",
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	fmt.Println(sent.Id)

	return nil
}
