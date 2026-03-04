package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrRateLimited = errors.New("rate limit exceeded")

type Service struct {
	repo      *Repository
	redis     *redis.Client
	apiKeyRPM int
}

func NewService(repo *Repository, redis *redis.Client, apiKeyRPM int) *Service {
	return &Service{repo: repo, redis: redis, apiKeyRPM: apiKeyRPM}
}

// Create generates a new API key, stores the hash, and returns the plaintext once.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateKeyRequest) (*APIKeyCreateResponse, error) {
	// Generate 32 random bytes → hex (64 chars)
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	hex64 := hex.EncodeToString(raw)
	plainKey := "sk_" + hex64
	prefix := hex64[:8]
	hash := hashKey(hex64)

	key, err := s.repo.Create(ctx, userID, req.Name, hash, prefix, req.Scopes, req.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &APIKeyCreateResponse{APIKey: *key, Key: plainKey}, nil
}

// ValidateKey checks the raw key (e.g. "sk_abc...") against the DB.
func (s *Service) ValidateKey(ctx context.Context, rawKey string) (*ValidatedKey, error) {
	// Strip "sk_" prefix to get hex64
	if len(rawKey) < 4 || rawKey[:3] != "sk_" {
		return nil, fmt.Errorf("invalid api key format")
	}
	hex64 := rawKey[3:]
	hash := hashKey(hex64)
	return s.repo.GetByHash(ctx, hash)
}

// CheckRateLimit enforces per-key per-minute rate limiting via Redis.
func (s *Service) CheckRateLimit(ctx context.Context, keyID uuid.UUID) error {
	if s.apiKeyRPM <= 0 {
		return nil
	}
	window := time.Now().Unix() / 60
	redisKey := fmt.Sprintf("ratelimit:apikey:%s:%d", keyID, window)

	count, err := s.redis.Incr(ctx, redisKey).Result()
	if err != nil {
		return nil // fail open on Redis error
	}
	if count == 1 {
		s.redis.Expire(ctx, redisKey, 2*time.Minute)
	}
	if int(count) > s.apiKeyRPM {
		return ErrRateLimited
	}
	return nil
}

// LogUsage asynchronously records an audit log entry (fire-and-forget).
func (s *Service) LogUsage(keyID uuid.UUID, method, path, ip string, statusCode int) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.repo.CreateLog(ctx, &APIKeyLog{
			APIKeyID:   keyID,
			Method:     method,
			Path:       path,
			StatusCode: statusCode,
			IP:         ip,
		})
		_ = s.repo.UpdateLastUsed(ctx, keyID)
	}()
}

func (s *Service) ListKeys(ctx context.Context, userID uuid.UUID) ([]*APIKey, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *Service) RevokeKey(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Revoke(ctx, id, userID)
}

func (s *Service) ListLogs(ctx context.Context, keyID, userID uuid.UUID, limit, offset int) ([]*APIKeyLog, int, error) {
	return s.repo.ListLogs(ctx, keyID, userID, limit, offset)
}

// hashKey returns the SHA-256 hex digest of the hex64 portion of a key.
func hashKey(hex64 string) string {
	sum := sha256.Sum256([]byte(hex64))
	return hex.EncodeToString(sum[:])
}
