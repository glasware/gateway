package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/glasware/glas-core/proto"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gorilla/websocket"
)

var usage = "Usage: %s <websocket address>\n"

func _main() error {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sc
		cancel()
	}()

	var args []string
	if args = os.Args[1:]; len(args) != 1 {
		return fmt.Errorf(usage, os.Args[0])
	}

	_url := url.URL{Scheme: "wss", Host: args[0], Path: "/api/connect"}
	fmt.Println("dialing " + _url.String())
	conn, _, err := websocket.DefaultDialer.Dial(_url.String(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			byt, err := json.Marshal(&pb.Input{
				Data: scanner.Text(),
			})
			if err != nil {
				errCh <- err
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, byt)
			if err != nil {
				errCh <- err
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if err != io.EOF {
				errCh <- err
			}
		}
	}()

	go func() {
		for {
			_, byt, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}

			var out pb.Output
			err = jsonpb.Unmarshal(bytes.NewReader(byt), &out)
			if err != nil {
				errCh <- err
				return
			}

			if out.Type != pb.Output_INSTRUCTION {
				fmt.Print(out.Data)
			}
		}
	}()

	select {
	case <-ctx.Done():
		break
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := _main(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
