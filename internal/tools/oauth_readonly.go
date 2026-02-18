package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/KafClaw/KafClaw/internal/config"
	"github.com/KafClaw/KafClaw/internal/skills"
)

// GoogleWorkspaceReadTool provides read-only Gmail/Calendar access.
type GoogleWorkspaceReadTool struct{}

// M365ReadTool provides read-only Outlook/Calendar access via Microsoft Graph.
type M365ReadTool struct{}

func NewGoogleWorkspaceReadTool() *GoogleWorkspaceReadTool { return &GoogleWorkspaceReadTool{} }
func NewM365ReadTool() *M365ReadTool                       { return &M365ReadTool{} }

func (t *GoogleWorkspaceReadTool) Name() string { return "google_workspace_read" }
func (t *GoogleWorkspaceReadTool) Tier() int    { return TierHighRisk }

func (t *M365ReadTool) Name() string { return "m365_read" }
func (t *M365ReadTool) Tier() int    { return TierHighRisk }

func (t *GoogleWorkspaceReadTool) Description() string {
	return "Read-only Google Workspace operations (Gmail and Calendar) using enrolled OAuth token."
}

func (t *M365ReadTool) Description() string {
	return "Read-only Microsoft 365 operations (mail and calendar) using enrolled OAuth token."
}

func (t *GoogleWorkspaceReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "Operation to run: gmail_list_messages | calendar_list_events | drive_list_files",
			},
			"profile": map[string]any{
				"type":        "string",
				"description": "OAuth profile name (default: default)",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of records to return (default 10, max 50)",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Optional Gmail search query (q=...)",
			},
			"calendar_id": map[string]any{
				"type":        "string",
				"description": "Calendar id (default: primary)",
			},
			"time_min": map[string]any{
				"type":        "string",
				"description": "Optional RFC3339 lower bound for calendar events",
			},
			"time_max": map[string]any{
				"type":        "string",
				"description": "Optional RFC3339 upper bound for calendar events",
			},
			"drive_query": map[string]any{
				"type":        "string",
				"description": "Optional Google Drive q= filter (defaults to non-trashed files)",
			},
		},
		"required": []string{"operation"},
	}
}

func (t *M365ReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "Operation to run: mail_list_messages | calendar_list_events | onedrive_list_children",
			},
			"profile": map[string]any{
				"type":        "string",
				"description": "OAuth profile name (default: default)",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of records to return (default 10, max 50)",
			},
			"select": map[string]any{
				"type":        "string",
				"description": "Optional Graph $select projection fields",
			},
			"time_min": map[string]any{
				"type":        "string",
				"description": "Optional RFC3339 lower bound for calendar events",
			},
			"time_max": map[string]any{
				"type":        "string",
				"description": "Optional RFC3339 upper bound for calendar events",
			},
			"drive_item_id": map[string]any{
				"type":        "string",
				"description": "Optional OneDrive item id (default: root)",
			},
		},
		"required": []string{"operation"},
	}
}

func (t *GoogleWorkspaceReadTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	if err := ensureSkillEnabled("google-workspace"); err != nil {
		return err.Error(), nil
	}
	op := strings.ToLower(strings.TrimSpace(GetString(params, "operation", "")))
	if op == "" {
		return "Error: operation is required", nil
	}
	profile := strings.TrimSpace(GetString(params, "profile", "default"))
	if profile == "" {
		profile = "default"
	}
	token, err := skills.GetOAuthAccessToken(skills.ProviderGoogleWorkspace, profile)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	maxResults := clamp(GetInt(params, "max_results", 10), 1, 50)
	client := &http.Client{Timeout: 30 * time.Second}
	switch op {
	case "gmail_list_messages":
		if !scopeHasAny(token.Scope, "https://www.googleapis.com/auth/gmail.readonly", "https://www.googleapis.com/auth/gmail.modify", "https://mail.google.com/") {
			return "Error: oauth scope missing Gmail read access; re-enroll with gmail.readonly or stronger scope", nil
		}
		q := strings.TrimSpace(GetString(params, "query", ""))
		u := "https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=" + strconv.Itoa(maxResults)
		if q != "" {
			u += "&q=" + url.QueryEscape(q)
		}
		return oauthGETJSON(ctx, client, u, token.AccessToken)
	case "calendar_list_events":
		if !scopeHasAny(token.Scope, "https://www.googleapis.com/auth/calendar.readonly", "https://www.googleapis.com/auth/calendar") {
			return "Error: oauth scope missing Calendar read access; re-enroll with calendar.readonly or stronger scope", nil
		}
		calendarID := strings.TrimSpace(GetString(params, "calendar_id", "primary"))
		if calendarID == "" {
			calendarID = "primary"
		}
		query := url.Values{}
		query.Set("singleEvents", "true")
		query.Set("orderBy", "startTime")
		query.Set("maxResults", strconv.Itoa(maxResults))
		if v := strings.TrimSpace(GetString(params, "time_min", "")); v != "" {
			query.Set("timeMin", v)
		}
		if v := strings.TrimSpace(GetString(params, "time_max", "")); v != "" {
			query.Set("timeMax", v)
		}
		u := "https://www.googleapis.com/calendar/v3/calendars/" + url.PathEscape(calendarID) + "/events?" + query.Encode()
		return oauthGETJSON(ctx, client, u, token.AccessToken)
	case "drive_list_files":
		if !scopeHasAny(token.Scope,
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/drive.metadata.readonly",
			"https://www.googleapis.com/auth/drive") {
			return "Error: oauth scope missing Drive read access; re-enroll with drive.readonly or drive.metadata.readonly", nil
		}
		query := url.Values{}
		query.Set("pageSize", strconv.Itoa(maxResults))
		query.Set("fields", "nextPageToken,files(id,name,mimeType,modifiedTime,webViewLink,size)")
		driveQ := strings.TrimSpace(GetString(params, "drive_query", "trashed=false"))
		if driveQ != "" {
			query.Set("q", driveQ)
		}
		u := "https://www.googleapis.com/drive/v3/files?" + query.Encode()
		return oauthGETJSON(ctx, client, u, token.AccessToken)
	default:
		return "Error: unsupported operation; use gmail_list_messages, calendar_list_events, or drive_list_files", nil
	}
}

func (t *M365ReadTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	if err := ensureSkillEnabled("m365"); err != nil {
		return err.Error(), nil
	}
	op := strings.ToLower(strings.TrimSpace(GetString(params, "operation", "")))
	if op == "" {
		return "Error: operation is required", nil
	}
	profile := strings.TrimSpace(GetString(params, "profile", "default"))
	if profile == "" {
		profile = "default"
	}
	token, err := skills.GetOAuthAccessToken(skills.ProviderM365, profile)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	maxResults := clamp(GetInt(params, "max_results", 10), 1, 50)
	client := &http.Client{Timeout: 30 * time.Second}
	switch op {
	case "mail_list_messages":
		if !scopeHasAny(token.Scope, "Mail.Read", "https://graph.microsoft.com/Mail.Read") {
			return "Error: oauth scope missing Mail.Read access; re-enroll with Mail.Read", nil
		}
		query := url.Values{}
		query.Set("$top", strconv.Itoa(maxResults))
		selectFields := strings.TrimSpace(GetString(params, "select", "id,subject,from,receivedDateTime,isRead"))
		if selectFields != "" {
			query.Set("$select", selectFields)
		}
		u := "https://graph.microsoft.com/v1.0/me/messages?" + query.Encode()
		return oauthGETJSON(ctx, client, u, token.AccessToken)
	case "calendar_list_events":
		if !scopeHasAny(token.Scope, "Calendars.Read", "https://graph.microsoft.com/Calendars.Read") {
			return "Error: oauth scope missing Calendars.Read access; re-enroll with Calendars.Read", nil
		}
		query := url.Values{}
		query.Set("$top", strconv.Itoa(maxResults))
		query.Set("$orderby", "start/dateTime")
		selectFields := strings.TrimSpace(GetString(params, "select", "id,subject,start,end,organizer"))
		if selectFields != "" {
			query.Set("$select", selectFields)
		}
		if v := strings.TrimSpace(GetString(params, "time_min", "")); v != "" {
			query.Set("$filter", "start/dateTime ge '"+v+"'")
		}
		u := "https://graph.microsoft.com/v1.0/me/events?" + query.Encode()
		return oauthGETJSON(ctx, client, u, token.AccessToken)
	case "onedrive_list_children":
		if !scopeHasAny(token.Scope, "Files.Read", "https://graph.microsoft.com/Files.Read") {
			return "Error: oauth scope missing Files.Read access; re-enroll with Files.Read", nil
		}
		itemID := strings.TrimSpace(GetString(params, "drive_item_id", ""))
		query := url.Values{}
		query.Set("$top", strconv.Itoa(maxResults))
		query.Set("$select", "id,name,folder,file,size,lastModifiedDateTime,webUrl")
		u := "https://graph.microsoft.com/v1.0/me/drive/root/children?" + query.Encode()
		if itemID != "" && !strings.EqualFold(itemID, "root") {
			u = "https://graph.microsoft.com/v1.0/me/drive/items/" + url.PathEscape(itemID) + "/children?" + query.Encode()
		}
		return oauthGETJSON(ctx, client, u, token.AccessToken)
	default:
		return "Error: unsupported operation; use mail_list_messages, calendar_list_events, or onedrive_list_children", nil
	}
}

func ensureSkillEnabled(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if !skills.EffectiveSkillEnabled(cfg, name) {
		return fmt.Errorf("skill %q is disabled; run `kafclaw skills enable-skill %s`", name, name)
	}
	return nil
}

func oauthGETJSON(ctx context.Context, client *http.Client, rawURL string, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("Error: provider API status %d: %s", resp.StatusCode, strings.TrimSpace(string(body))), nil
	}
	var pretty any
	if err := json.Unmarshal(body, &pretty); err != nil {
		return string(body), nil
	}
	out, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return string(body), nil
	}
	return string(out), nil
}

func scopeHasAny(scopeRaw string, accepted ...string) bool {
	if len(accepted) == 0 {
		return true
	}
	scopes := strings.Fields(strings.TrimSpace(scopeRaw))
	if len(scopes) == 0 {
		return false
	}
	have := map[string]struct{}{}
	for _, s := range scopes {
		have[strings.ToLower(strings.TrimSpace(s))] = struct{}{}
	}
	for _, a := range accepted {
		if _, ok := have[strings.ToLower(strings.TrimSpace(a))]; ok {
			return true
		}
	}
	return false
}

func clamp(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
