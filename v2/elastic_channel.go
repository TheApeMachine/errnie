package errnie

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type LogMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type ElasticLogger struct {
	client *elasticsearch.Client
}

func NewElasticLogger() LogChannel {
	return ElasticLogger{}
}

func (logChannel ElasticLogger) Panic(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.write("panic", msgs)
	return false
}

func (logChannel ElasticLogger) Fatal(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.write("fatal", msgs)
	return false
}

func (logChannel ElasticLogger) Critical(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.write("critical", msgs)
	return false
}

func (logChannel ElasticLogger) Error(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.write("error", msgs)
	return false
}

func (logChannel ElasticLogger) Info(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.write("info", msgs)
	return false
}

func (logChannel ElasticLogger) Warning(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.write("warning", msgs)
	return false
}

func (logChannel ElasticLogger) Debug(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.write("debug", msgs)
	return false
}

func (logChannel ElasticLogger) connect() *elasticsearch.Client {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			viper.GetViper().GetString("elasticsearch.host"),
		},
		Username: viper.GetViper().GetString("elasticsearch.username"),
		Password: viper.GetViper().GetString("elasticsearch.password"),
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
			TLSClientConfig: &tls.Config{
				MaxVersion:         tls.VersionTLS11,
				InsecureSkipVerify: true,
			},
		},
	})

	if err != nil || client == nil {
		log.Println("client was nil:", err)
		return nil
	}

	logChannel.client = client

	if viper.GetViper().GetString("elasticsearch.host") == "" {
		logChannel.client = nil
	}

	return logChannel.client
}

func (logChannel ElasticLogger) write(level string, msgs ...interface{}) {
	if logChannel.client == nil {
		_ = logChannel.connect()
	}

	if logChannel.client == nil {
		return
	}

	for _, msg := range msgs {
		jsonMsg, err := json.Marshal(LogMessage{Timestamp: time.Now(), Level: level, Message: fmt.Sprintf("%v", msg)})
		Handles(err).With(RECV)

		_, err = esapi.IndexRequest{
			Index:      viper.GetString("name"),
			DocumentID: uuid.New().String(),
			Body:       strings.NewReader(string(jsonMsg)),
		}.Do(context.Background(), logChannel.client)

		if err != nil {
			log.Println(err)
		}
	}
}
