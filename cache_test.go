package main

import (
	"testing"
	"encoding/json"
	"time"
)


func TestGetCacheKey(t *testing.T) {
	input := "{\"params\":[\"key\"]}"
	var scm Stratum_command_msg
	err := json.Unmarshal([]byte(input), &scm)
	if err != nil {
		t.Error("Input could not be parsed for test")
	}

	Key, Valid := GetCacheKey(scm.Params)

	if !Valid {
		t.Error("Cache Key was Not Valid")
	}
	if Key != "key" {
		t.Error("Cache Key was Not Parsed Correctly")
	}
}

func TestGetCacheKeyTooManyParams(t *testing.T) {
	input := "{\"params\":[\"key1\", \"key2\"]}"
	var scm Stratum_command_msg
	err := json.Unmarshal([]byte(input), &scm)
	if err != nil {
		t.Error("Input could not be parsed for test")
	}
	Key, Valid := GetCacheKey(scm.Params)

	if Valid {
		t.Error("Cache Key was Valid when it should not be")
	}
	if Key != "" {
		t.Error("Cache Key was not empty when it should be")
	}
}

func TestGetCacheKeyParamsNotList(t *testing.T) {
	input := "{\"params\":\"key\"}"
	var scm Stratum_command_msg
	err := json.Unmarshal([]byte(input), &scm)
	if err != nil {
		t.Error("Input could not be parsed for test")
	}
	Key, Valid := GetCacheKey(scm.Params)

	if Valid {
		t.Error("Cache Key was Valid when it should not be")
	}
	if Key != "" {
		t.Error("Cache Key was not empty when it should be")
	}
}

func TestLoadCache(t *testing.T) {
	var ch CommHub
	ch.CacheLoad = make(chan cache_load, 1)
	done := make(chan string)

	scm := Stratum_command_msg{}
	scm.Method = "blockchain.transaction.get"

	loadCache(scm, "pass", &ch)
	go func() {
		cl := <-ch.CacheLoad
		done <- cl.Resp
	}()
	go func() {
		time.Sleep(time.Second)
		done <- "timeout"
	}()
	result := <-done
	if result != "pass" {
		t.Error("Cache Load was not triggered")
	}
}

func TestLoadCacheNonCachedMethod(t *testing.T) {
	var ch CommHub
	ch.CacheLoad = make(chan cache_load, 1)
	done := make(chan string)

	scm := Stratum_command_msg{}
	scm.Method = "unknown.method"

	loadCache(scm, "fail", &ch)
	go func() {
		cl := <-ch.CacheLoad
		done <- cl.Resp
	}()
	go func() {
		time.Sleep(time.Second)
		done <- "pass"
	}()
	result := <-done
	if result != "pass" {
		t.Error("Cache Load was not triggered")
	}
}

func TestCheckCache(t *testing.T) {
	var ch CommHub
	ch.CacheReq = make(chan cache_req)

	scm := Stratum_command_msg{}
	scm.Method = "blockchain.transaction.get"

	go func() {
		cReq := <-ch.CacheReq
		cResp := cache_resp{true, cReq.stcmd.Method}
		cReq.retChan <-cResp
	}()

	result := checkCache(scm, &ch)
	if !result.CacheOK {
		t.Error("Cache result failure")
	}
	if result.CacheVal != "blockchain.transaction.get" {
		t.Error("Cache Value failure")
	}
}

func TestCheckCacheNonCachedMethod(t *testing.T) {
	var ch CommHub
	ch.CacheReq = make(chan cache_req)

	scm := Stratum_command_msg{}
	scm.Method = "unknown.method"

	result := checkCache(scm, &ch)
	if result.CacheOK {
		t.Error("Cache result does not match expected")
	}
	if result.CacheVal != "" {
		t.Error("Cache value does not match expected result")
	}
}
