package middleware

import (
	"encoding/base64"
	"log"
	"net/http"
 
)

var jwtSecret []byte

func init() {
	var err error
	jwtSecret, err = base64.StdEncoding.DecodeString("404E635266556A586E3272357538782F413F4428472B4B6250645367566B5970404E635266556A586E3272357538782F413F4428472B4B6250645367566B5970")
	if err != nil {
		log.Fatalf("failed to parse jwt secret: %v", err)
	}
}

func GetOwnerID(r *http.Request) string {
	userID := r.Header.Get("X-User-Id")
	// userRole := r.Header.Get("X-User-Role")
	// authMethod := r.Header.Get("X-Auth-Method")
	
	
	
	

	return userID
}





// package middleware

 
// func AuthMiddleware() gin.HandlerFunc {
//     return func(c *gin.Context) {
//         // already being set somewhere — keep it
//         c.Set("userID",     c.GetHeader("X-User-Id"))
//         c.Set("userRole",   c.GetHeader("X-User-Role"))
//         c.Set("authMethod", c.GetHeader("X-Auth-Method"))
//         c.Set("token", c.GetHeader("Authorization"))
 
//         c.Next()
//     }
// }