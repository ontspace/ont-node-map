package web

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"map/storage"
	"net/http"
)

func StartRestServer(port uint, disableCors bool) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	if !disableCors {
		r.Use(cors.Default())
	}
	r.LoadHTMLGlob("fe/dist/index.html")
	r.Static("/js", "fe/dist/js")
	r.Static("/css", "fe/dist/css")
	r.StaticFile("/favicon.ico", "fe/dist/favicon.ico")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	r.GET("/api/nodes", func(c *gin.Context) {
		nodes := storage.ListAllNodes()
		c.JSON(200,
			nodes,
		)
	})
	return r.Run(fmt.Sprintf(":%d", port))
}
