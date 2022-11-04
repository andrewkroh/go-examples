package main

import (
	"log"

	"layeh.com/radius"
	"layeh.com/radius/debug"
	"layeh.com/radius/rfc2865"
)

var (
	ServerUsername = "guest"
	ServerPassword = "bamboo"
)

func main() {
	handler := func(w radius.ResponseWriter, r *radius.Request) {
		log.Printf("received request: %v", debug.DumpRequestString(&debug.Config{Dictionary: debug.IncludedDictionary}, r))

		username := rfc2865.UserName_GetString(r.Packet)
		password := rfc2865.UserPassword_GetString(r.Packet)

		var code radius.Code
		if username == ServerUsername && password == ServerPassword {
			code = radius.CodeAccessAccept
		} else {
			code = radius.CodeAccessReject
		}
		log.Printf("Writing %v to %v", code, r.RemoteAddr)
		w.Write(r.Response(code))
	}

	server := radius.PacketServer{
		Handler:      radius.HandlerFunc(handler),
		SecretSource: radius.StaticSecretSource([]byte(`abc123`)),
	}

	log.Printf("Starting server on :1812")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
