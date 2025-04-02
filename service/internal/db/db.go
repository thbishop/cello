//go:generate moq -out ../../test/testhelpers/dbClientMock.go -pkg testhelpers . Client:DBClientMock

package db

import (
	"context"
	"fmt"

	"github.com/cello-proj/cello/internal/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/postgresql"
)

type ProjectEntry struct {
	ProjectID  string `db:"project" dynamodbav:"pk"`
	Repository string `db:"repository" dynamodbav:"repository"`
}

type TokenEntry struct {
	CreatedAt string `db:"created_at" dynamodbav:"created_at"`
	ExpiresAt string `db:"expires_at" dynamodbav:"expires_at"`
	ProjectID string `db:"project" dynamodbav:"-"` // ignore in ddb as it's in pk
	TokenID   string `db:"token_id" dynamodbav:"token_id"`
}

// IsEmpty returns whether a struct is empty.
func (t TokenEntry) IsEmpty() bool {
	return t == (TokenEntry{})
}

// Client allows for db crud operations
type Client interface {
	CreateProjectEntry(ctx context.Context, pe ProjectEntry) error
	DeleteProjectEntry(ctx context.Context, project string) error
	ReadProjectEntry(ctx context.Context, project string) (ProjectEntry, error)
	CreateTokenEntry(ctx context.Context, token types.Token) error
	DeleteTokenEntry(ctx context.Context, project, token string) error
	ReadTokenEntry(ctx context.Context, project, token string) (TokenEntry, error)
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

func (d SQLClient) DeleteTokenEntry(ctx context.Context, project, token string) error {
	sess, err := d.createSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.WithContext(ctx).Collection(TokenEntryDB).Find("token_id", token).Delete()
}

func (d SQLClient) ReadTokenEntry(ctx context.Context, project, token string) (TokenEntry, error) {
	res := TokenEntry{}
	sess, err := d.createSession()
	if err != nil {
		return res, err
	}
	defer sess.Close()

	// Note: We ignore the project parameter since token_id is unique in PostgreSQL
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

// DDBClient allows for db crud operations using DynamoDB
type DDBClient struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBClient(client *dynamodb.Client, tableName string) *DDBClient {
	return &DDBClient{
		client:    client,
		tableName: tableName,
	}
}

func (d *DDBClient) Health(ctx context.Context) error {
	// No-op as we don't want to incur AWS API costs just for health checks
	return nil
}

func (d *DDBClient) CreateProjectEntry(ctx context.Context, pe ProjectEntry) error {
	item, err := attributevalue.MarshalMap(pe)
	if err != nil {
		return fmt.Errorf("failed to marshal project entry: %w", err)
	}

	item["pk"] = &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", pe.ProjectID)}
	item["sk"] = &dynamodbtypes.AttributeValueMemberS{Value: "META"}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create project entry: %w", err)
	}

	return nil
}

func (d *DDBClient) ReadProjectEntry(ctx context.Context, project string) (ProjectEntry, error) {
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			"pk": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", project)},
			"sk": &dynamodbtypes.AttributeValueMemberS{Value: "META"},
		},
	})
	if err != nil {
		return ProjectEntry{}, fmt.Errorf("failed to get project entry: %w", err)
	}

	if result.Item == nil {
		return ProjectEntry{}, fmt.Errorf("project not found")
	}

	var pe ProjectEntry
	if err = attributevalue.UnmarshalMap(result.Item, &pe); err != nil {
		return ProjectEntry{}, fmt.Errorf("failed to unmarshal project entry: %w", err)
	}

	return pe, nil
}

func (d *DDBClient) DeleteProjectEntry(ctx context.Context, project string) error {
	// Query for all items with this project's pk
	queryResult, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("pk = :project"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":project": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", project)},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to query project entries: %w", err)
	}

	if len(queryResult.Items) == 0 {
		return nil
	}

	// Add all items to transaction for deletion
	var transactItems []dynamodbtypes.TransactWriteItem
	for _, item := range queryResult.Items {
		pk := item["pk"].(*dynamodbtypes.AttributeValueMemberS).Value
		sk := item["sk"].(*dynamodbtypes.AttributeValueMemberS).Value

		transactItems = append(transactItems, dynamodbtypes.TransactWriteItem{
			Delete: &dynamodbtypes.Delete{
				Key: map[string]dynamodbtypes.AttributeValue{
					"pk": &dynamodbtypes.AttributeValueMemberS{Value: pk},
					"sk": &dynamodbtypes.AttributeValueMemberS{Value: sk},
				},
				TableName: aws.String(d.tableName),
			},
		})
	}

	// Execute transaction to delete all items
	_, err = d.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		return fmt.Errorf("failed to delete items in transaction: %w", err)
	}

	return nil
}

func (d *DDBClient) CreateTokenEntry(ctx context.Context, token types.Token) error {
	te := TokenEntry{
		CreatedAt: token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
		ProjectID: token.ProjectID,
		TokenID:   token.ProjectToken.ID,
	}

	item, err := attributevalue.MarshalMap(te)
	if err != nil {
		return fmt.Errorf("failed to marshal token entry: %w", err)
	}

	// Add PK and SK for token entry
	item["pk"] = &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", token.ProjectID)}
	item["sk"] = &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("TOKEN#%s", token.ProjectToken.ID)}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create token entry: %w", err)
	}

	return nil
}

func (d *DDBClient) ReadTokenEntry(ctx context.Context, project, token string) (TokenEntry, error) {
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			"pk": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", project)},
			"sk": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("TOKEN#%s", token)},
		},
	})
	if err != nil {
		return TokenEntry{}, fmt.Errorf("failed to get token entry: %w", err)
	}

	if result.Item == nil {
		return TokenEntry{}, fmt.Errorf("token not found")
	}

	var te TokenEntry
	if err = attributevalue.UnmarshalMap(result.Item, &te); err != nil {
		return TokenEntry{}, fmt.Errorf("failed to unmarshal token entry: %w", err)
	}

	return te, nil
}

func (d *DDBClient) DeleteTokenEntry(ctx context.Context, project, token string) error {
	_, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			"pk": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", project)},
			"sk": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("TOKEN#%s", token)},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete token entry: %w", err)
	}

	return nil
}

func (d *DDBClient) ListTokenEntries(ctx context.Context, project string) ([]TokenEntry, error) {
	result, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("pk = :project AND begins_with(sk, :token_prefix)"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":project":      &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", project)},
			":token_prefix": &dynamodbtypes.AttributeValueMemberS{Value: "TOKEN#"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query token entries: %w", err)
	}

	var entries []TokenEntry
	err = attributevalue.UnmarshalListOfMaps(result.Items, &entries)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token entries: %w", err)
	}

	return entries, nil
}
