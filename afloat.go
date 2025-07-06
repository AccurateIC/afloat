package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/go-playground/validator/v10"

	"github.com/joho/godotenv"
)

// TODO: validator only parses first rule, 
// it should return error according to which rule failed

var validate = validator.New() // global validator instance

type ErrorResponse struct {
	Error       bool
	FailedField string
	Tag         string
	Value       interface{}
}

type XValidator struct {
	validator *validator.Validate
}

type GlobalErrorHandlerResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (v XValidator) Validate(data interface{}) []ErrorResponse {
	validationErrors := []ErrorResponse{}

	errs := validate.Struct(data)

	if errs != nil {
		for _, err := range errs.(validator.ValidationErrors) {
			var elem ErrorResponse

			elem.FailedField = err.Field()
			elem.Tag = err.Tag()
			elem.Value = err.Value()
			elem.Error = true

			validationErrors = append(validationErrors, elem)
		}
	}
	return validationErrors
}

func rootHandler(c *fiber.Ctx) error {
	return c.SendString("Hello, World!")
}

type PortCallRequestBody struct {
	Days   int16 `json:"days" validate:"required,min=1,max=15"`
}

type BerthCallRequestBody struct {
	Days   int16 `json:"days" validate:"required,min=1",max=15`
}

func portCallHandler(c *fiber.Ctx, validator *XValidator) error {

	body := new(PortCallRequestBody)
	if err := c.BodyParser(body); err != nil {
		log.Error(err)
		return err
	}

	if errs := validator.Validate(body); len(errs) > 0 && errs[0].Error {
		errMsgs := make([]string, 0)

		for _, err := range errs {
			errMsgs = append(errMsgs, fmt.Sprintf(
				"[%s]: '%v' | Needs to implement '%s'",
				err.FailedField,
				err.Value,
				err.Tag,
			))
		}

		return &fiber.Error{
			Code:    fiber.ErrBadRequest.Code,
			Message: strings.Join(errMsgs, " and "),
		}
	}

	API_KEY := os.Getenv("PORT_CALL_API_KEY")
	if API_KEY == "" {
		log.Error("Failed to fetch API key for Port Calls")
		return fiber.NewError(fiber.StatusInternalServerError, "Missing API Key for Port Calls")
	}

	BASE_URL := os.Getenv("MARINE_TRAFFIC_BASE_URL")
	if BASE_URL == "" {
		log.Error("Failed to fetch Base Marine Traffic URL.")
		return fiber.NewError(fiber.StatusInternalServerError, "Missing Marine Traffic base url")
	}

	msgType := "simple"
	protocol := "csv"

	toDate := time.Now()
	formattedToDate := toDate.Format("2006-01-02 15:04:05")
	encodedToDate := url.QueryEscape(formattedToDate)

	fromDate := toDate.Add(-time.Duration(body.Days) * 24 * time.Hour) // subtract `n` days from current date
	formattedFromDate := fromDate.Format("2006-01-02 15:04:05")
	encodedFromDate := url.QueryEscape(formattedFromDate)

	log.Info("From: ", formattedFromDate)
	log.Info("To: ", formattedToDate)

	URL := (BASE_URL + "/portcalls/" + API_KEY + "?v=6" +
		"&fromdate=" + encodedFromDate + "&todate=" + encodedToDate +
		"&msgtype=" + msgType + "&protocol=" + protocol)
	log.Info("PORT CALLS URL: " + URL)

	// call marine traffic api with `URL`
	response, respErr := http.Get(URL)
	if respErr != nil {
		log.Error("Failed to contact Marine Traffic Port Call API")
		return respErr
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		// if marine traffic returns error with status 429 (too many requests)
		if response.StatusCode == fiber.StatusTooManyRequests {
			// TODO: if data exists in cache, return it

			// else
			// TODO: tell the caller that how much time is remaining until they can
			// call the api again
			return c.Status(response.StatusCode).SendString("Too Many Requests")
		}

		// other error might be 401 (unauthorized)
		// for this there's not much we can do
		if response.StatusCode == fiber.StatusUnauthorized {
			return c.Status(response.StatusCode).SendString("Unauthorized")

		}

		// unexpected error case
		errBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error("Failed to read response body for port call", err)
			return fiber.NewError(response.StatusCode, "abcd")
			return err
		}
		return c.Status(response.StatusCode).SendString(string(errBody))
	}

	// here data exists
	csvData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Failed to read response body for port call", err)
		return err
	}

	c.Set("Content-Type", "text/csv")

	return c.Send(csvData)

}

func BerthCallHandler(c *fiber.Ctx, validator *XValidator) error {
	body := new(BerthCallRequestBody)
	if err := c.BodyParser(body); err != nil {
		log.Error(err)
		return err
	}


	if errs := validator.Validate(body); len(errs) > 0 && errs[0].Error {
		errMsgs := make([]string, 0)

		for _, err := range errs {
			errMsgs = append(errMsgs, fmt.Sprintf(
				"[%s]: '%v' | Needs to implement '%s'",
				err.FailedField,
				err.Value,
				err.Tag,
			))
		}

		return &fiber.Error{
			Code:    fiber.ErrBadRequest.Code,
			Message: strings.Join(errMsgs, " and "),
		}
	}


	API_KEY := os.Getenv("BERTH_CALL_API_KEY")
	if API_KEY == "" {
		log.Error("Missing BERTH_CALL_API_KEY")
		return fiber.NewError(fiber.StatusInternalServerError, "Missing API Key for Berth Calls")
	}

	BASE_URL := os.Getenv("MARINE_TRAFFIC_BASE_URL")
	if BASE_URL == "" {
		log.Error("Failed to fetch Base Marine Traffic URL.")
		return fiber.NewError(fiber.StatusInternalServerError, "Missing Marine Traffic base url")
	}

	msgType := "simple"
	protocol := "csv"

	toDate := time.Now()
	formattedToDate := toDate.Format("2006-01-02 15:04:05")
	encodedToDate := url.QueryEscape(formattedToDate)

	fromDate := toDate.Add(-time.Duration(body.Days) * 24 * time.Hour) // subtract `n` days from current date
	formattedFromDate := fromDate.Format("2006-01-02 15:04:05")
	encodedFromDate := url.QueryEscape(formattedFromDate)

	log.Info("From: ", formattedFromDate)
	log.Info("To: ", formattedToDate)

	URL := (BASE_URL + "/berth-calls/" + API_KEY + "?v=3" +
		"&fromdate=" + encodedFromDate + "&todate=" + encodedToDate +
		"&msgtype=" + msgType + "&protocol=" + protocol)
	log.Info("PORT CALLS URL: " + URL)

	// call marine traffic api with `URL`
	response, respErr := http.Get(URL)
	if respErr != nil {
		log.Error("Failed to contact Marine Traffic Port Call API")
		return respErr
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		// if marine traffic returns error with status 429 (too many requests)
		if response.StatusCode == fiber.StatusTooManyRequests {
			// TODO: if data exists in cache, return it

			// else
			// TODO: tell the caller that how much time is remaining until they can
			// call the api again
			return c.Status(response.StatusCode).SendString("Too Many Requests")
		}

		// other error might be 401 (unauthorized)
		// for this there's not much we can do
		if response.StatusCode == fiber.StatusUnauthorized {
			return c.Status(response.StatusCode).SendString("Unauthorized")

		}

		// unexpected error case
		errBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error("Failed to read response body for port call", err)
			return fiber.NewError(response.StatusCode, "abcd")
			return err
		}
		return c.Status(response.StatusCode).SendString(string(errBody))
	}

	// here data exists
	csvData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Failed to read response body for port call", err)
		return err
	}

	c.Set("Content-Type", "text/csv")

	return c.Send(csvData)

	
}

func main() {
	myValidator := &XValidator{
		validator: validate,
	}
	app := fiber.New(fiber.Config{ // create the fiber app
		// global custom error handler
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusBadRequest).JSON(GlobalErrorHandlerResp{
				Success: false,
				Message: err.Error(),
			})
		},
	})
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// global middlewares
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "${pid} ${locals:requestid} ${status} - ${method} ${path}​\n",
	}))
	app.Use(helmet.New())
	// app.Use(limiter.New(limiter.Config{
	// 	Max:               2,
	// 	Expiration:        5 * 60 * time.Second,
	// 	LimiterMiddleware: limiter.SlidingWindow{},
	// }))

	app.Get("/", rootHandler)
	app.Post("/api/portcall", func(c *fiber.Ctx) error {
		return portCallHandler(c, myValidator)
	})

	app.Post("/api/berthcall", func(c *fiber.Ctx) error {
		return BerthCallHandler(c, myValidator)
	})

	port := os.Getenv("PORT")
	app.Listen(":" + port)
}
