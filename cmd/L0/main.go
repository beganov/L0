package main

import "github.com/beganov/L0/internal/api"

func main() {
	router := api.SetupRouter()
	api.RouteRegister(router)
	if err := router.Run("localhost:8081"); err != nil {
		//	logger.Fatal().Err(err).Msg("Failed to run server")
	}

}
