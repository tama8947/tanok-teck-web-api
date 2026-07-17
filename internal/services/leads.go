package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tanok/tanok-web-api/internal/models"
)

func generateID(prefix string) string {
	b := make([]byte, 12)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func (s *Services) CaptureLead(ctx context.Context, req models.LeadCaptureRequest) (*models.LeadCaptureResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, fmt.Errorf("invalid email format")
	}

	if req.LeadMagnetID != "" {
		var magnetExists bool
		err := s.DB.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM "LeadMagnet" WHERE id = $1)`,
			req.LeadMagnetID,
		).Scan(&magnetExists)
		if err != nil || !magnetExists {
			return nil, fmt.Errorf("leadMagnetId not found")
		}
	}

	var existingID string
	err := s.DB.QueryRow(ctx,
		`SELECT id FROM "Lead" WHERE email = $1 LIMIT 1`,
		email,
	).Scan(&existingID)
	if err == nil {
		s.DB.Exec(ctx,
			`INSERT INTO "LeadEvent" (id, "leadId", type, referrer, "postId", "createdAt")
			 VALUES ($1, $2, 'repeat_capture', $3, NULL, NOW())`,
			generateID("le_"), existingID, sql.NullString{String: req.Referrer, Valid: req.Referrer != ""},
		)
		return &models.LeadCaptureResponse{ID: existingID, Status: "existing"}, nil
	}

	leadID := generateID("lead_")
	var company, referrer sql.NullString
	if req.Company != "" {
		company = sql.NullString{String: req.Company, Valid: true}
	}
	if req.Referrer != "" {
		referrer = sql.NullString{String: req.Referrer, Valid: true}
	}
	var magnetID sql.NullString
	if req.LeadMagnetID != "" {
		magnetID = sql.NullString{String: req.LeadMagnetID, Valid: true}
	}

	_, err = s.DB.Exec(ctx,
		`INSERT INTO "Lead" (id, name, email, company, referrer, "leadMagnetId", "createdAt")
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		leadID, name, email, company, referrer, magnetID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert lead: %w", err)
	}

	_, err = s.DB.Exec(ctx,
		`INSERT INTO "LeadEvent" (id, "leadId", type, referrer, "postId", "createdAt")
		 VALUES ($1, $2, 'lead_captured', $3, NULL, NOW())`,
		generateID("le_"), leadID, referrer,
	)
	if err != nil {
		return nil, fmt.Errorf("insert lead event: %w", err)
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.SendLeadConfirmation(bgCtx, email, name); err != nil {
			fmt.Printf("[leads] confirmation email failed: %v\n", err)
		}
	}()

	return &models.LeadCaptureResponse{ID: leadID, Status: "created"}, nil
}

func (s *Services) ListLeads(ctx context.Context, page, limit int) (*models.LeadListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM "Lead"`).Scan(&total)

	rows, err := s.DB.Query(ctx,
		`SELECT l.id, l.name, l.email, COALESCE(l.company, ''), COALESCE(l.referrer, ''),
		        COALESCE(lm.id, ''), COALESCE(lm.name, ''), COALESCE(lm.slug, ''),
		        l."createdAt"::text
		 FROM "Lead" l
		 LEFT JOIN "LeadMagnet" lm ON l."leadMagnetId" = lm.id
		 ORDER BY l."createdAt" DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query leads: %w", err)
	}
	defer rows.Close()

	var leads []models.LeadItem
	var leadIDs []string
	for rows.Next() {
		var item models.LeadItem
		var magnetID, magnetName, magnetSlug string
		if err := rows.Scan(&item.ID, &item.Name, &item.Email, &item.Company, &item.Referrer,
			&magnetID, &magnetName, &magnetSlug, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}
		if magnetID != "" {
			item.Magnet = &models.MagnetInfo{ID: magnetID, Name: magnetName, Slug: magnetSlug}
		}
		leads = append(leads, item)
		leadIDs = append(leadIDs, item.ID)
	}

	if len(leadIDs) > 0 {
		eventCounts := s.batchEventCounts(ctx, leadIDs)
		for i := range leads {
			leads[i].EventCount = eventCounts[leads[i].ID]
		}
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &models.LeadListResponse{
		Leads: leads,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *Services) GetLeadByID(ctx context.Context, id string) (*models.LeadDetail, error) {
	var detail models.LeadDetail
	var magnetID, magnetName, magnetSlug sql.NullString

	err := s.DB.QueryRow(ctx,
		`SELECT l.id, l.name, l.email, COALESCE(l.company, ''), COALESCE(l.referrer, ''),
		        lm.id, lm.name, lm.slug, l."createdAt"::text
		 FROM "Lead" l
		 LEFT JOIN "LeadMagnet" lm ON l."leadMagnetId" = lm.id
		 WHERE l.id = $1`,
		id,
	).Scan(&detail.ID, &detail.Name, &detail.Email, &detail.Company, &detail.Referrer,
		&magnetID, &magnetName, &magnetSlug, &detail.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("lead not found: %w", err)
	}

	if magnetID.Valid {
		detail.Magnet = &models.MagnetInfo{ID: magnetID.String, Name: magnetName.String, Slug: magnetSlug.String}
	}

	eventRows, err := s.DB.Query(ctx,
		`SELECT id, type, COALESCE("postId", ''), COALESCE(referrer, ''),
		        COALESCE(metadata, ''), "createdAt"::text
		 FROM "LeadEvent" WHERE "leadId" = $1 ORDER BY "createdAt" DESC`,
		id,
	)
	if err == nil {
		defer eventRows.Close()
		for eventRows.Next() {
			var evt models.LeadEventItem
			if err := eventRows.Scan(&evt.ID, &evt.Type, &evt.PostID, &evt.Referrer,
				&evt.Metadata, &evt.CreatedAt); err != nil {
				continue
			}
			detail.Events = append(detail.Events, evt)
		}
	}

	detail.EventCount = len(detail.Events)
	return &detail, nil
}

func (s *Services) GetMagnetBySlug(ctx context.Context, slug string) (*models.MagnetInfo, error) {
	var magnet models.MagnetInfo
	err := s.DB.QueryRow(ctx,
		`SELECT id, name, slug FROM "LeadMagnet" WHERE slug = $1`,
		slug,
	).Scan(&magnet.ID, &magnet.Name, &magnet.Slug)
	if err != nil {
		return nil, fmt.Errorf("magnet not found: %w", err)
	}
	return &magnet, nil
}

func (s *Services) batchEventCounts(ctx context.Context, leadIDs []string) map[string]int {
	counts := make(map[string]int)
	for _, id := range leadIDs {
		var c int
		if err := s.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM "LeadEvent" WHERE "leadId" = $1`, id,
		).Scan(&c); err == nil {
			counts[id] = c
		}
	}
	return counts
}

func (s *Services) SendLeadConfirmation(ctx context.Context, email, name string) error {
	apiKey := s.Config.ResendAPIKey
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}

	safeName := html.EscapeString(name)
	htmlBody := fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 640px; margin: 0 auto; color: #1c1917;">
  <div style="background: #0284c7; padding: 24px 32px; border-radius: 12px 12px 0 0;">
    <h1 style="color: white; margin: 0; font-size: 20px;">
      ¡Gracias, %s! / Thank you, %s!
    </h1>
    <p style="color: #bae6fd; margin: 4px 0 0; font-size: 14px;">
      Tanok Tech — AI Readiness Checklist
    </p>
  </div>
  <div style="background: #f8fafc; padding: 24px 32px; border: 1px solid #e2e8f0; border-top: none; border-radius: 0 0 12px 12px;">
    <p style="font-size: 15px; line-height: 1.6;">
      Hemos recibido tu interés en el <strong>Checklist de Preparación para IA</strong>.
      Pronto nos pondremos en contacto para hacer un diagnóstico personalizado de tu empresa.
    </p>
    <p style="font-size: 15px; line-height: 1.6;">
      We've received your interest in the <strong>AI Readiness Checklist</strong>.
      We'll get in touch soon to conduct a personalized diagnostic of your company.
    </p>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 20px 0;" />
    <p style="font-size: 13px; color: #64748b;">
      Tanok Tech — Software Development & AI Consulting<br />
      <a href="https://tanok-tech.com" style="color: #0284c7;">tanok-tech.com</a>
    </p>
  </div>
</div>`, safeName, safeName)

	return s.sendEmail(ctx, apiKey, email, "manager@tanok-tech.com",
		"¡Gracias por tu interés! / Thanks for your interest! — Tanok Tech", htmlBody, true)
}

func (s *Services) sendEmail(ctx context.Context, apiKey, to, bcc, subject, htmlBody string, withRetry bool) error {
	payload := map[string]interface{}{
		"from":    "Tanok Tech <manager@tanok-tech.com>",
		"to":      []string{to},
		"subject": subject,
		"html":    htmlBody,
	}
	if bcc != "" {
		payload["bcc"] = []string{bcc}
	}

	body, _ := json.Marshal(payload)

	err := doResendRequest(ctx, apiKey, body)
	if err != nil && withRetry {
		var apiErr *resendError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
			time.Sleep(1 * time.Second)
			return doResendRequest(ctx, apiKey, body)
		}
	}
	return err
}

type resendError struct {
	StatusCode int
	Message    string
}

func (e *resendError) Error() string {
	return fmt.Sprintf("resend error (status %d): %s", e.StatusCode, e.Message)
}

func doResendRequest(ctx context.Context, apiKey string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return &resendError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	var result struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(respBody, &result)
	if result.Error != nil {
		return fmt.Errorf("resend error: %s", result.Error.Message)
	}
	return nil
}

func init() {
	_ = generateID
}

var _ = sql.NullString{}
