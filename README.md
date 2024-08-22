## Geminipodcaster

Simple, but similar to one showed as prototype on Google I/O 2024.
Its a gemini podcast generator based on your context.

You can interrupt it and it will regenerate podcast based on your new intercation or just leave for host and guest to talk.


To run get an .env file with:
```
GEMINI_API_KEY=
GOOGLE_APPLICATION_CREDENTIALS=
```
To run the service, start a go gin server in ```/backend``` directory:

first get all required go modules
```
go mod tidy
```
then start the server
```
go run main.go
```

after that run the client in the ```/frontend``` directory ```npm run dev```.
