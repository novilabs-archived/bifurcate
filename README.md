# bifurcate
This is a program that runs multiple configured programs and forwards the signals along. This could make it easier to run something like consul-template and your configured program in the same docker container. It also forwards all of the stdout/stderr from the child programs

* * *
### Why?
There are many people that believe that only 1 process should run in a container at a time. I think that this makes a ton of since, because you want to monitor that the container is running not that everything inside the container is running well. Where this starts to make a little less sense is for something like [gunicorn](http://gunicorn.org/) which does really well at managing all of its sub processes. I think that it makes a lot of sense to allow something like gunicorn to run inside a container. This got me thinking that as long as the underlying processes are being monitored then it is fine.

There are many alternatives such as [supervisord](http://supervisord.org/), [systemd](http://www.freedesktop.org/wiki/Software/systemd/), etc.. but I was looking for something very light weight, and that would not restart any monitored child process. I did not want anything inside the docker container to get restarted because usually there is some kind of orchestartion around the conatiner itself and this could interfere with that. 

`bifurcate` is attempting to pretend that it is just one process so kills everything if any process dies, and will forward all signals to the underlying processes.

Also phusion and yelp talk about the [PID 1 zombie reading problem](https://blog.phusion.nl/2015/01/20/docker-and-the-pid-1-zombie-reaping-problem/), [dumb-init](https://engineeringblog.yelp.com/2016/01/dumb-init-an-init-for-docker.html) which can occur in a docker container that does not have a proper init system. As long as bifurcate is PID 1 then it will also reap these defunct/zombie processes.

* * *
### Configuration
Configuration is done in [json](http://json.org/). This is easy to have structure, and is natively supported by go, so does not require adding any more dependencies.

Right now everything exists under the "programs" key which is just a map of name of program to arguments describing programs.
```json
{
 "programs":{}
}
```

To add a program to be executed just add it to that object. Such as `sleep 10`. 
###### `args`
```json
{
 "programs": {
   "sleeper": {
     "args": ["/bin/sleep", "10"]
    }
  }
}
```
The first element in the list of arguments is expected to be the executable, it is best for this to be a full path, but it does not have to be. All of the environmental variables `bifurcate` runs with are passed to all of the child processes.

Just keep adding more uniquely named programs to run as needed.

Sometimes you want programs to run in a specific order, maybe one is generating the configuration file for another. Having to specify order can sometimes lead to bugs that are difficult to find. It could be that the race only occurs in the wrong order in production. Instead of specifying order of execution, `bifurcate` allows you to specify a specific file to exist before running a program.
###### `requires`
```json
{
  "programs": {
    "consul-template": { 
      "args": ["consul-template", "-template", "/config.conf.ctmpl:/config.conf"] 
    },
    "myapp": {
      "requires": [{"file": "/config.conf"}],
      "args": ["myapp", "-config", "/config.conf"]
    }
  }
}
```
In this example we are running [`consul-template`](https://github.com/hashicorp/consul-template) to get and keep a configuration file up to date for our application. We do not want the application to start up until the configuration file exists so we tell the program that it `requires` a `file` at a specified location to exist. This way even once consul template starts up, `bifurcate` waits until the config file exists to start our application.

* * *
To install, grab your platform dependent version from the releases page. Current latest version
https://github.com/novilabs/bifurcate/releases/latest

example installation into docker
```
FROM alpine:3.3

ENV BIFURCATE_VERSION 0.4.0
WORKDIR /usr/bin
RUN set -x\
 && apk add --update --virtual .deps\
    curl\
    tar\
 && curl -L https://github.com/novilabs/bifurcate/releases/download/v${BIFURCATE_VERSION}/bifurcate_${BIFURCATE_VERSION}_linux_amd64.tar.gz\
    | gunzip - | tar -x\
 && apk del .deps\
 && rm -rf /var/cache/apk/*
```

* * *
### Running the example.
```
BABY_NAME=quinn go run bifurcate.go resources/demo.json
```
`touch` baby-sleep file in the working directory to put the baby to sleep, and it will wake up after a while `exit 1` and kill the entire program.
