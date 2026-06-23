package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/crypto"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service exposes mail account and rule management plus IMAP connection testing.
type Service struct {
	db          *pgxpool.Pool
	accountRepo AccountRepository
	ruleRepo    RuleRepository
	enc         *crypto.Encryptor
}

// NewService creates the email service. enc seals IMAP passwords at rest.
func NewService(
	pool *pgxpool.Pool,
	accountRepo AccountRepository,
	ruleRepo RuleRepository,
	enc *crypto.Encryptor,
) *Service {
	return &Service{db: pool, accountRepo: accountRepo, ruleRepo: ruleRepo, enc: enc}
}

func clampPaging(limit, offset int) (int32, int32) {
	l := int32(limit)
	if l <= 0 {
		l = 100
	}
	o := int32(offset)
	if o < 0 {
		o = 0
	}
	return l, o
}

// ============================================================================
// Mail Accounts
// ============================================================================

type CreateAccountInput struct {
	Name     string
	ImapHost string
	ImapPort int
	Username string
	Password string
	Security Security
}

type UpdateAccountInput struct {
	Name     string
	ImapHost string
	ImapPort int
	Username string
	// Password, when nil, keeps the currently stored password.
	Password *string
	Security Security
}

type ListAccountsOutput struct {
	Items []api.MailAccount
	Total int
}

func (s *Service) ListAccounts(ctx context.Context, limit, offset int) (*ListAccountsOutput, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID
	queries := db.New(s.db)
	lim, off := clampPaging(limit, offset)

	total, err := queries.CountMailAccountProjectionsByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	accounts, err := queries.ListMailAccountProjectionsByOwner(ctx, db.ListMailAccountProjectionsByOwnerParams{
		OwnerID: ownerID, Lim: lim, Off: off,
	})
	if err != nil {
		return nil, err
	}
	return &ListAccountsOutput{
		Items: MapAccountProjectionSliceToAPI(accounts),
		Total: int(total),
	}, nil
}

func (s *Service) GetAccount(ctx context.Context, id uuid.UUID) (*api.MailAccount, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID
	account, err := s.loadOwnedAccountProjection(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	return MapAccountProjectionToAPI(&account), nil
}

func (s *Service) CreateAccount(ctx context.Context, input CreateAccountInput) (*api.MailAccount, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID

	encrypted, err := s.enc.EncryptString(input.Password)
	if err != nil {
		return nil, err
	}

	agg := &MailAccount{}
	if err := agg.Create(
		uuid.New(), ownerID, input.Name, input.ImapHost, input.ImapPort,
		input.Username, encrypted, input.Security, time.Now(),
	); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, err.Error())
	}
	if err := verifyConnection(
		input.ImapHost, input.ImapPort, input.Security, input.Username, input.Password,
	); err != nil {
		return nil, err
	}
	if err := s.accountRepo.Save(ctx, agg); err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	account, err := queries.GetMailAccountProjection(ctx, agg.ID)
	if err != nil {
		return nil, err
	}
	return MapAccountProjectionToAPI(&account), nil
}

func (s *Service) UpdateAccount(ctx context.Context, id uuid.UUID, input UpdateAccountInput) (*api.MailAccount, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID

	current, err := s.loadOwnedAccountProjection(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}

	encrypted := current.PasswordEncrypted
	var password string
	if input.Password != nil {
		password = *input.Password
		encrypted, err = s.enc.EncryptString(password)
		if err != nil {
			return nil, err
		}
	} else {
		password, err = s.enc.DecryptString(current.PasswordEncrypted)
		if err != nil {
			return nil, err
		}
	}

	agg, err := s.accountRepo.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := agg.Update(
		input.Name, input.ImapHost, input.ImapPort, input.Username,
		encrypted, input.Security, time.Now(),
	); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, err.Error())
	}
	if err := verifyConnection(
		input.ImapHost, input.ImapPort, input.Security, input.Username, password,
	); err != nil {
		return nil, err
	}
	if err := s.accountRepo.Save(ctx, agg); err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	account, err := queries.GetMailAccountProjection(ctx, id)
	if err != nil {
		return nil, err
	}
	return MapAccountProjectionToAPI(&account), nil
}

func (s *Service) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	ownerID := authn.RequireCurrentUser(ctx).ID
	if _, err := s.loadOwnedAccountProjection(ctx, id, ownerID); err != nil {
		return err
	}
	agg, err := s.accountRepo.LoadByID(ctx, id)
	if err != nil {
		return err
	}
	if err := agg.Delete(time.Now()); err != nil {
		return fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, err.Error())
	}
	return s.accountRepo.Save(ctx, agg)
}

// TestConnection verifies the IMAP credentials of an owned account.
func (s *Service) TestConnection(ctx context.Context, id uuid.UUID) (*api.TestMailAccountResponse, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID
	account, err := s.loadOwnedAccountProjection(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}

	password, err := s.enc.DecryptString(account.PasswordEncrypted)
	if err != nil {
		return nil, err
	}

	if err := verifyIMAPConnection(imapConfig{
		host:     account.ImapHost,
		port:     int(account.ImapPort),
		security: Security(account.Security),
		username: account.Username,
		password: password,
	}); err != nil {
		msg := err.Error()
		return &api.TestMailAccountResponse{Success: false, Message: &msg}, nil
	}

	return &api.TestMailAccountResponse{Success: true}, nil
}

// verifyConnection runs an IMAP connection test and wraps any failure in
// ErrConnectionTestFailed so callers can surface the reason to the user.
func verifyConnection(host string, port int, security Security, username, password string) error {
	if err := verifyIMAPConnection(imapConfig{
		host:     host,
		port:     port,
		security: security,
		username: username,
		password: password,
	}); err != nil {
		return fmt.Errorf("%w: %s", apperrors.ErrConnectionTestFailed, err.Error())
	}
	return nil
}

func (s *Service) loadOwnedAccountProjection(
	ctx context.Context, id, ownerID uuid.UUID,
) (db.MailAccountProjection, error) {
	account, err := db.New(s.db).GetMailAccountProjectionForOwner(ctx, db.GetMailAccountProjectionForOwnerParams{
		ID: id, OwnerID: ownerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.MailAccountProjection{}, apperrors.ErrNotFound
	}
	return account, err
}

// ============================================================================
// Mail Rules
// ============================================================================

type ListRulesOutput struct {
	Items []api.MailRule
	Total int
}

// RuleInput holds the request-shaped rule fields shared by create and update.
// Optional fields are pointers; toSpec resolves them into a validated RuleSpec,
// applying the rule defaults (order 0, enabled, folder INBOX).
type RuleInput struct {
	Name              string
	Order             *int
	Enabled           *bool
	Folder            *string
	FilterFrom        *string
	FilterSubject     *string
	MaxAgeDays        int
	AttachmentPattern *string
	Action            Action
	TargetDirectoryID *uuid.UUID
}

func (in RuleInput) toSpec() RuleSpec {
	order := 0
	if in.Order != nil {
		order = *in.Order
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	folder := "INBOX"
	if in.Folder != nil && *in.Folder != "" {
		folder = *in.Folder
	}
	return RuleSpec{
		Name:              in.Name,
		Order:             order,
		Enabled:           enabled,
		Folder:            folder,
		FilterFrom:        in.FilterFrom,
		FilterSubject:     in.FilterSubject,
		MaxAgeDays:        in.MaxAgeDays,
		AttachmentPattern: in.AttachmentPattern,
		Action:            in.Action,
		TargetDirectoryID: in.TargetDirectoryID,
	}
}

// ListAllRules returns every rule across all accounts owned by the current user.
func (s *Service) ListAllRules(ctx context.Context, limit, offset int) (*ListRulesOutput, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID

	queries := db.New(s.db)
	lim, off := clampPaging(limit, offset)
	total, err := queries.CountMailRuleProjectionsByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	rules, err := queries.ListMailRuleProjectionsByOwner(ctx, db.ListMailRuleProjectionsByOwnerParams{
		OwnerID: ownerID, Lim: lim, Off: off,
	})
	if err != nil {
		return nil, err
	}
	return &ListRulesOutput{
		Items: MapRuleProjectionSliceToAPI(rules),
		Total: int(total),
	}, nil
}

func (s *Service) GetRule(ctx context.Context, ruleID uuid.UUID) (*api.MailRule, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID
	rule, err := s.loadOwnedRuleProjection(ctx, ruleID, ownerID)
	if err != nil {
		return nil, err
	}
	return MapRuleProjectionToAPI(&rule), nil
}

func (s *Service) CreateRule(ctx context.Context, accountID uuid.UUID, in RuleInput) (*api.MailRule, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID
	if _, err := s.loadOwnedAccountProjection(ctx, accountID, ownerID); err != nil {
		return nil, err
	}
	spec := in.toSpec()
	if err := s.validateTargetDirectory(ctx, ownerID, spec.TargetDirectoryID); err != nil {
		return nil, err
	}

	agg := &MailRule{}
	if err := agg.Create(uuid.New(), accountID, spec, time.Now()); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, err.Error())
	}
	if err := s.ruleRepo.Save(ctx, agg); err != nil {
		return nil, err
	}

	rule, err := db.New(s.db).GetMailRuleProjection(ctx, agg.ID)
	if err != nil {
		return nil, err
	}
	return MapRuleProjectionToAPI(&rule), nil
}

// UpdateRule updates a rule. When accountID is non-nil, the rule is moved to
// that account (which must be owned by the caller); otherwise it keeps its
// current account.
func (s *Service) UpdateRule(
	ctx context.Context, ruleID uuid.UUID, accountID *uuid.UUID, in RuleInput,
) (*api.MailRule, error) {
	ownerID := authn.RequireCurrentUser(ctx).ID
	current, err := s.loadOwnedRuleProjection(ctx, ruleID, ownerID)
	if err != nil {
		return nil, err
	}
	spec := in.toSpec()
	if err := s.validateTargetDirectory(ctx, ownerID, spec.TargetDirectoryID); err != nil {
		return nil, err
	}

	targetAccountID := current.AccountID
	if accountID != nil && *accountID != current.AccountID {
		if _, err := s.loadOwnedAccountProjection(ctx, *accountID, ownerID); err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				return nil, fmt.Errorf("%w: target account not found", apperrors.ErrInvalidInput)
			}
			return nil, err
		}
		targetAccountID = *accountID
	}

	agg, err := s.ruleRepo.LoadByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if err := agg.Update(targetAccountID, spec, time.Now()); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, err.Error())
	}
	if err := s.ruleRepo.Save(ctx, agg); err != nil {
		return nil, err
	}

	rule, err := db.New(s.db).GetMailRuleProjection(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	return MapRuleProjectionToAPI(&rule), nil
}

func (s *Service) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	ownerID := authn.RequireCurrentUser(ctx).ID
	if _, err := s.loadOwnedRuleProjection(ctx, ruleID, ownerID); err != nil {
		return err
	}
	agg, err := s.ruleRepo.LoadByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if err := agg.Delete(time.Now()); err != nil {
		return fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, err.Error())
	}
	return s.ruleRepo.Save(ctx, agg)
}

// loadOwnedRuleProjection returns a rule only if its account belongs to ownerID;
// otherwise ErrNotFound, so cross-user access is indistinguishable from missing.
func (s *Service) loadOwnedRuleProjection(
	ctx context.Context, ruleID, ownerID uuid.UUID,
) (db.MailRuleProjection, error) {
	queries := db.New(s.db)
	rule, err := queries.GetMailRuleProjection(ctx, ruleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.MailRuleProjection{}, apperrors.ErrNotFound
	} else if err != nil {
		return db.MailRuleProjection{}, err
	}
	if _, err := s.loadOwnedAccountProjection(ctx, rule.AccountID, ownerID); err != nil {
		return db.MailRuleProjection{}, err
	}
	return rule, nil
}

// validateTargetDirectory ensures the target, when set, is a directory owned by
// the user.
func (s *Service) validateTargetDirectory(ctx context.Context, ownerID uuid.UUID, dirID *uuid.UUID) error {
	if dirID == nil {
		return nil
	}
	dir, err := db.New(s.db).GetFileProjectionForUser(ctx, db.GetFileProjectionForUserParams{
		UserID: ownerID, ID: *dirID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: target directory not found", apperrors.ErrInvalidInput)
	} else if err != nil {
		return err
	}
	if !dir.IsDirectory {
		return fmt.Errorf("%w: target is not a directory", apperrors.ErrInvalidInput)
	}
	return nil
}
