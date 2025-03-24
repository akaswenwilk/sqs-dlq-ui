package main

import (
	"context"
	"log"
	"net/http"

	"github.com/akaswenwilk/sqs-dlq-ui/handler"
	"github.com/akaswenwilk/sqs-dlq-ui/repo"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gorilla/mux"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	sqsClient := sqs.NewFromConfig(cfg)

	r := mux.NewRouter()
	r.PathPrefix("/api/").Handler(http.StripPrefix("/api", apiRouter(sqsClient)))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func apiRouter(sqsClient *sqs.Client) http.Handler {
	r := mux.NewRouter()
	sqsRepo := repo.New(sqsClient)
	h := handler.New(sqsRepo)
	r.HandleFunc("/queues", h.ListQueues).Methods("GET")
	r.HandleFunc("/queues/{queueName}/messages", h.ListMessages).Methods("GET")
	r.HandleFunc("/queues/{queueName}/messages/{messageID}/delete", h.DeleteMessage).Methods("POST")
	r.HandleFunc("/queues/{queueName}/messages/{messageID}/retry", h.RetryMessage).Methods("POST")
	r.HandleFunc("/queues/{queueName}/purge", h.PurgeQueue).Methods("POST")
	r.HandleFunc("/queues/{queueName}/retryAll", h.RetryAllMessages).Methods("POST")
	return r
}
