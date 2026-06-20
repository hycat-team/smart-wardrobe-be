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
<html lang="vi">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Closy API - Access Required</title>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body {
            background: #1b1b1b;
            color: #ffffff;
            font-family: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            overflow: hidden;
        }
        .card {
            background: #252525;
            border: 1px solid #333333;
            padding: 40px;
            border-radius: 16px;
            box-shadow: 0 15px 35px rgba(0, 0, 0, 0.6);
            text-align: center;
            max-width: 380px;
            width: 100%;
            box-sizing: border-box;
            animation: slideUp 0.5s cubic-bezier(0.16, 1, 0.3, 1);
        }
        @keyframes slideUp {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .logo-container {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            width: 64px;
            height: 64px;
            background: #49cc90;
            border-radius: 14px;
            margin-bottom: 24px;
            box-shadow: 0 6px 20px rgba(73, 204, 144, 0.2);
        }
        .logo-icon {
            color: #1b1b1b;
            font-size: 28px;
            font-weight: 700;
            user-select: none;
        }
        h2 {
            font-size: 22px;
            font-weight: 700;
            margin: 0 0 8px 0;
            letter-spacing: -0.02em;
            color: #ffffff;
        }
        p {
            font-size: 14px;
            color: #aaaaaa;
            margin: 0 0 32px 0;
            line-height: 1.5;
        }
        .input-group {
            position: relative;
            margin-bottom: 20px;
        }
        input {
            width: 100%;
            box-sizing: border-box;
            padding: 16px 20px;
            border: 1px solid #444444;
            border-radius: 10px;
            background: #1b1b1b;
            color: #ffffff;
            font-size: 15px;
            text-align: center;
            transition: all 0.2s ease;
            outline: none;
        }
        input:focus {
            border-color: #49cc90;
            box-shadow: 0 0 0 4px rgba(73, 204, 144, 0.15);
            background: #1e1e1e;
        }
        button {
            width: 100%;
            padding: 16px;
            border: none;
            border-radius: 10px;
            background: #49cc90;
            color: #1b1b1b;
            font-size: 15px;
            font-weight: 700;
            cursor: pointer;
            transition: all 0.2s ease;
            box-shadow: 0 4px 12px rgba(73, 204, 144, 0.25);
        }
        button:hover {
            transform: translateY(-1px);
            background: #3db87f;
            box-shadow: 0 6px 18px rgba(73, 204, 144, 0.35);
        }
        button:active {
            transform: translateY(0);
        }
        .error-msg {
            margin-top: 16px;
            font-size: 13px;
            color: #f87171;
            display: none;
            background: rgba(248, 113, 113, 0.08);
            padding: 10px 14px;
            border-radius: 8px;
            border: 1px solid rgba(248, 113, 113, 0.2);
            animation: shake 0.4s ease-in-out;
        }
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            25% { transform: translateX(-4px); }
            75% { transform: translateX(4px); }
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="logo-container">
            <span class="logo-icon">C</span>
        </div>
        <h2>Closy Swagger Access</h2>
        <p>Vui lòng cung cấp mã truy cập để xem tài liệu API chi tiết.</p>
        <div class="input-group">
            <input type="password" id="code" placeholder="Mã Access Code" autofocus onkeydown="if(event.key==='Enter') submitCode()">
        </div>
        <button onclick="submitCode()">Xác nhận truy cập</button>
        <div id="error" class="error-msg">Mã truy cập không chính xác!</div>
    </div>
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
