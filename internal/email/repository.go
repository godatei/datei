package email

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountRepository persists MailAccount aggregates.
type AccountRepository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*MailAccount, error)
	Save(ctx context.Context, account *MailAccount) error
}

// RuleRepository persists MailRule aggregates.
type RuleRepository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*MailRule, error)
	Save(ctx context.Context, rule *MailRule) error
}

type accountRepository struct {
	base events.GenericRepository
}

// NewAccountRepository creates a mail account repository.
func NewAccountRepository(pool *pgxpool.Pool, eventStore events.EventStore) AccountRepository {
	return &accountRepository{
		base: events.NewGenericRepository(pool, eventStore, "mail_account", updateAccountProjection),
	}
}

func (r *accountRepository) LoadByID(ctx context.Context, id uuid.UUID) (*MailAccount, error) {
	agg := &MailAccount{}
	if err := r.base.LoadByID(ctx, id, agg); err != nil {
		return nil, err
	}
	return agg, nil
}

func (r *accountRepository) Save(ctx context.Context, agg *MailAccount) error {
	return r.base.Save(ctx, agg)
}

type ruleRepository struct {
	base events.GenericRepository
}

// NewRuleRepository creates a mail rule repository.
func NewRuleRepository(pool *pgxpool.Pool, eventStore events.EventStore) RuleRepository {
	return &ruleRepository{
		base: events.NewGenericRepository(pool, eventStore, "mail_rule", updateRuleProjection),
	}
}

func (r *ruleRepository) LoadByID(ctx context.Context, id uuid.UUID) (*MailRule, error) {
	agg := &MailRule{}
	if err := r.base.LoadByID(ctx, id, agg); err != nil {
		return nil, err
	}
	return agg, nil
}

func (r *ruleRepository) Save(ctx context.Context, agg *MailRule) error {
	return r.base.Save(ctx, agg)
}

func updateAccountProjection(ctx context.Context, q *db.Queries, event events.DomainEvent) error {
	switch e := event.(type) {
	case MailAccountCreatedEvent:
		return updateProjectionForMailAccountCreated(ctx, q, &e)
	case MailAccountUpdatedEvent:
		return updateProjectionForMailAccountUpdated(ctx, q, &e)
	case MailAccountDeletedEvent:
		return updateProjectionForMailAccountDeleted(ctx, q, &e)
	default:
		return fmt.Errorf("unknown mail account event type: %s", event.EventType())
	}
}

func updateRuleProjection(ctx context.Context, q *db.Queries, event events.DomainEvent) error {
	switch e := event.(type) {
	case MailRuleCreatedEvent:
		return updateProjectionForMailRuleCreated(ctx, q, &e)
	case MailRuleUpdatedEvent:
		return updateProjectionForMailRuleUpdated(ctx, q, &e)
	case MailRuleDeletedEvent:
		return updateProjectionForMailRuleDeleted(ctx, q, &e)
	default:
		return fmt.Errorf("unknown mail rule event type: %s", event.EventType())
	}
}
