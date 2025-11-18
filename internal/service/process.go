package service

import (
	"batchRequestsRecover/internal/model"
	"batchRequestsRecover/internal/util"
	"crypto/tls"
	"fmt"
	"net/http"
)

const (
	httpSuccessMin = 200
	httpSuccessMax = 300
)

type ProcessService struct {
	config      model.Config
	args        model.CommandLineArgs
	httpService HttpService
}

func NewProcessService(config model.Config, args model.CommandLineArgs) *ProcessService {
	return &ProcessService{config: config, args: args, httpService: createHttpService(config, args)}
}

func (s *ProcessService) ProcessAll(records []http.Request) ([]string, []string, error) {
	respList := make([]string, 0, len(records))
	errList := make([]string, 0)
	for i, record := range records {

		responseMsg, err := s.processRecord(record, i)
		if err != nil {
			return respList, errList, fmt.Errorf("error processing record: %w", err)
		}

		if responseMsg.Type == model.SUCCESS {
			respList = append(respList, responseMsg.Message)
		} else {
			errList = append(errList, responseMsg.Message)
		}

		util.DelayFor(s.args.SleepMillis)
	}
	return respList, errList, nil
}

func (s *ProcessService) processRecord(record http.Request, index int) (res model.Response, err error) {

	response, status, err := s.httpService.call(record)
	if err != nil {
		return model.Response{Type: model.ERROR}, fmt.Errorf("error making request: %w", err)
	}

	formattedResponse := formatResponse(index, status, response)

	return createResponseFromStatus(status, formattedResponse), nil
}

func createResponseFromStatus(status int, message string) model.Response {
	if status >= httpSuccessMin && status < httpSuccessMax {
		return model.Response{Type: model.SUCCESS, Message: message}
	}
	return model.Response{Type: model.ERROR, Message: message}
}

func formatResponse(index, status int, response []byte) string {
	return fmt.Sprintf("%d-%d - %s", index, status, string(response))
}

func loadClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}
