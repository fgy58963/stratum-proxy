package main

import (
	"encoding/json"
	"strings"
	"log"
	"os"
)

var exiter func (code int) = os.Exit

func CheckJsonErrors(e error) {
	if e != nil {
		log.Println("Stratum from json string error")
		log.Println("Error:", e.Error())
		log.Println("Error decoding json string")
		exiter(1)
	}
}

type Stratum_command_msg struct {
	Id int `json:"id"`
	Params interface{} `json:"params"`
	Method string `json:"method"`
}

func (s *Stratum_command_msg) FromJsonString(in_string string) {
	CheckJsonErrors(json.Unmarshal([]byte(strings.TrimSpace(in_string)), s))
}

type Stratum_command_resp struct {
	Id int `json:"id"`
	Result interface{} `json:"result"`
	Params interface{} `json:"params,omitempty"`
	Method string `json:"method,omitempty"`
}

func (s *Stratum_command_resp) FromJsonString(in_string string) {
	CheckJsonErrors(json.Unmarshal([]byte(strings.TrimSpace(in_string)), s))
}

