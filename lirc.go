package lirc

import (
	"net"
	"bufio"
	"encoding/hex"
	"strings"
	"strconv"
	"log"
	"time"
	"errors"
)

type LircRouter struct {
	handlers map[remoteButton]Handle

	path string
	connection net.Conn
	writer *bufio.Writer
	reply chan LircReply
	receive chan LircEvent
}

type LircEvent struct {
	code uint64
	repeat int
	button string
	remote string
}

type LircReply struct {
	command string
	success int
	data_len int
	data []string
}

func Init(path string) (*LircRouter, error) {
	l := new(LircRouter)

	c, err := net.Dial("unix", path)

	if err != nil {
		return nil, err
	}

	l.path = path

	l.writer = bufio.NewWriter(c)
	l.reply  = make(chan LircReply)
	l.receive = make(chan LircEvent)

	scanner := bufio.NewScanner(c)
	go reader(scanner, l.receive, l.reply)

	return l, nil
}

func reader(scanner *bufio.Scanner, receive chan LircEvent, reply chan LircReply) {
	const (
		RECEIVE = iota
		REPLY = iota
		MESSAGE = iota
		STATUS = iota
		DATA_START = iota
		DATA_LEN = iota
		DATA = iota
		END = iota
	)

	var message LircReply
	state := RECEIVE
	data_cnt := 0

	for scanner.Scan() {
		line := scanner.Text()

		switch state {
		case RECEIVE:
			if line == "BEGIN" {
				state = REPLY
			} else {
				r := strings.Split(line, " ")
				c, err := hex.DecodeString(r[0])
			        if err != nil {
					log.Println("Invalid lirc broadcats message received - code not parseable")
					continue
				}
		        	if (len(c) != 8) {
					log.Println("Invalid lirc broadcats message received - code has wrong length")
					continue
				}

				var code uint64
				code = 0
				for i:=0; i< 8; i++ {
					code &= uint64(c[i]) << uint(8*i)	
				}

				var event LircEvent
				event.repeat, err = strconv.Atoi(r[1])
			        if err != nil {
					log.Println("Invalid lirc broadcats message received - invalid repeat count")
				}
				event.code = code
				event.button = r[2]
				event.remote = r[3]
				receive <- event
			}
		case REPLY:
			message.command = line
			message.success = 0
			message.data_len = 0
			message.data = message.data[:0]
			state = STATUS
		case STATUS:
			if line == "SUCCESS" {
				message.success = 1
				state = DATA_START
			} else if line == "END" {
				message.success = 1
				state = RECEIVE
				reply <- message
			} else if line == "ERROR" {
				message.success = 0
				state = DATA_START
			} else {
				log.Println("Invalid lirc reply message received - invalid status")
				state = RECEIVE
			}
		case DATA_START:
			if line == "END" {
				state = RECEIVE
				reply <- message
			} else if line == "DATA" {
				state = DATA_LEN
			} else {
				log.Println("Invalid lirc reply message received - invalid data start")
				state = RECEIVE
			}
		case DATA_LEN:
			data_cnt = 0
			var err error
			message.data_len, err = strconv.Atoi(line)
			if err != nil {
				log.Println("Invalid lirc reply message received - invalid data len")
				state = RECEIVE
			} else {
				state = DATA
			}
		case DATA:
			if data_cnt < message.data_len {
				message.data = append(message.data, line)
			}
			data_cnt++
			if data_cnt == message.data_len {
				state = END
			}
		case END:
			state = RECEIVE
			if line == "END" {
				reply <- message
			} else {
				log.Println("Invalid lirc reply message received - invalid end")
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("error reading from lircd socket")
	}
}

func (l *LircRouter) Command(command string) LircReply {
	l.writer.WriteString(command + "\n")
	l.writer.Flush()
	
	reply := <- l.reply

	return reply
}

func (l *LircRouter) Send(command string) error {
	reply := l.Command("SEND_ONCE " + command)
	if reply.success == 0 {
		return errors.New(strings.Join(reply.data, " "))
	}
	return nil
}

func (l *LircRouter) SendLong(command string, duration string) error {
	d, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	reply := l.Command("SEND_START " + command)
	if reply.success == 0 {
		return errors.New(strings.Join(reply.data, " "))
	}
	time.Sleep(d)
	reply  = l.Command("SEND_STOP " + command)
	if reply.success == 0 {
		return errors.New(strings.Join(reply.data, " "))
	}

	return nil
}

func (l *LircRouter) Close() {
	l.connection.Close()
}
