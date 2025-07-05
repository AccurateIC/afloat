package main

import "github.com/gofiber/fiber/v2"

func rootHandler(c *fiber.Ctx) error {
	return c.SendString("Hello, World!")
}
func main() {
	app := fiber.New()

	app.Get("/", rootHandler)

	app.Listen(":3000")
}
