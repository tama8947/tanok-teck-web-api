package services

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tanok/tanok-web-api/internal/models"
)

func getDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") || strings.Contains(ua, "playbook") || strings.Contains(ua, "silk") {
		return "tablet"
	}
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipod") ||
		strings.Contains(ua, "android") || strings.Contains(ua, "blackberry") || strings.Contains(ua, "windows phone") {
		return "mobile"
	}
	return "desktop"
}

func parseBrowser(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "edg/") {
		return "Edge"
	}
	if strings.Contains(ua, "opr/") || strings.Contains(ua, "opera") {
		return "Opera"
	}
	if strings.Contains(ua, "chrome/") && !strings.Contains(ua, "chromium") {
		return "Chrome"
	}
	if strings.Contains(ua, "firefox/") {
		return "Firefox"
	}
	if strings.Contains(ua, "safari/") && !strings.Contains(ua, "chrome") {
		return "Safari"
	}
	if strings.Contains(ua, "chromium/") {
		return "Chromium"
	}
	return "Unknown"
}

func parseOS(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "windows nt 10") {
		return "Windows 10"
	}
	if strings.Contains(ua, "windows nt 6.3") {
		return "Windows 8.1"
	}
	if strings.Contains(ua, "windows") {
		return "Windows"
	}
	if strings.Contains(ua, "mac os x") {
		return "macOS"
	}
	if strings.Contains(ua, "android") {
		return "Android"
	}
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod") {
		return "iOS"
	}
	if strings.Contains(ua, "linux") {
		return "Linux"
	}
	return "Unknown"
}

func hashVisitorID(ip, userAgent string) string {
	h := md5.Sum([]byte(ip + "-" + userAgent))
	return fmt.Sprintf("%x", h)
}

func (s *Services) TrackEvent(ctx context.Context, req models.TrackEventRequest, visitorID, userAgent, geo string) (*models.TrackResponse, error) {
	if req.Type == "" || req.Path == "" {
		return nil, fmt.Errorf("missing required fields: type, path")
	}

	validTypes := map[models.EventType]bool{
		models.EventPageview: true,
		models.EventClick:    true,
		models.EventScroll:   true,
		models.EventDuration: true,
	}
	if !validTypes[req.Type] {
		return nil, fmt.Errorf("invalid event type: %s", req.Type)
	}

	device := getDeviceType(userAgent)
	browser := parseBrowser(userAgent)
	os := parseOS(userAgent)
	country, city, region := parseGeo(geo)

	metadataStr := "{}"
	if req.Metadata != nil {
		if b, err := json.Marshal(req.Metadata); err == nil {
			metadataStr = string(b)
		}
	}

	isPageview := req.Type == models.EventPageview
	resolvedSessionID := req.SessionID
	isBounce := false

	if isPageview {
		if req.SessionID != "" {
			var existingStartTime sql.NullTime
			var existingPageCount int
			var existingBounce bool
			err := s.DB.QueryRow(ctx,
				`SELECT "startTime", "pageCount", bounce FROM "Session" WHERE id = $1`,
				req.SessionID,
			).Scan(&existingStartTime, &existingPageCount, &existingBounce)

			if err == nil && existingStartTime.Valid {
				durationSec := int(time.Since(existingStartTime.Time).Seconds())
				isBounce = existingPageCount == 1 && durationSec < 10

				_, err := s.DB.Exec(ctx,
					`UPDATE "Session" SET "pageCount" = "pageCount" + 1, "endTime" = NOW(),
					 bounce = $2 WHERE id = $1`,
					req.SessionID, existingBounce || isBounce,
				)
				if err != nil {
					return nil, fmt.Errorf("update session: %w", err)
				}
			}
		} else {
			resolvedSessionID = generateID("sess_")
			_, err := s.DB.Exec(ctx,
				`INSERT INTO "Session" (id, "visitorId", "startTime", "endTime", "pageCount", bounce)
				 VALUES ($1, $2, NOW(), NOW(), 1, false)`,
				resolvedSessionID, visitorID,
			)
			if err != nil {
				return nil, fmt.Errorf("create session: %w", err)
			}
		}
	}

	var element, referrer, utmSource, utmMedium, utmCampaign sql.NullString
	if req.Element != "" {
		element = sql.NullString{String: req.Element, Valid: true}
	}
	if req.Referrer != "" {
		referrer = sql.NullString{String: req.Referrer, Valid: true}
	}
	if req.UtmSource != "" {
		utmSource = sql.NullString{String: req.UtmSource, Valid: true}
	}
	if req.UtmMedium != "" {
		utmMedium = sql.NullString{String: req.UtmMedium, Valid: true}
	}
	if req.UtmCampaign != "" {
		utmCampaign = sql.NullString{String: req.UtmCampaign, Valid: true}
	}

	var sessionID sql.NullString
	if resolvedSessionID != "" {
		sessionID = sql.NullString{String: resolvedSessionID, Valid: true}
	}

	var x, y, scrollDepth sql.NullInt64
	if req.X != nil {
		x = sql.NullInt64{Int64: int64(*req.X), Valid: true}
	}
	if req.Y != nil {
		y = sql.NullInt64{Int64: int64(*req.Y), Valid: true}
	}
	if req.ScrollDepth != nil {
		scrollDepth = sql.NullInt64{Int64: int64(*req.ScrollDepth), Valid: true}
	}

	var countryVal, cityVal, regionVal sql.NullString
	if country != "" && country != "unknown" {
		countryVal = sql.NullString{String: country, Valid: true}
	}
	if city != "" && city != "unknown" {
		cityVal = sql.NullString{String: city, Valid: true}
	}
	if region != "" && region != "unknown" {
		regionVal = sql.NullString{String: region, Valid: true}
	}

	_, err := s.DB.Exec(ctx,
		`INSERT INTO "Event" (id, type, path, "visitorId", "sessionId", country, city, region,
		 element, metadata, "userAgent", referrer, device, browser, os,
		 x, y, "scrollDepth", "utmSource", "utmMedium", "utmCampaign", "createdAt")
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,NOW())`,
		generateID("evt_"), req.Type, req.Path, visitorID, sessionID,
		countryVal, cityVal, regionVal,
		element, metadataStr, userAgent, referrer,
		device, browser, os,
		x, y, scrollDepth,
		utmSource, utmMedium, utmCampaign,
	)
	if err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}

	today := time.Now().UTC().Format("2006-01-02")

	if isPageview {
		s.upsertDailyStats(ctx, today, req.Path, country, true, false, scrollDepth)
	} else if req.Type == models.EventClick {
		s.upsertDailyStats(ctx, today, req.Path, country, false, true, sql.NullInt64{})
	}

	return &models.TrackResponse{Success: true, SessionID: resolvedSessionID}, nil
}

func (s *Services) upsertDailyStats(ctx context.Context, date, path, country string, isPageview, isClick bool, scrollDepth sql.NullInt64) {
	var existingID string
	err := s.DB.QueryRow(ctx,
		`SELECT id FROM "DailyStats" WHERE date = $1 AND path = $2`,
		date, path,
	).Scan(&existingID)

	if err == nil {
		if isPageview {
			s.DB.Exec(ctx,
				`UPDATE "DailyStats" SET pageviews = pageviews + 1, "uniqueViews" = "uniqueViews" + 1
				 WHERE id = $1`, existingID)
		}
		if isClick {
			s.DB.Exec(ctx,
				`UPDATE "DailyStats" SET clicks = clicks + 1 WHERE id = $1`, existingID)
		}
	} else {
		pv := 0
		uv := 0
		clicks := 0
		if isPageview {
			pv = 1
			uv = 1
		}
		if isClick {
			clicks = 1
		}
		var topCountry sql.NullString
		if country != "" && country != "unknown" {
			topCountry = sql.NullString{String: country, Valid: true}
		}
		s.DB.Exec(ctx,
			`INSERT INTO "DailyStats" (id, date, path, pageviews, "uniqueViews", clicks, "avgDuration", "topCountry", "createdAt")
			 VALUES ($1, $2, $3, $4, $5, $6, 0, $7, NOW())`,
			generateID("ds_"), date, path, pv, uv, clicks, topCountry,
		)
	}
}

func parseGeo(geo string) (country, city, region string) {
	var data map[string]string
	if err := json.Unmarshal([]byte(geo), &data); err != nil {
		return "unknown", "unknown", "unknown"
	}
	return data["country"], data["city"], data["region"]
}

func (s *Services) GetStats(ctx context.Context, days int) (*models.StatsResponse, error) {
	if days <= 0 {
		days = 7
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	var pageviews, uniqueVisitors, totalClicks, avgDuration int
	s.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(pageviews),0), COUNT(DISTINCT "visitorId")
		 FROM "Event" WHERE "createdAt" >= $1 AND type = 'pageview'`,
		since,
	).Scan(&pageviews, &uniqueVisitors)

	s.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(clicks),0), COALESCE(AVG("avgDuration"),0)
		 FROM "DailyStats" WHERE date >= $1`,
		since,
	).Scan(&totalClicks, &avgDuration)

	topPages := s.queryTopN(ctx,
		`SELECT path, COUNT(*) as cnt FROM "Event"
		 WHERE "createdAt" >= $1 AND type = 'pageview'
		 GROUP BY path ORDER BY cnt DESC LIMIT 10`, since)
	topCountries := s.queryTopN(ctx,
		`SELECT COALESCE(country,'unknown'), COUNT(*) as cnt FROM "Event"
		 WHERE "createdAt" >= $1 AND type = 'pageview'
		 GROUP BY country ORDER BY cnt DESC LIMIT 10`, since)
	topDevices := s.queryTopN(ctx,
		`SELECT COALESCE(device,'unknown'), COUNT(*) as cnt FROM "Event"
		 WHERE "createdAt" >= $1 AND type = 'pageview'
		 GROUP BY device ORDER BY cnt DESC LIMIT 5`, since)

	stats := &models.StatsResponse{
		Summary: models.StatsSummary{
			TotalPageviews:  pageviews,
			TotalUniqueViews: uniqueVisitors,
			TotalClicks:     totalClicks,
			AvgDuration:     avgDuration,
			Period:          fmt.Sprintf("%dd", days),
		},
		PageviewsByDate: make(map[string]int),
	}

	pageviewsByDate, _ := s.queryDailyPageviews(ctx, since)
	stats.PageviewsByDate = pageviewsByDate

	for _, t := range topPages {
		stats.TopPages = append(stats.TopPages, models.TopPage{Path: t.Key, Pageviews: t.Count})
	}
	for _, t := range topCountries {
		stats.TopCountries = append(stats.TopCountries, models.TopCountry{Country: t.Key, Pageviews: t.Count})
	}
	for _, t := range topDevices {
		stats.DeviceBreakdown = append(stats.DeviceBreakdown, models.TopDevice{Device: t.Key, Pageviews: t.Count})
	}
	return stats, nil
}

type topItem struct {
	Key   string
	Count int
}

func (s *Services) queryTopN(ctx context.Context, query string, args ...interface{}) []topItem {
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var items []topItem
	for rows.Next() {
		var item topItem
		if rows.Scan(&item.Key, &item.Count) == nil {
			items = append(items, item)
		}
	}
	return items
}

func (s *Services) GetEvents(ctx context.Context, limit int, typeFilter, pathFilter string) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT id, type, path, "visitorId", COALESCE(element,''), COALESCE(referrer,''),
	          COALESCE(device,''), COALESCE(browser,''), COALESCE(os,''), "createdAt"::text
	          FROM "Event" WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if typeFilter != "" {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, typeFilter)
		argIdx++
	}
	if pathFilter != "" {
		query += fmt.Sprintf(" AND path ILIKE $%d", argIdx)
		args = append(args, "%"+pathFilter+"%")
		argIdx++
	}
	query += fmt.Sprintf(" ORDER BY \"createdAt\" DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var id, typ, path, visitorID, element, referrer, device, browser, os, createdAt string
		if err := rows.Scan(&id, &typ, &path, &visitorID, &element, &referrer, &device, &browser, &os, &createdAt); err != nil {
			continue
		}
		events = append(events, map[string]interface{}{
			"id": id, "type": typ, "path": path, "visitorId": visitorID,
			"element": element, "referrer": referrer, "device": device,
			"browser": browser, "os": os, "createdAt": createdAt,
		})
	}
	return events, nil
}

func (s *Services) GetTopElements(ctx context.Context, days, limit int) ([]models.TopElementItem, error) {
	if days <= 0 {
		days = 7
	}
	if limit <= 0 {
		limit = 20
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	items := s.queryTopN(ctx,
		`SELECT COALESCE(element,'unknown'), COUNT(*) as cnt FROM "Event"
		 WHERE "createdAt" >= $1 AND type = 'click'
		 GROUP BY element ORDER BY cnt DESC LIMIT $2`,
		since, limit,
	)
	var result []models.TopElementItem
	for _, item := range items {
		result = append(result, models.TopElementItem{Element: item.Key, Clicks: item.Count})
	}
	return result, nil
}

func (s *Services) GetEngagement(ctx context.Context, days int) (*models.EngagementData, error) {
	if days <= 0 {
		days = 7
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	var totalSessions, bounceCount int
	s.DB.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN bounce = true THEN 1 ELSE 0 END),0)
		 FROM "Session" WHERE "startTime" >= $1`,
		since,
	).Scan(&totalSessions, &bounceCount)

	var avgScrollDepth float64
	s.DB.QueryRow(ctx,
		`SELECT COALESCE(AVG("scrollDepth"),0) FROM "Event"
		 WHERE "createdAt" >= $1 AND "scrollDepth" IS NOT NULL`,
		since,
	).Scan(&avgScrollDepth)

	bounceRate := float64(0)
	if totalSessions > 0 {
		bounceRate = float64(bounceCount) / float64(totalSessions) * 100
	}
	engagementScore := float64(0)
	if totalSessions > 0 {
		engagementScore = (100 - bounceRate)*0.4 + (avgScrollDepth)*0.6
		if engagementScore > 100 {
			engagementScore = 100
		}
	}

	return &models.EngagementData{
		BounceRate:      bounceRate,
		AvgScrollDepth:  avgScrollDepth,
		EngagementScore: engagementScore,
		TotalSessions:   totalSessions,
		BounceCount:     bounceCount,
	}, nil
}

func (s *Services) GetHeatmap(ctx context.Context, path string) ([]models.HeatmapPoint, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT x, y, COUNT(*) as cnt FROM "Event"
		 WHERE path = $1 AND type = 'click' AND x IS NOT NULL AND y IS NOT NULL
		 GROUP BY x, y ORDER BY cnt DESC LIMIT 500`,
		path,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []models.HeatmapPoint
	for rows.Next() {
		var p models.HeatmapPoint
		if err := rows.Scan(&p.X, &p.Y, &p.Count); err == nil {
			points = append(points, p)
		}
	}
	return points, nil
}

func (s *Services) GetTrafficSources(ctx context.Context, days int) ([]models.TrafficSourceItem, error) {
	if days <= 0 {
		days = 7
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	rows, err := s.DB.Query(ctx,
		`SELECT COALESCE(referrer,''), COUNT(*) as cnt FROM "Event"
		 WHERE "createdAt" >= $1 AND type = 'pageview'
		 GROUP BY referrer`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	direct := 0
	google := 0
	social := 0
	other := 0
	for rows.Next() {
		var ref string
		var cnt int
		if rows.Scan(&ref, &cnt) != nil {
			continue
		}
		switch {
		case ref == "" || ref == "direct":
			direct += cnt
		case strings.Contains(ref, "google."):
			google += cnt
		case strings.Contains(ref, "facebook.") || strings.Contains(ref, "twitter.") ||
			strings.Contains(ref, "linkedin.") || strings.Contains(ref, "instagram."):
			social += cnt
		default:
			other += cnt
		}
	}

	return []models.TrafficSourceItem{
		{Source: "direct", Count: direct, Label: "Direct"},
		{Source: "google", Count: google, Label: "Google"},
		{Source: "social", Count: social, Label: "Social"},
		{Source: "other", Count: other, Label: "Other"},
	}, nil
}

func (s *Services) AnalyzeWithAI(ctx context.Context, query string) (*models.AnalyzeResponse, error) {
	if s.Config.MiniMaxAPIKey == "" {
		return nil, fmt.Errorf("MINIMAX_API_KEY is not configured")
	}

	systemPrompt := `You are a web analytics assistant for Tanok Tech. Analyze analytics data and provide insights in a concise, professional manner. Use the data provided to answer questions about traffic, user behavior, conversions, and trends. Always respond in Spanish unless the user asks in English.`

	payload := map[string]interface{}{
		"model": "MiniMax-Text-01",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": query},
		},
		"temperature": 0.7,
		"max_tokens":  1024,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.minimax.io/v1/text/chatcompletion_v2", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Config.MiniMaxAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Reply string `json:"reply"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(respBody, &result)

	analysis := result.Reply
	if analysis == "" && len(result.Choices) > 0 {
		analysis = result.Choices[0].Message.Content
	}
	if analysis == "" {
		analysis = "No se pudo generar un análisis en este momento."
	}

	return &models.AnalyzeResponse{Analysis: analysis}, nil
}

func (s *Services) queryDailyPageviews(ctx context.Context, since string) (map[string]int, error) {
	result := make(map[string]int)
	rows, err := s.DB.Query(ctx,
		`SELECT date, SUM(pageviews)::int FROM "DailyStats" WHERE date >= $1 GROUP BY date ORDER BY date`,
		since,
	)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err == nil {
			result[date] = count
		}
	}
	return result, nil
}

var _ = html.EscapeString
