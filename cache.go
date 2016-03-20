package main

import (
	"log"
	"strings"
	"time"
	"encoding/json"
)

type cache_resp struct {
	CacheOK bool
	CacheVal string
}

type cache_req struct {
	stcmd Stratum_command_msg
	retChan chan cache_resp
}

type cache_load struct {
	StCmdMsg Stratum_command_msg
	Resp string
}

type cache_exp struct {
	Data string
	ExpTime time.Time
}

func GetCacheKey(i interface{}) (string, bool) {
	switch p := i.(type) {
	case []interface{}:
		if len(p) == 1 {
			switch q := p[0].(type) {
			case string:
				return q, true
			}
		}
	}
	return "", false
}

func MapCacheRequest(key string, Cache map[interface{}]string) cache_resp {
	resp := cache_resp{false, ""}
	cached_data, ok := Cache[key]
	if ok {
		log.Println("Data in Cache:", strings.TrimSpace(cached_data))
		resp = cache_resp{true, cached_data}
	}
	return resp
}

func MapCacheRequestWithExpire(key string, Cache map[interface{}]cache_exp) cache_resp {
	resp := cache_resp{false, ""}
	cached_data, ok := Cache[key]
	if ok {
		log.Println("Data in Cache:", strings.TrimSpace(cached_data.Data))
		if cached_data.ExpTime.Before(time.Now()) {
			log.Println("Data in Cache Expired:", time.Now().Sub(cached_data.ExpTime).Seconds())
		} else {
			resp = cache_resp{true, cached_data.Data}
		}
	}
	return resp
}

func CacheRequest(scm Stratum_command_msg, TransactionCache map[interface{}]string, UnspentCache map[interface{}]cache_exp, NumBlocks int) cache_resp {
	resp := cache_resp{false, ""}
	key, keyok := GetCacheKey(scm.Params)
	switch scm.Method {
	case "blockchain.transaction.get":
		if keyok {
			resp = MapCacheRequest(key, TransactionCache)
		}
	case "blockchain.address.listunspent":
		if keyok {
			resp = MapCacheRequestWithExpire(key, UnspentCache)
		}
	case "blockchain.numblocks.subscribe":
		sr := Stratum_command_resp{
			Id: 1,
			Result: NumBlocks,
		}
		msg, err := json.Marshal(sr)
		if err != nil {
			log.Fatal("json error numblock subscription", err)
		}
		resp = cache_resp{true, string(msg)}
	}
	return resp
}

func GetIntFromParams(input interface{}) (int, bool) {
	switch p := input.(type) {
	case []interface{}:
		switch q := p[0].(type) {
		case int:
			return q, true
		}
	}
	return 0, false
}

func CacheManager(ch *CommHub, exp time.Duration) {
	TransactionCache := make(map[interface{}]string)
	UnspentCache := make(map[interface{}]cache_exp)
	NumBlocks := 0
	for {
		select {
		case cr := <-ch.CacheReq:
			scm := cr.stcmd
			resp := CacheRequest(scm, TransactionCache, UnspentCache, NumBlocks)
			cr.retChan <-resp
		case cl := <-ch.CacheLoad:
			key, keyok := GetCacheKey(cl.StCmdMsg.Params)
			switch cl.StCmdMsg.Method {
			case "blockchain.transaction.get":
				if keyok {
					TransactionCache[key] = cl.Resp
				}
			case "blockchain.address.listunspent":
				if keyok {
					exptime := time.Now().Add(exp)
					unsp_data := cache_exp{cl.Resp, exptime}
					UnspentCache[key] = unsp_data
				}
			case "blockchain.numblocks.subscribe":
				i, ok := GetIntFromParams(cl.StCmdMsg.Params)
				if ok {
					NumBlocks = i
				}
			}
		}
	}
}

func checkCache(stcmd Stratum_command_msg, ch *CommHub) cache_resp {
	switch stcmd.Method {
	case "blockchain.transaction.get", "blockchain.address.listunspent", "blockchain.numblocks.subscribe":
		log.Println("Checking Cache for Method:", stcmd.Method)
		cacheRespChan := make(chan cache_resp)
		ch.CacheReq <-cache_req{stcmd, cacheRespChan}
		CacheResp := <-cacheRespChan
		return CacheResp
	default:
		log.Println("Cache not enabled for method:", stcmd.Method)
		return cache_resp{false, ""}
	}
}

func loadCache(stcmd Stratum_command_msg, resp string, ch *CommHub) {
	switch stcmd.Method {
	case "blockchain.transaction.get", "blockchain.address.listunspent", "blockchain.numblocks.subscribe":
		log.Println("Add to Cache", strings.TrimSpace(resp))
		ch.CacheLoad <- cache_load{stcmd, resp}
	}
}
