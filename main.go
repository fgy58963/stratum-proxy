package main

import (
	"net"
	"log"
	"bufio"
	"flag"
	"strings"
	"encoding/json"
	"time"
)

type incoming_conn struct {
	Conn net.Conn
	MsgIds map[int]Stratum_command_msg
	ConnId int
	Subscriptions map[string]struct{}
}

type server_out struct {
	strat_cmd Stratum_command_msg
	ping bool
	incConn incoming_conn
}

type CommHub struct {
	MsgMap map[int]int //map message to conn id
	IncomingConns []incoming_conn
	CacheLoad chan cache_load
	CacheReq chan cache_req
	ServerIn chan string
	ServerOut chan server_out
}



func serverRespHandler(ch *CommHub) {
	for {
		select {
		case m := <-ch.ServerIn:
			var s Stratum_command_resp
			s.FromJsonString(m)
			if s.Method == "blockchain.numblocks.subscribe" {
				stcmdmsg := Stratum_command_msg{1, s.Params, s.Method}
				loadCache(stcmdmsg, m, ch)
				for _, n := range ch.IncomingConns {
					_, found := n.Subscriptions["blockchain.numblocks.subscribe"]
					if found {
						n.Conn.Write(append([]byte(m), []byte("\n")...))
					}
				}
			} else {
				for _, n := range ch.IncomingConns {
					log.Println("conn number:", n.ConnId)
					stcmdmsg, ok := n.MsgIds[s.Id]
					if ok {
						loadCache(stcmdmsg, m, ch)
						s.Id = stcmdmsg.Id
						newMsg, err := json.Marshal(s)
						if err != nil {
							log.Fatal("JSON Error,", err)
						}
						n.Conn.Write(append(newMsg, []byte("\n")...))
					}
				}
			}
		}
	}
}

func handleConnection(inConn incoming_conn, ch *CommHub) {
	for {
		message, ConnErr := bufio.NewReader(inConn.Conn).ReadString('\n')
		if ConnErr != nil {
			log.Println("Listen Connection Read Error")
			log.Println(ConnErr)
			// remove connection from incoming connection list
			for i, n := range ch.IncomingConns {
				if n.ConnId == inConn.ConnId {
					ch.IncomingConns = append(ch.IncomingConns[:i], ch.IncomingConns[i+1:]...)
					break
				}
			}
			// close connection
			inConn.Conn.Close()
			return
		}
		if len(message) > 0 {
			var stratCmd Stratum_command_msg
			stratCmd.FromJsonString(message)
			if stratCmd.Method == "server.version" {
				log.Println("Client version message")
				sr := Stratum_command_resp{
					Id: stratCmd.Id,
					Result: "1.0",
				}
				msg, _ := json.Marshal(sr)
				inConn.Conn.Write(append(msg, []byte("\n")...))
				continue
			}
			if stratCmd.Method == "blockchain.numblocks.subscribe" {
				_, already_sub := inConn.Subscriptions["blockchain.numblocks.subscribe"]
				if already_sub {
					CacheResp := checkCache(stratCmd, ch)
					if CacheResp.CacheOK {
						var sr Stratum_command_resp
						sr.FromJsonString(CacheResp.CacheVal)
						sr.Id = stratCmd.Id
						newMsg, err := json.Marshal(sr)
						if err != nil {
							log.Fatal("json marshal err", err)
						}
						inConn.Conn.Write(append(newMsg, []byte("\n")...))
					} else {
						log.Println("ERROR: cached response not found (should be found)")
					}
					continue
				} else {
					inConn.Subscriptions["blockchain.numblocks.subscribe"] = struct{}{}
				}
			}
			log.Println("Method Call:", stratCmd.Method)
			log.Println("Params:", stratCmd.Params)
			CacheResp := checkCache(stratCmd, ch)
			if CacheResp.CacheOK {
				var sr Stratum_command_resp
				sr.FromJsonString(CacheResp.CacheVal)
				sr.Id = stratCmd.Id
				newMsg, err := json.Marshal(sr)
				if err != nil {
					log.Fatal("json marshal err", err)
				}
				inConn.Conn.Write(append(newMsg, []byte("\n")...))
			} else {
				ch.ServerOut <- server_out{stratCmd, false, inConn}
			}
		}
	}
}

func ServerPing(ch *CommHub) {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		log.Println("Sending Server Ping (version message)")
		stratCmd := Stratum_command_msg{1, []string{""}, "server.version"}
		ch.ServerOut <- server_out{stratCmd, true, incoming_conn{}}
	}
}

func outgoingConn(conn net.Conn, ch *CommHub) {
	outId := 0
	for {
		select {
		case so := <-ch.ServerOut:
			stcmd := so.strat_cmd
			if !so.ping {
				ch.MsgMap[outId] = so.incConn.ConnId
				so.incConn.MsgIds[outId] = stcmd
			}
			stcmd.Id = outId
			newMsg, err := json.Marshal(stcmd)
			if err != nil {
				log.Fatal("JSON Marshal Error", err)
			}
			conn.Write(append(newMsg, []byte("\n")...))
			outId++
		}
	}
}

func outgoingConnResp(conn net.Conn, ch *CommHub) {
	for {
		message, _ := bufio.NewReader(conn).ReadString('\n')
		if len(message) > 0 {
			log.Println("incoming conn:", strings.TrimSpace(message))
			ch.ServerIn <-message
		}
	}
}

func main () {
	var host = flag.String("host", "ecdsa.net:50001", "electrum server host to connect to")
	flag.Parse()
	var commhub = CommHub {
		MsgMap: make(map[int]int),
		IncomingConns: make([]incoming_conn, 0),
		CacheLoad: make(chan cache_load),
		CacheReq: make(chan cache_req),
		ServerIn: make(chan string),
		ServerOut: make(chan server_out),
	}
	log.Println("Connecting to:", *host)
	outConn, err := net.Dial("tcp", *host)
	if err != nil {
		log.Fatal("Connect Error", err)
	}
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("error listening", err)
	}
	defer ln.Close()
	cacheExpTime, err := time.ParseDuration("5m")
	if err != nil {
		log.Fatal("Cache Exp Time Parse Failure")
	}
	go CacheManager(&commhub, cacheExpTime)
	go outgoingConn(outConn, &commhub)
	go outgoingConnResp(outConn, &commhub)
	go serverRespHandler(&commhub)
	go ServerPing(&commhub)
	connId := 0
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("error listening", err)
		}
		var incConn = incoming_conn {
			Conn: conn,
			MsgIds: make(map[int]Stratum_command_msg),
			ConnId: connId,
			Subscriptions: make(map[string]struct{}),
		}
		commhub.IncomingConns = append(commhub.IncomingConns, []incoming_conn{incConn}...)
		connId++
		go handleConnection(incConn, &commhub)
	}
}
