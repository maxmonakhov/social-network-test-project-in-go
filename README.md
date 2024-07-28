# Social-network test project

## App description


## How to run the app


## API enpoints



## Additional info
Application is a test project, not applicable for production use.

Many aspects are simplified, like
- storing session in memory
- usage of native http/net library with corresponding drawbacks as it's too basic
- not using proper file structure as I don't know yet which one is common in Go projects
- i've never worked with mongo before so was trying to make it SQL'ish by using separatins of entities to collections as to different SQL tables. This approach is probably wrong

**TLDR**: the code is really shitty because of with experience of writing Go project and using MongoDB.

## Things to improve
- [ ] Use onion architecture - separate application to layers:
  - [ ] Separate controllers, service and db access layers - introduce Service files and repositories
  - [ ] Use dependency inversion to pass data access objects
- [ ] Move all hardcoded configs to envirounment variables
- [ ] Introduce Redis for storing auth sessions
- [ ] Add framework to avoid mess with HTTP methods and URL path params parsing