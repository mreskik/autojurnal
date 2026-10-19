package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"autojurnal/internal/module/master/users"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionNotFound    = errors.New("session not found")
)

// last_used_at hanya di-update kalau sudah lewat interval ini, biar tidak write tiap request
const touchInterval = time.Minute

type Service struct {
	repo      *Repository
	userRepo  *users.Repository
	ttl       time.Duration
	dummyHash []byte
}

func NewService(repo *Repository, userRepo *users.Repository, ttl time.Duration) *Service {
	// dipakai saat username tidak ditemukan, supaya waktu respons sama
	// dengan password salah (mencegah tebak username lewat timing)
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)
	return &Service{repo: repo, userRepo: userRepo, ttl: ttl, dummyHash: dummy}
}

func (s *Service) Login(ctx context.Context, req LoginRequest, ip, userAgent string) (LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return LoginResponse{}, err
	}

	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if errors.Is(err, users.ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(req.Password))
		return LoginResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResponse{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	token, err := generateToken()
	if err != nil {
		return LoginResponse{}, err
	}
	session := &Session{
		UserID:    user.ID,
		TokenHash: hashToken(token),
		IPAddress: truncate(ip, 45),
		UserAgent: truncate(userAgent, 255),
		ExpiresAt: time.Now().Add(s.ttl),
	}
	if err := s.repo.Create(ctx, session); err != nil {
		return LoginResponse{}, err
	}

	if err := s.repo.DeleteExpiredByUser(ctx, user.ID); err != nil {
		log.Printf("auth: cleanup expired sessions: %v", err)
	}

	return LoginResponse{
		Token:     token,
		ExpiresAt: session.ExpiresAt,
		User:      users.ToUserResponse(user),
	}, nil
}

// Authenticate memvalidasi token dari header dan mengembalikan session + user-nya.
func (s *Service) Authenticate(ctx context.Context, token string) (*Session, *users.User, error) {
	if token == "" {
		return nil, nil, ErrUnauthorized
	}
	session, err := s.repo.FindActiveByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, nil, err
	}

	// user yang sudah dihapus (soft delete) otomatis tidak lolos
	user, err := s.userRepo.FindByID(ctx, session.UserID)
	if errors.Is(err, users.ErrUserNotFound) {
		return nil, nil, ErrUnauthorized
	}
	if err != nil {
		return nil, nil, err
	}

	if time.Since(session.LastUsedAt) > touchInterval {
		if err := s.repo.Touch(ctx, session.ID); err != nil {
			log.Printf("auth: touch session: %v", err)
		}
	}
	return session, user, nil
}

func (s *Service) Logout(ctx context.Context, session *Session) error {
	return s.repo.DeleteByID(ctx, session.UserID, session.ID)
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.repo.DeleteByUser(ctx, userID)
}

func (s *Service) ListSessions(ctx context.Context, current *Session) ([]SessionResponse, error) {
	list, err := s.repo.ListActiveByUser(ctx, current.UserID)
	if err != nil {
		return nil, err
	}
	result := make([]SessionResponse, 0, len(list))
	for i := range list {
		result = append(result, ToSessionResponse(&list[i], current.ID))
	}
	return result, nil
}

func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	return s.repo.DeleteByID(ctx, userID, sessionID)
}

// generateToken: 32 byte acak -> 43 karakter base64url.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
