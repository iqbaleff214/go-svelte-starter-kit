package user

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

var (
	ErrInvalidCurrentPassword = errors.New("current password is incorrect")
	ErrInvalidImageType       = errors.New("avatar must be JPEG, PNG, or WebP")
	ErrImageTooLarge          = errors.New("avatar must be 2MB or less")
)

const (
	maxAvatarBytes = 2 << 20 // 2 MB
	maxAvatarDim   = 256
	uploadsDir     = "uploads/avatars"
)

type Service struct {
	repo    *Repository
	baseURL string
}

func NewService(repo *Repository, baseURL string) *Service {
	return &Service{repo: repo, baseURL: baseURL}
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	log.Println(u)
	return toProfileResponse(u), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*ProfileResponse, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	displayName := u.DisplayName
	if req.DisplayName != "" {
		displayName = req.DisplayName
	}
	bio := req.Bio

	updated, err := s.repo.UpdateProfile(ctx, userID, displayName, bio)
	if err != nil {
		return nil, err
	}
	return toProfileResponse(updated), nil
}

func (s *Service) UploadAvatar(ctx context.Context, userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > maxAvatarBytes {
		return "", ErrImageTooLarge
	}

	src, format, err := image.Decode(file)
	if err != nil {
		return "", ErrInvalidImageType
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return "", ErrInvalidImageType
	}

	resized := resizeImage(src, maxAvatarDim)

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("create uploads dir: %w", err)
	}

	filename := uuid.New().String() + ".jpg"
	path := filepath.Join(uploadsDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create avatar file: %w", err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, resized, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("encode avatar: %w", err)
	}

	avatarURL := s.baseURL + "/uploads/avatars/" + filename
	if err := s.repo.UpdateAvatar(ctx, userID, avatarURL); err != nil {
		return "", err
	}
	return avatarURL, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return ErrInvalidCurrentPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.repo.UpdatePassword(ctx, userID, string(hash))
}

func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	return s.repo.SoftDelete(ctx, userID)
}

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID, currentSessionID string) ([]*SessionResponse, error) {
	sessions, err := s.repo.ListActiveSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*SessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		result = append(result, &SessionResponse{
			ID:         sess.ID.String(),
			UserAgent:  sess.UserAgent,
			IPAddress:  sess.IPAddress,
			LastSeenAt: sess.LastSeenAt,
			CreatedAt:  sess.CreatedAt,
			IsCurrent:  sess.ID.String() == currentSessionID,
		})
	}
	return result, nil
}

func (s *Service) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return ErrNotFound
	}
	return s.repo.RevokeSession(ctx, sid, userID)
}

func (s *Service) RevokeAllOtherSessions(ctx context.Context, userID uuid.UUID, currentSessionID string) error {
	sid, err := uuid.Parse(currentSessionID)
	if err != nil {
		return nil
	}
	return s.repo.RevokeAllOtherSessions(ctx, userID, sid)
}

// resizeImage scales img down so neither dimension exceeds maxDim, preserving aspect ratio.
func resizeImage(src image.Image, maxDim int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= maxDim && h <= maxDim {
		return src
	}

	var dw, dh int
	if w > h {
		dw = maxDim
		dh = h * maxDim / w
	} else {
		dh = maxDim
		dw = w * maxDim / h
	}

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func toProfileResponse(u *User) *ProfileResponse {
	return &ProfileResponse{
		ID:              u.ID.String(),
		Email:           u.Email,
		DisplayName:     u.DisplayName,
		AvatarURL:       u.AvatarURL,
		Bio:             u.Bio,
		EmailVerifiedAt: u.EmailVerifiedAt,
		TwoFAEnabled:    u.TwoFAEnabled,
		CreatedAt:       u.CreatedAt,
	}
}
