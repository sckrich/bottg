package mtproto

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
)

type MTProtoConfig struct {
	APIID      int
	APIHash    string
	SessionDir string
}

type Session struct {
	*tg.Client
	client  *telegram.Client
	auth    *auth.Client
	session telegram.SessionStorage
}

func NewSession(cfg MTProtoConfig) (*Session, error) {
	sessionStorage := &telegram.FileSessionStorage{
		Path: fmt.Sprintf("%s/session.json", cfg.SessionDir),
	}

	opts := telegram.Options{
		SessionStorage: sessionStorage,
		DCList:         dcs.Prod(),
	}

	client := telegram.NewClient(cfg.APIID, cfg.APIHash, opts)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Run(ctx, func(ctx context.Context) error {
		authClient := client.Auth()
		status, err := authClient.Status(ctx)
		if err != nil {
			return err
		}
		if !status.Authorized {
			return fmt.Errorf("not authorized")
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	return &Session{
		Client:  client.API(),
		client:  client,
		auth:    client.Auth(),
		session: sessionStorage,
	}, nil
}

func (s *Session) AuthByPhone(ctx context.Context, phone string) (string, error) {
	sentCode, err := s.auth.SendCode(ctx, phone, auth.SendCodeOptions{})
	if err != nil {
		return "", fmt.Errorf("send code failed: %w", err)
	}

	code, ok := sentCode.(*tg.AuthSentCode)
	if !ok {
		return "", fmt.Errorf("unexpected sent code type: %T", sentCode)
	}

	return code.PhoneCodeHash, nil
}

func (s *Session) ConfirmAuth(ctx context.Context, phone, code, codeHash string) error {
	_, err := s.auth.SignIn(ctx, phone, code, codeHash)
	return err
}

func (s *Session) Close() {
	if s.client != nil {
	}
}
