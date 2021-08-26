package main

import (
	"gochart/src/scharts"
	"net/http"
)

func main() {

	http.HandleFunc("/chart", scharts.CMChart)
	http.ListenAndServe(":8082", nil)

}
