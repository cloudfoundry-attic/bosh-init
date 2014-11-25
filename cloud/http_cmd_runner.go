package cloud

import (
	"encoding/json"
	"fmt"
	"time"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	bmhttpclient "github.com/cloudfoundry/bosh-micro-cli/deployer/httpclient"
	"io/ioutil"
)

type httpCmdRunner struct {
	deploymentUUID string
	endpoint       string
	httpClient     bmhttpclient.HTTPClient
	logger         boshlog.Logger
	logTag         string
}

func NewHTTPCmdRunner(deploymentUUID string, endpoint string, httpClient bmhttpclient.HTTPClient, logger boshlog.Logger) CPICmdRunner {
	return &httpCmdRunner{
		deploymentUUID: deploymentUUID,
		endpoint:       endpoint,
		httpClient:     httpClient,
		logger:         logger,
		logTag:         "httpCmdRunner",
	}
}

type taskCreatedResponse struct {
	TaskID string `json:"task_id"`
}

type taskStatusResponse struct {
	State  string
	Result string
}

func (c *httpCmdRunner) Run(method string, args ...interface{}) (CmdOutput, error) {
	cmdInput := CmdInput{
		Method:    method,
		Arguments: args,
		Context: CmdContext{
			DirectorUUID: c.deploymentUUID,
		},
	}
	inputBytes, err := json.Marshal(cmdInput)
	if err != nil {
		return CmdOutput{}, bosherr.WrapError(err, "Marshalling external CPI command input %#v", cmdInput)
	}

	response, err := c.httpClient.Post(c.endpoint, inputBytes)
	if err != nil {
		return CmdOutput{}, bosherr.WrapError(err, "Marshalling external CPI command input %#v", cmdInput)
	}
	body, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()

	var taskResponse taskCreatedResponse
	err = json.Unmarshal(body, &taskResponse)
	if err != nil {
		return CmdOutput{}, bosherr.WrapError(err, "Unmarshaling task response '%s'", string(body))
	}

	var attempt int
	var result string
	for attempt = 0; attempt < 100; attempt++ {
		response, err := c.httpClient.Get(fmt.Sprintf("%s/tasks/%s", c.endpoint, taskResponse.TaskID))
		body, err := ioutil.ReadAll(response.Body)
		defer response.Body.Close()

		var taskStatusResponse taskStatusResponse
		err = json.Unmarshal(body, &taskStatusResponse)
		if err != nil {
			return CmdOutput{}, bosherr.WrapError(err, "Unmarshaling external CPI command result '%s'", string(body))
		}

		if taskStatusResponse.State == "finished" {
			result = taskStatusResponse.Result
			break
		}
		time.Sleep(5 * time.Second)
	}

	if result == "" {
		return CmdOutput{}, bosherr.New("Timed out getting task result")
	}

	cmdOutput := CmdOutput{}
	if err != nil {
		return CmdOutput{}, bosherr.WrapError(err, "Reading response body")
	}

	err = json.Unmarshal([]byte(result), &cmdOutput)
	if err != nil {
		return CmdOutput{}, bosherr.WrapError(err, "Unmarshalling external CPI command output: '%s'", result)
	}

	c.logger.Debug(c.logTag, cmdOutput.Log)

	if cmdOutput.Error != nil {
		return cmdOutput, bosherr.New("External CPI command for method `%s' returned an error: %s", method, cmdOutput.Error)
	}

	return cmdOutput, err
}
