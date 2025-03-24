package model

type Message struct {
	MessageId        string            `json:"messageId"`
	Body             string            `json:"body"`
	ReceiptHandle    string            `json:"receiptHandle"`
	Attributes       map[string]string `json:"attributes"`
	SystemAttributes map[string]string `json:"systemAttributes"`
}
