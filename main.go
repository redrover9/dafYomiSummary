package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
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

func main() {
	dyTextStr := getDafYomi()
	var subscriptionKey string = "e8e116026eb640fdb86d20c0ac1f319e"
	var endpoint string = "https://dafyomisummary.cognitiveservices.azure.com"

	const uriPath = "/text/analytics/v3.2-preview.1/analyze"

	var uri = endpoint + uriPath

	data := []map[string]string{
		{"id": "1", "language": "en", "text": dyTextStr},
	}
	params := []map[string]string{
		{"model-version": "latest", "sentenceCount": "3", "sortBy": "Offset"},
	}
	documents, err := json.Marshal(&data)
	if err != nil {
		fmt.Printf("Error marshaling data: %v\n", err)
		return
	}
	tasks, err := json.Marshal(&params)
	if err != nil {
		fmt.Printf("Error marshaling data: %v\n", err)
		return
	}
	r := strings.NewReader("{\"tasks\":{\"extractiveSummarizationTasks\":{\"parameters\":" + string(tasks) + "}},\"analysisInput\":{\"documents\":" + string(documents) + "}}")
	client := &http.Client{
		Timeout: time.Second * 2,
	}
	req, err := http.NewRequest("POST", uri, r)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Ocp-Apim-Subscription-Key", subscriptionKey)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error on request: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}
	var f interface{}
	json.Unmarshal(body, &f)
	jsonFormatted, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		fmt.Printf("Error producing JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonFormatted))
}
