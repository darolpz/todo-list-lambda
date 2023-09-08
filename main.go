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
	StartCommand   Command = "start"
	AddTaskCommand Command = "add_task"
	AddListCommand Command = "add_list"
	TaskCommand    Command = "tasks"
	ListCommand    Command = "list"
	BotCommand             = "bot_command"
)

type CallbackData struct {
	UserID string `json:"user_id"`
	TaskID string `json:"task_id"`
}

type Update struct {
	UpdateID      int            `json:"update_id"`
	Message       Message        `json:"message"`
	CallbackQuery *CallBackQuery `json:"callback_query,omitempty"`
}

type CallBackQuery struct {
	Data    string  `json:"data"`
	From    User    `json:"from"`
	Message Message `json:"message"`
}

type Entity struct {
	Type string `json:"type"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type Chat struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}
type Message struct {
	MessageID int      `json:"message_id"`
	From      User     `json:"from"`
	Chat      Chat     `json:"chat"`
	Date      int      `json:"date"`
	Text      string   `json:"text"`
	Entities  []Entity `json:"entities,omitempty"`
}

func (u Update) IsCommand() bool {
	return u.Message.Entities != nil && u.Message.Entities[0].Type == BotCommand
}

func (u Update) IsCallback() bool {
	return u.CallbackQuery != nil
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
	fmt.Printf("request: %+v\n", request.Body)
	update := Update{}
	if err := json.Unmarshal([]byte(request.Body), &update); err != nil {
		log.Printf("could not unmarshal request: %s\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
		}, nil
	}

	fmt.Printf("update: %+v\n", update)
	if !validateUser(update) {
		log.Printf("unauthorized user: user id %s", update.Message.From.Username)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
		}, nil
	}

	if update.IsCommand() {
		if err := h.handleCommand(ctx, update); err != nil {
			log.Printf("could not handle command: %s\n", err)
			// _ = sendMessage(ctx, fmt.Sprintf("could not handle command: %s\n", err), update.Message.Chat.ID)
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
			}, nil
		}
	}

	if update.IsCallback() {
		if err := h.handleCallback(ctx, update); err != nil {
			log.Printf("could not handle callback: %s\n", err)
			// _ = sendMessage(ctx, fmt.Sprintf("could not handle command: %s\n", err), update.Message.Chat.ID)
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
			}, nil
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "message processed succesfully",
	}, nil
}

func (h handler) handleCommand(ctx context.Context, u Update) error {
	if !u.IsCommand() {
		return errors.New("update is not a command")
	}

	parts := strings.Fields(u.Message.Text)

	command := Command(parts[0][1:])
	switch command {
	case AddTaskCommand:
		return h.AddTask(ctx, u.Message)
	case TaskCommand:
		return h.GetTaskList(ctx)
	case StartCommand:
		return nil
	default:
		return errors.New("invalid command")
	}
}

func (h handler) handleCallback(ctx context.Context, u Update) error {
	taskID := u.CallbackQuery.Data
	if err := h.dynamoDB.DeleteTask(ctx, fmt.Sprintf("%d", u.CallbackQuery.From.ID), taskID); err != nil {
		log.Printf("could not delete item: %s\n", err)
		return err
	}

	if err := sendMessage(ctx, "task deleted succesfully", u.CallbackQuery.Message.Chat.ID); err != nil {
		log.Printf("could not send message: %s\n", err)
		return err
	}
	return nil
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

		err = sendMessage(ctx, t.Text, chatID, WithCallback(string(t.TaskID)))
		if err != nil {
			log.Printf("could not send task: %s\n", err)
			return err
		}
	}
	return nil
}

func validateUser(u Update) bool {
	// This is a personal project, I validate if the user id is my own user id and if it is my own chat
	return (strconv.Itoa(u.Message.From.ID) == USER_ID && strconv.Itoa(u.Message.Chat.ID) == USER_ID) ||
		(strconv.Itoa(u.CallbackQuery.From.ID) == USER_ID && strconv.Itoa(u.CallbackQuery.Message.Chat.ID) == USER_ID)
}
