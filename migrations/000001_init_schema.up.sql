CREATE TABLE IF NOT EXISTS "User" (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    password    TEXT NOT NULL,
    name        TEXT,
    permissions TEXT NOT NULL DEFAULT '{"dashboard": true}',
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "Event" (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    type         TEXT NOT NULL,
    path         TEXT NOT NULL,
    "visitorId"  TEXT NOT NULL,
    "sessionId"  TEXT,
    country      TEXT,
    city         TEXT,
    region       TEXT,
    element      TEXT,
    metadata     TEXT NOT NULL DEFAULT '{}',
    "userAgent"  TEXT,
    referrer     TEXT,
    device       TEXT,
    browser      TEXT,
    os           TEXT,
    x            INTEGER,
    y            INTEGER,
    "scrollDepth" INTEGER,
    "utmSource"  TEXT,
    "utmMedium"  TEXT,
    "utmCampaign" TEXT,
    "createdAt"  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON "Event" ("createdAt");
CREATE INDEX IF NOT EXISTS idx_events_type ON "Event" (type);
CREATE INDEX IF NOT EXISTS idx_events_path ON "Event" (path);
CREATE INDEX IF NOT EXISTS idx_events_visitor_id ON "Event" ("visitorId");
CREATE INDEX IF NOT EXISTS idx_events_session_id ON "Event" ("sessionId");

CREATE TABLE IF NOT EXISTS "DailyStats" (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    date        TEXT NOT NULL,
    path        TEXT NOT NULL,
    pageviews   INTEGER NOT NULL DEFAULT 0,
    "uniqueViews" INTEGER NOT NULL DEFAULT 0,
    clicks      INTEGER NOT NULL DEFAULT 0,
    "avgDuration" INTEGER NOT NULL DEFAULT 0,
    "avgScrollDepth" INTEGER NOT NULL DEFAULT 0,
    "bounceCount"    INTEGER NOT NULL DEFAULT 0,
    "totalSessions" INTEGER NOT NULL DEFAULT 0,
    "scrollDepthBreakdown" TEXT NOT NULL DEFAULT '{}',
    "topCountry" TEXT,
    "createdAt"  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT date_path UNIQUE (date, path)
);

CREATE TABLE IF NOT EXISTS "Session" (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    "visitorId" TEXT NOT NULL,
    "startTime" TIMESTAMPTZ NOT NULL,
    "endTime"   TIMESTAMPTZ,
    "pageCount" INTEGER NOT NULL DEFAULT 1,
    bounce      BOOLEAN NOT NULL DEFAULT false,
    country     TEXT,
    device      TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_visitor_id ON "Session" ("visitorId");
CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON "Session" ("startTime");

CREATE TABLE IF NOT EXISTS "Post" (
    id                  TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    title               TEXT NOT NULL,
    slug                TEXT NOT NULL,
    excerpt             TEXT,
    content             TEXT NOT NULL,
    "coverImage"        TEXT,
    status              TEXT NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED')),
    "publishedAt"       TIMESTAMPTZ,
    "createdAt"         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt"         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "authorId"          TEXT NOT NULL REFERENCES "User"(id),
    locale              TEXT NOT NULL DEFAULT 'es',
    "translationGroupId" TEXT,
    "metaTitle"         TEXT,
    "metaDescription"   TEXT,
    "ogImage"           TEXT,
    "generatedBy"       TEXT,
    "generationPrompt"  TEXT,
    "indexedAt"         TIMESTAMPTZ,
    "indexNowSubmittedAt" TIMESTAMPTZ,
    "sourceUrl"         TEXT,
    "sourceName"        TEXT,
    "linkedinSnippet"   TEXT,
    "linkedinHashtags"  TEXT[] DEFAULT '{}',
    UNIQUE(slug, locale)
);
CREATE INDEX IF NOT EXISTS idx_posts_locale_status ON "Post" (locale, status, "publishedAt");
CREATE INDEX IF NOT EXISTS idx_posts_translation_group ON "Post" ("translationGroupId");

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW."updatedAt" = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
DROP TRIGGER IF EXISTS update_posts_updated_at ON "Post";
CREATE TRIGGER update_posts_updated_at BEFORE UPDATE ON "Post"
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS "CronRun" (
    id             TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    slot           TEXT NOT NULL UNIQUE,
    status         TEXT NOT NULL DEFAULT 'running',
    "postId"       TEXT,
    error          TEXT,
    "agentPipeline" JSONB,
    "createdAt"    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cron_runs_slot ON "CronRun" (slot);

CREATE TABLE IF NOT EXISTS "CronTopic" (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    "esText"    TEXT NOT NULL,
    "enText"    TEXT NOT NULL,
    "searchQuery" TEXT,
    "usedAt"    TIMESTAMPTZ,
    "useCount"  INTEGER NOT NULL DEFAULT 0,
    "isActive"  BOOLEAN NOT NULL DEFAULT true,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cron_topics_used_at ON "CronTopic" ("usedAt", "isActive");
CREATE INDEX IF NOT EXISTS idx_cron_topics_active ON "CronTopic" ("isActive", "useCount");

CREATE TABLE IF NOT EXISTS "PostLink" (
    id             TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    "sourcePostId" TEXT NOT NULL,
    "targetPostId" TEXT NOT NULL,
    score          DOUBLE PRECISION NOT NULL DEFAULT 0,
    "createdAt"    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE("sourcePostId", "targetPostId")
);
CREATE INDEX IF NOT EXISTS idx_post_links_source ON "PostLink" ("sourcePostId");
CREATE INDEX IF NOT EXISTS idx_post_links_target ON "PostLink" ("targetPostId");

CREATE TABLE IF NOT EXISTS "Lead" (
    id             TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL,
    email          TEXT NOT NULL,
    company        TEXT,
    referrer       TEXT,
    "leadMagnetId" TEXT REFERENCES "LeadMagnet"(id),
    "createdAt"    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_leads_email ON "Lead" (email);
CREATE INDEX IF NOT EXISTS idx_leads_created_at ON "Lead" ("createdAt");

CREATE TABLE IF NOT EXISTS "LeadMagnet" (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "LeadEvent" (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    "postId"    TEXT,
    "leadId"    TEXT REFERENCES "Lead"(id) ON DELETE SET NULL,
    type        TEXT NOT NULL DEFAULT 'cta_click',
    referrer    TEXT,
    metadata    TEXT,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_lead_events_post_type ON "LeadEvent" ("postId", type);
CREATE INDEX IF NOT EXISTS idx_lead_events_lead ON "LeadEvent" ("leadId");
