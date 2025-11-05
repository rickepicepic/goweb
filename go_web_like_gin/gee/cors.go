package gee

import "net/http"

// CORS is a middleware that sets permissive CORS headers (suitable for development).
// For production, tighten the allowed origins, methods, and headers as needed.
func CORS() HandlerFunc {
	return func(c *Context) {
		c.SetHeader("Access-Control-Allow-Origin", "*")
		c.SetHeader("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.SetHeader("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With")
		c.SetHeader("Access-Control-Max-Age", "86400")

		if c.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
