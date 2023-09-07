package main

import (
	"context"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	TaskStatusTODO = "TODO"
	TaskStatusDONE = "DONE"
	TasksTable     = "tasks"
)

type Task struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	Date      string `json:"date"`
	Text      string `json:"text"`
	ChatID    string `json:"chat_id"`
}

type DynamoDB interface {
	AddTaskToDynamoDB(ctx context.Context, m Message) error
	ListPendingTasks(ctx context.Context) ([]Task, error)
}

type dynamoDB struct {
	client *dynamodb.Client
}

func NewDynamoDBRepository() (DynamoDB, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("could not load default config: %s\n", err)
		return nil, err
	}

	// Create a DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	return &dynamoDB{
		client: client,
	}, nil
}

func (d *dynamoDB) AddTaskToDynamoDB(ctx context.Context, m Message) error {
	// Define the DynamoDB PutItem input
	input := &dynamodb.PutItemInput{
		TableName: aws.String("tasks"),
		Item:      itemDataToAttributeValueMap(m),
	}

	// Perform the PutItem operation
	_, err := d.client.PutItem(ctx, input)
	if err != nil {
		log.Printf("could not put item in dynamo db: %s\n", err)
		return err
	}

	return nil
}

func (d *dynamoDB) ListPendingTasks(ctx context.Context) ([]Task, error) {
	// Define the query parameters using expression builder
	query := &dynamodb.QueryInput{
		TableName:              aws.String(TasksTable),
		KeyConditionExpression: aws.String("#s = :value"), // Use an alias for "status"
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":value": &types.AttributeValueMemberS{Value: "TODO"},
		},
		ExpressionAttributeNames: map[string]string{
			"#s": "status", // Define an alias for the "status" attribute
		},
	}

	result, err := d.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	// Iterate over the items in result.Items
	for _, item := range result.Items {
		// Access attribute values by attribute name
		task := Task{
			MessageID: item["message_id"].(*types.AttributeValueMemberS).Value,
			Status:    item["status"].(*types.AttributeValueMemberS).Value,
			Date:      item["date"].(*types.AttributeValueMemberS).Value,
			Text:      item["text"].(*types.AttributeValueMemberS).Value,
			ChatID:    item["chat_id"].(*types.AttributeValueMemberS).Value,
		}
		tasks = append(tasks, task)
	}

	log.Printf("tasks: %+v", tasks)

	return tasks, nil
}

func itemDataToAttributeValueMap(m Message) map[string]types.AttributeValue {
	attrMap := make(map[string]types.AttributeValue)

	// Convert Message fields to DynamoDB attribute values
	attrMap["message_id"] = &types.AttributeValueMemberS{Value: strconv.Itoa(m.MessageID)}
	attrMap["status"] = &types.AttributeValueMemberS{Value: TaskStatusTODO}

	attrMap["text"] = &types.AttributeValueMemberS{Value: m.Text}
	attrMap["date"] = &types.AttributeValueMemberS{Value: strconv.Itoa(m.Date)}
	attrMap["chat_id"] = &types.AttributeValueMemberS{Value: strconv.Itoa(m.Chat.ID)}

	return attrMap
}
