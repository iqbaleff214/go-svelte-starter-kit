package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/404nfid/go-svelte-starter-kit/internal/notification"
	"github.com/404nfid/go-svelte-starter-kit/internal/rbac"
	"github.com/404nfid/go-svelte-starter-kit/pkg/config"
	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
)

type Service struct {
	repo            *Repository
	client          anthropic.Client
	model           string
	geminiKey       string
	geminiModel     string
	defaultProvider string
	sysPrompt       string
	tools           []Tool
	ttl             config.AIConfig
}

func NewService(repo *Repository, cfg config.AIConfig, notifRepo *notification.Repository, rbacRepo *rbac.Repository) *Service {
	client := anthropic.NewClient(option.WithAPIKey(cfg.AnthropicKey))
	tools := buildTools(notifRepo, rbacRepo)
	return &Service{
		repo:            repo,
		client:          client,
		model:           cfg.Model,
		geminiKey:       cfg.GeminiKey,
		geminiModel:     cfg.GeminiModel,
		defaultProvider: cfg.Provider,
		sysPrompt:       cfg.SystemPrompt,
		tools:           tools,
		ttl:             cfg,
	}
}

func (s *Service) toolParams() []anthropic.ToolUnionParam {
	params := make([]anthropic.ToolUnionParam, len(s.tools))
	for i, t := range s.tools {
		params[i] = t.Param
	}
	return params
}

func (s *Service) findTool(name string) *Tool {
	for i := range s.tools {
		if s.tools[i].Param.OfTool != nil && s.tools[i].Param.OfTool.Name == name {
			return &s.tools[i]
		}
	}
	return nil
}

// Chat dispatches to the appropriate provider.
func (s *Service) Chat(ctx context.Context, userID uuid.UUID, claims *token.Claims, req ChatRequest, w http.ResponseWriter) error {
	provider := req.Provider
	if provider == "" {
		provider = s.defaultProvider
	}
	if provider == "gemini" {
		return s.chatWithGemini(ctx, userID, req, w)
	}
	return s.chatWithAnthropic(ctx, userID, claims, req, w)
}

// chatWithAnthropic runs the agentic streaming loop using the Anthropic API.
func (s *Service) chatWithAnthropic(ctx context.Context, userID uuid.UUID, claims *token.Claims, req ChatRequest, w http.ResponseWriter) error {
	var conv *Conversation
	var err error
	if req.ConversationID != nil && *req.ConversationID != "" {
		id, parseErr := uuid.Parse(*req.ConversationID)
		if parseErr != nil {
			return fmt.Errorf("invalid conversation_id")
		}
		conv, err = s.repo.GetConversation(ctx, id, userID)
	} else {
		title := truncate(req.Message, 60)
		conv, err = s.repo.CreateConversation(ctx, userID, title, s.model, []ChatMessage{})
	}
	if err != nil {
		return fmt.Errorf("load conversation: %w", err)
	}

	writeSSE(w, map[string]any{"type": "start", "conversation_id": conv.ID.String()})
	flush(w)

	apiMessages := msgsToParams(conv.Messages)
	apiMessages = append(apiMessages, anthropic.NewUserMessage(anthropic.NewTextBlock(req.Message)))

	storedMessages := append(conv.Messages, ChatMessage{Role: "user", Content: req.Message})

	var assistantText strings.Builder
	var totalTokens int

	for {
		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(s.model),
			MaxTokens: 8192,
			Messages:  apiMessages,
			Tools:     s.toolParams(),
		}
		if s.sysPrompt != "" {
			params.System = []anthropic.TextBlockParam{{Text: s.sysPrompt}}
		}

		stream := s.client.Messages.NewStreaming(ctx, params)

		var accMsg anthropic.Message
		assistantText.Reset()

		for stream.Next() {
			event := stream.Current()
			if accErr := accMsg.Accumulate(event); accErr != nil {
				continue
			}
			if ev, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
				if delta, ok := ev.Delta.AsAny().(anthropic.TextDelta); ok {
					assistantText.WriteString(delta.Text)
					writeSSE(w, map[string]any{"type": "delta", "text": delta.Text})
					flush(w)
				}
			}
		}

		if err := stream.Err(); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			writeSSE(w, map[string]any{"type": "error", "message": "Stream error"})
			flush(w)
			return err
		}

		totalTokens += int(accMsg.Usage.OutputTokens)

		switch accMsg.StopReason {
		case anthropic.StopReasonEndTurn, anthropic.StopReasonMaxTokens, anthropic.StopReasonStopSequence:
			text := assistantText.String()
			if text != "" {
				storedMessages = append(storedMessages, ChatMessage{Role: "assistant", Content: text})
			}
			_ = s.repo.UpdateConversation(ctx, conv.ID, storedMessages, totalTokens)
			writeSSE(w, map[string]any{"type": "done", "token_usage": totalTokens})
			flush(w)
			return nil

		case anthropic.StopReasonToolUse:
			apiMessages = append(apiMessages, accMsg.ToParam())

			var toolResults []anthropic.ContentBlockParamUnion
			for _, block := range accMsg.Content {
				if block.Type != "tool_use" {
					continue
				}
				toolUse := block.AsToolUse()
				result := s.executeTool(ctx, claims, toolUse.Name, toolUse.Input)
				toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, result, false))
			}

			if len(toolResults) > 0 {
				apiMessages = append(apiMessages, anthropic.NewUserMessage(toolResults...))
			}

		default:
			text := assistantText.String()
			if text != "" {
				storedMessages = append(storedMessages, ChatMessage{Role: "assistant", Content: text})
			}
			_ = s.repo.UpdateConversation(ctx, conv.ID, storedMessages, totalTokens)
			writeSSE(w, map[string]any{"type": "done", "token_usage": totalTokens})
			flush(w)
			return nil
		}
	}
}

// chatWithGemini streams a response using the Google Gemini REST API.
func (s *Service) chatWithGemini(ctx context.Context, userID uuid.UUID, req ChatRequest, w http.ResponseWriter) error {
	if s.geminiKey == "" {
		writeSSE(w, map[string]any{"type": "error", "message": "Gemini API key not configured"})
		flush(w)
		return fmt.Errorf("gemini key not set")
	}

	var conv *Conversation
	var err error
	modelLabel := "gemini/" + s.geminiModel
	if req.ConversationID != nil && *req.ConversationID != "" {
		id, parseErr := uuid.Parse(*req.ConversationID)
		if parseErr != nil {
			return fmt.Errorf("invalid conversation_id")
		}
		conv, err = s.repo.GetConversation(ctx, id, userID)
	} else {
		title := truncate(req.Message, 60)
		conv, err = s.repo.CreateConversation(ctx, userID, title, modelLabel, []ChatMessage{})
	}
	if err != nil {
		return fmt.Errorf("load conversation: %w", err)
	}

	writeSSE(w, map[string]any{"type": "start", "conversation_id": conv.ID.String()})
	flush(w)

	// Build Gemini request body
	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Role  string       `json:"role"`
		Parts []geminiPart `json:"parts"`
	}
	type geminiSysInstruction struct {
		Parts []geminiPart `json:"parts"`
	}
	type geminiGenConfig struct {
		MaxOutputTokens int `json:"maxOutputTokens"`
	}
	type geminiRequest struct {
		Contents          []geminiContent       `json:"contents"`
		SystemInstruction *geminiSysInstruction `json:"systemInstruction,omitempty"`
		GenerationConfig  geminiGenConfig       `json:"generationConfig"`
	}

	var contents []geminiContent
	for _, m := range conv.Messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{Role: role, Parts: []geminiPart{{Text: m.Content}}})
	}
	contents = append(contents, geminiContent{Role: "user", Parts: []geminiPart{{Text: req.Message}}})

	gemReq := geminiRequest{
		Contents:         contents,
		GenerationConfig: geminiGenConfig{MaxOutputTokens: 8192},
	}
	if s.sysPrompt != "" {
		gemReq.SystemInstruction = &geminiSysInstruction{Parts: []geminiPart{{Text: s.sysPrompt}}}
	}

	reqBody, _ := json.Marshal(gemReq)
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s",
		s.geminiModel, s.geminiKey,
	)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		writeSSE(w, map[string]any{"type": "error", "message": "Gemini request failed"})
		flush(w)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeSSE(w, map[string]any{"type": "error", "message": fmt.Sprintf("Gemini API error (status %d)", resp.StatusCode)})
		flush(w)
		return fmt.Errorf("gemini status %d", resp.StatusCode)
	}

	// Parse SSE stream from Gemini
	var geminiEvent struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			TotalTokenCount int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	var assistantText strings.Builder
	var totalTokens int

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		if err := json.Unmarshal([]byte(data), &geminiEvent); err != nil {
			continue
		}
		if len(geminiEvent.Candidates) > 0 {
			for _, part := range geminiEvent.Candidates[0].Content.Parts {
				if part.Text != "" {
					assistantText.WriteString(part.Text)
					writeSSE(w, map[string]any{"type": "delta", "text": part.Text})
					flush(w)
				}
			}
			if geminiEvent.UsageMetadata.TotalTokenCount > 0 {
				totalTokens = geminiEvent.UsageMetadata.TotalTokenCount
			}
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return err
	}

	storedMessages := append(conv.Messages, ChatMessage{Role: "user", Content: req.Message})
	if text := assistantText.String(); text != "" {
		storedMessages = append(storedMessages, ChatMessage{Role: "assistant", Content: text})
	}
	_ = s.repo.UpdateConversation(ctx, conv.ID, storedMessages, totalTokens)
	writeSSE(w, map[string]any{"type": "done", "token_usage": totalTokens})
	flush(w)

	return nil
}

func (s *Service) executeTool(ctx context.Context, claims *token.Claims, name string, inputRaw json.RawMessage) string {
	tool := s.findTool(name)
	if tool == nil {
		return fmt.Sprintf(`"unknown tool: %s"`, name)
	}
	var input ToolInput
	if len(inputRaw) > 0 {
		_ = json.Unmarshal(inputRaw, &input)
	}
	if input == nil {
		input = ToolInput{}
	}
	result, err := tool.Execute(ctx, claims, input)
	if err != nil {
		return fmt.Sprintf(`"tool error: %s"`, err.Error())
	}
	return result
}

func (s *Service) ListConversations(ctx context.Context, userID uuid.UUID) ([]*ConversationSummary, error) {
	return s.repo.ListConversations(ctx, userID)
}

func (s *Service) GetConversation(ctx context.Context, id, userID uuid.UUID) (*Conversation, error) {
	return s.repo.GetConversation(ctx, id, userID)
}

func (s *Service) DeleteConversation(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.DeleteConversation(ctx, id, userID)
}

func (s *Service) PurgeOldConversations(ctx context.Context) error {
	return s.repo.PurgeOldConversations(ctx, s.ttl.ConversationTTL)
}

// ---- helpers ----

func msgsToParams(msgs []ChatMessage) []anthropic.MessageParam {
	params := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "user" {
			params = append(params, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		} else if m.Role == "assistant" {
			params = append(params, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	return params
}

func writeSSE(w http.ResponseWriter, data any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", b)
}

func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
