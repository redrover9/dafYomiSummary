package main

import (
	"bytes"
	"encoding/json"
	"fmt"
    	"github.com/dghubble/go-twitter/twitter"
    	"github.com/dghubble/oauth1"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Document struct {
	Language string `json:"language"`
	ID       int    `json:"id"`
	Text     string `json:"text"`
}

type ExtractiveSummarizationTask struct {
	Parameters map[string]interface{} `json:"parameters"`
}

type AnalysisInput struct {
	Documents []Document `json:"documents"`
}

type Request struct {
	Input AnalysisInput            `json:"analysisInput"`
	Tasks map[string][]interface{} `json:"tasks"`
}

type Result struct {
	Created time.Time      `json:"createdDateTime"`
	Updated time.Time      `json:"lastUpdateDateTime"`
	Errors  []interface{}  `json:"errors"`
	Status  string         `json:"status"`
	Tasks   map[string][]interface{} `json:"tasks"`
}

const (
	subscriptionKey = "123"
	endpoint        = "https://abc.cognitiveservices.azure.com"
	uriPath         = "/text/analytics/v3.2-preview.1/analyze"
	apiURL          = endpoint + uriPath
)
func getUrl() struct {
	URL string "json:\"url\""
} {
	calResp, err := http.Get("https://sefaria.org/api/calendars")
	if err != nil {
		log.Fatalln(err)
	}
	calBody, err := ioutil.ReadAll(calResp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	type All struct {
		CalendarItems []struct {
			URL string `json:"url"`
		} `json:"calendar_items"`
	}
	var allCal All
	err = json.Unmarshal(calBody, &allCal)
	if err != nil {
		log.Fatalln(err)
	}
	incompleteURL := allCal.CalendarItems[2]
	return incompleteURL
}

func regexGarbage() string {
	urlStruct := getUrl()
	urlStr := fmt.Sprintf("%#v", urlStruct)
	re, _ := regexp.Compile("\"[A-Z][a-z]*.[0-9]*\"")
	urlOnly := re.FindString(urlStr)
	urlOnlyMinusQuotes := urlOnly[1 : len(urlOnly)-1]
	return urlOnlyMinusQuotes
}

func getDafYomi() string {
	fullUrl := "https://sefaria.org/api/texts/" + regexGarbage()
	dafYomi, err := http.Get(fullUrl)
	if err != nil {
		log.Fatalln(err)
	}
	dafYomiBody, err := ioutil.ReadAll(dafYomi.Body)
	if err != nil {
		log.Fatalln(err)
	}
	type DY struct {
		DYText []interface{} `json:"text"`
	}
	var dy DY
	err = json.Unmarshal(dafYomiBody, &dy)
	if err != nil {
		log.Fatalln(err)
	}
	dyText := dy.DYText
	var psukim []interface{}
	for _, daf := range dyText {
		psukim = append(psukim, daf.([]interface{})...)
	}
	psukimStrSl := make([]string, len(psukim))
	for i, v := range psukim {
		psukimStrSl[i] = fmt.Sprintf("%v", v)
	}
	psukimStr := strings.Join(psukimStrSl, " ")
	return psukimStr
}

func getHttpClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
	}
}

func makeHttpRequest(verb, url string, body interface{}) *http.Request {
	var err error
	var bodyAsJson []byte

	if body != nil {
		bodyAsJson, err = json.Marshal(body)
		if err != nil {
			log.Fatalln(err)
		}
	}

	var httpRequest *http.Request
	if body != nil {
		httpRequest, err = http.NewRequest(verb, url, bytes.NewBuffer(bodyAsJson))
	} else {
		httpRequest, err = http.NewRequest(verb, url, nil)
	}

	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return nil
	}

	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("Ocp-Apim-Subscription-Key", subscriptionKey)

	return httpRequest
}

func main() {
	config := oauth1.NewConfig("123", "456")
	token := oauth1.NewToken("789", "012")
	twitterHTTPClient := config.Client(oauth1.NoContext, token)
	twitterClient := twitter.NewClient(twitterHTTPClient)


	input := Request{
		Input: AnalysisInput{
			Documents: []Document{
				{
					Language: "en",
					ID:       1,
					Text:     getDafYomi(),
				},
			},
		},
		Tasks: map[string][]interface{}{
			"extractiveSummarizationTasks": {
				ExtractiveSummarizationTask{
					Parameters: map[string]interface{}{
						"model-version": "latest",
						"sentenceCount": 2,
						"sortBy":        "Offset",
					},
				},
			},
		},
	}

	fmt.Printf("Issuing POST of analysis request to %s\n", apiURL)

	request := makeHttpRequest("POST", apiURL, input)
	httpClient := getHttpClient()
	resp, err := httpClient.Do(request)
	if err != nil {
		fmt.Printf("Error on request: %v\n", err)
		return
	}

	resultsUrl := resp.Header["Operation-Location"][0]
	fmt.Printf("Request was successful. Got results URL back: %s\n", resultsUrl)

	for {
		request = makeHttpRequest("GET", resultsUrl, nil)
		result := checkAnalysisTaskResults(resultsUrl)
		finish := false

		switch result.Status {
		case "notStarted":
			fmt.Println("Task hasn't started yet, sleeping...")
			time.Sleep(time.Second * 5)
			continue

		case "running":
			fmt.Println("Task is running...")
			time.Sleep(time.Second * 5)
			continue

		case "succeeded":
			fmt.Println("Task was successful")
			extractiveSummarizationTasks := result.Tasks["extractiveSummarizationTasks"]
			for _, value := range extractiveSummarizationTasks {
				valueStr := fmt.Sprint(value)
				re, _ := regexp.Compile("text:[^]]*.")
				sentences := re.FindAllString(valueStr, 3)
				for j, sentence := range sentences {
					sentence = strings.ReplaceAll(sentence, "text:", "")
					sentence = strings.ReplaceAll(sentence, "<b>", "")
					sentence = strings.ReplaceAll(sentence, "</b>", "")
					sentence = strings.ReplaceAll(sentence, "]", "")
					sentences[j] = sentence
				}
				sentencesStr := strings.Join(sentences, " ")
				fmt.Println(sentencesStr)
				tweet, resp, err := twitterClient.Statuses.Update(sentencesStr, nil)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(tweet)
				fmt.Println(resp)
			}
			finish = true
			break

		default:
			fmt.Printf("Unknown task status: %s\n", result.Status)
			finish = true
			break
		}

		if finish {
			break
		}
	}

	fmt.Println("Finished.")
}

func checkAnalysisTaskResults(url string) *Result {
	fmt.Printf("issuing GET to results URL to fetch results: %s\n", url)

	httpClient := getHttpClient()
	resultsRequest := makeHttpRequest("GET", url, nil)
	resp, err := httpClient.Do(resultsRequest)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return nil
	}

	var r *Result
	json.Unmarshal(body, &r)
	return r
}
