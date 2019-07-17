package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const serverAddress = "https://integration.agent.exec.itsupport247.net/agent/v1/%s/execute"

//const serverAddress = "http://localhost:8081/agent/v1/%s/execute"

var failedEndpints = make([]string, 0)
var count = 1

type RequestBody struct {
	Files     []string `json:"files"`
	Endpoints []string `json:"endpoints"`
}

func main() {
	router := gin.Default()
	router.POST("/upload-plugin/reterive-logs", func(c *gin.Context) {

		var loginCmd RequestBody
		c.BindJSON(&loginCmd)

		fmt.Println(loginCmd)

		response := make([]string, 0)

		response, _ = sendMessage(loginCmd.Endpoints, loginCmd.Files, response)

		fmt.Println(response)

		c.JSON(http.StatusOK, gin.H{"user": loginCmd})

	})
	router.Run(":8080")
}

func sendMessage(partial []string, files []string, resp []string) ([]string, error) { // SendMessage
	for _, endpoint := range partial {
		payload := createPayload(endpoint, files)
		req, err := http.NewRequest("POST", fmt.Sprintf(serverAddress, endpoint), payload)
		if err != nil {
			fmt.Println(time.Now(), " Error while creating request ", err, " for endpoint ", endpoint)
			failedEndpints = append(failedEndpints, endpoint)
		}
		req.Header.Add("content-type", "application/json")
		req.Header.Add("cache-control", "no-cache")

		res, err := createClient().Do(req)
		if err != nil {
			fmt.Println(time.Now(), " Error while geting response ", err, " for endpoint ", endpoint)
			failedEndpints = append(failedEndpints, endpoint)
		}
		if res != nil && res.Body != nil {
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Println(time.Now(), " Error while Reading message ", err, " for endpoint ", endpoint)
				failedEndpints = append(failedEndpints, endpoint)
			}
			resp = append(resp, string(body))
		}
	}
	time.Sleep(10 * time.Second)
	return resp, nil
}

func createPayload(endpoint string, files []string) io.Reader {
	count++
	uploadAddress := "https://agent.service.itsupport247.net"
	message := "\t{\"requestID\":\"r" + strconv.Itoa(count) + "\", \"fileServerURL\":\"" + uploadAddress + "/agent/v1/" + endpoint + "/fileupload\", \"protocol\":\"HTTPS\", \"srcPath\":" + strings.Join(files, ",") + ", \"destPath\":\"\"}\r\n"
	encodedString := base64.StdEncoding.EncodeToString([]byte(message))
	payload := "{\n  \"name\": \"Agent Core Logs\",\n  \"type\": \"SCHEDULE\",\n  \"version\": \"2.0\",\n  \"timestampUTC\": \"2018-09-10T12:24:53.489110938Z\",\n  \"path\": \"/filetransfer/ftp/upload\",\n  \"message\": \"{\\\"task\\\": \\\"/schedule/filetransfer/ftp/upload\\\", \\\"executeNow\\\": \\\"true\\\", \\\"taskInput\\\":\\\"" + encodedString + "\\\"}\"\n}\n"
	return strings.NewReader(payload)
}

func readEndpoints(path string) ([]string, error) {
	value, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, err
	}

	endpoints := strings.Replace(string(value), "\n", ",", -1)
	endpoints = strings.Replace(string(value), "\r", ",", -1)
	endpoints = strings.Replace(string(value), "\r\n", ",", -1)
	endpoints = strings.Replace(string(value), " ", ",", -1)
	re := regexp.MustCompile(`\r?\n`)
	endpoints = re.ReplaceAllString(endpoints, ",")
	return strings.Split(endpoints, ","), nil
}

var client *http.Client

func createClient() *http.Client {
	if client == nil {
		tlsConfig := &tls.Config{
			InsecureSkipVerify:     true,
			SessionTicketsDisabled: false,
			ClientSessionCache:     tls.NewLRUClientSessionCache(1),
		}

		transport := &http.Transport{
			TLSClientConfig: tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 3 * time.Second,
			}).DialContext,
			MaxIdleConns:          1,
			IdleConnTimeout:       time.Minute,
			TLSHandshakeTimeout:   3 * time.Second,
			ExpectContinueTimeout: 3 * time.Second,
			MaxIdleConnsPerHost:   2,
		}
		client = &http.Client{
			Timeout:   time.Minute,
			Transport: transport,
		}
	}
	return client
}
