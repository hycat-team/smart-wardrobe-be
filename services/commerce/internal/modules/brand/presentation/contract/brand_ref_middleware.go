package contract

import "github.com/gin-gonic/gin"

type IBrandRefMiddleware interface {
	Path(brandIDParam string) gin.HandlerFunc
	PublicPath(brandIDParam string) gin.HandlerFunc
	Query(brandIDParam string) gin.HandlerFunc
}
