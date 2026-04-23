package services

import (
	"fmt"

	"github.com/resend/resend-go/v3"
)

// re_UapR6ViR_34ED5n3Cy71Mzf6NrEcpuqot
func SendMail(destination, confimation_token, email_type string) error {
	client := resend.NewClient("re_UapR6ViR_34ED5n3Cy71Mzf6NrEcpuqot")
	var email_type_value string
	var displayMessage string

	switch email_type {
	case "ResetPassword":
		email_type_value = "setpassword"
		displayMessage = "Restablecer contraseña"
		break
	case "Register":
		email_type_value = "verifyaccount"
		displayMessage = "Verificar cuenta"
		break
	}

	params := &resend.SendEmailRequest{
		From: "Acme <onboarding@resend.dev>",
		To:   []string{destination},
		Html: fmt.Sprintf(`<a href="http://127.0.0.1:5173/auth/%s?token=%s">%s c:</a>`,
			email_type_value, confimation_token, displayMessage),
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
