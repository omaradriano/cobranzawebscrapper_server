package services

import (
	"fmt"

	"github.com/omaradriano/cobranzawebscrapper_server/env"
	"github.com/resend/resend-go/v3"
)

type emailConfig struct {
	Subject     string
	Title       string
	Message     string
	ButtonText  string
	ButtonURL   string
	FooterNote  string
}

func SendMail(destination, token, emailType string) error {
	var config emailConfig

	switch emailType {
	case "Register":
		config = emailConfig{
			Subject:    "Verifica tu cuenta - GoAgent",
			Title:      "Verificación de cuenta",
			Message:    "Se ha creado tu cuenta en GoAgent. Para activarla y comenzar a sincronizar tus pólizas, confirma tu correo electrónico.",
			ButtonText: "Verificar mi cuenta",
			ButtonURL:  fmt.Sprintf("%s/auth/verifyaccount?token=%s", env.Envs.MailDestinationServer, token),
			FooterNote: "Este enlace expira en 30 días. Si no creaste esta cuenta, ignora este correo.",
		}
	case "ResetPassword":
		config = emailConfig{
			Subject:    "Restablece tu contraseña - GoAgent",
			Title:      "Restablecimiento de contraseña",
			Message:    "Recibimos una solicitud para restablecer la contraseña de tu cuenta. Si fuiste tú, haz clic en el botón para crear una nueva contraseña.",
			ButtonText: "Restablecer contraseña",
			ButtonURL:  fmt.Sprintf("%s/auth/setpassword?token=%s&setpasswordmode=resetpassword", env.Envs.MailDestinationWeb, token),
			FooterNote: "Este enlace expira en 60 minutos. Si no solicitaste esto, ignora este correo.",
		}
	default:
		return fmt.Errorf("tipo de email no reconocido: %s", emailType)
	}

	return sendWithTemplate(destination, config)
}

func SendCustomMail(destination, message string) error {
	config := emailConfig{
		Subject:    "Contraseña actualizada - GoAgent",
		Title:      "Contraseña actualizada",
		Message:    message,
		ButtonText: "",
		FooterNote: "Si no realizaste este cambio, contacta a soporte inmediatamente.",
	}

	return sendWithTemplate(destination, config)
}

func sendWithTemplate(destination string, config emailConfig) error {
	client := resend.NewClient(env.Envs.ResendToken)

	params := &resend.SendEmailRequest{
		From:    "GoAgent <notificaciones@goagent.com.mx>",
		To:      []string{destination},
		Subject: config.Subject,
		Html:    buildEmailHTML(config),
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		Log.ErrorMessage(fmt.Sprintf("Error enviando email a %s: %s", destination, err.Error()))
		return err
	}

	return nil
}

func buildEmailHTML(config emailConfig) string {
	buttonSection := ""
	if config.ButtonText != "" {
		buttonSection = fmt.Sprintf(`
			<tr>
				<td align="center" style="padding: 24px 0;">
					<a href="%s"
						style="display: inline-block; background-color: #155dfc; color: #ffffff;
						font-size: 16px; font-weight: 600; text-decoration: none;
						padding: 14px 32px; border-radius: 8px;">
						%s
					</a>
				</td>
			</tr>`,
			config.ButtonURL, config.ButtonText)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; background-color: #f1f5f9; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;">
	<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color: #f1f5f9; padding: 40px 20px;">
		<tr>
			<td align="center">
				<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width: 520px; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.08);">
					<!-- Header -->
					<tr>
						<td style="background-color: #1d293d; padding: 24px; text-align: center;">
							<span style="font-size: 22px; font-weight: 700; color: #ffffff; letter-spacing: 0.5px;">GoAgent</span>
						</td>
					</tr>
					<!-- Body -->
					<tr>
						<td style="padding: 36px 32px;">
							<table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
								<tr>
									<td style="padding-bottom: 16px;">
										<h1 style="margin: 0; font-size: 20px; font-weight: 600; color: #1e293b;">%s</h1>
									</td>
								</tr>
								<tr>
									<td style="padding-bottom: 8px;">
										<p style="margin: 0; font-size: 15px; line-height: 1.6; color: #475569;">%s</p>
									</td>
								</tr>
								%s
								<tr>
									<td style="padding-top: 16px; border-top: 1px solid #e2e8f0;">
										<p style="margin: 0; font-size: 13px; color: #94a3b8; line-height: 1.5;">%s</p>
									</td>
								</tr>
							</table>
						</td>
					</tr>
					<!-- Footer -->
					<tr>
						<td style="background-color: #f8fafc; padding: 20px 32px; text-align: center; border-top: 1px solid #e2e8f0;">
							<p style="margin: 0; font-size: 12px; color: #94a3b8;">GoAgent - Administración de pólizas</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`,
		config.Title, config.Message, buttonSection, config.FooterNote)
}
