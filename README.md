## Geminipodcaster

Similar to one showed as prototype on Google I/O 2024.
Its a gemini podcast generator based on your context.

You can interrupt it and it will regenerate podcast based on your new intercation or just leave for host and guest to talk.

Its a simple messy example with hard coded context and user interaction prompt. But it makes api calls to gemini api to generate and regenerate podcast.
 
To be done:
- client that will send requests with context and user interaction prompt
- Audio handling
- ....
- ....


To run get an .env file with:
```
GEMINI_API_KEY=
```

run to get all required go modules
```
go mod tidy
```
and then run gin server 
```
go run main.go
```

GET /start
- Starts the geminipodcaster

GET /interact 
- Will do the user interaction