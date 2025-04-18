test:
	go test ./tests/...

coverage:
	go test -coverpkg=./ast,./command,./lexer,./parser,./printer,./style,./token ./tests/... -cover
