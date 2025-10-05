package service

import (
	"batchRequestsRecover/internal/model"
	"fmt"
	"io"
	"math/rand"
	"net/http"
)

type HttpService interface {
	call(record http.Request) ([]byte, int, error)
}

type HttpServiceMock struct {
	config model.Config
	args   model.CommandLineArgs
}

type HttpServiceReal struct {
	config model.Config
	args   model.CommandLineArgs
}

func createHttpService(config model.Config, args model.CommandLineArgs) HttpService {
	if args.DryRun {
		return &HttpServiceMock{config: config, args: args}
	}
	return &HttpServiceReal{config: config, args: args}
}

func (service *HttpServiceMock) call(record http.Request) ([]byte, int, error) {
	println("--- Start Request ---")
	println("Dry run, skipping request")
	println("Request URL: ", record.URL.String())
	println("Request Method: ", record.Method)
	println("--- End Request ---")
	//todo implement dry run more randomly
	//get a random number between 1 and 10
	index := rand.Intn(10)

	if (index % 3) == 0 {
		return []byte("BadRequest"), 400, nil
	} else {
		return []byte("Success"), 200, nil
	}
}

func (service *HttpServiceReal) call(record http.Request) ([]byte, int, error) {
	recClient := loadClient()
	resp, err := recClient.Do(&record)
	if err != nil {
		return nil, 0, fmt.Errorf("error making request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading response: %w", err)
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, 0, fmt.Errorf("error closing response body: %w", err)
	}
	fmt.Println("Status:", resp.Status)

	return body, resp.StatusCode, nil
}
