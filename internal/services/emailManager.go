package services

import (
	"fmt"

	"github.com/omaradriano/cobranzawebscrapper_server/env"
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
		destination_router = env.Envs.MailDestinationWeb
		break
	case "Register":
		destination_router = env.Envs.MailDestinationServer
		email_type_value = "verifyaccount"
		displayMessage = "Verificar cuenta"
		break
	}

	params := &resend.SendEmailRequest{
		From: "notificaciones@goagent.com.mx",
		To:   []string{destination},
		Html: fmt.Sprintf(`<a href="%s/auth/%s?token=%s&setpasswordmode=resetpassword">%s c:</a>`,
			destination_router, email_type_value, confimation_token, displayMessage),
		Subject: "Confirmación de cuenta para GoAgent",
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
		From:    "notificaciones@goagent.com.mx",
		To:      []string{destination},
		Html:    fmt.Sprintf(`<p>%s</p>`, message),
		Subject: "Confirmación de cambio de contraseña para GoAgent",
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	fmt.Println(sent.Id)

	return nil
}
