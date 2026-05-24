# Existing Gin or Echo App

GoMyAdmin exposes a standard `http.Handler`, so frameworks can mount it through their HTTP adapter.

## Gin

```go
adminHandler := adminServer.Handler()

router.Any("/admin/*path", func(c *gin.Context) {
	adminHandler.ServeHTTP(c.Writer, c.Request)
})
```

## Echo

```go
adminHandler := adminServer.Handler()

e.Any("/admin/*", func(c echo.Context) error {
	adminHandler.ServeHTTP(c.Response(), c.Request())
	return nil
})
```

Use the same `server.Config` options shown in the chi example: pass either `DatabaseURL`, `Pool`, or a custom `Store`.
