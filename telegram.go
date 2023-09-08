package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type ResponseBody struct {
	ChatID      int                   `json:"chat_id"`
	Text        string                `json:"text"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type sendMessageOptions func(*ResponseBody)

func sendMessage(ctx context.Context, message string, chatID int, options ...sendMessageOptions) error {
	resBody := ResponseBody{
		ChatID: chatID,
		Text:   message,
	}

	for _, opt := range options {
		opt(&resBody)
	}

	// Create the JSON body from the struct
	resBytes, err := json.Marshal(resBody)
	if err != nil {
		log.Printf("could not marshal response: %s\n", err)
		return err
	}

	// Send a post request with your token
	response, err := http.Post(
		SEND_ENDPOINT,
		"application/json",
		bytes.NewBuffer(resBytes))
	if err != nil {
		log.Printf("could not send message: %s\n", err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Printf("unexpected status: %s\n", err)
		body, _ := io.ReadAll(response.Body)
		log.Printf("body: %s", string(body))
		return err
	}

	return nil
}

func WithCallback(taskID string) sendMessageOptions {
	return func(rb *ResponseBody) {
		// Crear un botón inline
		inlineKeyboard := [][]InlineKeyboardButton{
			{
				{Text: "❌", CallbackData: taskID},
			},
		}
		rb.ReplyMarkup = &InlineKeyboardMarkup{InlineKeyboard: inlineKeyboard}
	}
}
