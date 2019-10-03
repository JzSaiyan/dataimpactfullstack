package main

import (
	"os"
	"time"

	"./mongodb"
	"./users"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func init() {
	// check mongo connection
	_, err := mongodb.GetMongoDBClient()
	if err != nil {
		os.Exit(-1)
	}
}

func main() {
	gin.SetMode(gin.DebugMode)
	router := gin.New()

	// recovery middleware
	router.Use(gin.Recovery())

	// cors configuration
	router.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{`GET`, `POST`, `PUT`, `DELETE`},
		AllowHeaders:    []string{"Origin", "Authorization", "Content-Length", "Content-Type"},
		MaxAge:          12 * time.Hour,
	}))

	// init users routes
	users.InitUsersRoutes(router)
	router.Run(":8080")
}
