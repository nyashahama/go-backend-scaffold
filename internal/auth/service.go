package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/nyashahama/go-backend-scaffold/db/gen"
	"github.com/nyashahama/go-backend-scaffold/internal/notification"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/database"
)

// Sentinel errors.
var (
	ErrEmailExists          = errors.New("email already registered")
	ErrInvalidEmail         = errors.New("invalid email address")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrInvalidOrgSelection  = errors.New("invalid org selection")
	ErrInvalidToken         = errors.New("invalid or expired token")
	ErrOrgSelectionRequired = errors.New("org selection required")
	ErrWeakPassword         = errors.New("password does not meet complexity requirements")
	ErrWrongPassword        = errors.New("current password is incorrect")
)

// Response types returned by the service.

type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserInfo `json:"user"`
	ExpiresIn    int      `json:"expires_in"` // seconds
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type MeResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	OrgID    string `json:"org_id"`
	OrgName  string `json:"org_name"`
	Role     string `json:"role"`
}

// Servicer is the interface Handler depends on (enables test mocks).
type Servicer interface {
	Register(ctx context.Context, email, password, fullName string) (*AuthResponse, error)
	Login(ctx context.Context, email, password, orgID string) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID, orgID string) (*MeResponse, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, password string) error
	ChangePassword(ctx context.Context, userID, currentPassword, nextPassword string) error
}

// Service implements Servicer.
type Service struct {
	db            *database.Pool
	cache         *redis.Client
	sender        notification.Sender
	appBaseURL    string
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

func NewService(
	db *database.Pool,
	cache *redis.Client,
	sender notification.Sender,
	jwtSecret, appBaseURL string,
	jwtExpiry, refreshExpiry time.Duration,
) *Service {
	return &Service{
		db:            db,
		cache:         cache,
		sender:        sender,
		jwtSecret:     jwtSecret,
		appBaseURL:    appBaseURL,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (s *Service) Register(ctx context.Context, email, password, fullName string) (*AuthResponse, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	_, err = s.db.Q.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	var user dbgen.User
	var org dbgen.Org

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		var txErr error
		user, txErr = q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        email,
			PasswordHash: string(hash),
			FullName:     fullName,
		})
		if txErr != nil {
			if isUniqueEmailViolation(txErr) {
				return ErrEmailExists
			}
			return txErr
		}
		org, txErr = q.CreateOrg(ctx, "")
		if txErr != nil {
			return txErr
		}
		_, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: user.ID,
			OrgID:  org.ID,
			Role:   string(RoleAdmin),
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, org.ID, string(RoleAdmin))
}

func (s *Service) Login(ctx context.Context, email, password, orgID string) (*AuthResponse, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.db.Q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	memberships, err := s.db.Q.ListOrgMembershipsByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, ErrInvalidCredentials
	}

	m, err := selectMembership(memberships, orgID)
	if err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, user, m.OrgID, m.Role)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	var (
		user            dbgen.User
		newRefreshToken string
		m               dbgen.OrgMembership
	)

	err := database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		rt, txErr := q.ConsumeRefreshToken(ctx, hashOpaqueToken(refreshToken))
		if txErr != nil {
			if errors.Is(txErr, pgx.ErrNoRows) {
				return ErrInvalidToken
			}
			return txErr
		}

		user, txErr = q.GetUserByID(ctx, rt.UserID)
		if txErr != nil {
			if errors.Is(txErr, pgx.ErrNoRows) {
				return ErrInvalidToken
			}
			return txErr
		}

		m, txErr = q.GetOrgMembershipByUser(ctx, dbgen.GetOrgMembershipByUserParams{
			UserID: user.ID,
			OrgID:  rt.OrgID,
		})
		if txErr != nil {
			if errors.Is(txErr, pgx.ErrNoRows) {
				return ErrInvalidToken
			}
			return txErr
		}

		newRefreshToken, txErr = GenerateRefreshToken()
		if txErr != nil {
			return txErr
		}

		_, txErr = q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
			Token:     hashOpaqueToken(newRefreshToken),
			UserID:    user.ID,
			OrgID:     m.OrgID,
			ExpiresAt: time.Now().Add(s.refreshExpiry),
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	accessToken, err := GenerateAccessToken(user.ID.String(), m.OrgID.String(), m.Role, user.TokenVersion, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}

	return &RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		rt, err := q.ConsumeRefreshToken(ctx, hashOpaqueToken(refreshToken))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		if err := q.IncrementUserTokenVersion(ctx, rt.UserID); err != nil {
			return err
		}
		return q.RevokeAllUserRefreshTokens(ctx, rt.UserID)
	})
}

func (s *Service) Me(ctx context.Context, userID, orgID string) (*MeResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.db.Q.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	org, err := s.db.Q.GetOrg(ctx, oid)
	if err != nil {
		return nil, err
	}
	membership, err := s.db.Q.GetOrgMembershipByUser(ctx, dbgen.GetOrgMembershipByUserParams{
		UserID: uid,
		OrgID:  oid,
	})
	if err != nil {
		return nil, err
	}

	return &MeResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		FullName: user.FullName,
		OrgID:    org.ID.String(),
		OrgName:  org.Name,
		Role:     membership.Role,
	}, nil
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil
	}

	user, err := s.db.Q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // silent — prevents email enumeration
	}
	if err != nil {
		return err
	}

	token, err := GenerateRefreshToken()
	if err != nil {
		return err
	}

	key := passwordResetCacheKey(token)
	if err := s.cache.Set(ctx, key, user.ID.String(), time.Hour).Err(); err != nil {
		return err
	}

	if s.appBaseURL == "" {
		return nil // no URL configured; skip email (dev/test)
	}

	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", s.appBaseURL, token)
	if err := s.sender.SendPasswordReset(ctx, user.Email, resetURL); err != nil {
		slog.Default().Error("failed to send password reset email",
			"error", err,
			"user_id", user.ID.String(),
			"email", user.Email,
		)
		return nil
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}

	key := passwordResetCacheKey(token)
	userIDStr, err := s.cache.GetDel(ctx, key).Result()
	if err != nil {
		return ErrInvalidToken
	}

	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		if txErr := q.UpdateUserPassword(ctx, dbgen.UpdateUserPasswordParams{
			ID:           uid,
			PasswordHash: string(hash),
		}); txErr != nil {
			return txErr
		}
		if txErr := q.IncrementUserTokenVersion(ctx, uid); txErr != nil {
			return txErr
		}
		return q.RevokeAllUserRefreshTokens(ctx, uid)
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, nextPassword string) error {
	if err := validatePassword(nextPassword); err != nil {
		return err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidToken
	}

	user, err := s.db.Q.GetUserByID(ctx, uid)
	if err != nil {
		return err
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); compareErr != nil {
		return ErrWrongPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(nextPassword), 12)
	if err != nil {
		return err
	}

	return database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		if txErr := q.UpdateUserPassword(ctx, dbgen.UpdateUserPasswordParams{
			ID:           uid,
			PasswordHash: string(hash),
		}); txErr != nil {
			return txErr
		}
		if txErr := q.IncrementUserTokenVersion(ctx, uid); txErr != nil {
			return txErr
		}
		return q.RevokeAllUserRefreshTokens(ctx, uid)
	})
}

// issueTokens creates an access + refresh token pair and persists the refresh token.
func (s *Service) issueTokens(ctx context.Context, user dbgen.User, orgID uuid.UUID, role string) (*AuthResponse, error) {
	accessToken, err := GenerateAccessToken(user.ID.String(), orgID.String(), role, user.TokenVersion, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	_, err = s.db.Q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		Token:     hashOpaqueToken(refreshToken),
		UserID:    user.ID,
		OrgID:     orgID,
		ExpiresAt: time.Now().Add(s.refreshExpiry),
	})
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
		User: UserInfo{
			ID:       user.ID.String(),
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}

func isUniqueEmailViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func selectMembership(memberships []dbgen.OrgMembership, orgID string) (dbgen.OrgMembership, error) {
	if len(memberships) == 0 {
		return dbgen.OrgMembership{}, ErrInvalidCredentials
	}

	if orgID == "" {
		if len(memberships) == 1 {
			return memberships[0], nil
		}
		return dbgen.OrgMembership{}, ErrOrgSelectionRequired
	}

	requestedOrgID, err := uuid.Parse(orgID)
	if err != nil {
		return dbgen.OrgMembership{}, ErrInvalidOrgSelection
	}

	for _, membership := range memberships {
		if membership.OrgID == requestedOrgID {
			return membership, nil
		}
	}

	return dbgen.OrgMembership{}, ErrInvalidOrgSelection
}
