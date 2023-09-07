package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
)

func sendMessage(ctx context.Context, message string, chatID int) error {
	resBody := ResponseBody{
		ChatID: chatID,
		Text:   message,
	}

	// Create the JSON body from the struct
	resBytes, err := json.Marshal(resBody)
	if err != nil {
		log.Printf("could not marshal response: %s\n", err)
		return err
	}

	// Send a post request with your token
	sendMessageRes, err := http.Post(
		SEND_ENDPOINT,
		"application/json",
		bytes.NewBuffer(resBytes))
	if err != nil {
		log.Printf("could not send message: %s\n", err)
		return err
	}
	defer sendMessageRes.Body.Close()

	if sendMessageRes.StatusCode != http.StatusOK {
		log.Printf("unexpected status: %s\n", err)
		return err
	}

	return nil
}
