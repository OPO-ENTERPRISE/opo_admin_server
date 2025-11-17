package services

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"strconv"

	"opo_admin_server/internal/config"
)

// EmailService maneja el envío de emails
type EmailService struct {
	cfg config.Config
}

// NewEmailService crea una nueva instancia del servicio de email
func NewEmailService(cfg config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendDeactivationEmail envía un email de confirmación para darse de baja
func (s *EmailService) SendDeactivationEmail(email, token string) error {
	if s.cfg.SMTPHost == "" || s.cfg.SMTPUser == "" || s.cfg.SMTPPassword == "" {
		log.Printf("⚠️ SMTP no configurado, simulando envío de email a %s", email)
		log.Printf("📧 Token de desactivación: %s", token)
		return nil
	}

	// Construir URL de confirmación
	confirmURL := fmt.Sprintf("%s%s/users/deactivate-confirm?token=%s", s.cfg.AppBaseURL, s.cfg.APIBasePath, token)

	// Renderizar template HTML del email
	emailHTML, err := s.renderDeactivationEmailTemplate(confirmURL)
	if err != nil {
		return fmt.Errorf("error renderizando template: %v", err)
	}

	// Configurar mensaje
	from := s.cfg.SMTPFrom
	if from == "" {
		from = s.cfg.SMTPUser
	}

	to := []string{email}
	subject := "Confirmación de baja de cuenta"
	body := emailHTML

	// Construir mensaje MIME
	message := fmt.Sprintf("From: %s\r\n", from)
	message += fmt.Sprintf("To: %s\r\n", email)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += body

	// Conectar y enviar
	port, err := strconv.Atoi(s.cfg.SMTPPort)
	if err != nil {
		return fmt.Errorf("puerto SMTP inválido: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, port)
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	err = smtp.SendMail(addr, auth, from, to, []byte(message))
	if err != nil {
		return fmt.Errorf("error enviando email: %v", err)
	}

	log.Printf("✅ Email de desactivación enviado a %s", email)
	return nil
}

// renderDeactivationEmailTemplate renderiza el template HTML del email
func (s *EmailService) renderDeactivationEmailTemplate(confirmURL string) (string, error) {
	tmpl := `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Confirmación de baja</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .header h1 {
            color: #2c3e50;
            margin: 0;
        }
        .content {
            margin-bottom: 30px;
        }
        .content p {
            margin-bottom: 15px;
        }
        .button-container {
            text-align: center;
            margin: 30px 0;
        }
        .button {
            display: inline-block;
            padding: 12px 30px;
            background-color: #e74c3c;
            color: #ffffff;
            text-decoration: none;
            border-radius: 5px;
            font-weight: bold;
            transition: background-color 0.3s;
        }
        .button:hover {
            background-color: #c0392b;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            font-size: 12px;
            color: #777;
            text-align: center;
        }
        .warning {
            background-color: #fff3cd;
            border-left: 4px solid #ffc107;
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Confirmación de baja de cuenta</h1>
        </div>
        <div class="content">
            <p>Hemos recibido una solicitud para darte de baja de nuestra plataforma.</p>
            <p>Si has sido tú quien ha solicitado la baja, por favor haz clic en el siguiente botón para confirmar:</p>
            
            <div class="button-container">
                <a href="{{.ConfirmURL}}" class="button">Confirmar baja de cuenta</a>
            </div>
            
            <div class="warning">
                <strong>⚠️ Advertencia:</strong> Esta acción desactivará tu cuenta permanentemente. 
                Si no has solicitado esta baja, puedes ignorar este email.
            </div>
            
            <p>Si el botón no funciona, copia y pega el siguiente enlace en tu navegador:</p>
            <p style="word-break: break-all; color: #3498db;">{{.ConfirmURL}}</p>
            
            <p><strong>Este enlace expirará en 24 horas.</strong></p>
        </div>
        <div class="footer">
            <p>Este es un email automático, por favor no respondas a este mensaje.</p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]string{
		"ConfirmURL": confirmURL,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

