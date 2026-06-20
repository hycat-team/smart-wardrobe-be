package middleware

import (
	"net/http"
	"smart-wardrobe-be/config"
	"strings"

	"github.com/gin-gonic/gin"
)

// SwaggerLockMiddleware restricts access to Swagger UI and API spec docs unless a valid access code is presented
func SwaggerLockMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only enforce Swagger access code check in production environment
		if cfg.Server.Env != "production" {
			c.Next()
			return
		}

		expectedBase64 := cfg.Server.SwaggerAccessCode
		if expectedBase64 == "" {
			c.Next()
			return
		}

		cookieVal, err := c.Cookie("swagger_session")
		if err == nil && cookieVal == expectedBase64 {
			c.Next()
			return
		}

		// If accessing the main UI page, serve a simple, unstyled HTML page for entering the code
		path := c.Request.URL.Path
		if strings.Contains(path, "index.html") || path == "/swagger" || path == "/swagger/" {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Swagger API - Access Required</title>
</head>
<body>
    <h2>Nhập Mã Truy Cập Swagger</h2>
    <input type="password" id="code" placeholder="Mã Base64">
    <button onclick="submitCode()">Xác nhận</button>
    <div id="error" style="color: red; display: none; margin-top: 10px;">Mã truy cập không chính xác!</div>
    <script>
        function submitCode() {
            var code = document.getElementById("code").value;
            document.cookie = "swagger_session=" + code + ";path=/;max-age=604800";
            location.reload();
        }
        var cookies = document.cookie.split("; ");
        var hasSession = false;
        for (var i = 0; i < cookies.length; i++) {
            var parts = cookies[i].split("=");
            if (parts[0] === "swagger_session") {
                hasSession = true;
                break;
            }
        }
        if (hasSession) {
            document.getElementById("error").style.display = "block";
        }
    </script>
</body>
</html>`)
			c.Abort()
			return
		}

		// Return unauthorized for doc.json, swagger.json, and other static assets
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access denied. Invalid swagger_session."})
		c.Abort()
	}
}
