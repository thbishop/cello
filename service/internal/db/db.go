//go:generate moq -out ../../test/testhelpers/dbClientMock.go -pkg testhelpers . Client:DBClientMock

package db

import (
	"context"
	"time"

	"github.com/cello-proj/cello/internal/types"

	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/postgresql"
)

type ProjectEntry struct {
	ProjectID  string `db:"project"`
	Repository string `db:"repository"`
}

type TokenEntry struct {
	CreatedAt string `db:"created_at"`
	ExpiresAt string `db:"expires_at"`
	ProjectID string `db:"project"`
	TokenID   string `db:"token_id"`
}

// IsEmpty returns whether a struct is empty.
func (t TokenEntry) IsEmpty() bool {
	return t == (TokenEntry{})
}

type TargetEntry struct {
	CreatedAt  time.Time        `db:"created_at"`
	Name       string           `db:"name"`
	Project    string           `db:"project"`
	Properties postgresql.JSONB `db:"properties"`
	Type       string           `db:"type"`
	UpdatedAt  time.Time        `db:"updated_at"`
}

// Client allows for db crud operations
type Client interface {
	CreateProjectEntry(ctx context.Context, pe ProjectEntry) error
	DeleteProjectEntry(ctx context.Context, project string) error
	ReadProjectEntry(ctx context.Context, project string) (ProjectEntry, error)
	CreateTargetEntry(ctx context.Context, project string, target types.Target) error
	CreateIfMissingTargetEntry(ctx context.Context, project string, target types.Target) error
	DeleteTargetEntry(ctx context.Context, project, targetName string) error
	ListTargetEntries(ctx context.Context, project string) ([]TargetEntry, error)
	ReadTargetEntry(ctx context.Context, project, targetName string) (TargetEntry, error)
	UpdateTargetEntry(ctx context.Context, project string, target types.Target) error
	UpsertTargetEntry(ctx context.Context, project string, target types.Target) error
	CreateTokenEntry(ctx context.Context, token types.Token) error
	DeleteTokenEntry(ctx context.Context, token string) error
	ReadTokenEntry(ctx context.Context, token string) (TokenEntry, error)
	ListTokenEntries(ctx context.Context, project string) ([]TokenEntry, error)
	Health(ctx context.Context) error
}

// SQLClient allows for db crud operations using postgres db
type SQLClient struct {
	host     string
	database string
	user     string
	password string
	options  map[string]string
}

const (
	ProjectEntryDB = "projects"
	TargetEntryDB  = "targets"
	TokenEntryDB   = "tokens"
)

func NewSQLClient(host, database, user, password string, options map[string]string) (SQLClient, error) {
	return SQLClient{
		host:     host,
		database: database,
		user:     user,
		password: password,
		options:  options,
	}, nil
}

func (d SQLClient) createSession() (db.Session, error) {
	settings := postgresql.ConnectionURL{
		Host:     d.host,
		Database: d.database,
		User:     d.user,
		Password: d.password,
		Options:  d.options,
	}

	return postgresql.Open(settings)
}

func (d SQLClient) Health(ctx context.Context) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.WithContext(ctx).Ping()
}

func (d SQLClient) CreateProjectEntry(ctx context.Context, pe ProjectEntry) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.WithContext(ctx).Tx(func(sess db.Session) error {
		if err := sess.Collection(ProjectEntryDB).Find("project", pe.ProjectID).Delete(); err != nil {
			return err
		}

		if _, err = sess.Collection(ProjectEntryDB).Insert(pe); err != nil {
			return err
		}

		return nil
	})
}

func (d SQLClient) ReadProjectEntry(ctx context.Context, project string) (ProjectEntry, error) {
	res := ProjectEntry{}

	sess, err := d.createSession()
	if err != nil {
		return res, err
	}
	defer sess.Close()

	err = sess.WithContext(ctx).Collection(ProjectEntryDB).Find("project", project).One(&res)
	return res, err
}

func (d SQLClient) DeleteProjectEntry(ctx context.Context, project string) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.WithContext(ctx).Collection(ProjectEntryDB).Find("project", project).Delete()
}

func (d SQLClient) CreateTokenEntry(ctx context.Context, token types.Token) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	err = sess.WithContext(ctx).Tx(func(sess db.Session) error {
		res := TokenEntry{
			CreatedAt: token.CreatedAt,
			ExpiresAt: token.ExpiresAt,
			ProjectID: token.ProjectID,
			TokenID:   token.ProjectToken.ID,
		}

		if _, err = sess.Collection(TokenEntryDB).Insert(res); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (d SQLClient) DeleteTokenEntry(ctx context.Context, token string) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.WithContext(ctx).Collection(TokenEntryDB).Find("token_id", token).Delete()
}

func (d SQLClient) ReadTokenEntry(ctx context.Context, token string) (TokenEntry, error) {
	res := TokenEntry{}
	sess, err := d.createSession()
	if err != nil {
		return res, err
	}
	defer sess.Close()

	err = sess.WithContext(ctx).Collection(TokenEntryDB).Find("token_id", token).One(&res)
	return res, err
}

func (d SQLClient) ListTokenEntries(ctx context.Context, project string) ([]TokenEntry, error) {
	res := []TokenEntry{}

	sess, err := d.createSession()
	if err != nil {
		return res, err
	}
	defer sess.Close()

	err = sess.WithContext(ctx).Collection(TokenEntryDB).Find("project", project).OrderBy("-created_at").All(&res)
	return res, err
}

func (d SQLClient) CreateTargetEntry(ctx context.Context, project string, target types.Target) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.WithContext(ctx).Tx(func(sess db.Session) error {
		now := time.Now().UTC()

		res := TargetEntry{
			CreatedAt:  now,
			Name:       target.Name,
			Project:    project,
			Properties: postgresql.JSONB{V: target.Properties},
			Type:       target.Type,
			UpdatedAt:  now,
		}

		if _, err = sess.Collection(TargetEntryDB).Insert(res); err != nil {
			return err
		}
		return nil
	})
}

func (d SQLClient) CreateIfMissingTargetEntry(ctx context.Context, project string, target types.Target) error {
	// See if it exists
	if _, err := d.ReadTargetEntry(ctx, project, target.Name); err != nil {
		// Doesn't exist, create
		if err == db.ErrNoMoreRows {
			return d.CreateTargetEntry(ctx, project, target)
		}
		return err
	}
	return nil
}

func (d SQLClient) ReadTargetEntry(ctx context.Context, project, targetName string) (TargetEntry, error) {
	res := TargetEntry{}
	sess, err := d.createSession()
	if err != nil {
		return res, err
	}
	defer sess.Close()

	err = sess.WithContext(ctx).Collection(TargetEntryDB).Find(db.Cond{"project": project, "name": targetName}).One(&res)
	return res, err
}

func (d SQLClient) UpdateTargetEntry(ctx context.Context, project string, target types.Target) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.WithContext(ctx).Tx(func(sess db.Session) error {
		now := time.Now().UTC()

		data := map[string]interface{}{
			"properties": postgresql.JSONB{V: target.Properties},
			"updated_at": now,
		}

		return sess.Collection(TargetEntryDB).Find(db.Cond{"project": project, "name": target.Name}).Update(data)
	})
}

func (d SQLClient) DeleteTargetEntry(ctx context.Context, project, targetName string) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.WithContext(ctx).Collection(TargetEntryDB).Find(db.Cond{"project": project, "name": targetName}).Delete()
}

func (d SQLClient) ListTargetEntries(ctx context.Context, project string) ([]TargetEntry, error) {
	res := []TargetEntry{}
	sess, err := d.createSession()
	if err != nil {
		return res, err
	}
	defer sess.Close()

	err = sess.WithContext(ctx).Collection(TargetEntryDB).Find(db.Cond{"project": project}).All(&res)
	return res, err
}

func (d SQLClient) UpsertTargetEntry(ctx context.Context, project string, target types.Target) error {
	// See if it exists
	if _, err := d.ReadTargetEntry(ctx, project, target.Name); err != nil {
		// Doesn't exist, create
		if err == db.ErrNoMoreRows {
			return d.CreateTargetEntry(ctx, project, target)
		}
		return err
	}

	return d.UpdateTargetEntry(ctx, project, target)
}
