package auth

import (
	"fmt"

	jwt "github.com/appleboy/gin-jwt"
	"github.com/gin-gonic/gin"

	"strings"
)

func allowed(perm, level string, claims map[string]interface{}) bool {
	fmt.Println(perm, level)
	for _, course := range claims["courses"].([]interface{}) {
		if perm == course.(map[string]interface{})["courseID"] && level == course.(map[string]interface{})["enrollmentType"] {
			return true
		}
	}

	return false
}

func determineLevel(route string) string {
	if strings.Contains(route, "create") {
		return "teacher"
	}

	if strings.Contains(route, "submit") {
		return "student"
	}

	return ""
}

// Authorizator a default function for a gin jwt, that authorizes a user.
func Authorizator(d interface{}, c *gin.Context) bool {
	claims := jwt.ExtractClaims(c)
	// aid := c.Param("aid")
	cid := c.Param("cid")

	userShouldBe := determineLevel(c.Request.URL.String())
	fmt.Println("usb", userShouldBe)

	if cid != "" {
		return allowed(cid, userShouldBe, claims)
	}

	return true
}
