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
	"github.com/google/uuid"
)

// ---- OpenRouter (OpenAI-compatible) types ----

type orMessage struct {
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	ToolCalls  []orToolCall `json:"tool_calls,omitempty"`
}

type orToolCall struct {
	ID       string     `json:"id"`
	Type     string     `json:"type"`
	Function orFuncCall `json:"function"`
}

type orFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type orTool struct {
	Type     string    `json:"type"`
	Function orToolDef `json:"function"`
}

type orToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type orRequest struct {
	Model     string      `json:"model"`
	Messages  []orMessage `json:"messages"`
	Stream    bool        `json:"stream"`
	MaxTokens int         `json:"max_tokens,omitempty"`
	Tools     []orTool    `json:"tools,omitempty"`
}

type orChunk struct {
	Choices []orChunkChoice `json:"choices"`
	Usage   *orUsage        `json:"usage,omitempty"`
}

type orChunkChoice struct {
	Delta        orDelta `json:"delta"`
	FinishReason string  `json:"finish_reason"`
}

type orDelta struct {
	Content   string            `json:"content,omitempty"`
	ToolCalls []orToolCallDelta `json:"tool_calls,omitempty"`
}

type orToolCallDelta struct {
	Index    int         `json:"index"`
	ID       string      `json:"id,omitempty"`
	Type     string      `json:"type,omitempty"`
	Function orFuncDelta `json:"function"`
}

type orFuncDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type orUsage struct {
	TotalTokens int `json:"total_tokens"`
}

// ---- Service ----

type Service struct {
	repo      *Repository
	apiKey    string
	model     string
	sysPrompt string
	tools     []Tool
	ttl       config.AIConfig
}

func NewService(repo *Repository, cfg config.AIConfig, notifRepo *notification.Repository, rbacRepo *rbac.Repository) *Service {
	return &Service{
		repo:      repo,
		apiKey:    cfg.OpenRouterKey,
		model:     cfg.Model,
		sysPrompt: cfg.SystemPrompt,
		tools:     buildTools(notifRepo, rbacRepo),
		ttl:       cfg,
	}
}

func (s *Service) orTools() []orTool {
	result := make([]orTool, 0, len(s.tools))
	for _, t := range s.tools {
		result = append(result, orTool{
			Type: "function",
			Function: orToolDef{
				Name:        t.Def.Name,
				Description: t.Def.Description,
				Parameters:  t.Def.Parameters,
			},
		})
	}
	return result
}

func (s *Service) findTool(name string) *Tool {
	for i := range s.tools {
		if s.tools[i].Def.Name == name {
			return &s.tools[i]
		}
	}
	return nil
}

func (s *Service) Chat(ctx context.Context, userID uuid.UUID, claims *token.Claims, req ChatRequest, w http.ResponseWriter) error {
	if s.apiKey == "" {
		writeSSE(w, map[string]any{"type": "error", "message": "OpenRouter API key not configured"})
		flush(w)
		return fmt.Errorf("openrouter key not set")
	}

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

	messages := make([]orMessage, 0, len(conv.Messages)+2)
	if s.sysPrompt != "" {
		messages = append(messages, orMessage{Role: "system", Content: s.sysPrompt})
	}
	for _, m := range conv.Messages {
		messages = append(messages, orMessage{Role: m.Role, Content: m.Content})
	}
	messages = append(messages, orMessage{Role: "user", Content: req.Message})

	storedMessages := append(conv.Messages, ChatMessage{Role: "user", Content: req.Message})
	var totalTokens int
	var assistantText strings.Builder

	for {
		orReq := orRequest{
			Model:     s.model,
			Messages:  messages,
			Stream:    true,
			MaxTokens: 8192,
		}
		if len(s.tools) > 0 {
			orReq.Tools = s.orTools()
		}

		reqBody, _ := json.Marshal(orReq)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(reqBody))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
		httpReq.Header.Set("HTTP-Referer", "https://github.com/404nfid/go-svelte-starter-kit")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			writeSSE(w, map[string]any{"type": "error", "message": "OpenRouter request failed"})
			flush(w)
			return err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			writeSSE(w, map[string]any{"type": "error", "message": fmt.Sprintf("OpenRouter API error (status %d)", resp.StatusCode)})
			flush(w)
			return fmt.Errorf("openrouter status %d", resp.StatusCode)
		}

		assistantText.Reset()
		toolCallMap := map[int]*orToolCall{}
		var finishReason string

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := line[6:]
			if data == "[DONE]" {
				break
			}
			var chunk orChunk
			if json.Unmarshal([]byte(data), &chunk) != nil {
				continue
			}
			if chunk.Usage != nil && chunk.Usage.TotalTokens > 0 {
				totalTokens = chunk.Usage.TotalTokens
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
			if choice.Delta.Content != "" {
				assistantText.WriteString(choice.Delta.Content)
				writeSSE(w, map[string]any{"type": "delta", "text": choice.Delta.Content})
				flush(w)
			}
			for _, tc := range choice.Delta.ToolCalls {
				existing, ok := toolCallMap[tc.Index]
				if !ok {
					existing = &orToolCall{Type: "function"}
					toolCallMap[tc.Index] = existing
				}
				if tc.ID != "" {
					existing.ID = tc.ID
				}
				if tc.Function.Name != "" {
					existing.Function.Name = tc.Function.Name
				}
				existing.Function.Arguments += tc.Function.Arguments
			}
		}
		resp.Body.Close()

		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			return err
		}

		// Collect tool calls in index order
		toolCalls := make([]orToolCall, len(toolCallMap))
		for i := range len(toolCallMap) {
			if tc, ok := toolCallMap[i]; ok {
				toolCalls[i] = *tc
			}
		}

		if finishReason == "tool_calls" && len(toolCalls) > 0 {
			messages = append(messages, orMessage{Role: "assistant", ToolCalls: toolCalls})
			for _, tc := range toolCalls {
				result := s.executeTool(ctx, claims, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
				messages = append(messages, orMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    result,
				})
			}
			continue
		}

		if text := assistantText.String(); text != "" {
			storedMessages = append(storedMessages, ChatMessage{Role: "assistant", Content: text})
		}
		_ = s.repo.UpdateConversation(ctx, conv.ID, storedMessages, totalTokens)
		writeSSE(w, map[string]any{"type": "done", "token_usage": totalTokens})
		flush(w)
		return nil
	}
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
