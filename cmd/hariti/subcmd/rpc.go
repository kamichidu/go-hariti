package subcmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/textproto"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli"
)

// http://www.jsonrpc.org/specification
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	Id      interface{} `json:"id,omitempty"`
}

type Response struct {
	Id    interface{} `json:"id"`
	Value interface{} `json:"value"`
}

func call(req *Request) (*Response, error) {
	if req.JSONRPC != "2.0" {
		return nil, fmt.Errorf("Object key \"jsonrpc\" must be exact \"2.0\"", req.JSONRPC)
	}
	if req.Id != nil {
		value, err := onInvocation(req.Method, req.Params)
		if err != nil {
			return nil, fmt.Errorf("Internal error: %s", err)
		}
		return &Response{
			Id:    req.Id,
			Value: value,
		}, nil
	} else {
		return nil, onNotification(req.Method, req.Params)
	}
}

func onNotification(method string, params interface{}) error {
	log.Printf("Notification: %s(%v)", method, params)
	return nil
}

func onInvocation(method string, params interface{}) (interface{}, error) {
	log.Printf("Invocation: %s(%v)", method, params)
	return nil, nil
}

func rpcAction(c *cli.Context) error {
	w := c.App.Writer
	r := bufio.NewReader(os.Stdin)
	logger := log.New(c.App.ErrWriter, "[jsonrpc] ", log.LstdFlags|log.Lshortfile)

	responseCh := make(chan *Response)
	go func() {
		encoder := json.NewEncoder(w)
		for {
			response, active := <-responseCh
			if !active {
				break
			}
			if err := encoder.Encode(response); err != nil {
				logger.Printf("Can't write response: %s", err)
			}
		}
	}()

	for {
		header, err := textproto.NewReader(r).ReadMIMEHeader()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				logger.Printf("Can't read request header: %s", err)
				continue
			}
		}

		var contentLength int64
		if header.Get("Content-Length") != "" {
			cl := strings.TrimSpace(header.Get("Content-Length"))
			n, err := strconv.ParseInt(cl, 10, 64)
			if err != nil {
				logger.Printf("Can't parse Content-Length header: %s", err)
				continue
			}
			contentLength = n
		}

		body, err := ioutil.ReadAll(io.LimitReader(r, contentLength))
		if err != nil {
			logger.Printf("Can't read request payload: %s", err)
			continue
		}
		if len(body) < 2 {
			logger.Printf("Ignore too small input: %d", len(body))
			continue
		}

		logger.Printf("read request header: %#v", header)
		logger.Printf("read request body: %s", string(body))
		req := new(Request)
		if err = json.Unmarshal(body, req); err != nil {
			logger.Printf("Can't decode body: %s", err)
			continue
		}

		response, err := call(req)
		if err != nil {
			logger.Printf("Internal error: %s", err)
			continue
		}
		if response != nil {
			responseCh <- response
		}
	}
	return nil
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "rpc",
		Usage:     "Run hariti as rpc server",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "sock",
				Usage: "Socket `TYPE` for RPC Connection (choices: pipe)",
				Value: "pipe",
			},
		},
		Action: rpcAction,
	})
}
