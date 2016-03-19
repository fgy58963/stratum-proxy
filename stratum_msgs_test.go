package main

import (
	"testing"
	"errors"
	"os"
	"encoding/json"
)

func TestCheckJsonErrors(t *testing.T) {
	var exited bool = false
	exiter = func(code int) {
		exited = true
	}
	e := errors.New("test")
	CheckJsonErrors(e)

	if exited == false {
		t.Errorf("Error in CheckJsonErrors()")
	}
	exiter = os.Exit
}

func TestStratumCommandMsgNormal(t *testing.T) {
	var exited bool = false
	exiter = func(code int) {
		exited = true
	}
	var input string = "{\"id\":9999,\"params\":[\"asdf\",\"asdf2\"],\"method\":\"cmd\"}"
	var scm = new(Stratum_command_msg)
	scm.FromJsonString(input)
	if exited == true {
		t.Errorf("Error in FromJsonString")
	}
	if scm.Id != 9999 {
		t.Errorf("Error parsing Id")
	}
	if scm.Method != "cmd" {
		t.Errorf("Error parsing Method")
	}
	switch p := scm.Params.(type) {
	case []interface{}:
		switch p0 := p[0].(type) {
		case string:
			if p0 != "asdf" {
				t.Errorf("Error parsing Params value")
			}
		default:
			t.Errorf("Error parsing Params string type")
		}
	default:
		t.Errorf("Error parsing Params slice type")
	}
}

func TestStratumCommandMsgMissingParams(t *testing.T) {
	var exited bool = false
	exiter = func(code int) {
		exited = true
	}
	var input string = "{\"id\":9999,\"method\":\"cmd\"}"
	var scm = new(Stratum_command_msg)
	scm.FromJsonString(input)
	if exited == true {
		t.Errorf("Error in FromJsonString")
	}
	if scm.Id != 9999 {
		t.Errorf("Error parsing Id")
	}
	if scm.Method != "cmd" {
		t.Errorf("Error parsing Method")
	}
	if scm.Params != nil {
		t.Errorf("Error parsing Params")
	}
}

func TestStratumCommandMsgInvalidId(t *testing.T) {
	var exited bool = false
	exiter = func(code int) {
		exited = true
	}
	var input string = "{\"id\":\"999\"}"
	var scm = new(Stratum_command_msg)
	scm.FromJsonString(input)
	if exited == false {
		t.Errorf("Error in FromJsonString")
	}
}

func TestStratumCommandMsgMarshal(t *testing.T) {
	var expected string = "{\"id\":9999,\"params\":null,\"method\":\"cmd\"}"
	var scm = Stratum_command_msg{
		Id: 9999,
		Method: "cmd",
	}
	output, _ := json.Marshal(scm)
	if string(output) != expected {
		t.Errorf("Error Marshalling Json, output: %s", output)
	}
}

func TestStratumCommandResp(t *testing.T) {
	var input string = "{\"id\":9999,\"result\":\"response\"}"
	var scr = new(Stratum_command_resp)
	scr.FromJsonString(input)
	if scr.Id != 9999 {
		t.Errorf("Error parsing stratum response")
	}
	output, _ := json.Marshal(scr)
	if string(output) != input {
		t.Errorf("Error Marshalling Json, output: %s", output)
	}
}

