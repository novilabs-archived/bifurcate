# bifurcate
This is a program that runs multiple configured programs and forwards the signals along. This could make it easier to run say consul-template and your configured program in the same docker container. It also forwards all of the stdout/stderr from the child programs

Running the example
```
BABY_NAME=quinn go run bifurcate.go resources/demo.json
```
