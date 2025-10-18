package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	botToken string
	client   *http.Client
	baseURL  string
}

type SendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

type SendMessageResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	Result      struct {
		MessageID int `json:"message_id"`
	} `json:"result,omitempty"`
}

type GetUpdatesResponse struct {
	OK     bool `json:"ok"`
	Result []struct {
		UpdateID int `json:"update_id"`
		Message  struct {
			Chat struct {
				ID    int64  `json:"id"`
				Title string `json:"title,omitempty"`
			} `json:"chat"`
		} `json:"message,omitempty"`
	} `json:"result"`
}

func NewService(botToken string) *Service {
	return &Service{
		botToken: botToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.telegram.org/bot" + botToken,
	}
}

func (s *Service) SendMessage(chatID, message string) error {
	if s.botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	
	if chatID == "" {
		return fmt.Errorf("chat ID not provided")
	}
	
	if message == "" {
		return fmt.Errorf("message is empty")
	}
	
	// Check message length
	if len(message) > 4096 {
		return fmt.Errorf("message too long (%d characters, max 4096)", len(message))
	}
	
	request := SendMessageRequest{
		ChatID:                chatID,
		Text:                  message,
		ParseMode:             "Markdown",
		DisableWebPagePreview: true,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}
	
	url := s.baseURL + "/sendMessage"
	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	
	var response SendMessageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}
	
	if !response.OK {
		return fmt.Errorf("telegram API error: %s", response.Description)
	}
	
	return nil
}

func (s *Service) SendAlert(chatID, alertMessage string) error {
	alert := fmt.Sprintf("🚨 *Winnipeg Tech Events Alert*\n\n%s\n\n_Time: %s_", 
		alertMessage, time.Now().Format("2006-01-02 15:04:05 MST"))
	
	return s.SendMessage(chatID, alert)
}

func (s *Service) TestConnection() error {
	if s.botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	
	url := s.baseURL + "/getMe"
	resp, err := s.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to test connection: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

func (s *Service) GetChatInfo(chatID string) (map[string]interface{}, error) {
	if s.botToken == "" {
		return nil, fmt.Errorf("bot token not configured")
	}
	
	url := s.baseURL + "/getChat"
	request := map[string]string{
		"chat_id": chatID,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}
	
	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	
	return response, nil
}

func (s *Service) FormatMessage(events []map[string]interface{}) string {
	if len(events) == 0 {
		return "📅 No upcoming events found for Winnipeg tech community."
	}
	
	now := time.Now()
	dateStr := now.Format("Monday, January 2, 2006")
	
	message := fmt.Sprintf("🚀 *Winnipeg Tech Events - %s*\n\n", dateStr)
	
	// Group events by time period
	groups := groupEventsForTelegram(events)
	
	for period, periodEvents := range groups {
		if len(periodEvents) > 0 {
			message += fmt.Sprintf("*%s:*\n", period)
			for _, event := range periodEvents {
				name := getString(event, "name")
				url := getString(event, "url")
				startTime := getString(event, "start_time")
				venue := getString(event, "venue")
				price := getString(event, "price")
				
				message += fmt.Sprintf("• %s\n", escapeMarkdown(name))
				
				if startTime != "" {
					if t, err := time.Parse(time.RFC3339, startTime); err == nil {
						timeStr := t.Format("Jan 2 at 3:04 PM")
						message += fmt.Sprintf("  📅 %s\n", timeStr)
					}
				}
				
				if venue != "" {
					message += fmt.Sprintf("  📍 %s\n", escapeMarkdown(venue))
				}
				
				if price != "" && price != "Free" {
					message += fmt.Sprintf("  💰 %s\n", escapeMarkdown(price))
				}
				
				if url != "" {
					message += fmt.Sprintf("  🔗 [View Event](%s)\n", url)
				}
				
				message += "\n"
			}
		}
	}
	
	message += "\n_Shared via Winnipeg Tech Events Tracker_"
	
	return message
}

func groupEventsForTelegram(events []map[string]interface{}) map[string][]map[string]interface{} {
	now := time.Now()
	groups := map[string][]map[string]interface{}{
		"Today":     {},
		"This Week": {},
		"Next Week": {},
		"Later":     {},
	}
	
	for _, event := range events {
		if startTimeStr, ok := event["start_time"].(string); ok {
			if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
				if isSameDay(startTime, now) {
					groups["Today"] = append(groups["Today"], event)
				} else if isThisWeek(startTime) {
					groups["This Week"] = append(groups["This Week"], event)
				} else if isNextWeek(startTime) {
					groups["Next Week"] = append(groups["Next Week"], event)
				} else {
					groups["Later"] = append(groups["Later"], event)
				}
			}
		}
	}
	
	// Remove empty groups
	for key, group := range groups {
		if len(group) == 0 {
			delete(groups, key)
		}
	}
	
	return groups
}

func isSameDay(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && date1.YearDay() == date2.YearDay()
}

func isThisWeek(date time.Time) bool {
	now := time.Now()
	startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
	endOfWeek := startOfWeek.AddDate(0, 0, 6)
	return date.After(startOfWeek) && date.Before(endOfWeek.Add(24*time.Hour))
}

func isNextWeek(date time.Time) bool {
	now := time.Now()
	startOfNextWeek := now.AddDate(0, 0, 7-int(now.Weekday()))
	endOfNextWeek := startOfNextWeek.AddDate(0, 0, 6)
	return date.After(startOfNextWeek) && date.Before(endOfNextWeek.Add(24*time.Hour))
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func escapeMarkdown(text string) string {
	// Escape special Markdown characters
	replacements := map[string]string{
		"_": "\\_",
		"*": "\\*",
		"[": "\\[",
		"]": "\\]",
		"(": "\\(",
		")": "\\)",
		"~": "\\~",
		"`": "\\`",
		">": "\\>",
		"#": "\\#",
		"+": "\\+",
		"-": "\\-",
		"=": "\\=",
		"|": "\\|",
		"{": "\\{",
		"}": "\\}",
		".": "\\.",
		"!": "\\!",
	}
	
	result := text
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	
	return result
}
