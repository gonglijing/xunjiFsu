package handlers

import (
	"log"
	"net/http"
)

func writeServerErrorWithLog(w http.ResponseWriter, def APIErrorDef, err error) {
	if err != nil {
		if def.Code != "" {
			log.Printf("%s: %v", def.Code, err)
		} else {
			log.Printf("%s: %v", def.Message, err)
		}
	}
	WriteServerErrorDef(w, def)
}
