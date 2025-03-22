package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gorilla/mux"
)

type QueueInfo struct {
	URL          string `json:"url"`
	Name         string `json:"name"`
	MessageCount int    `json:"messageCount"`
}

type Message struct {
	MessageId     string `json:"messageId"`
	Body          string `json:"body"`
	ReceiptHandle string `json:"receiptHandle"`
}

var sqsClient *sqs.Client

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	sqsClient = sqs.NewFromConfig(cfg)

	r := mux.NewRouter()
	r.PathPrefix("/api/").Handler(http.StripPrefix("/api", apiRouter()))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func apiRouter() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/queues", listQueuesHandler).Methods("GET")
	r.HandleFunc("/queues/{queueName}/messages", listMessagesHandler).Methods("GET")
	r.HandleFunc("/queues/{queueName}/messages/{messageId}/delete", deleteMessageHandler).Methods("POST")
	r.HandleFunc("/queues/{queueName}/messages/{messageId}/retry", retryMessageHandler).Methods("POST")
	r.HandleFunc("/queues/{queueName}/purge", purgeQueueHandler).Methods("POST")
	r.HandleFunc("/queues/{queueName}/retryAll", retryAllMessagesHandler).Methods("POST")
	return r
}

func listQueuesHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 {
		size = 10
	}
	result, err := sqsClient.ListQueues(context.TODO(), &sqs.ListQueuesInput{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var queues []QueueInfo
	for _, url := range result.QueueUrls {
		attrs, err := sqsClient.GetQueueAttributes(context.TODO(), &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(url),
			AttributeNames: []sqs.QueueAttributeName{sqs.QueueAttributeNameApproximateNumberOfMessages},
		})
		count := 0
		if err == nil {
			count, _ = strconv.Atoi(attrs.Attributes[string(sqs.QueueAttributeNameApproximateNumberOfMessages)])
		}
		segments := strings.Split(url, "/")
		name := segments[len(segments)-1]
		queues = append(queues, QueueInfo{
			URL:          url,
			Name:         name,
			MessageCount: count,
		})
	}
	start := (page - 1) * size
	end := start + size
	if start > len(queues) {
		start = len(queues)
	}
	if end > len(queues) {
		end = len(queues)
	}
	json.NewEncoder(w).Encode(queues[start:end])
}

func listMessagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queueName := vars["queueName"]
	queueURL, err := getQueueURLByName(queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	result, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     1,
		AttributeNames:      []sqs.QueueAttributeName{"All"},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var messages []Message
	for _, m := range result.Messages {
		messages = append(messages, Message{
			MessageId:     *m.MessageId,
			Body:          *m.Body,
			ReceiptHandle: *m.ReceiptHandle,
		})
	}
	json.NewEncoder(w).Encode(messages)
}

func deleteMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queueName := vars["queueName"]
	queueURL, err := getQueueURLByName(queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	receiptHandle := r.URL.Query().Get("receiptHandle")
	if receiptHandle == "" {
		http.Error(w, "receiptHandle required", http.StatusBadRequest)
		return
	}
	_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func retryMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queueName := vars["queueName"]
	queueURL, err := getQueueURLByName(queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	receiptHandle := r.URL.Query().Get("receiptHandle")
	if receiptHandle == "" {
		http.Error(w, "receiptHandle required", http.StatusBadRequest)
		return
	}
	_, err = sqsClient.ChangeMessageVisibility(context.TODO(), &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     aws.String(receiptHandle),
		VisibilityTimeout: 0,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func purgeQueueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queueName := vars["queueName"]
	queueURL, err := getQueueURLByName(queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = sqsClient.PurgeQueue(context.TODO(), &sqs.PurgeQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func retryAllMessagesHandler(w http.ResponseWriter, r *http.Request) {
	// Not implemented: SQS doesn't provide a bulk retry. Custom implementation required.
	w.WriteHeader(http.StatusNotImplemented)
}

func getQueueURLByName(name string) (string, error) {
	result, err := sqsClient.ListQueues(context.TODO(), &sqs.ListQueuesInput{})
	if err != nil {
		return "", err
	}
	for _, url := range result.QueueUrls {
		segments := strings.Split(url, "/")
		if segments[len(segments)-1] == name {
			return url, nil
		}
	}
	return "", http.ErrNoLocation
}
