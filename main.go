package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Command string

var (
	BOT_TOKEN     = os.Getenv("TELEGRAM_BOT_TOKEN")
	SEND_ENDPOINT = fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", BOT_TOKEN)
	USER_ID       = os.Getenv("USER_ID")
)

const (
	AddTaskCommand Command = "add_task"
	AddListCommand Command = "add_list"
	TaskCommand    Command = "tasks"
	ListCommand    Command = "list"
	BotCommand             = "bot_command"
)

type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}
type Message struct {
	MessageID int `json:"message_id"`
	From      struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"from"`
	Chat struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"chat"`
	Date     int    `json:"date"`
	Text     string `json:"text"`
	Entities []struct {
		Type string `json:"type"`
	} `json:"entities"`
}

func (m Message) IsCommand() bool {
	return m.Entities[0].Type == BotCommand
}

type ResponseBody struct {
	ChatID int    `json:"chat_id"`
	Text   string `json:"text"`
}

type handler struct {
	dynamoDB DynamoDB
}

func newHandler() (handler, error) {
	dynamoDB, err := NewDynamoDBRepository()
	if err != nil {
		return handler{}, err
	}
	return handler{
		dynamoDB: dynamoDB,
	}, nil
}

func main() {
	handler, err := newHandler()
	if err != nil {
		log.Printf("could not create handler: %s\n", err)
		panic(err)
	}
	lambda.Start(handler.Handle)
}

func (h handler) CleanQueue(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("cleaning messages")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "message processed succesfully",
	}, nil
}

func (h handler) Handle(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	update := Update{}
	if err := json.Unmarshal([]byte(request.Body), &update); err != nil {
		log.Printf("could not unmarshal request: %s\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	if !validateUser(update.Message) {
		log.Printf("unauthorized user: user id %s", update.Message.From.Username)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
		}, nil
	}

	log.Printf("message: %+v", update.Message)
	if err := h.handleCommand(ctx, update.Message); err != nil {
		log.Printf("could not handle command: %s\n", err)
		// _ = sendMessage(ctx, fmt.Sprintf("could not handle command: %s\n", err), update.Message.Chat.ID)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "message processed succesfully",
	}, nil
}

func (h handler) handleCommand(ctx context.Context, m Message) error {
	if !m.IsCommand() {
		return errors.New("update is not a command")
	}

	parts := strings.Fields(m.Text)

	command := Command(parts[0][1:])
	switch command {
	case AddTaskCommand:
		return h.AddTask(ctx, m)
	case AddListCommand:
		return AddMarketList(m.Text)
	case TaskCommand:
		return h.GetTaskList(ctx)
	case ListCommand:
		return GetMarketList()
	default:
		return errors.New("invalid command")
	}
}

func (h handler) AddTask(ctx context.Context, m Message) error {
	// remove command from text
	parts := strings.Fields(m.Text)
	if len(parts) <= 1 {
		return errors.New("task is empty")
	}

	m.Text = strings.Join(parts[1:], " ")

	if err := h.dynamoDB.AddTaskToDynamoDB(ctx, m); err != nil {
		log.Printf("could not add task to dynamo db: %s\n", err)
		return err
	}

	if err := sendMessage(ctx, "new task created succesfully", m.Chat.ID); err != nil {
		log.Printf("could not send message: %s\n", err)
		return err
	}
	return nil
}

func (h handler) GetTaskList(ctx context.Context) error {
	tasks, err := h.dynamoDB.ListPendingTasks(ctx)
	if err != nil {
		log.Printf("could not list tasks to dynamo db: %s\n", err)
		return err
	}

	for _, t := range tasks {
		chatID, err := strconv.Atoi(t.ChatID)
		if err != nil {
			log.Printf("chat id is not a number: %s", err)
			return err
		}
		err = sendMessage(ctx, t.Text, chatID)
		if err != nil {
			log.Printf("could not send task: %s\n", err)
			return err
		}
	}
	return nil
}

func AddMarketList(task string) error {
	return nil
}

func GetMarketList() error {
	return nil
}

func validateUser(m Message) bool {
	// This is a personal project, I validate if the user id is my own user id and if it is my own chat
	return strconv.Itoa(m.From.ID) == USER_ID && strconv.Itoa(m.Chat.ID) == USER_ID
}
