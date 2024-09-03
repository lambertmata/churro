## Churro

Churro is another go http router aiming to be simple. Although the standard library already include a router with path variable capabilities and other awesome packages already exists. The router is heavily inspired on Laravel Router and allows to define routes using expressive syntax.


### Routes

```go
r := churro.NewRouter()
r.Get("invoices")
r.Get("orders", handler)
r.Get("orders/:id", handler)
r.Post("orders", handler)
r.Delete("orders/:id", handler)
```

#### Path variables
This is how you can access matched path variables from your handler
```go
r := churro.NewRouter()

userHandler := func(w http.ResponseWriter, req *http.Request) {
    userId := GetPathParam(req, "id") 
    slog.Info("got user id", "UserID", userId)
}

r.Get("user/:id", userHandler)
```

### Groups
Routes can be grouped in order to provide common functionalities.
```go
r.Group(func (g) {
    g.Get("/", handler)
    g.Get("/:id", handler)
    g.Post("/", handler)
}).Prefix("orders")
```
You can also nest groups.

### Middlewares
You can define global middlewares, group middlewares and route middlewares
```go
r := churro.NewRouter()

logger := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		slog.Info(fmt.Sprintf("%s %s", req.Method, req.requestURI))
        next.ServeHTTP(w, req)
    })
}
```
#### Global middlewares

Global middleware will be applied to all child routes.
```go
r := churro.NewRouter()
r.Middleware(logger)
```
#### Route middlewares
Route middlewares will applied only to the route which it is assigned to.
```go
r := churro.NewRouter()
r.Get("orders").Middleware(logger)
```
#### Group middlewares
Group middlewares will be applied to all group routes.
```go
r := churro.NewRouter()
r.Get("orders").Middleware(logger)
```