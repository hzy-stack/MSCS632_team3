package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"go-chat-server/internal/chat"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	hub := chat.NewHub()

	handler := chat.NewHandler(hub)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	e.GET("/ws", handler.HandleWebSocket)
	e.POST("/logout", handler.HandleLogout)

	e.Start(":8080")
}
