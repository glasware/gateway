package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/glasware/gateway/config"
	"github.com/glasware/gateway/internal"
	"github.com/ingcr3at1on/x/sigctx"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	if err := sigctx.StartWith(func(ctx context.Context) error {
		if err := internal.SetupRoutes(new(config.Config), e.Group("/api")); err != nil {
			return err
		}

		// FIXME: make this path more friendly...
		// e.Static("/", "./static/terminal.html")

		addr := ":8080"
		return e.Start(addr)
	}); err != nil {
		if err != http.ErrServerClosed {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}

	if err := e.Shutdown(context.Background()); err != nil {
		fmt.Println(err.Error())
	}
}
