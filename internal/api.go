package internal

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/glasware/gateway/config"
	glas "github.com/glasware/glas-core"
	"github.com/glasware/glas-core/ansi"
	pb "github.com/glasware/glas-core/proto"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gorilla/websocket"
	"github.com/justanotherorganization/l5424"
	"github.com/justanotherorganization/l5424/x5424"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
)

var upgrader websocket.Upgrader

func init() {
	upgrader.CheckOrigin = func(req *http.Request) bool {
		return true
	}
}

// SetupRoutes sets up the API routes.
func SetupRoutes(config *config.Config, g *echo.Group) error {
	if err := config.Validate(); err != nil {
		return errors.Wrap(err, "config.Validate")
	}

	g.GET("/ready", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	g.GET("/connect", makeConnectHandler(config))

	return nil
}

func makeConnectHandler(cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			errMsg := "error upgrading request to websocket"
			cfg.Logger.Log(x5424.Severity, l5424.ErrorLvl, errMsg, err.Error())
			return echo.NewHTTPError(http.StatusInternalServerError, errMsg)
		}
		defer ws.Close()

		var wg sync.WaitGroup
		errCh := make(chan error, 1)
		inCh := make(chan *pb.Input)
		outCh := make(chan *pb.Output)

		// Read from Glas.
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				out := <-outCh
				if out != nil {
					// FIXME: I really don't like this way of breaking out ANSI instructions
					// into something to pass over the websocket...
					out.Data = ansi.ReplaceCodes(out.Data)
					if strings.HasPrefix(out.Data, ansi.Instruction) {
						out.Type = pb.Output_INSTRUCTION
						fields := strings.Split(out.Data, ansi.Separator)
						out.Data = strings.TrimPrefix(fields[0], ansi.Instruction)

						if err := send(ws, out); err != nil {
							errCh <- err
							return
						}

						if len(fields) > 1 {
							for _, f := range fields[1:] {
								if err := send(ws, &pb.Output{
									Type: pb.Output_BUFFERED,
									Data: f,
								}); err != nil {
									errCh <- err
									return
								}
							}
						}

						continue
					}

					if err := send(ws, out); err != nil {
						errCh <- err
						return
					}
				}
			}
		}()

		g, err := glas.New(&glas.Config{
			Input:  inCh,
			Output: outCh,
		})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(c.Request().Context())

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := g.Start(ctx, cancel); err != nil {
				errCh <- err
			}
		}()

		// Write to Glas.
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, byt, err := ws.ReadMessage()
				if err != nil {
					errCh <- err
					return
				}

				var in pb.Input
				err = jsonpb.Unmarshal(bytes.NewReader(byt), &in)
				if err != nil {
					fmt.Println(err)
					errCh <- err
					return
				}

				inCh <- &in
			}
		}()

		select {
		case <-c.Request().Context().Done():
			break
		case <-ctx.Done():
			cancel()
			break
		case err := <-errCh:
			if err != nil {
				cancel()
				if websocket.IsUnexpectedCloseError(
					err,
					websocket.CloseGoingAway,
				) {
					return echo.NewHTTPError(http.StatusInternalServerError, err)
				}
			}
		}

		if err := ws.WriteMessage(websocket.TextMessage, []byte("closing connection")); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "error writing to websocket"))
		}

		wg.Wait()
		return c.NoContent(http.StatusOK)
	}
}

func send(ws *websocket.Conn, out *pb.Output) error {
	m := new(jsonpb.Marshaler)
	var buf bytes.Buffer
	if err := m.Marshal(&buf, out); err != nil {
		return err
	}

	return ws.WriteMessage(websocket.TextMessage, buf.Bytes())
}
