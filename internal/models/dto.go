package models

import "time"

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Success bool        `json:"success"`
	User    *UserPublic `json:"user"`
}

type UserPublic struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type AuthResponse struct {
	User *UserPublic `json:"user"`
}

type PostSummary struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	Slug                string `json:"slug"`
	Excerpt             string `json:"excerpt"`
	CoverImage          string `json:"coverImage"`
	PublishedAt         string `json:"publishedAt"`
	Locale              string `json:"locale"`
	MetaTitle           string `json:"metaTitle"`
	MetaDescription     string `json:"metaDescription"`
	TranslationGroupID  string `json:"translationGroupId"`
}

type PostDetail struct {
	PostSummary
	Content      string `json:"content"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	OgImage      string `json:"ogImage"`
	GeneratedBy  string `json:"generatedBy"`
	SourceURL    string `json:"sourceUrl"`
	SourceName   string `json:"sourceName"`
	AuthorID     string `json:"authorId"`
}

type PostListResponse struct {
	Posts   []PostSummary `json:"posts"`
	Page    int           `json:"page"`
	Limit   int           `json:"limit"`
	Total   int64         `json:"total"`
	HasMore bool          `json:"hasMore"`
}

type PostResponse struct {
	Post *PostDetail `json:"post"`
}

type SiblingResponse struct {
	Slug   string `json:"slug"`
	Locale string `json:"locale"`
}

type EventType string

const (
	EventPageview EventType = "pageview"
	EventClick    EventType = "click"
	EventScroll   EventType = "scroll"
	EventDuration EventType = "duration"
)

type TrackEventRequest struct {
	Type        EventType              `json:"type" binding:"required"`
	Path        string                 `json:"path" binding:"required"`
	Metadata    map[string]interface{} `json:"metadata"`
	Element     string                 `json:"element"`
	Referrer    string                 `json:"referrer"`
	X           *int                   `json:"x"`
	Y           *int                   `json:"y"`
	ScrollDepth *int                   `json:"scrollDepth"`
	SessionID   string                 `json:"sessionId"`
	UtmSource   string                 `json:"utmSource"`
	UtmMedium   string                 `json:"utmMedium"`
	UtmCampaign string                 `json:"utmCampaign"`
}

type TrackResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"sessionId,omitempty"`
}

type LeadCaptureRequest struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required"`
	Company      string `json:"company"`
	Referrer     string `json:"referrer"`
	LeadMagnetID string `json:"leadMagnetId"`
}

type LeadCaptureResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type LeadItem struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	Company    string       `json:"company"`
	Referrer   string       `json:"referrer"`
	Magnet     *MagnetInfo  `json:"magnet"`
	EventCount int          `json:"eventCount"`
	CreatedAt  string       `json:"createdAt"`
}

type MagnetInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type LeadListResponse struct {
	Leads      []LeadItem   `json:"leads"`
	Pagination Pagination   `json:"pagination"`
}

type LeadDetail struct {
	LeadItem
	Events []LeadEventItem `json:"events"`
}

type LeadEventItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	PostID    string `json:"postId"`
	Referrer  string `json:"referrer"`
	Metadata  string `json:"metadata"`
	CreatedAt string `json:"createdAt"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

type StatsResponse struct {
	Summary         StatsSummary      `json:"summary"`
	PageviewsByDate map[string]int    `json:"pageviewsByDate"`
	TopPages        []TopPage         `json:"topPages"`
	TopCountries    []TopCountry      `json:"topCountries"`
	DeviceBreakdown []TopDevice       `json:"deviceBreakdown"`
}

type StatsSummary struct {
	TotalPageviews  int    `json:"totalPageviews"`
	TotalUniqueViews int   `json:"totalUniqueViews"`
	TotalClicks     int    `json:"totalClicks"`
	AvgDuration     int    `json:"avgDuration"`
	Period          string `json:"period"`
}

type TopPage struct {
	Path      string `json:"path"`
	Pageviews int    `json:"pageviews"`
}

type TopCountry struct {
	Country   string `json:"country"`
	Pageviews int    `json:"pageviews"`
}

type TopDevice struct {
	Device    string `json:"device"`
	Pageviews int    `json:"pageviews"`
}

type TrafficSourceItem struct {
	Source    string `json:"source"`
	Count     int    `json:"count"`
	Label     string `json:"label"`
}

type TopElementItem struct {
	Element string `json:"element"`
	Clicks  int    `json:"clicks"`
}

type EngagementData struct {
	BounceRate       float64 `json:"bounceRate"`
	AvgScrollDepth   float64 `json:"avgScrollDepth"`
	EngagementScore  float64 `json:"engagementScore"`
	TotalSessions    int     `json:"totalSessions"`
	BounceCount      int     `json:"bounceCount"`
}

type HeatmapPoint struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Count int `json:"count"`
}

type AnalyzeRequest struct {
	Query string `json:"query" binding:"required"`
}

type AnalyzeResponse struct {
	Analysis string `json:"analysis"`
}

type QuestionnaireRequest struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required"`
	Company string `json:"company"`
	Message string `json:"message"`
}

type CronResponse struct {
	Status  string `json:"status"`
	PostID  string `json:"postId,omitempty"`
	Slot    string `json:"slot"`
	Message string `json:"message,omitempty"`
}

type RateLimitResponse struct {
	Error      string `json:"error"`
	RetryAfter int    `json:"retryAfter"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type PostStatus string

const (
	PostStatusDraft     PostStatus = "DRAFT"
	PostStatusPublished PostStatus = "PUBLISHED"
	PostStatusArchived  PostStatus = "ARCHIVED"
)

type Locale string

const (
	LocaleES Locale = "es"
	LocaleEN Locale = "en"
)

// DB-compatible Post struct (used only by repository/service layer)
type PostRow struct {
	ID                  string
	Title               string
	Slug                string
	Excerpt             *string
	Content             string
	CoverImage          *string
	Status              string
	PublishedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	AuthorID            string
	Locale              string
	TranslationGroupID  *string
	MetaTitle           *string
	MetaDescription     *string
	OgImage             *string
	GeneratedBy         *string
	GenerationPrompt    *string
	IndexedAt           *time.Time
	IndexNowSubmittedAt *time.Time
	SourceURL           *string
	SourceName          *string
	LinkedinSnippet     *string
	LinkedinHashtags    []string
}

type UserRow struct {
	ID          string
	Email       string
	Password    string
	Name        *string
	Permissions string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
