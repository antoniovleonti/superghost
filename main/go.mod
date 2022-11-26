module main

go 1.17

replace sgserver => ../server

replace superghost => ../superghost

require (
	github.com/go-chi/chi/v5 v5.0.7 // indirect
	sgserver v0.0.0-00010101000000-000000000000 // indirect
	superghost v0.0.0-00010101000000-000000000000 // indirect
)
