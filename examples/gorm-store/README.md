# GORM Store

```go
gormDB, err := gorm.Open(mysql.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
if err != nil {
	return err
}

app := admin.New("Acme Admin")
app.Resource(User{}).
	TableName("users").
	Field("ID").String().Primary().Readonly().
	Field("Email").Email().Searchable().Sortable()

store, err := gormstore.MySQL(gormDB, app)
if err != nil {
	return err
}

adminServer, err := server.New(ctx, server.Config{
	App:          app,
	Store:        store,
	SessionStore: redisstore.New(redisClient),
	Authenticate: authenticateAdmin,
})
```

The GORM adapter reuses GORM's underlying `*sql.DB`, so connection pooling and driver configuration stay owned by your app.
