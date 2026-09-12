package chat_service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type IService interface {
	Post(ctx context.Context, req PostRequest) (domain.ChatMessage, error)
	History(ctx context.Context) []domain.ChatMessage
}

type PostRequest struct {
	AuthorName string
	AuthorID   string
	CountryID  string
	Text       string
	UserAgent  string
}

func New(
	storage domain.Storage,
	countryChecker domain.CountryChecker,
	timeProvider cptime.Provider,
	config Config,
) *Service {
	if timeProvider == nil {
		timeProvider = cptime.ActualProvider{}
	}

	return &Service{
		storage:        storage,
		countryChecker: countryChecker,
		timeProvider:   timeProvider,
		config:         config.withDefaults(),
	}
}

type Service struct {
	storage        domain.Storage
	countryChecker domain.CountryChecker
	timeProvider   cptime.Provider
	config         Config
}

func (s *Service) Post(ctx context.Context, req PostRequest) (domain.ChatMessage, error) {
	name, err := sanitize(req.AuthorName, s.config.MaxNameLength)
	if err != nil {
		return domain.ChatMessage{}, fmt.Errorf("%w: author name: %w", domain.ErrInvalidMessage, err)
	}

	text, err := sanitize(req.Text, s.config.MaxTextLength)
	if err != nil {
		return domain.ChatMessage{}, fmt.Errorf("%w: text: %w", domain.ErrInvalidMessage, err)
	}

	if !s.countryChecker.CheckCountry(req.CountryID) {
		return domain.ChatMessage{}, fmt.Errorf("%w: invalid country code %q", domain.ErrInvalidMessage, req.CountryID)
	}

	ip := cpctx.GetSourceIP(ctx)

	message := domain.ChatMessage{
		ID:         uuid.NewString(),
		SentAt:     s.timeProvider.Now(),
		AuthorName: name,
		AuthorTag:  s.tag(ip),
		CountryID:  req.CountryID,
		Text:       text,
	}

	record := domain.ChatRecord{
		Message:   message,
		AuthorID:  truncate(req.AuthorID, maxAuthorIDLength),
		IP:        ip,
		UserAgent: truncate(req.UserAgent, maxUserAgentLength),
	}

	if err := s.storage.Append(ctx, record); err != nil {
		return domain.ChatMessage{}, fmt.Errorf("failed to store chat message: %w", err)
	}

	return message, nil
}

func (s *Service) History(ctx context.Context) []domain.ChatMessage {
	return s.storage.History(ctx)
}

func (s *Service) tag(ip string) string {
	sum := sha256.Sum256([]byte(s.config.TagSalt + "\x00" + ip))
	return hex.EncodeToString(sum[:])[:tagLength]
}

const (
	tagLength          = 6
	maxAuthorIDLength  = 64
	maxUserAgentLength = 256
)

func sanitize(value string, maxLength int) (string, error) {
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("not valid UTF-8")
	}

	var b strings.Builder
	for _, r := range value {
		if r == '\t' {
			r = ' '
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}

	cleaned := strings.TrimSpace(b.String())
	if cleaned == "" {
		return "", fmt.Errorf("empty")
	}

	if utf8.RuneCountInString(cleaned) > maxLength {
		return "", fmt.Errorf("longer than %d characters", maxLength)
	}

	return cleaned, nil
}

func truncate(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}
