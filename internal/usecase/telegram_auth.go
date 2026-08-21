package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

const (
	TelegramOIDCModeLogin = "login"
	TelegramOIDCModeLink  = "link"
	telegramPasswordHash  = "!telegram"
)

type TelegramOIDCClaims struct {
	Subject     string
	Username    string
	DisplayName string
	PictureURL  string
}

type TelegramOIDCClient interface {
	Exchange(ctx context.Context, code string, codeVerifier string, redirectURI string, nonce string) (TelegramOIDCClaims, error)
}

type TelegramAuthConfig struct {
	Enabled          bool
	ClientID         string
	AuthorizationURL string
	RedirectURI      string
	FlowTTL          time.Duration
}

type TelegramBeginCommand struct {
	Mode   string
	UserID *entity.AuthUserID
}

type TelegramCompleteCommand struct {
	State         string
	Code          string
	CurrentUserID *entity.AuthUserID
}

type TelegramCompleteResult struct {
	Mode   string
	UserID entity.AuthUserID
}

type AuthIdentityDTO struct {
	Provider    entity.AuthProvider `json:"provider"`
	Username    string              `json:"username,omitempty"`
	DisplayName string              `json:"display_name,omitempty"`
	PictureURL  string              `json:"picture_url,omitempty"`
}

type TelegramAuthService struct {
	identities AuthIdentityRepository
	flows      AuthOIDCFlowRepository
	users      AuthUserRepository
	txManager  TxManager
	userIDGen  AuthUserIDGenerator
	tokenGen   TokenGenerator
	tokenHash  TokenHasher
	client     TelegramOIDCClient
	clock      Clock
	config     TelegramAuthConfig
}

func NewTelegramAuthService(
	identities AuthIdentityRepository,
	flows AuthOIDCFlowRepository,
	users AuthUserRepository,
	txManager TxManager,
	userIDGen AuthUserIDGenerator,
	tokenGen TokenGenerator,
	tokenHash TokenHasher,
	client TelegramOIDCClient,
	clock Clock,
	config TelegramAuthConfig,
) *TelegramAuthService {
	if clock == nil {
		clock = SystemClock{}
	}
	if config.FlowTTL <= 0 {
		config.FlowTTL = 10 * time.Minute
	}
	if config.AuthorizationURL == "" {
		config.AuthorizationURL = "https://oauth.telegram.org/auth"
	}
	return &TelegramAuthService{
		identities: identities,
		flows:      flows,
		users:      users,
		txManager:  txManager,
		userIDGen:  userIDGen,
		tokenGen:   tokenGen,
		tokenHash:  tokenHash,
		client:     client,
		clock:      clock,
		config:     config,
	}
}

func (s *TelegramAuthService) Enabled() bool {
	return s != nil && s.config.Enabled && s.client != nil
}

func (s *TelegramAuthService) Begin(ctx context.Context, cmd TelegramBeginCommand) (string, error) {
	if !s.Enabled() {
		return "", entity.ErrTelegramAuthDisabled
	}
	if cmd.Mode != TelegramOIDCModeLogin && cmd.Mode != TelegramOIDCModeLink {
		return "", entity.ErrOIDCFlowInvalid
	}
	if cmd.Mode == TelegramOIDCModeLink && cmd.UserID == nil {
		return "", entity.ErrUnauthorized
	}

	state, err := s.tokenGen.NewToken()
	if err != nil {
		return "", err
	}
	verifier, err := s.tokenGen.NewToken()
	if err != nil {
		return "", err
	}
	nonce, err := s.tokenGen.NewToken()
	if err != nil {
		return "", err
	}
	now := s.clock.Now()
	flow := &entity.AuthOIDCFlow{
		StateHash: s.tokenHash.HashToken(state), Mode: cmd.Mode, UserID: cmd.UserID,
		CodeVerifier: verifier, Nonce: nonce, RedirectURI: s.config.RedirectURI,
		CreatedAt: now, ExpiresAt: now.Add(s.config.FlowTTL),
	}
	if err := s.txManager.RunInTx(ctx, func(tx Tx) error { return s.flows.SaveOIDCFlow(tx, flow) }); err != nil {
		return "", err
	}

	challenge := sha256.Sum256([]byte(verifier))
	params := url.Values{
		"client_id":             {s.config.ClientID},
		"redirect_uri":          {s.config.RedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	return s.config.AuthorizationURL + "?" + params.Encode(), nil
}

func (s *TelegramAuthService) Complete(ctx context.Context, cmd TelegramCompleteCommand) (*TelegramCompleteResult, error) {
	if !s.Enabled() {
		return nil, entity.ErrTelegramAuthDisabled
	}
	if strings.TrimSpace(cmd.State) == "" || strings.TrimSpace(cmd.Code) == "" {
		return nil, entity.ErrOIDCFlowInvalid
	}

	var flow *entity.AuthOIDCFlow
	if err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		var err error
		flow, err = s.flows.ConsumeOIDCFlow(tx, s.tokenHash.HashToken(cmd.State), s.clock.Now())
		return err
	}); err != nil {
		return nil, err
	}
	if flow.Mode == TelegramOIDCModeLink && (flow.UserID == nil || cmd.CurrentUserID == nil || *flow.UserID != *cmd.CurrentUserID) {
		return nil, entity.ErrUnauthorized
	}

	claims, err := s.client.Exchange(ctx, cmd.Code, flow.CodeVerifier, flow.RedirectURI, flow.Nonce)
	if err != nil || strings.TrimSpace(claims.Subject) == "" {
		return nil, entity.ErrOIDCTokenInvalid
	}

	now := s.clock.Now()
	identity := &entity.AuthIdentity{
		Provider: entity.AuthProviderTelegram, Subject: claims.Subject,
		Username: claims.Username, DisplayName: claims.DisplayName, PictureURL: claims.PictureURL,
		CreatedAt: now, UpdatedAt: now,
	}
	result := &TelegramCompleteResult{Mode: flow.Mode}
	err = s.txManager.RunInTx(ctx, func(tx Tx) error {
		if flow.Mode == TelegramOIDCModeLink {
			identity.UserID = *flow.UserID
			if err := s.identities.SaveIdentity(tx, identity); err != nil {
				return err
			}
			result.UserID = *flow.UserID
			return nil
		}

		existing, findErr := s.identities.FindIdentity(tx, entity.AuthProviderTelegram, claims.Subject)
		if findErr == nil {
			result.UserID = existing.UserID
			return nil
		}
		if !errors.Is(findErr, entity.ErrAuthIdentityNotFound) {
			return findErr
		}

		userID := s.userIDGen.New()
		emailHash := sha256.Sum256([]byte(claims.Subject))
		user, newErr := entity.NewAuthUser(
			userID,
			fmt.Sprintf("telegram-%x@telegram.invalid", emailHash[:12]),
			telegramPasswordHash,
			entity.AuthRoleUser,
			now,
		)
		if newErr != nil {
			return newErr
		}
		if err := s.users.Save(tx, user); err != nil {
			return err
		}
		identity.UserID = userID
		if err := s.identities.SaveIdentity(tx, identity); err != nil {
			return err
		}
		result.UserID = userID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *TelegramAuthService) ListIdentities(ctx context.Context, userID entity.AuthUserID) ([]AuthIdentityDTO, error) {
	var identities []entity.AuthIdentity
	if err := s.txManager.RunInTx(ctx, func(tx Tx) error {
		var err error
		identities, err = s.identities.ListIdentities(tx, userID)
		return err
	}); err != nil {
		return nil, err
	}
	result := make([]AuthIdentityDTO, 0, len(identities))
	for _, identity := range identities {
		result = append(result, AuthIdentityDTO{
			Provider: identity.Provider, Username: identity.Username,
			DisplayName: identity.DisplayName, PictureURL: identity.PictureURL,
		})
	}
	return result, nil
}

func (s *TelegramAuthService) Unlink(ctx context.Context, userID entity.AuthUserID) error {
	return s.txManager.RunInTx(ctx, func(tx Tx) error {
		user, err := s.users.FindUserByID(tx, userID)
		if err != nil {
			return err
		}
		if user.PasswordHash == telegramPasswordHash {
			return entity.ErrLastAuthMethod
		}
		return s.identities.DeleteIdentity(tx, userID, entity.AuthProviderTelegram)
	})
}
