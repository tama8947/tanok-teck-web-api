package services

import (
	"context"
	"fmt"
	"html"

	"github.com/tanok/tanok-web-api/internal/models"
)

func (s *Services) SendWebQuestionnaire(ctx context.Context, req models.QuestionnaireRequest) error {
	apiKey := s.Config.ResendAPIKey
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}

	safeEmpresa := html.EscapeString(req.Company)
	safeContacto := html.EscapeString(req.Name)
	safeEmail := html.EscapeString(req.Email)
	safeMensaje := html.EscapeString(req.Message)

	htmlBody := fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 640px; margin: 0 auto; color: #1c1917;">
  <div style="background: #0284c7; padding: 24px 32px; border-radius: 12px 12px 0 0;">
    <h1 style="color: white; margin: 0; font-size: 20px;">Nuevo Cuestionario Web Recibido</h1>
    <p style="color: #bae6fd; margin: 4px 0 0; font-size: 14px;">%s — Tanok Tech</p>
  </div>
  <div style="background: #f8fafc; padding: 24px 32px; border: 1px solid #e2e8f0; border-top: none; border-radius: 0 0 12px 12px;">
    <table style="width: 100%%; border-collapse: collapse; margin-bottom: 24px;">
      <tr><td style="padding: 6px 0; font-weight: bold; color: #64748b; width: 140px;">Empresa</td><td style="padding: 6px 0;">%s</td></tr>
      <tr><td style="padding: 6px 0; font-weight: bold; color: #64748b;">Contacto</td><td style="padding: 6px 0;">%s</td></tr>
      <tr><td style="padding: 6px 0; font-weight: bold; color: #64748b;">Email</td><td style="padding: 6px 0;"><a href="mailto:%s" style="color: #0284c7;">%s</a></td></tr>
    </table>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 16px 0;" />
    <h2 style="font-size: 16px; color: #0c4a6e; margin-bottom: 12px;">Mensaje</h2>
    <p style="font-size: 14px; line-height: 1.6; color: #1c1917;">%s</p>
  </div>
</div>`, safeEmpresa, safeEmpresa, safeContacto, safeEmail, safeEmail, safeMensaje)

	to := "manager@tanok-tech.com"
	if req.Email != "" {
		to = "ortalla_9@hotmail.com"
	}

	return s.sendEmail(ctx, apiKey, to, "",
		fmt.Sprintf("Cuestionario Web — %s", req.Company), htmlBody, false)
}

func (s *Services) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	apiKey := s.Config.ResendAPIKey
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}
	return s.sendEmail(ctx, apiKey, to, "", subject, htmlBody, true)
}

var _ = html.EscapeString
