package ai

import (
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
	repo      *Repository
	client    anthropic.Client
	model     string
	sysPrompt string
	tools     []Tool
	ttl       config.AIConfig
}

func NewService(repo *Repository, cfg config.AIConfig, notifRepo *notification.Repository, rbacRepo *rbac.Repository) *Service {
	client := anthropic.NewClient(option.WithAPIKey(cfg.AnthropicKey))
	tools := buildTools(notifRepo, rbacRepo)
	return &Service{
		repo:      repo,
		client:    client,
		model:     cfg.Model,
		sysPrompt: cfg.SystemPrompt,
		tools:     tools,
		ttl:       cfg,
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

// Chat runs the agentic streaming loop, writing SSE events to w.
func (s *Service) Chat(ctx context.Context, userID uuid.UUID, claims *token.Claims, req ChatRequest, w http.ResponseWriter) error {
	// Load or create conversation
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

	// Emit start event
	writeSSE(w, map[string]any{"type": "start", "conversation_id": conv.ID.String()})
	flush(w)

	// Build API message list from stored history
	apiMessages := msgsToParams(conv.Messages)
	apiMessages = append(apiMessages, anthropic.NewUserMessage(anthropic.NewTextBlock(req.Message)))

	// storedMessages tracks only user/assistant (no tool internals) for persistence
	storedMessages := append(conv.Messages, ChatMessage{Role: "user", Content: req.Message})

	var assistantText strings.Builder
	var totalTokens int

	// Agentic loop
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
				return nil // client disconnected
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
			// persist
			_ = s.repo.UpdateConversation(ctx, conv.ID, storedMessages, totalTokens)
			writeSSE(w, map[string]any{"type": "done", "token_usage": totalTokens})
			flush(w)
			return nil

		case anthropic.StopReasonToolUse:
			// Append the assistant turn (with tool_use blocks) to API messages
			apiMessages = append(apiMessages, accMsg.ToParam())

			// Execute tools and collect results
			var toolResults []anthropic.ContentBlockParamUnion
			for _, block := range accMsg.Content {
				if block.Type != "tool_use" {
					continue
				}
				toolUse := block.AsToolUse()
				result := s.executeTool(ctx, claims, toolUse.Name, toolUse.Input)
				toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, result, false))
			}

			// Append tool results as user turn
			if len(toolResults) > 0 {
				apiMessages = append(apiMessages, anthropic.NewUserMessage(toolResults...))
			}
			// continue loop

		default:
			// Unexpected stop reason — treat as done
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

// helpers

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
