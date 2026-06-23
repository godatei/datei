package email

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
	"github.com/glasskube/pkg/seekbuf"
	"github.com/godatei/datei/internal/crypto"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Poller fetches mail from every configured account, applies rules, ingests
// matching attachments as files, and performs each rule's action.
type Poller struct {
	db      *pgxpool.Pool
	fileSvc *file.Service
	enc     *crypto.Encryptor
}

// NewPoller creates the IMAP poller. enc decrypts stored account passwords.
func NewPoller(pool *pgxpool.Pool, fileSvc *file.Service, enc *crypto.Encryptor) *Poller {
	return &Poller{db: pool, fileSvc: fileSvc, enc: enc}
}

// RunAll polls every configured account once. Per-account and per-message
// failures are logged and skipped so one bad mailbox never blocks the rest.
func (p *Poller) RunAll(ctx context.Context) {
	accounts, err := db.New(p.db).ListMailAccountProjectionsAll(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "email poller: failed to list accounts", "error", err)
		return
	}
	slog.DebugContext(ctx, "email poller: starting run", "accounts", len(accounts))
	for i := range accounts {
		account := accounts[i]
		if err := p.pollAccount(ctx, &account); err != nil {
			slog.ErrorContext(ctx, "email poller: account failed", "account_id", account.ID, "error", err)
		}
	}
}

func (p *Poller) pollAccount(ctx context.Context, account *db.MailAccountProjection) error {
	rules, err := db.New(p.db).ListEnabledMailRuleProjectionsByAccount(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("list rules: %w", err)
	}
	if len(rules) == 0 {
		slog.DebugContext(ctx, "email poller: account has no enabled rules", "account_id", account.ID)
		return nil
	}
	slog.DebugContext(ctx, "email poller: polling account",
		"account_id", account.ID, "host", account.ImapHost, "rules", len(rules))

	password, err := p.enc.DecryptString(account.PasswordEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt password: %w", err)
	}

	client, err := dialAndLogin(imapConfig{
		host:     account.ImapHost,
		port:     int(account.ImapPort),
		security: Security(account.Security),
		username: account.Username,
		password: password,
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Logout().Wait()
		_ = client.Close()
	}()

	for i := range rules {
		rule := rules[i]
		if err := p.processRule(ctx, client, account, &rule); err != nil {
			slog.ErrorContext(ctx, "email poller: rule failed",
				"account_id", account.ID, "rule_id", rule.ID, "error", err)
		}
	}
	return nil
}

func (p *Poller) processRule(
	ctx context.Context,
	client *imapclient.Client,
	account *db.MailAccountProjection,
	rule *db.MailRuleProjection,
) error {
	selectData, err := client.Select(rule.Folder, nil).Wait()
	if err != nil {
		return fmt.Errorf("select folder %q: %w", rule.Folder, err)
	}
	uidValidity := int64(selectData.UIDValidity)

	criteria := &imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
		Since:   time.Now().AddDate(0, 0, -int(rule.MaxAgeDays)),
	}
	searchData, err := client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	uids := searchData.AllUIDs()
	slog.DebugContext(ctx, "email poller: searched folder",
		"account_id", account.ID, "rule_id", rule.ID, "folder", rule.Folder, "candidates", len(uids))
	for _, uid := range uids {
		if err := p.processMessage(ctx, client, account, rule, uidValidity, uid); err != nil {
			slog.ErrorContext(ctx, "email poller: message failed",
				"account_id", account.ID, "rule_id", rule.ID, "uid", uint32(uid), "error", err)
		}
	}
	return nil
}

func (p *Poller) processMessage(
	ctx context.Context,
	client *imapclient.Client,
	account *db.MailAccountProjection,
	rule *db.MailRuleProjection,
	uidValidity int64,
	uid imap.UID,
) error {
	queries := db.New(p.db)
	exists, err := queries.ProcessedMessageExists(ctx, db.ProcessedMessageExistsParams{
		AccountID: account.ID, Folder: rule.Folder, UidValidity: uidValidity, ImapUid: int64(uid),
	})
	if err != nil {
		return fmt.Errorf("check processed: %w", err)
	}
	if exists {
		slog.DebugContext(ctx, "email poller: skipping already-processed message",
			"account_id", account.ID, "rule_id", rule.ID, "uid", uint32(uid))
		return nil
	}

	matches, err := messageMatches(client, rule, uid)
	if err != nil {
		return err
	}
	if !matches {
		slog.DebugContext(ctx, "email poller: message did not match filters",
			"account_id", account.ID, "rule_id", rule.ID, "uid", uint32(uid))
		return nil
	}

	count, err := p.ingestAttachments(ctx, client, account, rule, uid)
	if err != nil {
		return err
	}
	// Per paperless semantics, only act on mails we actually consume documents
	// from. Non-matching mails are left untouched for other rules to evaluate.
	if count == 0 {
		slog.DebugContext(ctx, "email poller: message had no matching attachments",
			"account_id", account.ID, "rule_id", rule.ID, "uid", uint32(uid))
		return nil
	}
	slog.DebugContext(ctx, "email poller: ingested attachments",
		"account_id", account.ID, "rule_id", rule.ID, "uid", uint32(uid), "attachments", count)

	// Record as processed before performing the action so a failed action never
	// causes the attachments to be ingested twice on the next poll.
	if err := queries.InsertProcessedMessage(ctx, db.InsertProcessedMessageParams{
		AccountID: account.ID, Folder: rule.Folder, UidValidity: uidValidity, ImapUid: int64(uid),
	}); err != nil {
		return fmt.Errorf("record processed: %w", err)
	}

	if err := performAction(client, Action(rule.Action), uid); err != nil {
		return fmt.Errorf("perform action: %w", err)
	}
	return nil
}

// messageMatches reports whether the message satisfies the rule's envelope
// filters. When the rule has no filters it returns true without fetching the
// envelope, avoiding a round trip. Fetching only the envelope (not the body)
// keeps non-matching mail untouched and out of memory.
func messageMatches(client *imapclient.Client, rule *db.MailRuleProjection, uid imap.UID) (bool, error) {
	if !ruleHasFilters(rule) {
		return true, nil
	}
	buffers, err := client.Fetch(imap.UIDSetNum(uid), &imap.FetchOptions{Envelope: true}).Collect()
	if err != nil {
		return false, fmt.Errorf("fetch envelope: %w", err)
	}
	if len(buffers) == 0 {
		return false, nil
	}
	return messageMatchesFilters(buffers[0].Envelope, rule), nil
}

func ruleHasFilters(rule *db.MailRuleProjection) bool {
	hasFrom := rule.FilterFrom != nil && strings.TrimSpace(*rule.FilterFrom) != ""
	hasSubject := rule.FilterSubject != nil && strings.TrimSpace(*rule.FilterSubject) != ""
	return hasFrom || hasSubject
}

// ingestAttachments fetches the message body and ingests every matching
// attachment, returning the count ingested.
func (p *Poller) ingestAttachments(
	ctx context.Context,
	client *imapclient.Client,
	account *db.MailAccountProjection,
	rule *db.MailRuleProjection,
	uid imap.UID,
) (int, error) {
	cmd := client.Fetch(imap.UIDSetNum(uid), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{{}},
	})
	defer func() { _ = cmd.Close() }()

	msg := cmd.Next()
	if msg == nil {
		return 0, errors.New("message not found")
	}
	for {
		item := msg.Next()
		if item == nil {
			return 0, nil
		}
		body, ok := item.(imapclient.FetchItemDataBodySection)
		if ok && body.Literal != nil {
			return p.ingestFromLiteral(ctx, account, rule, body.Literal)
		}
	}
}

// ingestFromLiteral drains the streamed body into a disk-backed buffer, then
// parses it and ingests matching attachments. Draining to disk first keeps the
// full message out of memory and releases the IMAP connection (and its read
// timeout) before the slower uploads to object storage run.
func (p *Poller) ingestFromLiteral(
	ctx context.Context,
	account *db.MailAccountProjection,
	rule *db.MailRuleProjection,
	literal io.Reader,
) (int, error) {
	buf, err := seekbuf.New(literal)
	if err != nil {
		return 0, fmt.Errorf("buffer message body: %w", err)
	}
	defer func() { _ = buf.Destroy() }()
	reader, err := buf.Get()
	if err != nil {
		return 0, fmt.Errorf("open message buffer: %w", err)
	}
	defer func() { _ = reader.Close() }()

	mr, err := mail.CreateReader(reader)
	if err != nil {
		return 0, fmt.Errorf("parse message: %w", err)
	}

	var count int
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return count, fmt.Errorf("read part: %w", err)
		}

		header, ok := part.Header.(*mail.AttachmentHeader)
		if !ok {
			continue
		}
		filename, err := header.Filename()
		if err != nil || filename == "" {
			continue
		}
		if !attachmentMatches(filename, rule.AttachmentPattern) {
			continue
		}
		contentType, _, _ := header.ContentType()
		slog.DebugContext(ctx, "email poller: ingesting attachment",
			"account_id", account.ID, "rule_id", rule.ID, "filename", filename, "content_type", contentType)
		if _, err := p.fileSvc.CreateFileForOwner(ctx, account.OwnerID, file.CreateFileInput{
			ParentID:    rule.TargetDirectoryID,
			Reader:      part.Body,
			FileName:    filename,
			ContentType: contentType,
		}); err != nil {
			return count, fmt.Errorf("ingest attachment %q: %w", filename, err)
		}
		count++
	}
	return count, nil
}

// attachmentMatches reports whether filename matches the rule's comma-separated
// glob patterns. An empty pattern matches everything.
func attachmentMatches(filename string, pattern *string) bool {
	if pattern == nil || strings.TrimSpace(*pattern) == "" {
		return true
	}
	name := strings.ToLower(filename)
	for _, raw := range strings.Split(*pattern, ",") {
		glob := strings.ToLower(strings.TrimSpace(raw))
		if glob == "" {
			continue
		}
		if ok, _ := path.Match(glob, name); ok {
			return true
		}
	}
	return false
}

func messageMatchesFilters(env *imap.Envelope, rule *db.MailRuleProjection) bool {
	if env == nil {
		return false
	}
	if rule.FilterFrom != nil {
		needle := strings.ToLower(strings.TrimSpace(*rule.FilterFrom))
		if needle != "" && !fromMatches(env.From, needle) {
			return false
		}
	}
	if rule.FilterSubject != nil {
		needle := strings.ToLower(strings.TrimSpace(*rule.FilterSubject))
		if needle != "" && !strings.Contains(strings.ToLower(env.Subject), needle) {
			return false
		}
	}
	return true
}

func fromMatches(addrs []imap.Address, needle string) bool {
	for i := range addrs {
		if strings.Contains(strings.ToLower(addrs[i].Addr()), needle) ||
			strings.Contains(strings.ToLower(addrs[i].Name), needle) {
			return true
		}
	}
	return false
}

func performAction(client *imapclient.Client, action Action, uid imap.UID) error {
	switch action {
	case ActionMarkAsRead:
		flags := &imap.StoreFlags{Op: imap.StoreFlagsAdd, Flags: []imap.Flag{imap.FlagSeen}, Silent: true}
		if err := client.Store(imap.UIDSetNum(uid), flags, nil).Close(); err != nil {
			return fmt.Errorf("mark as read: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}
