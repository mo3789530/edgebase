package handler

import (
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) DocsHTML(c *fiber.Ctx) error {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>EdgeBase Control Plane API</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: 'Roboto', sans-serif;
        }
    </style>
</head>
<body>
    <redoc spec-url='/openapi.yaml'></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@latest/bundles/redoc.standalone.js"></script>
</body>
</html>
	`
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (h *Handler) OpenAPISpec(c *fiber.Ctx) error {
	return c.SendFile("./internal/docs/openapi.yaml")
}
