package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
