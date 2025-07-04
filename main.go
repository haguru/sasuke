package main

import (
	"github.com/haguru/sasuke/config"
	"github.com/haguru/sasuke/internal/app"
)

func main() {

	// create and initialize the app
	app, err := app.NewApp(config.CONFIG_PATH)
	if err != nil {
		panic(err) // handle error appropriately in production code
	}

	// run the app
	// This will start the server and handle routes as defined in the app package.
	// The Run method can be used to start the server or perform other tasks.
	// In this case, it will start the server and listen for incoming requests.
	// If there are any errors during the server startup, they will be handled appropriately.
	err = app.Run()
	if err != nil {
		panic(err) // handle error appropriately in production code
	}
}
