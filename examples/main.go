package main

import (
	"fmt"
	churro "github.com/lambertmata/churro"
	"log/slog"
	"net/http"
	"time"
)

type WrappedResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (w *WrappedResponseWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func NewWrappedResponseWriter(w http.ResponseWriter) *WrappedResponseWriter {
	return &WrappedResponseWriter{ResponseWriter: w, StatusCode: 200}
}

func LogRequests() churro.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()

			responseWriter := NewWrappedResponseWriter(w)

			next.ServeHTTP(responseWriter, req)

			duration := time.Since(start)

			slog.Info(fmt.Sprintf("%s %s %s %d %dms", req.RemoteAddr, req.Method, req.URL, responseWriter.StatusCode, duration.Milliseconds()))
		})
	}
}

func MiddlewareA() churro.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			slog.Info("Middleware A")
			next.ServeHTTP(w, req)
		})
	}
}

func main() {
	router := churro.NewRouter()

	router.Middlewares(LogRequests(), MiddlewareA())

	churro.Get(router, "/api/2", func(ctx *churro.Context) error {
		slog.Info("API 2")
		return ctx.SendString("API 2 response")
	})
	router.Get("/api", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(request.RequestURI))
	}).Middlewares(LogRequests())

	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(request.RequestURI))
	})

	router.Group(func(gRouter *churro.Router) {
		gRouter.Get("/status", func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte(request.RequestURI))
		})

		gRouter.Post("/channel/{channel}", func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte(request.RequestURI))
		})

	}).Prefix("/ws")

	router.Post("/api/channel/{channel}", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(request.RequestURI))
	})

	slog.Info("Server started")
	_ = http.ListenAndServe(":8888", router)

}
