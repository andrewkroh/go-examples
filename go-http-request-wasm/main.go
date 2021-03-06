package main

import (
	"bytes"
	"fmt"
	"net/http"
)

func GetJson(url string) (json string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = fmt.Errorf("response status code: %d", resp.StatusCode)
		return
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return
	}
	json = buf.String()
	return
}

func main() {
	json, err := GetJson("https://raw.githubusercontent.com/brandiqa/json-examples/master/src/db.json")
	if err != nil {
		panic(err)
	}
	fmt.Println(json)
}
