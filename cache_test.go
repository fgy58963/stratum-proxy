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
		time.Sleep(20 * time.Millisecond)
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
		time.Sleep(20 * time.Millisecond)
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

func TestMapCacheRequest(t *testing.T) {
	TestCache := make(map[interface{}]string)
	TestKey := "testkey1"
	TestCache[TestKey] = "value"
	resp := MapCacheRequest(TestKey, TestCache)
	if !resp.CacheOK {
		t.Error("Cache Value Not found when it should be")
	}
	if resp.CacheVal != "value" {
		t.Error("Cache Value Not What it should be")
	}
	resp = MapCacheRequest("testkey2", TestCache)
	if resp.CacheOK {
		t.Error("Cache Value Found when it shouldn't be")
	}
	if resp.CacheVal != "" {
		t.Error("Cache Value not blank when it should be")
	}
}

func TestMapCacheRequestWithExpire(t *testing.T) {
	TestCache := make(map[interface{}]cache_exp)
	CurrentTime := time.Now()
	TestKey := "testkey1"
	CacheValue := cache_exp{
		Data: "value",
		ExpTime: CurrentTime.Add(-1 * time.Minute),
	}
	TestCache[TestKey] = CacheValue
	resp := MapCacheRequestWithExpire(TestKey, TestCache)
	if resp.CacheOK {
		t.Error("Cache Value should be expired")
	}
	if resp.CacheVal != "" {
		t.Error("Cache Value should be blank when value is expired")
	}
	CacheValue = cache_exp{
		Data: "value2",
		ExpTime: CurrentTime.Add(time.Minute),
	}
	TestCache[TestKey] = CacheValue
	resp = MapCacheRequestWithExpire(TestKey, TestCache)
	if !resp.CacheOK {
		t.Error("Cache Value should be found")
	}
	if resp.CacheVal != "value2" {
		t.Error("Cache value should be set")
	}
	resp = MapCacheRequestWithExpire("testkey2", TestCache)
	if resp.CacheOK {
		t.Error("Cache Value should not be found because key does not exist")
	}
	if resp.CacheVal != "" {
		t.Error("Cache Value should be blank because key does not exist")
	}
}

func TestCacheRequest(t *testing.T) {
	TestCache1 := make(map[interface{}]string)
	TestCache2 := make(map[interface{}]cache_exp)
	NumBlocks := 1
	scm := Stratum_command_msg{
		Id: 1,
		Method: "unknown.method",
	}
	resp := CacheRequest(scm, TestCache1, TestCache2, NumBlocks)
	if resp.CacheOK {
		t.Error("Cache value found when it should not be")
	}
	if resp.CacheVal != "" {
		t.Error("Cache value not blank when it should be")
	}
	scm = Stratum_command_msg{
		Method: "blockchain.transaction.get",
		Params: []interface{}{"key1"},
	}
	TestCache1["key1"] = "value1"
	resp = CacheRequest(scm, TestCache1, TestCache2, NumBlocks)
	if !resp.CacheOK {
		t.Error("Cache value not found when it should be, value1")
	}
	if resp.CacheVal != "value1" {
		t.Error("Cache Value returned incorrect, should be value1")
	}
	TestCache2["key1"] = cache_exp{
		Data: "value2",
		ExpTime: time.Now().Add(time.Minute),
	}
	scm.Method = "blockchain.address.listunspent"
	resp = CacheRequest(scm, TestCache1, TestCache2, NumBlocks)
	if !resp.CacheOK {
		t.Error("Cache value not found when it should be, value2")
	}
	if resp.CacheVal != "value2" {
		t.Error("Cache value not what it was expected to be, value2")
	}
	scm.Method = "blockchain.numblocks.subscribe"
	resp = CacheRequest(scm, TestCache1, TestCache2, NumBlocks)
	if !resp.CacheOK {
		t.Error("Cache value for numblocks not found when it should be")
	}
	if resp.CacheVal != "{\"id\":1,\"result\":1}" {
		t.Errorf("Cache value not what it was expected to be: %s", resp.CacheVal)
	}
}

func TestGetIntFromParams(t *testing.T) {
	input := []interface{}{1}
	i, ok := GetIntFromParams(input)
	if !ok {
		t.Error("Get int from params failed when it shouldn't")
	}
	if i != 1 {
		t.Error("Get int from params returned the wrong value")
	}
	input2 := 1
	i, ok = GetIntFromParams(input2)
	if ok {
		t.Error("Get int from params did not fail when it should")
	}
	if i != 0 {
		t.Error("Default value for int from params is not 0")
	}
}
