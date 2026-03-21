package app

import "net/http"

func registerUserRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /users", apiDeps.user.GetUsers)
	api.HandleFunc("POST /users", apiDeps.user.CreateUser)
	api.HandleFunc("PUT /users/{id}", apiDeps.user.UpdateUser)
	api.HandleFunc("DELETE /users/{id}", apiDeps.user.DeleteUser)
	api.HandleFunc("PUT /users/password", apiDeps.user.ChangePassword)
}
