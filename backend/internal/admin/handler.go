package admin

import (
	"context"
	"errors"
	"log"
	"long/internal/admin/model"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gin-gonic/gin"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const maxRequestBytes = 1 << 20

func NewHandler(store Store) gin.HandlerFunc {
	config := Config{Resolvers: &Resolver{Store: store}}
	config.Complexity.Query.Images = func(child int, input *model.ImageQueryInput) int {
		limit := 25
		if input != nil {
			limit = input.Limit
		}
		return listComplexity(child, limit)
	}
	config.Complexity.Query.Users = func(child int, input *model.UserQueryInput) int {
		limit := 25
		if input != nil {
			limit = input.Limit
		}
		return listComplexity(child, limit)
	}
	server := handler.New(NewExecutableSchema(config))
	server.AddTransport(transport.POST{})
	server.SetQueryCache(lru.New[*ast.QueryDocument](100))
	server.SetParserTokenLimit(10000)
	server.Use(extension.FixedComplexityLimit(5000))
	server.SetRecoverFunc(func(ctx context.Context, value any) error {
		log.Printf("admin GraphQL panic: %v", value)
		return gqlerror.Errorf("Unable to complete admin query")
	})
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if c.Request.ContentLength > maxRequestBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"errors": []gin.H{{"message": "Request body too large"}}})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBytes)
		server.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
	}
}

func listComplexity(child, limit int) int {
	if limit < 1 || limit > 100 {
		return 5001
	}
	return 100 + limit*child
}

func validatePage(offset, limit int, search string) error {
	if offset < 0 || int64(offset) > 2147483647 {
		return gqlerror.Errorf("offset must be between 0 and 2147483647")
	}
	if limit < 1 || limit > 100 {
		return gqlerror.Errorf("limit must be between 1 and 100")
	}
	if len(search) > 255 {
		return gqlerror.Errorf("search must be at most 255 bytes")
	}
	return nil
}

func searchParams(value string) (search, pattern string, id int64) {
	search = strings.TrimSpace(value)
	id, err := strconv.ParseInt(search, 10, 64)
	if err != nil || id <= 0 {
		id = 0
	}
	// Treat SQL wildcard characters as literal search text.
	pattern = "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search) + "%"
	return
}

func queryError(ctx context.Context, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		return gqlerror.Errorf("Admin query timed out")
	}
	log.Printf("admin GraphQL database error: %v", err)
	return gqlerror.Errorf("Unable to load admin data")
}
